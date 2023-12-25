package server

import (
	"github.com/codecat/go-enet"
)

type Message struct {
	text        string
	sender      *Client
	channel     uint8
	packetFlags enet.PacketFlags
}

func NewMessage(sender *Client, text string) *Message {
	return &Message{
		sender:      sender,
		text:        text,
		channel:     0,
		packetFlags: enet.PacketFlagReliable,
	}
}

func NewMessageUnreliable(sender *Client, text string) *Message {
	return &Message{
		sender:      sender,
		text:        text,
		channel:     0,
		packetFlags: enet.PacketFlagUnsequenced,
	}
}

func (m *Message) String() string {
	return m.text
}
