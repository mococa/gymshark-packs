package store

import (
	"sort"
	"sync"
)

// InMemoryStore is a thread-safe in-memory implementation of PackSizeStore.
type InMemoryStore struct {
	mu    sync.RWMutex
	sizes map[int]struct{}
}

// NewInMemoryStore creates a new in-memory store with the given default pack sizes.
func NewInMemoryStore(defaultSizes []int) *InMemoryStore {
	s := &InMemoryStore{sizes: make(map[int]struct{})}
	for _, size := range defaultSizes {
		s.sizes[size] = struct{}{}
	}
	return s
}

func (s *InMemoryStore) GetAll() ([]int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]int, 0, len(s.sizes))
	for size := range s.sizes {
		result = append(result, size)
	}
	sort.Ints(result)
	return result, nil
}

func (s *InMemoryStore) Add(size int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.sizes[size]; exists {
		return ErrPackSizeExists
	}
	s.sizes[size] = struct{}{}
	return nil
}

// Remove deletes a pack size. Returns ErrPackSizeNotFound if missing,
// ErrLastPackSize if it is the only remaining size.
func (s *InMemoryStore) Remove(size int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.sizes[size]; !exists {
		return ErrPackSizeNotFound
	}
	if len(s.sizes) <= 1 {
		return ErrLastPackSize
	}
	delete(s.sizes, size)
	return nil
}

// Exists reports whether size is present. Not part of PackSizeStore; used by tests.
func (s *InMemoryStore) Exists(size int) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.sizes[size]
	return ok
}
