package main

import (
	"reflect"
	. "shared"
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
	println("servers bullet inst type: ", reflect.TypeOf(Inst[Bullet]{}).String())
	server := ws.StartServer(messageHandler)
	server.Poll(time.Second/30.0, func() {
		server.Lock.Lock()
		numPlayers := ws.Players.Len()
		if numPlayers == 0 {
			server.Lock.Unlock()
			return
		}
		mPlayers := make([]Upd[Player], numPlayers)
		i := 0
		for idx, player := range ws.Players.Data {
			if !ws.Players.Valid[idx] {
				continue
			}
			mPlayers[i] = Upd[Player]{Id: idx, V: player}
			i++
		}

		gravity := glm.Vec3{0, -9.8, 0}
		gravity = gravity.Mul(1.0 / 30.0)
		for id, bullet := range ws.Bullets.Data {
			if !ws.Bullets.Valid[id] {
				continue
			}
			step := bullet.Vel.Mul(1.0 / 30.0)
			bullet.Position = bullet.Position.Add(&step)
			bullet.Vel = bullet.Vel.Add(&gravity)
			if bullet.Position[1] < 0 {
				ws.Bullets.Remove(id)
				// delete(ws.Bullets, id)
				ws.DestroyedBullets = append(ws.DestroyedBullets, Deinst[Bullet]{Id: id})
			} else {
				ws.Bullets.Data[id] = bullet
			}
		}
		server.Lock.Unlock()

		playersBytes := EncodeSubmessage(mPlayers)
		newBulletsBytes := EncodeSubmessage(ws.NewBullets)
		destroyBulletsBytes := EncodeSubmessage(ws.DestroyedBullets)

		ws.NewBullets = make([]Inst[Bullet], 0)
		ws.DestroyedBullets = make([]Deinst[Bullet], 0)

		// message := playersBytes
		message := append(playersBytes, newBulletsBytes...)
		message = append(message, destroyBulletsBytes...)
		server.Broadcast(message)
	})
}

func messageHandler(server *ws.Server, id int, message []byte) {
	// fmt.Println(string(message))
	messageType := message[0]
	switch messageType {
	case 0:
		{
			d := (*Player)(unsafe.Pointer(&message[1]))
			server.Lock.Lock()
			ws.Players.Data[d.Id] = *d
			server.Lock.Unlock()
		}
	case 1:
		{
			bullet := (*Bullet)(unsafe.Pointer(&message[1]))
			server.Lock.Lock()
			bulletId := ws.Bullets.Emplace(*bullet)
			ws.NewBullets = append(ws.NewBullets, Inst[Bullet]{Id: bulletId, V: *bullet})
			server.Lock.Unlock()
		}
	}
}
