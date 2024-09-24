package shared

import "reflect"

type ServerClientMap []uint32

func (s *ServerClientMap) get(id uint32) *uint32 {
	for id >= uint32(len(*s)) {
		*s = append(*s, 0)
	}
	return &(*s)[id]
}

// a struct to store changes to the storage to be broadcast by the server
type changes[T any] struct {
	Insts   []Inst[T]
	Deinsts []Deinst[T]
}

func NewChanges[T any]() *changes[T] {
	return &changes[T]{
		make([]Inst[T], 0),
		make([]Deinst[T], 0),
	}
}

type Storage[T any] struct {
	mapping   ServerClientMap
	changes   *changes[T]
	available []uint32 // Todo: use min heap
	Valid     []bool
	Data      []T
}

func NewStorage[T any](hasChanges bool) *Storage[T] {
	var ch *changes[T]
	if hasChanges {
		ch = NewChanges[T]()
	}
	return &Storage[T]{
		mapping:   make(ServerClientMap, 0),
		changes:   ch,
		available: make([]uint32, 0),
		Valid:     make([]bool, 0),
		Data:      make([]T, 0),
	}
}

func (s *Storage[T]) Emplace(v T) uint32 {
	var id uint32
	if len(s.available) > 0 {
		id = s.available[0]
		s.available = s.available[1:]
		s.Data[id] = v
		s.Valid[id] = true
	} else {
		id = uint32(len(s.Data))
		s.Data = append(s.Data, v)
		s.Valid = append(s.Valid, true)
	}
	if s.changes != nil {
		s.changes.Insts = append(s.changes.Insts, Inst[T]{Id: id, V: v})
	}
	return id
}

func (s *Storage[T]) Remove(id uint32) {
	s.available = append(s.available, id)
	s.Valid[id] = false
	if s.changes != nil {
		s.changes.Deinsts = append(s.changes.Deinsts, Deinst[T]{Id: id})
	}
}

func (s *Storage[T]) Len() int {
	return len(s.Data) - len(s.available)
}

type StorageBase interface {
	// Update()
	SyncInst(s *Submessage, b []byte) uintptr
	SyncDeinst(s *Submessage, b []byte) uintptr
	SyncUpd(s *Submessage, b []byte) uintptr
	GetId() uint32
	Encode() []byte
	EncodeUpdates() []byte
	EncodeInst() []byte
}

func (s *Storage[T]) GetId() uint32 {
	var a T
	return Hash(reflect.TypeOf(a).String())
}
func (s *Storage[T]) SyncInst(submessage *Submessage, b []byte) uintptr {
	a, offset := DecodeSubmessage[Inst[T]](b)
	for _, v := range a {
		id := s.Emplace(v.V)
		*s.mapping.get(v.Id) = id
	}
	return offset
}
func (s *Storage[T]) SyncDeinst(submessage *Submessage, b []byte) uintptr {
	a, offset := DecodeSubmessage[Deinst[T]](b)
	for _, v := range a {
		id := *s.mapping.get(v.Id)
		s.Remove(id)
	}
	return offset
}
func (s *Storage[T]) SyncUpd(submessage *Submessage, b []byte) uintptr {
	a, offset := DecodeSubmessage[Upd[T]](b)
	for _, v := range a {
		s.Data[*s.mapping.get(v.Id)] = v.V
	}
	return offset
}
func (s *Storage[T]) Encode() []byte {
	b := make([]byte, 0)
	if s.changes != nil {
		if len(s.changes.Insts) > 0 {
			b = append(b, EncodeSubmessage(s.changes.Insts)...)
			s.changes.Insts = make([]Inst[T], 0)
		}
		if len(s.changes.Deinsts) > 0 {
			b = append(b, EncodeSubmessage(s.changes.Deinsts)...)
			s.changes.Deinsts = make([]Deinst[T], 0)
		}
	}
	return b
}
func (s *Storage[T]) EncodeUpdates() []byte {
	msg := make([]Upd[T], 0)
	for i, v := range s.Data {
		if s.Valid[i] {
			msg = append(msg, Upd[T]{Id: uint32(i), V: v})
		}
	}
	return EncodeSubmessage(msg)
}
func (s *Storage[T]) EncodeInst() []byte {
	msg := make([]Inst[T], 0)
	for i, v := range s.Data {
		if s.Valid[i] {
			msg = append(msg, Inst[T]{Id: uint32(i), V: v})
		}
	}
	return EncodeSubmessage(msg)
}

// func (s *Storage[T]) Update() {
// 	for i, v := range s.Data {
// 		if s.Valid[i] {
// 			s.update(&v, uint32(i))
// 		}
// 	}
// }

type ECS struct {
	Entities map[uint32]*StorageBase
}

func NewECS() *ECS {
	return &ECS{
		make(map[uint32]*StorageBase),
	}
}
func Register[T any](e *ECS, a T, isServer bool) {
	s := NewStorage[T](isServer)
	e.add(s.GetId(), s)
}

// func (e *ECS) Update() {
// 	for _, s := range e.Entities {
// 		s.Update()
// 	}
// }
func (e *ECS) add(id uint32, s StorageBase) {
	e.Entities[id] = &s
}

func GetStorage[T any](e *ECS) *Storage[T] {
	var a T
	return (*e.Entities[Hash(reflect.TypeOf(a).String())]).(*Storage[T])
	// return e.Entities[Hash(reflect.TypeOf(a).String())].(*Storage[T])
}
