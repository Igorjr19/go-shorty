package storage

import (
	"context"
	"errors"

	"github.com/Igorjr19/go-shorty/internal/entity"
)

var (
	ErrNotFound   = errors.New("link not found")
	ErrCodeExists = errors.New("link code already exists")
)

type Storage interface {
	Save(ctx context.Context, link entity.Link) error
	Load(ctx context.Context, code string) (entity.Link, error)
	Visit(ctx context.Context, code string) (entity.Link, error)
}
