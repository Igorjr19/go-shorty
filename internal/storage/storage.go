package storage

import (
	"errors"

	"github.com/Igorjr19/go-shorty/internal/entity"
)

var (
	ErrNotFound   = errors.New("link not found")
	ErrCodeExists = errors.New("link code already exists")
)

type Storage interface {
	Save(entity.Link) error
	Load(code string) (entity.Link, error)
}
