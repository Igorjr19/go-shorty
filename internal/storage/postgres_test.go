package storage

import (
	"database/sql"
	"errors"
	"os"
	"testing"
	"time"

	_ "github.com/lib/pq"

	"github.com/Igorjr19/go-shorty/internal/entity"
	"github.com/Igorjr19/go-shorty/internal/migrate"
)

func newTestPostgresStorage(t *testing.T) *PostgresStorage {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set, skipping postgres integration test")
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	t.Cleanup(func() { db.Close() })

	if err := db.Ping(); err != nil {
		t.Fatalf("db.Ping() error = %v", err)
	}

	if err := migrate.NewMigrator(db, "../../migrations").Up(); err != nil {
		t.Fatalf("migrations failed: %v", err)
	}

	return NewPostgresStorage(db)
}

func cleanupLink(t *testing.T, s *PostgresStorage, code string) {
	t.Helper()
	t.Cleanup(func() {
		s.db.Exec("DELETE FROM links WHERE code = $1", code)
	})
}

func TestPostgresStorage_SaveAndLoad(t *testing.T) {
	s := newTestPostgresStorage(t)
	link := entity.Link{Code: "tst001", OriginalURL: "https://example.com", CreatedAt: time.Now()}
	cleanupLink(t, s, link.Code)

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

func TestPostgresStorage_LoadNotFound(t *testing.T) {
	s := newTestPostgresStorage(t)

	_, err := s.Load("missing")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Load() error = %v, want %v", err, ErrNotFound)
	}
}

func TestPostgresStorage_SaveDuplicateCode(t *testing.T) {
	s := newTestPostgresStorage(t)
	link := entity.Link{Code: "tst002", OriginalURL: "https://example.com", CreatedAt: time.Now()}
	cleanupLink(t, s, link.Code)

	if err := s.Save(link); err != nil {
		t.Fatalf("first Save() error = %v", err)
	}

	if err := s.Save(link); !errors.Is(err, ErrCodeExists) {
		t.Errorf("second Save() error = %v, want %v", err, ErrCodeExists)
	}
}
