package main

func main() {
	config := DefaultServerConfig()
	server := NewServer(config)
	server.Listen()
}
