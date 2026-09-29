package shortener

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/Igorjr19/go-shorty/internal/entity"
	"github.com/Igorjr19/go-shorty/internal/storage"
)

const (
	codeLength     = 6
	codeAlphabet   = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	maxSaveRetries = 5
)

var ErrCodeGenerationFailed = errors.New("could not generate a unique code")

type Service struct {
	storage storage.Storage
}

func NewService(storage storage.Storage) *Service {
	return &Service{
		storage: storage,
	}
}

func (s *Service) Shorten(url string) (string, error) {
	for range maxSaveRetries {
		code, err := generateCode()
		if err != nil {
			return "", err
		}

		link := entity.Link{
			Code:        code,
			OriginalURL: url,
			CreatedAt:   time.Now(),
		}

		err = s.storage.Save(link)
		if errors.Is(err, storage.ErrCodeExists) {
			continue
		}
		if err != nil {
			return "", err
		}

		return code, nil
	}

	return "", ErrCodeGenerationFailed
}

func (s *Service) Resolve(code string) (string, error) {
	link, err := s.storage.Load(code)

	if err != nil {
		return "", err
	}

	return link.OriginalURL, nil
}

func generateCode() (string, error) {
	alphabetSize := big.NewInt(int64(len(codeAlphabet)))

	codeBytes := make([]byte, codeLength)
	for i := range codeBytes {
		n, err := rand.Int(rand.Reader, alphabetSize)
		if err != nil {
			return "", fmt.Errorf("failed to generate random code: %w", err)
		}
		codeBytes[i] = codeAlphabet[n.Int64()]
	}

	return string(codeBytes), nil
}
