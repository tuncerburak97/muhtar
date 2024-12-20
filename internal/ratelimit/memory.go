package ratelimit

import (
	"context"
	"sync"
	"time"
)

// MemoryStore implements Store interface using in-memory storage
type MemoryStore struct {
	mu    sync.RWMutex
	data  map[string]*window
	clean *time.Ticker
}

type window struct {
	count     int
	resetTime time.Time
}

// NewMemoryStore creates a new memory-based store
func NewMemoryStore(cleanupInterval time.Duration) *MemoryStore {
	store := &MemoryStore{
		data:  make(map[string]*window),
		clean: time.NewTicker(cleanupInterval),
	}

	go store.cleanup()
	return store
}

func (s *MemoryStore) cleanup() {
	for range s.clean.C {
		s.mu.Lock()
		now := time.Now()
		for key, w := range s.data {
			if now.After(w.resetTime) {
				delete(s.data, key)
			}
		}
		s.mu.Unlock()
	}
}

func (s *MemoryStore) Get(ctx context.Context, key string) (int, time.Time, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if w, exists := s.data[key]; exists {
		if time.Now().After(w.resetTime) {
			return 0, time.Now(), nil
		}
		return w.count, w.resetTime, nil
	}
	return 0, time.Now(), nil
}

func (s *MemoryStore) Increment(ctx context.Context, key string, resetTime time.Time) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	if w, exists := s.data[key]; exists {
		if now.After(w.resetTime) {
			w.count = 1
			w.resetTime = resetTime
		} else {
			w.count++
		}
		return w.count, nil
	}

	s.data[key] = &window{
		count:     1,
		resetTime: resetTime,
	}
	return 1, nil
}

func (s *MemoryStore) Reset(ctx context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, key)
	return nil
}

func (s *MemoryStore) Close() error {
	s.clean.Stop()
	s.mu.Lock()
	s.data = nil
	s.mu.Unlock()
	return nil
}
