package main

import (
	"time"
	"unsafe"
	"wgpu_server/ws"

	"github.com/EngoEngine/glm"
)

func Encode[T any](v []T) []byte {
	if len(v) == 0 {
		return []byte{}
	}
	return unsafe.Slice((*byte)(unsafe.Pointer(&v[0])), len(v)*int(unsafe.Sizeof(v[0])))
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
		gravity := glm.Vec3{0, -9.8, 0}
		gravity = gravity.Mul(1.0 / 30.0)
		for id, bullet := range ws.Bullets {
			step := bullet.Vel.Mul(1.0 / 30.0)
			bullet.Position = bullet.Position.Add(&step)
			bullet.Vel = bullet.Vel.Add(&gravity)
			if bullet.Position[1] < 0 {
				delete(ws.Bullets, id)
				ws.DestroyedBullets = append(ws.DestroyedBullets, id)
			} else {
				ws.Bullets[id] = bullet
			}
		}
		newBullets := make([]ws.Shoot, len(ws.NewBullets))
		i = 0
		for _, bullet := range ws.NewBullets {
			newBullets[i] = bullet
			i++
		}

		newBulletsBytes := Encode(ws.NewBullets)
		numNewBulletBytes := Encode([]int{len(newBullets)})
		newBulletsBytes = append(numNewBulletBytes, newBulletsBytes...)

		destroyBullets := Encode(ws.DestroyedBullets)
		numDestroyBulletBytes := Encode([]int{len(ws.DestroyedBullets)})
		destroyBullets = append(numDestroyBulletBytes, destroyBullets...)

		ws.NewBullets = make([]ws.Shoot, 0)
		ws.DestroyedBullets = make([]int, 0)

		server.Lock.Unlock()
		playersBytes := Encode(mPlayers)
		message := append([]byte{byte(numPlayers)}, playersBytes...)
		message = append(message, newBulletsBytes...)
		message = append(message, destroyBullets...)
		server.Broadcast(message)
	})
}

func messageHandler(server *ws.Server, id int, message []byte) {
	// fmt.Println(string(message))
	messageType := message[0]
	switch messageType {
	case 0:
		{
			d := (*ws.Player)(unsafe.Pointer(&message[1]))
			// fmt.Printf("Received player data: %v, %v, %v\n", d.Position, d.Rotation.W, d.Rotation.V)
			server.Lock.Lock()
			ws.Players[id] = *d
			server.Lock.Unlock()
			// m := ws.Message{Client: id, Data: *d}
			// server.WriteMessage(id, (*[unsafe.Sizeof(m)]byte)(unsafe.Pointer(&m))[:])
		}
	case 1:
		{
			d := (*ws.Shoot)(unsafe.Pointer(&message[1]))
			dir := d.Direction.Normalized()
			vel := dir.Mul(d.Speed)
			bullet := ws.Bullet{Position: d.Position, Vel: vel}
			server.Lock.Lock()
			bulletId := ws.BulletId
			ws.BulletId++
			ws.Bullets[bulletId] = bullet
			ws.NewBullets = append(ws.NewBullets, ws.Shoot{Position: d.Position, Direction: d.Direction, Speed: d.Speed, Id: bulletId})
			server.Lock.Unlock()
		}
	}
}
