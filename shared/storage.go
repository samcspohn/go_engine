package shared

type Storage[T any] struct {
	available []int // Todo: use min heap
	Valid     []bool
	Data      []T
}

func NewStorage[T any]() *Storage[T] {
	return &Storage[T]{
		make([]int, 0),
		make([]bool, 0),
		make([]T, 0),
	}
}

func (s *Storage[T]) Emplace(v T) int {
	if len(s.available) > 0 {
		id := s.available[0]
		s.available = s.available[1:]
		s.Data[id] = v
		s.Valid[id] = true
		return id
	} else {
		id := len(s.Data)
		s.Data = append(s.Data, v)
		s.Valid = append(s.Valid, true)
		return id
	}
}

func (s *Storage[T]) Remove(id int) {
	s.available = append(s.available, id)
	s.Valid[id] = false
}

func (s *Storage[T]) Len() int {
	return len(s.Data) - len(s.available)
}
