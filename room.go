package main

import (
	"errors"
	"math"
	"math/rand"
	"time"
)

const (
	ROOM_ID_ALPHABET = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	ROOM_ID_LENGHT   = 6

	NOTICE_TYPE = "tcp"

	CLIENT_JOINED_NOTICE = "JOINED"
	CLIENT_LEFT_NOTICE   = "LEFT"
)

// Unique room id.
type RoomID string

// Room is used to send messages between its members.
type Room struct {
	id       RoomID
	capacity int
	members  map[ClientID]*Client
}

// Creates new empty room.
func NewRoom(capacity int) *Room {
	if capacity < 1 {
		capacity = math.MaxInt64
	}
	id := genRoomID()
	return &Room{
		id:       id,
		members:  make(map[ClientID]*Client),
		capacity: capacity,
	}
}

// Adds new member to the room.
func (r *Room) Join(c *Client) error {
	if len(r.members) >= r.capacity {
		return errors.New("ERR: Room is full")
	}

	c.room = r
	r.members[c.id] = c
	joinedMessage := NewMessage(c, CLIENT_JOINED_NOTICE)
	r.Broadcast(joinedMessage)

	return nil
}

// Removes member from room.
func (r *Room) Leave(c *Client) {
	m := NewMessage(c, CLIENT_LEFT_NOTICE)
	r.Broadcast(m)

	delete(r.members, c.id)

	c.room = nil
}

// Sends messages to all participants in the room except the sender.
func (r *Room) Broadcast(m *Message) {
	for _, member := range r.members {
		if member != m.sender {
			member.outgoing <- m
		}
	}
}

// Generates new RoomID
func genRoomID() RoomID {
	b := make([]byte, ROOM_ID_LENGHT)
	rand.Seed(time.Now().UnixNano())
	for i := range b {
		b[i] = ROOM_ID_ALPHABET[rand.Intn(len(ROOM_ID_ALPHABET))]
	}

	return RoomID(b)
}
