package ws

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Accepting all requests
	},
}

type Server struct {
	clients       map[int]*websocket.Conn
	handleMessage func(server *Server, id int, message []byte) // New message handler
	idGen         int
	lock          sync.Mutex
}

func StartServer(handleMessage func(server *Server, id int, message []byte)) *Server {
	server := Server{
		make(map[int]*websocket.Conn),
		handleMessage,
		0,
		sync.Mutex{},
	}

	http.HandleFunc("/", server.echo)
	go http.ListenAndServe(":8080", nil)

	return &server
}
func (server *Server) Poll() {
	for {
		time.Sleep(1 * time.Second)
	}
}

func (server *Server) echo(w http.ResponseWriter, r *http.Request) {
	connection, _ := upgrader.Upgrade(w, r, nil)
	id := server.idGen
	server.idGen++
	server.clients[id] = connection // Save the connection using it as a key
	connection.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("%d", id)))
	// server.WriteMessage([]byte(fmt.Sprintf("create: %d", id)))

	for {
		mt, message, err := connection.ReadMessage()

		if err != nil || mt == websocket.CloseMessage {
			break // Exit the loop if the client tries to close the connection or the connection is interrupted
		}

		go server.handleMessage(server, id, message)
	}

	delete(server.clients, id) // Removing the connection

	connection.Close()
	// server.WriteMessage([]byte(fmt.Sprintf("destroy: %d", id)))
}

func (server *Server) WriteMessage(client int, message []byte) {
	server.lock.Lock()

	newMessage := append([]byte(fmt.Sprintf("%d, ", client)), message...)
	for _, conn := range server.clients {
		println(string(newMessage))
		err := conn.WriteMessage(websocket.TextMessage, newMessage)
		if err != nil {
			println("Error writing message")
		}
	}
	server.lock.Unlock()
}
