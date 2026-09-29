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

	if err := s.Save(t.Context(), link); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	got, err := s.Load(t.Context(), link.Code)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got.OriginalURL != link.OriginalURL {
		t.Errorf("Load().OriginalURL = %q, want %q", got.OriginalURL, link.OriginalURL)
	}
}

func TestMemoryStorage_LoadNotFound(t *testing.T) {
	s := NewMemoryStorage()

	_, err := s.Load(t.Context(), "missing")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Load() error = %v, want %v", err, ErrNotFound)
	}
}

func TestMemoryStorage_Visit(t *testing.T) {
	s := NewMemoryStorage()
	link := entity.Link{Code: "abc123", OriginalURL: "https://example.com", CreatedAt: time.Now()}
	if err := s.Save(t.Context(), link); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	for want := int64(1); want <= 3; want++ {
		got, err := s.Visit(t.Context(), link.Code)
		if err != nil {
			t.Fatalf("Visit() error = %v", err)
		}
		if got.Visits != want {
			t.Errorf("Visit().Visits = %d, want %d", got.Visits, want)
		}
	}

	loaded, _ := s.Load(t.Context(), link.Code)
	if loaded.Visits != 3 {
		t.Errorf("Load().Visits = %d, want 3", loaded.Visits)
	}
}

func TestMemoryStorage_VisitNotFound(t *testing.T) {
	s := NewMemoryStorage()

	_, err := s.Visit(t.Context(), "missing")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Visit() error = %v, want %v", err, ErrNotFound)
	}
}

func TestMemoryStorage_SaveDuplicateCode(t *testing.T) {
	s := NewMemoryStorage()
	link := entity.Link{Code: "abc123", OriginalURL: "https://example.com", CreatedAt: time.Now()}

	if err := s.Save(t.Context(), link); err != nil {
		t.Fatalf("first Save() error = %v", err)
	}

	link.OriginalURL = "https://other.com"
	if err := s.Save(t.Context(), link); !errors.Is(err, ErrCodeExists) {
		t.Fatalf("second Save() error = %v, want %v", err, ErrCodeExists)
	}

	got, _ := s.Load(t.Context(), link.Code)
	if got.OriginalURL != "https://example.com" {
		t.Errorf("stored link was overwritten: OriginalURL = %q", got.OriginalURL)
	}
}
