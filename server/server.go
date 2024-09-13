package main

import (
	"time"
	"unsafe"
	"wgpu_server/ws"
)

func Encode[T any](v []T, len int) []byte {
	return unsafe.Slice((*byte)(unsafe.Pointer(&v[0])), len*int(unsafe.Sizeof(v[0])))
}
func main() {
	server := ws.StartServer(messageHandler)
	server.Poll(time.Second/30.0, func() {
		server.Lock.Lock()
		numPlayers := len(ws.Players)
		if numPlayers == 0 {
			server.Lock.Unlock()
			return
		}
		mPlayers := make([]ws.Message, numPlayers)
		i := 0
		for id, player := range ws.Players {
			mPlayers[i] = ws.Message{Client: id, Data: player}
			i++
		}
		server.Lock.Unlock()
		// for i := range mPlayers {
		// 	player := mPlayers[i]
		// 	fmt.Printf("Player %d: %v, %v\n", player.Client, player.Data.Position, player.Data.Rotation)
		// }
		// println()

		// println(mPlayers)
		// println(unsafe.Sizeof(Players))
		// PlayersBytes := (*[unsafe.Sizeof(mPlayers)]byte)(unsafe.Pointer(&mPlayers[0]))[:]
		// playersBytes := unsafe.Slice((*byte)(unsafe.Pointer(&mPlayers[0])), len(mPlayers)*int(unsafe.Sizeof(mPlayers[0])))
		playersBytes := Encode(mPlayers, len(mPlayers))
		message := append([]byte{byte(numPlayers)}, playersBytes...)
		server.WriteMessage(-1, message)

		// for id, player := range players {
		// 	// m := ws.Message{Client: id, Data: player}
		// 	server.WriteMessage(id, (*[unsafe.Sizeof(m)]byte)(unsafe.Pointer(&m))[:])
		// }
	})
	// server.WriteMessage([]byte("Hello"))
	// for {
	// 	server.WriteMessage([]byte("Hello"))
	// }
}

func messageHandler(server *ws.Server, id int, message []byte) {
	// fmt.Println(string(message))
	messageType := message[0]
	switch messageType {
	case 0:
		{
			d := (*ws.PlayerData)(unsafe.Pointer(&message[1]))
			// fmt.Printf("Received player data: %v, %v, %v\n", d.Position, d.Rotation.W, d.Rotation.V)
			server.Lock.Lock()
			ws.Players[id] = *d
			server.Lock.Unlock()
			// m := ws.Message{Client: id, Data: *d}
			// server.WriteMessage(id, (*[unsafe.Sizeof(m)]byte)(unsafe.Pointer(&m))[:])
		}
	}
}
