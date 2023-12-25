package server

import (
	"errors"
	"math"
	"strconv"
	"strings"
)

type HandlerPrefix string

// Used for handling server commands
type Handler interface {
	handle(m *Message) error
}

// Calls when client wants to create new room.
type createRoomHandler struct {
	server *Server
}

// Creates a new server command handler that calls when client wants to create new room.
func NewCreateRoomHandler(s *Server) *createRoomHandler {
	h := &createRoomHandler{
		server: s,
	}

	return h
}

func (h *createRoomHandler) handle(m *Message) error {
	room_capacity := math.MaxInt64
	mess_parts := strings.Split(m.text, " ")
	if len(mess_parts) > 2 {
		return errors.New("Invalid command format")
	}

	if len(mess_parts) > 1 {
		capacity_str := mess_parts[1]
		capacity_param, err := strconv.Atoi(capacity_str)
		if err != nil {
			return errors.New("Invalid command format")
		}
		if capacity_param > 0 {
			room_capacity = capacity_param
		}
	}

	creator := m.sender
	h.server.CreateRoom(creator, room_capacity)

	return nil
}

// Calls when client wants to join to existing room by RoomID.
type joinRoomHandler struct {
	server *Server
}

// Creates a new server command handler that calls when client wants to join to existing room by RoomID.
func NewJoinRoomHandler(s *Server) *joinRoomHandler {
	h := &joinRoomHandler{
		server: s,
	}

	return h
}

func (h *joinRoomHandler) handle(m *Message) error {
	client := m.sender
	args := strings.Split(m.text, " ")[1:]

	if len(args) < 1 {
		return errors.New("Invalid command format")
	}

	roomID := strings.TrimSuffix(args[0], "\n")
	h.server.JoinRoom(client, RoomID(roomID))

	return nil
}

// Calls when client wants to leave the room.
type leaveRoomHandler struct {
	server *Server
}

// Creates a new server command handler that calls when client wants to leave the room.
func NewLeaveRoomHandler(s *Server) *leaveRoomHandler {
	h := &leaveRoomHandler{
		server: s,
	}

	return h
}

func (h *leaveRoomHandler) handle(m *Message) error {
	h.server.LeaveRoom(m.sender)

	return nil
}
