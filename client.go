package main

import (
	"log"
	"math/rand"
	"time"

	"github.com/codecat/go-enet"
)

const (
	CLIENT_ID_ALPHABET = "AaBbCcDdEeFfGgHhIiJjKkLlMmNnOoPpQqRrSsTtUuVvWwXxYyZz0123456789"
	CLIENT_ID_LENGHT   = 16
)

// Unique client id.
type ClientID string

// Client implements the interaction of a separate client connected to the server.
type Client struct {
	id       ClientID
	room     *Room
	incoming chan *Message
	outgoing chan *Message
	peer     enet.Peer
}

// Creates new client.
func NewClient(peer enet.Peer) *Client {
	id := genClientID()

	client := &Client{
		id:       id,
		room:     nil,
		incoming: make(chan *Message),
		outgoing: make(chan *Message),
		peer:     peer,
	}

	return client
}

// Starts read and write data from client connection.
func (c *Client) Listen() {
	go c.writeLoop()
}

func (c *Client) Quit() {
	c.peer.Disconnect(0)
	log.Printf("The client %s has left the server", c.id)
}

// Writes data to the client connection.
func (c *Client) writeLoop() {
	for message := range c.outgoing {
		err := c.peer.SendString(message.text, message.channel, message.packetFlags)
		if err != nil {
			log.Println("Sending string:", err.Error())
		}
	}
}

// Generates new ClientID
func genClientID() ClientID {
	b := make([]byte, CLIENT_ID_LENGHT)
	rand.Seed(time.Now().UnixNano())
	for i := range b {
		b[i] = CLIENT_ID_ALPHABET[rand.Intn(len(CLIENT_ID_ALPHABET))]
	}

	return ClientID(b)
}
