package main

import (
	"wgpu_server/ws"
)

func main() {
	server := ws.StartServer(messageHandler)
	server.Poll()
	// server.WriteMessage([]byte("Hello"))
	// for {
	// 	server.WriteMessage([]byte("Hello"))
	// }
}

func messageHandler(server *ws.Server, id int, message []byte) {
	// fmt.Println(string(message))
	server.WriteMessage(id, message)
}
