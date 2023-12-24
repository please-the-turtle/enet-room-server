package main

import (
	"log"
)

func main() {
	config := DefaultServerConfig()
	server := NewServer(config)
	log.Printf("The server is running on the port %d", server.config.Port)
	server.Listen()
}
