package shared

import (
	"bytes"
	"encoding/gob"
	"hash/fnv"
	"reflect"
	"unsafe"

	"github.com/EngoEngine/glm"
)

type Player struct {
	Position glm.Vec3
	Rotation glm.Quat
	Id       int
}
type Bullet struct {
	Position glm.Vec3
	Vel      glm.Vec3
}

// type Shoot struct {
// 	Position  glm.Vec3
// 	Direction glm.Vec3
// 	Speed     float32
// 	Id        int
// }

type Op interface {
	GetId() uint32
	GetOp() int
}

// instantiate
type Inst[T any] struct {
	Id uint32
	V  T
}

func (i Inst[T]) GetId() uint32 {
	var a T
	return Hash(reflect.TypeOf(a).String())
}
func (i Inst[T]) GetOp() int {
	return OpInstantiate
}

type InstReq[T any] struct {
	V T
}
type UpdReq[T any] struct {
	Id uint32
	V  T
}

// deinstantiate
type Deinst[T any] struct {
	Id uint32
}

func (i Deinst[T]) GetId() uint32 {
	var a T
	return Hash(reflect.TypeOf(a).String())
}
func (i Deinst[T]) GetOp() int {
	return OpDeinstantiate
}

// update
type Upd[T any] struct {
	Id uint32
	V  T
}

func (i Upd[T]) GetId() uint32 {
	var a T
	return Hash(reflect.TypeOf(a).String())
}
func (i Upd[T]) GetOp() int {
	return OpUpdate
}

const (
	OpInstantiate   = iota
	OpUpdate        = iota
	OpDeinstantiate = iota
)

type Submessage struct {
	T        uint32
	Op       int
	NumBytes uintptr
}

func Hash(s string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(s))
	return h.Sum32()
}

func EncodeSubmessage[T Op](v []T) []byte { // todo: use gob on v part
	var vbytes []byte
	if len(v) == 0 {
		vbytes = []byte{}
	} else {

		b := bytes.Buffer{}
		enc := gob.NewEncoder(&b)
		err := enc.Encode(v)
		if err != nil {
			panic(err)
		}
		vbytes = b.Bytes()
	}
	var a T
	s := Submessage{a.GetId(), a.GetOp(), uintptr(len(vbytes))}
	return append(unsafe.Slice((*byte)(unsafe.Pointer(&s)), int(unsafe.Sizeof(s))), vbytes...)
}
func DecodeSubmessage[T Op](b []byte) ([]T, uintptr) {
	s := *(*Submessage)(unsafe.Pointer(&b[0]))
	if s.NumBytes == 0 {
		return []T{}, unsafe.Sizeof(s)
	} else {
		sSize := unsafe.Sizeof(s)
		b = b[sSize : sSize+s.NumBytes]
		var v []T
		gob.NewDecoder(bytes.NewReader(b)).Decode(&v)
		// var t T
		// v := unsafe.Slice((*T)(unsafe.Pointer(&b[int(unsafe.Sizeof(s))])), s.NumBytes/int(unsafe.Sizeof(t)))
		return v, unsafe.Sizeof(s) + s.NumBytes
	}
}

// func Decode[T any](s *Submessage) []T {
// 	if s.Num == 0 {
// 		return []T{}
// 	}
// 	return unsafe.Slice((*T)(unsafe.Pointer(&s.V[0])), s.Num)
// }
