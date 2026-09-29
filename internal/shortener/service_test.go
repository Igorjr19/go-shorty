package shortener

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/Igorjr19/go-shorty/internal/entity"
	"github.com/Igorjr19/go-shorty/internal/storage"
)

type fakeStorage struct {
	*storage.MemoryStorage
	saveErrs  []error
	saveCalls int
}

func newFakeStorage(saveErrs ...error) *fakeStorage {
	return &fakeStorage{MemoryStorage: storage.NewMemoryStorage(), saveErrs: saveErrs}
}

func (f *fakeStorage) Save(ctx context.Context, link entity.Link) error {
	f.saveCalls++
	if f.saveCalls <= len(f.saveErrs) && f.saveErrs[f.saveCalls-1] != nil {
		return f.saveErrs[f.saveCalls-1]
	}
	return f.MemoryStorage.Save(ctx, link)
}

func TestService_ShortenAndResolve(t *testing.T) {
	svc := NewService(storage.NewMemoryStorage())
	original := "https://example.com/some/path"

	code, err := svc.Shorten(t.Context(), original)
	if err != nil {
		t.Fatalf("Shorten() error = %v", err)
	}
	if len(code) != codeLength {
		t.Errorf("len(code) = %d, want %d", len(code), codeLength)
	}

	got, err := svc.Resolve(t.Context(), code)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if got != original {
		t.Errorf("Resolve() = %q, want %q", got, original)
	}
}

func TestService_ShortenInvalidURL(t *testing.T) {
	tests := []struct {
		name string
		url  string
	}{
		{name: "javascript scheme", url: "javascript:alert(1)"},
		{name: "empty", url: ""},
		{name: "relative path", url: "/some/path"},
		{name: "missing scheme", url: "example.com"},
		{name: "ftp scheme", url: "ftp://example.com/file"},
		{name: "missing host", url: "http:///path"},
		{name: "garbage", url: "not a url"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewService(storage.NewMemoryStorage())

			_, err := svc.Shorten(t.Context(), tt.url)
			if !errors.Is(err, ErrInvalidURL) {
				t.Errorf("Shorten(%q) error = %v, want %v", tt.url, err, ErrInvalidURL)
			}
		})
	}
}

func TestService_ShortenValidURL(t *testing.T) {
	tests := []struct {
		name string
		url  string
	}{
		{name: "http", url: "http://example.com"},
		{name: "https with path and query", url: "https://example.com/a/b?c=d#e"},
		{name: "with port", url: "http://localhost:8080/path"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewService(storage.NewMemoryStorage())

			if _, err := svc.Shorten(t.Context(), tt.url); err != nil {
				t.Errorf("Shorten(%q) error = %v", tt.url, err)
			}
		})
	}
}

func TestService_ShortenGeneratesValidCodes(t *testing.T) {
	svc := NewService(storage.NewMemoryStorage())

	for range 100 {
		code, err := svc.Shorten(t.Context(), "https://example.com")
		if err != nil {
			t.Fatalf("Shorten() error = %v", err)
		}
		for _, c := range code {
			if !strings.ContainsRune(codeAlphabet, c) {
				t.Fatalf("code %q has character %q outside the alphabet", code, c)
			}
		}
	}
}

func TestService_ShortenRetriesOnCollision(t *testing.T) {
	fake := newFakeStorage(storage.ErrCodeExists, storage.ErrCodeExists)
	svc := NewService(fake)

	code, err := svc.Shorten(t.Context(), "https://example.com")
	if err != nil {
		t.Fatalf("Shorten() error = %v", err)
	}
	if fake.saveCalls != 3 {
		t.Errorf("saveCalls = %d, want 3", fake.saveCalls)
	}

	got, err := svc.Resolve(t.Context(), code)
	if err != nil || got != "https://example.com" {
		t.Errorf("Resolve(%q) = %q, %v", code, got, err)
	}
}

func TestService_ShortenGivesUpAfterMaxRetries(t *testing.T) {
	errs := make([]error, maxSaveRetries)
	for i := range errs {
		errs[i] = storage.ErrCodeExists
	}
	fake := newFakeStorage(errs...)
	svc := NewService(fake)

	_, err := svc.Shorten(t.Context(), "https://example.com")
	if !errors.Is(err, ErrCodeGenerationFailed) {
		t.Errorf("Shorten() error = %v, want %v", err, ErrCodeGenerationFailed)
	}
	if fake.saveCalls != maxSaveRetries {
		t.Errorf("saveCalls = %d, want %d", fake.saveCalls, maxSaveRetries)
	}
}

func TestService_ShortenReturnsStorageError(t *testing.T) {
	storageErr := errors.New("connection refused")
	fake := newFakeStorage(storageErr)
	svc := NewService(fake)

	_, err := svc.Shorten(t.Context(), "https://example.com")
	if !errors.Is(err, storageErr) {
		t.Errorf("Shorten() error = %v, want %v", err, storageErr)
	}
	if fake.saveCalls != 1 {
		t.Errorf("saveCalls = %d, want 1 (should not retry on other errors)", fake.saveCalls)
	}
}

func TestService_ResolveCountsVisits(t *testing.T) {
	store := storage.NewMemoryStorage()
	svc := NewService(store)

	code, err := svc.Shorten(t.Context(), "https://example.com")
	if err != nil {
		t.Fatalf("Shorten() error = %v", err)
	}

	for range 3 {
		if _, err := svc.Resolve(t.Context(), code); err != nil {
			t.Fatalf("Resolve() error = %v", err)
		}
	}

	link, _ := store.Load(t.Context(), code)
	if link.Visits != 3 {
		t.Errorf("Visits = %d, want 3", link.Visits)
	}
}

func TestService_Stats(t *testing.T) {
	svc := NewService(storage.NewMemoryStorage())

	code, err := svc.Shorten(t.Context(), "https://example.com")
	if err != nil {
		t.Fatalf("Shorten() error = %v", err)
	}
	svc.Resolve(t.Context(), code)
	svc.Resolve(t.Context(), code)

	link, err := svc.Stats(t.Context(), code)
	if err != nil {
		t.Fatalf("Stats() error = %v", err)
	}
	if link.Visits != 2 {
		t.Errorf("Visits = %d, want 2", link.Visits)
	}
	if link.OriginalURL != "https://example.com" {
		t.Errorf("OriginalURL = %q, want %q", link.OriginalURL, "https://example.com")
	}
}

func TestService_StatsDoesNotCountVisit(t *testing.T) {
	svc := NewService(storage.NewMemoryStorage())

	code, _ := svc.Shorten(t.Context(), "https://example.com")
	svc.Stats(t.Context(), code)

	link, _ := svc.Stats(t.Context(), code)
	if link.Visits != 0 {
		t.Errorf("Visits = %d, want 0", link.Visits)
	}
}

func TestService_ResolveNotFound(t *testing.T) {
	svc := NewService(storage.NewMemoryStorage())

	_, err := svc.Resolve(t.Context(), "missing")
	if !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("Resolve() error = %v, want %v", err, storage.ErrNotFound)
	}
}
