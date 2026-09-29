package storage

import (
	"errors"
	"testing"
	"time"

	"github.com/Igorjr19/go-shorty/internal/entity"
)

func TestMemoryStorage_SaveAndLoad(t *testing.T) {
	s := NewMemoryStorage()
	link := entity.Link{Code: "abc123", OriginalURL: "https://example.com", CreatedAt: time.Now()}

	if err := s.Save(link); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	got, err := s.Load(link.Code)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got.OriginalURL != link.OriginalURL {
		t.Errorf("Load().OriginalURL = %q, want %q", got.OriginalURL, link.OriginalURL)
	}
}

func TestMemoryStorage_LoadNotFound(t *testing.T) {
	s := NewMemoryStorage()

	_, err := s.Load("missing")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Load() error = %v, want %v", err, ErrNotFound)
	}
}

func TestMemoryStorage_SaveDuplicateCode(t *testing.T) {
	s := NewMemoryStorage()
	link := entity.Link{Code: "abc123", OriginalURL: "https://example.com", CreatedAt: time.Now()}

	if err := s.Save(link); err != nil {
		t.Fatalf("first Save() error = %v", err)
	}

	link.OriginalURL = "https://other.com"
	if err := s.Save(link); !errors.Is(err, ErrCodeExists) {
		t.Fatalf("second Save() error = %v, want %v", err, ErrCodeExists)
	}

	got, _ := s.Load(link.Code)
	if got.OriginalURL != "https://example.com" {
		t.Errorf("stored link was overwritten: OriginalURL = %q", got.OriginalURL)
	}
}
