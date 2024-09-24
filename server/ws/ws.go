package ws

import (
	"fmt"
	"net/http"
	"shared"
	"sync"
	"time"

	"github.com/EngoEngine/glm"
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
	// idGen         int
	Lock sync.Mutex
}

func StartServer(handleMessage func(server *Server, id int, message []byte)) *Server {
	server := Server{
		make(map[int]*websocket.Conn),
		handleMessage,
		// 0,
		sync.Mutex{},
	}

	http.HandleFunc("/", server.echo)
	go http.ListenAndServe(":8080", nil)

	return &server
}
func (server *Server) Poll(dur time.Duration, f func()) {
	for {
		time.Sleep(dur)
		f()
	}
}

var ECS = shared.NewECS()

func (server *Server) echo(w http.ResponseWriter, r *http.Request) {
	connection, _ := upgrader.Upgrade(w, r, nil)
	// id := server.idGen
	// server.idGen++
	server.Lock.Lock()

	players := shared.GetStorage[shared.Player](ECS)
	bullets := shared.GetStorage[shared.Bullet](ECS)
	msg := []byte{}
	msg = append(msg, players.EncodeInst()...) // Send all players to the new client except the new player

	player := shared.Player{Position: glm.Vec3{0, 0, 0}, Rotation: glm.Quat{W: 0, V: glm.Vec3{0, 0, 1}}}
	id := int(players.Emplace(player))
	server.clients[id] = connection // Save the connection using it as a key
	connection.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("%d", id)))

	players.Data[id].Id = id

	msg = append(msg, bullets.EncodeInst()...)
	server.Lock.Unlock()
	connection.WriteMessage(websocket.BinaryMessage, msg)

	for {
		mt, message, err := connection.ReadMessage()

		if err != nil || mt == websocket.CloseMessage {
			break // Exit the loop if the client tries to close the connection or the connection is interrupted
		}

		go server.handleMessage(server, id, message)
	}
	players.Remove(uint32(id)) // Removing the player from the server
	delete(server.clients, id) // Removing the connection

	connection.Close()
}

func (server *Server) Broadcast(message []byte) {
	server.Lock.Lock()
	for _, conn := range server.clients {
		err := conn.WriteMessage(websocket.BinaryMessage, message)
		if err != nil {
			println("Error writing message")
		}
	}
	server.Lock.Unlock()
}
