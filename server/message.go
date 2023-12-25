package server

import (
	"github.com/codecat/go-enet"
)

type Message struct {
	text        string
	sender      *client
	channel     uint8
	packetFlags enet.PacketFlags
}

func NewMessage(sender *client, text string) *Message {
	return &Message{
		sender:      sender,
		text:        text,
		channel:     0,
		packetFlags: enet.PacketFlagReliable,
	}
}

func NewMessageUnreliable(sender *client, text string) *Message {
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
