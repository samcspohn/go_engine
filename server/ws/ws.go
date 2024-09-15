package ws

import (
	"fmt"
	"net/http"
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
	idGen         int
	Lock          sync.Mutex
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
func (server *Server) Poll(dur time.Duration, f func()) {
	for {
		time.Sleep(dur)
		f()
	}
}

var Players = make(map[int]Player)
var Bullets = make(map[int]Bullet)
var NewBullets = make([]Shoot, 0)
var DestroyedBullets = make([]int, 0)
var BulletId = 0

func (server *Server) echo(w http.ResponseWriter, r *http.Request) {
	connection, _ := upgrader.Upgrade(w, r, nil)
	id := server.idGen
	server.idGen++
	server.clients[id] = connection // Save the connection using it as a key
	connection.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("%d", id)))
	Players[id] = Player{glm.Vec3{0, 0, 0}, glm.Quat{W: 0, V: glm.Vec3{0, 0, 1}}}
	// server.WriteMessage([]byte(fmt.Sprintf("create: %d", id)))

	for {
		mt, message, err := connection.ReadMessage()

		if err != nil || mt == websocket.CloseMessage {
			break // Exit the loop if the client tries to close the connection or the connection is interrupted
		}

		go server.handleMessage(server, id, message)
	}
	delete(Players, id)
	delete(server.clients, id) // Removing the connection

	connection.Close()
	// server.WriteMessage([]byte(fmt.Sprintf("destroy: %d", id)))
}

type Player struct {
	Position glm.Vec3
	Rotation glm.Quat
}
type Bullet struct {
	Position glm.Vec3
	Vel      glm.Vec3
}
type Shoot struct {
	Position  glm.Vec3
	Direction glm.Vec3
	Speed     float32
	Id        int
}

type Message struct {
	Client int
	Data   Player
}

func (server *Server) Broadcast(message []byte) {
	server.Lock.Lock()

	// newMessage := append([]byte(fmt.Sprintf("%d, ", client)), message...)
	for _, conn := range server.clients {
		// println(string(newMessage))
		err := conn.WriteMessage(websocket.BinaryMessage, message)
		if err != nil {
			println("Error writing message")
		}
	}
	server.Lock.Unlock()
}
