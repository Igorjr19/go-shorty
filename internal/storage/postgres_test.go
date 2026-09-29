package storage

import (
	"database/sql"
	"errors"
	"os"
	"sync"
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

func TestPostgresStorage_PreservesCreatedAtInstant(t *testing.T) {
	s := newTestPostgresStorage(t)
	saoPaulo := time.FixedZone("UTC-3", -3*60*60)
	createdAt := time.Date(2026, 9, 28, 21, 0, 0, 0, saoPaulo)
	link := entity.Link{Code: "tst005", OriginalURL: "https://example.com", CreatedAt: createdAt}
	cleanupLink(t, s, link.Code)

	if err := s.Save(t.Context(), link); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	got, err := s.Load(t.Context(), link.Code)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !got.CreatedAt.Equal(createdAt) {
		t.Errorf("CreatedAt = %v, want %v (diff %v)", got.CreatedAt, createdAt, got.CreatedAt.Sub(createdAt))
	}
}

func TestPostgresStorage_LoadNotFound(t *testing.T) {
	s := newTestPostgresStorage(t)

	_, err := s.Load(t.Context(), "missing")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Load() error = %v, want %v", err, ErrNotFound)
	}
}

func TestPostgresStorage_Visit(t *testing.T) {
	s := newTestPostgresStorage(t)
	link := entity.Link{Code: "tst003", OriginalURL: "https://example.com", CreatedAt: time.Now()}
	cleanupLink(t, s, link.Code)

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
		if got.OriginalURL != link.OriginalURL {
			t.Errorf("Visit().OriginalURL = %q, want %q", got.OriginalURL, link.OriginalURL)
		}
	}

	loaded, err := s.Load(t.Context(), link.Code)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded.Visits != 3 {
		t.Errorf("Load().Visits = %d, want 3", loaded.Visits)
	}
}

func TestPostgresStorage_VisitConcurrent(t *testing.T) {
	s := newTestPostgresStorage(t)
	link := entity.Link{Code: "tst004", OriginalURL: "https://example.com", CreatedAt: time.Now()}
	cleanupLink(t, s, link.Code)

	if err := s.Save(t.Context(), link); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	const visitors = 50
	var wg sync.WaitGroup
	for range visitors {
		wg.Go(func() {
			if _, err := s.Visit(t.Context(), link.Code); err != nil {
				t.Errorf("Visit() error = %v", err)
			}
		})
	}
	wg.Wait()

	loaded, _ := s.Load(t.Context(), link.Code)
	if loaded.Visits != visitors {
		t.Errorf("Visits = %d, want %d", loaded.Visits, visitors)
	}
}

func TestPostgresStorage_VisitNotFound(t *testing.T) {
	s := newTestPostgresStorage(t)

	_, err := s.Visit(t.Context(), "missing")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Visit() error = %v, want %v", err, ErrNotFound)
	}
}

func TestPostgresStorage_SaveDuplicateCode(t *testing.T) {
	s := newTestPostgresStorage(t)
	link := entity.Link{Code: "tst002", OriginalURL: "https://example.com", CreatedAt: time.Now()}
	cleanupLink(t, s, link.Code)

	if err := s.Save(t.Context(), link); err != nil {
		t.Fatalf("first Save() error = %v", err)
	}

	if err := s.Save(t.Context(), link); !errors.Is(err, ErrCodeExists) {
		t.Errorf("second Save() error = %v, want %v", err, ErrCodeExists)
	}
}
