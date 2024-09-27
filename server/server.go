package main

import (
	"shared"
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
	Register(ws.ECS, Player{}, true)
	Register(ws.ECS, Bullet{}, true)
	server := ws.StartServer(messageHandler)
	players := GetStorage[Player](ws.ECS)
	bullets := GetStorage[Bullet](ws.ECS)
	server.Poll(time.Second/30.0, func() {
		server.Lock.Lock()
		numPlayers := players.Len()
		if numPlayers == 0 {
			server.Lock.Unlock()
			return
		}

		gravity := glm.Vec3{0, -9.8, 0}
		gravity = gravity.Mul(1.0 / 30.0)
		for id, bullet := range bullets.Data {
			if !bullets.Valid[id] {
				continue
			}
			step := bullet.Vel.Mul(1.0 / 30.0)
			bullet.Position = bullet.Position.Add(&step)
			bullet.Vel = bullet.Vel.Add(&gravity)
			if bullet.Position[1] < -3 {
				bullets.Remove(uint32(id))
			} else {
				bullets.Data[id] = bullet
			}
		}
		message := []byte{}
		for _, stor := range ws.ECS.Entities {
			message = append(message, (*stor).Encode()...)
		}
		message = append(message, players.EncodeUpdates()...)
		server.Lock.Unlock()
		server.Broadcast(message)
	})
}

func messageHandler(server *ws.Server, id int, message []byte) {

	offset := uintptr(0)
	server.Lock.Lock()
	for offset < uintptr(len(message)) {
		t := (*shared.Submessage)(unsafe.Pointer(&message[offset]))
		i := uintptr(unsafe.Sizeof(shared.Submessage{}))
		if t.NumBytes > 0 {
			switch t.Op {
			case shared.OpUpdate:
				i = (*ws.ECS.Entities[t.T]).SyncUpdServer(t, message[offset:])
			case shared.OpInstantiate:
				i = (*ws.ECS.Entities[t.T]).SyncInstServer(t, message[offset:])
				// case shared.OpDeinstantiate:
				// 	i = (*ws.ECS.Entities[t.T]).SyncDeinst(t, message[offset:])
			}
		}
		offset += i
	}
	server.Lock.Unlock()

	// fmt.Println(string(message))
	// messageType := message[0]
	// switch messageType {
	// case 0:
	// 	{
	// 		d := (*Player)(unsafe.Pointer(&message[1]))
	// 		server.Lock.Lock()
	// 		players := GetStorage[Player](ws.ECS)
	// 		players.Data[d.Id] = *d
	// 		server.Lock.Unlock()
	// 	}
	// case 1:
	// 	{
	// 		bullet := (*Bullet)(unsafe.Pointer(&message[1]))
	// 		server.Lock.Lock()
	// 		bullets := GetStorage[Bullet](ws.ECS)
	// 		bullets.Emplace(*bullet)
	// 		// ws.NewBullets = append(ws.NewBullets, Inst[Bullet]{Id: bulletId, V: *bullet})
	// 		server.Lock.Unlock()
	// 	}
	// }
}
