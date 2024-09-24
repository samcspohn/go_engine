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

var Players = shared.NewStorage[shared.Player]()
var Bullets = shared.NewStorage[shared.Bullet]()
var NewBullets = make([]shared.Inst[shared.Bullet], 0)
var DestroyedBullets = make([]shared.Deinst[shared.Bullet], 0)
var NewPlayers = make([]shared.Inst[shared.Player], 0)
var DestroyedPlayers = make([]int, 0)

func (server *Server) echo(w http.ResponseWriter, r *http.Request) {
	connection, _ := upgrader.Upgrade(w, r, nil)
	// id := server.idGen
	// server.idGen++
	player := shared.Player{Position: glm.Vec3{0, 0, 0}, Rotation: glm.Quat{W: 0, V: glm.Vec3{0, 0, 1}}}
	id := Players.Emplace(player)
	server.clients[id] = connection // Save the connection using it as a key
	connection.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("%d", id)))

	Players.Data[id].Id = id

	instBullets := make([]shared.Inst[shared.Bullet], 0)
	for i, bullet := range Bullets.Data {
		if !Bullets.Valid[i] {
			continue
		}
		instBullets = append(instBullets, shared.Inst[shared.Bullet]{Id: i, V: bullet})
	}
	instPlayers := make([]shared.Inst[shared.Player], 0)
	for i, player := range Players.Data {
		if !Players.Valid[i] {
			continue
		}
		instPlayers = append(instPlayers, shared.Inst[shared.Player]{Id: i, V: player})
	}
	bytes := shared.EncodeSubmessage(instPlayers)
	bytes = append(bytes, shared.EncodeSubmessage(instBullets)...)
	connection.WriteMessage(websocket.BinaryMessage, bytes)

	for {
		mt, message, err := connection.ReadMessage()

		if err != nil || mt == websocket.CloseMessage {
			break // Exit the loop if the client tries to close the connection or the connection is interrupted
		}

		go server.handleMessage(server, id, message)
	}
	// delete(Players, id)
	Players.Remove(id)
	bytes = shared.EncodeSubmessage([]shared.Deinst[shared.Player]{{Id: id}})
	server.Broadcast(bytes)
	// DestroyedPlayers = append(DestroyedPlayers, _id)
	delete(server.clients, id) // Removing the connection

	connection.Close()
	// server.WriteMessage([]byte(fmt.Sprintf("destroy: %d", id)))
}

// type Message struct {
// 	Client int
// 	Data   shared.Player
// }

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
