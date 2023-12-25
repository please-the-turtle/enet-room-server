package server

import (
	"errors"
	"math"
	"strconv"
	"strings"
)

type HandlerPrefix string

type Handler interface {
	handle(m *Message) error
}

// Calls when client wants to create new room.
type CreateRoomHandler struct {
	server *Server
}

func NewCreateRoomHandler(s *Server) *CreateRoomHandler {
	h := &CreateRoomHandler{
		server: s,
	}

	return h
}

func (h *CreateRoomHandler) handle(m *Message) error {
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
type JoinRoomHandler struct {
	server *Server
}

func NewJoinRoomHandler(s *Server) *JoinRoomHandler {
	h := &JoinRoomHandler{
		server: s,
	}

	return h
}

func (h *JoinRoomHandler) handle(m *Message) error {
	client := m.sender
	args := strings.Split(m.text, " ")[1:]

	if len(args) < 1 {
		return errors.New("Invalid command format")
	}

	roomID := strings.TrimSuffix(args[0], "\n")
	h.server.JoinRoom(client, RoomID(roomID))

	return nil
}

type LeaveRoomHandler struct {
	server *Server
}

func NewLeaveRoomHandler(s *Server) *LeaveRoomHandler {
	h := &LeaveRoomHandler{
		server: s,
	}

	return h
}

func (h *LeaveRoomHandler) handle(m *Message) error {
	h.server.LeaveRoom(m.sender)

	return nil
}
