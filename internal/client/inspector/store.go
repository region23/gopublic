package inspector

import (
	"sync"
)

// Store defines the interface for storing HTTP exchanges.
type Store interface {
	// Add adds a new exchange and returns its ID.
	Add(exchange HTTPExchange) int64
	// Get retrieves an exchange by ID.
	Get(id int64) (*HTTPExchange, bool)
	// List returns all exchanges, newest first.
	List() []HTTPExchange
	// Clear removes all exchanges.
	Clear()
	// Count returns the number of stored exchanges.
	Count() int
}

// InMemoryStore implements Store with an in-memory ring buffer.
type InMemoryStore struct {
	mu        sync.RWMutex
	exchanges []HTTPExchange
	nextID    int64
	maxSize   int
}

// NewInMemoryStore creates a new in-memory store with the specified max size.
func NewInMemoryStore(maxSize int) *InMemoryStore {
	if maxSize <= 0 {
		maxSize = 100
	}
	return &InMemoryStore{
		exchanges: make([]HTTPExchange, 0, maxSize),
		maxSize:   maxSize,
	}
}

// Add adds a new exchange to the store (thread-safe).
// The exchange ID is set automatically. Returns the assigned ID.
func (s *InMemoryStore) Add(exchange HTTPExchange) int64 {
	s.mu.Lock()
	defer s.mu.Unlock()

	exchange.ID = s.nextID
	s.nextID++

	// Prepend to list (newest first)
	// Use efficient prepend by creating a new slice only when necessary
	if len(s.exchanges) >= s.maxSize {
		// Shift elements to make room, drop oldest
		copy(s.exchanges[1:], s.exchanges[:len(s.exchanges)-1])
		s.exchanges[0] = exchange
	} else {
		// Prepend by creating new slice with extra capacity
		newExchanges := make([]HTTPExchange, len(s.exchanges)+1, s.maxSize)
		newExchanges[0] = exchange
		copy(newExchanges[1:], s.exchanges)
		s.exchanges = newExchanges
	}

	return exchange.ID
}

// Get retrieves an exchange by ID (thread-safe).
func (s *InMemoryStore) Get(id int64) (*HTTPExchange, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for i := range s.exchanges {
		if s.exchanges[i].ID == id {
			// Return a copy to prevent mutation
			ex := s.exchanges[i]
			return &ex, true
		}
	}
	return nil, false
}

// List returns all exchanges (thread-safe).
// Returns a copy to prevent mutation of internal state.
func (s *InMemoryStore) List() []HTTPExchange {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]HTTPExchange, len(s.exchanges))
	copy(result, s.exchanges)
	return result
}

// Clear removes all exchanges (thread-safe).
func (s *InMemoryStore) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.exchanges = s.exchanges[:0]
	// Note: nextID is not reset to avoid ID collisions if old references exist
}

// Count returns the number of stored exchanges.
func (s *InMemoryStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.exchanges)
}
