package storage

import (
	"fmt"
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

func (m *MemoryStorage) Save(link entity.Link) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[link.Code] = link
	return nil
}

func (m *MemoryStorage) Load(code string) (entity.Link, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	link, exists := m.data[code]
	if !exists {
		return entity.Link{}, ErrNotFound
	}
	return link, nil
}

var ErrNotFound = fmt.Errorf("link not found")
