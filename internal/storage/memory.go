package storage

import (
	"context"
	"sync"

	"github.com/Igorjr19/go-shorty/internal/entity"
)

type MemoryStorage struct {
	data map[string]entity.Link
	mu   sync.RWMutex
}

func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		data: make(map[string]entity.Link),
	}
}

func (m *MemoryStorage) Save(_ context.Context, link entity.Link) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.data[link.Code]; exists {
		return ErrCodeExists
	}
	m.data[link.Code] = link
	return nil
}

func (m *MemoryStorage) Load(_ context.Context, code string) (entity.Link, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	link, exists := m.data[code]
	if !exists {
		return entity.Link{}, ErrNotFound
	}
	return link, nil
}
