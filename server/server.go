package server

import (
	"strings"
	"sync"

	"github.com/codecat/go-enet"

	"github.com/please-the-turtle/enet-room-server/logging"
)

const (
	ERROR_NOTICE_PREFIX = "ERR: "

	SERVER_FULL_NOTICE            = ERROR_NOTICE_PREFIX + "The server is full\n"
	CREATING_ROOM_FAILED_NOTICE   = ERROR_NOTICE_PREFIX + "Creating room failed\n"
	CLIENT_ALREADY_IN_ROOM_NOTICE = ERROR_NOTICE_PREFIX + "Client already in room\n"
	ROOM_NOT_EXISTS_NOTICE        = ERROR_NOTICE_PREFIX + "Room not exists\n"
	NOT_A_COMMAND_NOTICE          = ERROR_NOTICE_PREFIX + "Command not exists\n"
)

var wg sync.WaitGroup

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
	rooms    map[RoomID]*room
	clients  map[enet.Peer]*client
	handlers map[HandlerPrefix]Handler
	incoming chan *Message
}

// Creates new server.
func NewServer(config *ServerConfig) *Server {
	s := &Server{
		config:   config,
		rooms:    make(map[RoomID]*room),
		clients:  make(map[enet.Peer]*client),
		handlers: make(map[HandlerPrefix]Handler),
		incoming: make(chan *Message),
	}

	s.handlers[HandlerPrefix("ROOM")] = NewCreateRoomHandler(s)
	s.handlers[HandlerPrefix("JOIN")] = NewJoinRoomHandler(s)
	s.handlers[HandlerPrefix("LEAV")] = NewLeaveRoomHandler(s)

	return s
}

// Adds new client to the server.
func (s *Server) Join(c *client) {
	if len(s.clients) >= s.config.MaxClients {
		c.peer.SendString(SERVER_FULL_NOTICE, 0, enet.PacketFlagReliable)
		c.quit()
		return
	}

	s.clients[c.peer] = c
	go func() {
		for message := range c.incoming {
			s.incoming <- message
		}
		s.Disconnect(c)
	}()

	c.listen()
	c.outgoing <- NewMessage(c, string(c.id))

	logging.Info("New client joined on server")
}

// Disconnects client from server.
func (s *Server) Disconnect(c *client) {
	c.quit()
	s.LeaveRoom(c)
	delete(s.clients, c.peer)
}

func (s *Server) Listen() {
	logging.Info("Server started on port", s.config.Port)
	wg.Add(1)
	go s.listen()
	go func() {
		for {
			message := <-s.incoming
			s.parse(message)
		}
	}()
	wg.Wait()
}

// Parses the message for processing.
// If the message text begins with a command symbol,
// then it tries to select the appropriate handler in accordance with HandlerPrefix.
func (s *Server) parse(m *Message) {
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
		logging.Errorf("HandlePrefix '%s' incorrect.", handlerPrefix)
		return
	}

	err := handler.handle(m)
	if err != nil {
		m.sender.outgoing <- NewMessage(m.sender, ERROR_NOTICE_PREFIX+err.Error())
		logging.Error(err)
	}
}

// Sends the message to clients from the same room
func (s *Server) Send(m *Message) {
	if m.sender == nil {
		return
	}

	if m.sender.room == nil {
		logging.Error("Message not sent: Client not in the room")
		return
	}

	m.sender.room.broadcast(m)
}

func (s *Server) CreateRoom(c *client, roomCapacity int) {
	if c.room != nil {
		c.outgoing <- NewMessage(c, CLIENT_ALREADY_IN_ROOM_NOTICE)
		logging.Error("Creating a room: Client already in another room")
		return
	}

	room := NewRoom(roomCapacity)
	if s.rooms[room.id] != nil {
		c.outgoing <- NewMessage(c, CREATING_ROOM_FAILED_NOTICE)
		logging.Errorf("Creating room with id %s failed", room.id)
		return
	}

	s.rooms[room.id] = room
	logging.Infof("New room with id %s created", room.id)
	room.join(c)

	c.outgoing <- NewMessage(c, string(c.room.id))
}

func (s *Server) JoinRoom(c *client, roomID RoomID) {
	room, prs := s.rooms[roomID]

	if !prs {
		c.outgoing <- NewMessage(c, ROOM_NOT_EXISTS_NOTICE)
		logging.Errorf("Joining to the room: Room with id %s not exists", roomID)
		return
	}

	if c.room != nil {
		c.outgoing <- NewMessage(c, CLIENT_ALREADY_IN_ROOM_NOTICE)
		logging.Errorf("ERROR: Joining to the room: Client already in another room")
		return
	}

	err := room.join(c)
	if err != nil {
		c.outgoing <- NewMessage(c, err.Error())
		return
	}

	c.outgoing <- NewMessage(c, "JOINED")
	logging.Infof("Client %s joined to the room %s", c.id, room.id)
}

func (s *Server) LeaveRoom(c *client) {
	room := c.room

	if room == nil {
		return
	}

	room.leave(c)
	logging.Infof("Client with id %s left room %s", c.id, room.id)
	c.outgoing <- NewMessage(c, "LEFT")
	if len(room.members) == 0 {
		s.DeleteRoom(room)
	}
}

func (s *Server) DeleteRoom(r *room) {
	logging.Infof("Room with id %s deleted.", r.id)
	delete(s.rooms, r.id)
}

func (s *Server) listen() {
	defer wg.Done()

	enet.Initialize()
	defer enet.Deinitialize()

	host, err := enet.NewHost(enet.NewListenAddress(s.config.Port), uint64(s.config.MaxClients), 0, 0, 0)
	if err != nil {
		logging.Error("Couldn't create host: ", err.Error())
		return
	}
	defer host.Destroy()

	for {
		event := host.Service(0)

		switch event.GetType() {
		case enet.EventNone:
			continue

		case enet.EventConnect:
			client := NewClient(event.GetPeer())
			s.Join(client)

		case enet.EventDisconnect:
			client, prs := s.clients[event.GetPeer()]
			if !prs {
				logging.Warning("ENet EventDisconnect: client not present.")
				continue
			}
			s.Disconnect(client)

		case enet.EventReceive:
			client, prs := s.clients[event.GetPeer()]
			if !prs {
				logging.Warning("ENet EventRecieve: client not present.")
				continue
			}

			packet := event.GetPacket()
			data := packet.GetData()
			text := string(data)
			message := NewMessage(client, text)
			message.channel = event.GetChannelID()
			message.packetFlags = packet.GetFlags()
			s.incoming <- message

			packet.Destroy()
		}
	}
}
