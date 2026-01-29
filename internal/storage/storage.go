package storage

import "github.com/Igorjr19/go-shorty/internal/entity"

type Storage interface {
	Save(entity.Link) error
	Load(code string) (entity.Link, error)
}
