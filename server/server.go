package main

import (
	"fmt"
	"wgpu_server/ws"
)

func main() {
	server := ws.StartServer(messageHandler)

	for {
		server.WriteMessage([]byte("Hello"))
	}
}

func messageHandler(message []byte) {
	fmt.Println(string(message))
}
