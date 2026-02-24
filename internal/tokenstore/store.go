package tokenstore

import "sync"

type Value struct {
	WTS         string
	PartialVal  uint64
	Name        string
	Low         uint64
	Mid         uint64
	High        uint64
}

type Store struct {
	mu     sync.RWMutex
	values map[int32]Value
}

func New() *Store {
	return &Store{values: make(map[int32]Value)}
}

func (s *Store) Get(tokenID int32) (Value, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.values[tokenID]
	return v, ok
}

func (s *Store) Set(tokenID int32, v Value) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.values[tokenID] = v
}

func (s *Store) UpsertIfNewer(tokenID int32, next Value) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	cur, ok := s.values[tokenID]
	if !ok || isNewer(next.WTS, cur.WTS) {
		s.values[tokenID] = next
		return true
	}
	return false
}

func isNewer(left, right string) bool {
	if right == "" {
		return left != ""
	}
	if left == "" {
		return false
	}
	return left > right
}
