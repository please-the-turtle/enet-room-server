package main

import (
	"strings"
	"sync"

	"github.com/codecat/go-enet"
)

const (
	UDP_BUFFER_SIZE_BYTES = 256
	TCP_BUFFER_SIZE_BYTES = 256

	ERROR_NOTICE_PREFIX = "ERR: "

	SERVER_FULL_NOTICE            = ERROR_NOTICE_PREFIX + "The server is full\n"
	CREATING_ROOM_FAILED_NOTICE   = ERROR_NOTICE_PREFIX + "Creating room failed\n"
	CLIENT_ALREADY_IN_ROOM_NOTICE = ERROR_NOTICE_PREFIX + "Client already in room\n"
	ROOM_NOT_EXISTS_NOTICE        = ERROR_NOTICE_PREFIX + "Room not exists\n"
	NOT_A_COMMAND_NOTICE          = ERROR_NOTICE_PREFIX + "Command not exists\n"
)

var (
	wg      sync.WaitGroup
	loggers *Loggers
)

func init() {
	loggers = NewLoggers()
}

// Server configuration data.
// MaxClients store the maximum number of clients served.
// If maxClients lesser than 1, then number of clients isn't limited.
type ServerConfig struct {
	MaxClients    int
	Port          uint16
	CommandPrefix string
}

// Provides default server settings.
func DefaultServerConfig() *ServerConfig {
	return &ServerConfig{
		MaxClients:    100,
		Port:          8095,
		CommandPrefix: ":",
	}
}

// Serves ENet connections and controls sending messages
// between clients inside one room.
type Server struct {
	config   *ServerConfig
	rooms    map[RoomID]*Room
	clients  map[enet.Peer]*Client
	handlers map[HandlerPrefix]Handler
	incoming chan *Message
}

// Creates new server.
func NewServer(config *ServerConfig) *Server {
	s := &Server{
		config:   config,
		rooms:    make(map[RoomID]*Room),
		clients:  make(map[enet.Peer]*Client),
		handlers: make(map[HandlerPrefix]Handler),
		incoming: make(chan *Message),
	}

	s.handlers[HandlerPrefix("ROOM")] = NewCreateRoomHandler(s)
	s.handlers[HandlerPrefix("JOIN")] = NewJoinRoomHandler(s)
	s.handlers[HandlerPrefix("LEAV")] = NewLeaveRoomHandler(s)

	return s
}

// Adds new client to the server.
func (s *Server) Join(c *Client) {
	if len(s.clients) >= s.config.MaxClients {
		c.peer.SendString(SERVER_FULL_NOTICE, 0, enet.PacketFlagReliable)
		c.Quit()
		return
	}

	s.clients[c.peer] = c
	go func() {
		for message := range c.incoming {
			s.incoming <- message
		}
		s.Disconnect(c)
	}()

	c.Listen()
	c.outgoing <- NewMessage(c, string(c.id)+"\n")

	loggers.Info("New client joined on server")
}

// Disconnects client from server.
func (s *Server) Disconnect(c *Client) {
	c.Quit()
	s.LeaveRoom(c)
	delete(s.clients, c.peer)
}

func (s *Server) Listen() {
	loggers.Info("Server started on port", s.config.Port)
	wg.Add(1)
	go s.listen()
	go func() {
		for {
			message := <-s.incoming
			s.Parse(message)
		}
	}()
	wg.Wait()
}

// Parses the message for processing.
// If the message text begins with a command symbol,
// then it tries to select the appropriate handler in accordance with HandlerPrefix.
func (s *Server) Parse(m *Message) {
	if !strings.HasPrefix(m.text, s.config.CommandPrefix) {
		s.Send(m)
		return
	}

	handlerPrefix := m.text[len(s.config.CommandPrefix):]
	handlerPrefix, _, _ = strings.Cut(handlerPrefix, " ")
	handlerPrefix = strings.TrimSuffix(handlerPrefix, "\n")
	handler, prs := s.handlers[HandlerPrefix(handlerPrefix)]

	if !prs {
		m.sender.outgoing <- NewMessage(m.sender, NOT_A_COMMAND_NOTICE)
		loggers.Errorf("HandlePrefix '%s' incorrect.", handlerPrefix)
		return
	}

	err := handler.handle(m)
	if err != nil {
		m.sender.outgoing <- NewMessage(m.sender, ERROR_NOTICE_PREFIX+err.Error())
		loggers.Error(err)
	}
}

// Sends the message to clients from the same room
func (s *Server) Send(m *Message) {
	if m.sender == nil {
		return
	}

	if m.sender.room == nil {
		loggers.Error("Message not sent: Client not in the room")
		return
	}

	m.sender.room.Broadcast(m)
}

func (s *Server) CreateRoom(c *Client, roomCapacity int) {
	if c.room != nil {
		c.outgoing <- NewMessage(c, CLIENT_ALREADY_IN_ROOM_NOTICE)
		loggers.Error("Creating a room: Client already in another room")
		return
	}

	room := NewRoom(roomCapacity)
	if s.rooms[room.id] != nil {
		c.outgoing <- NewMessage(c, CREATING_ROOM_FAILED_NOTICE)
		loggers.Errorf("Creating room with id %s failed", room.id)
		return
	}

	s.rooms[room.id] = room
	loggers.Infof("New room with id %s created", room.id)
	room.Join(c)

	c.outgoing <- NewMessage(c, string(c.room.id)+"\n")
}

func (s *Server) JoinRoom(c *Client, roomID RoomID) {
	room, prs := s.rooms[roomID]

	if !prs {
		c.outgoing <- NewMessage(c, ROOM_NOT_EXISTS_NOTICE)
		loggers.Errorf("Joining to the room: Room with id %s not exists\n", roomID)
		return
	}

	if c.room != nil {
		c.outgoing <- NewMessage(c, CLIENT_ALREADY_IN_ROOM_NOTICE)
		loggers.Errorf("ERROR: Joining to the room: Client already in another room")
		return
	}

	err := room.Join(c)
	if err != nil {
		c.outgoing <- NewMessage(c, err.Error())
		return
	}

	c.outgoing <- NewMessage(c, "JOINED\n")
	loggers.Infof("Client %s joined to the room %s\n", c.id, room.id)
}

func (s *Server) LeaveRoom(c *Client) {
	room := c.room

	if room == nil {
		return
	}

	room.Leave(c)
	loggers.Infof("Client with id %s left room %s", c.id, room.id)
	c.outgoing <- NewMessage(c, "LEFT\n")
	if len(room.members) == 0 {
		s.DeleteRoom(room)
	}
}

func (s *Server) DeleteRoom(r *Room) {
	loggers.Infof("Room with id %s deleted.", r.id)
	delete(s.rooms, r.id)
}

func (s *Server) listen() {
	defer wg.Done()

	enet.Initialize()
	defer enet.Deinitialize()

	host, err := enet.NewHost(enet.NewListenAddress(s.config.Port), uint64(s.config.MaxClients), 0, 0, 0)
	if err != nil {
		loggers.Error("Couldn't create host: ", err.Error())
		return
	}
	defer host.Destroy()

	for {
		event := host.Service(1000)

		switch event.GetType() {
		case enet.EventNone:
			continue

		case enet.EventConnect:
			client := NewClient(event.GetPeer())
			s.Join(client)

		case enet.EventDisconnect:
			client, prs := s.clients[event.GetPeer()]
			if !prs {
				loggers.Warning("ENet EventDisconnect: client not present.")
				continue
			}
			s.Disconnect(client)

		case enet.EventReceive:
			client, prs := s.clients[event.GetPeer()]
			if !prs {
				loggers.Warning("ENet EventRecieve: client not present.")
				continue
			}
			packet := event.GetPacket()
			data := packet.GetData()
			defer packet.Destroy()
			defer func() {
				loggers.Info("DEFERRED CALL INSIDE SWITCH")
			}()
			text := string(data)

			message := NewMessage(client, text)
			message.channel = event.GetChannelID()
			message.packetFlags = packet.GetFlags()
			s.incoming <- message
		}
	}
}
