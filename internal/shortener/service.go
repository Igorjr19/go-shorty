package shortener

import (
	"math/rand"
	"time"

	"github.com/Igorjr19/go-shorty/internal/entity"
	"github.com/Igorjr19/go-shorty/internal/storage"
)

type Service struct {
	storage storage.Storage
}

func NewService(storage storage.Storage) *Service {
	return &Service{
		storage: storage,
	}
}

func (s *Service) Shorten(url string) (string, error) {
	const codeLength = 6
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	codeBytes := make([]byte, codeLength)
	for i := range codeBytes {
		codeBytes[i] = letters[rand.Intn(len(letters))]
	}
	code := string(codeBytes)

	link := entity.Link{
		Code:        code,
		OriginalURL: url,
		CreatedAt:   time.Now(),
	}

	if err := s.storage.Save(link); err != nil {
		return "", err
	}

	return code, nil
}

func (s *Service) Resolve(code string) (string, error) {
	link, err := s.storage.Load(code)

	if err != nil {
		return "", err
	}

	return link.OriginalURL, nil
}
