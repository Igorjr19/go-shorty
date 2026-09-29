package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/Igorjr19/go-shorty/internal/entity"
	"github.com/Igorjr19/go-shorty/internal/logger"
	"github.com/Igorjr19/go-shorty/internal/shortener"
	"github.com/Igorjr19/go-shorty/internal/storage"
)

func TestMain(m *testing.M) {
	logger.Init("test")
	os.Exit(m.Run())
}

func newTestServer(t *testing.T) *http.ServeMux {
	t.Helper()
	return newTestServerWithBaseURL(t, "")
}

func newTestServerWithBaseURL(t *testing.T, baseURL string) *http.ServeMux {
	t.Helper()

	handler := NewHandler(shortener.NewService(storage.NewMemoryStorage()), baseURL)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /shorten", handler.ShortenURL)
	mux.HandleFunc("GET /{code}", handler.ResolveURL)
	return mux
}

func TestShortenURL_Created(t *testing.T) {
	mux := newTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/shorten", strings.NewReader(`{"url":"https://example.com"}`))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d (body: %q)", rec.Code, http.StatusCreated, rec.Body.String())
	}
}

func TestShortenURL_ReturnsShortURL(t *testing.T) {
	tests := []struct {
		name    string
		baseURL string
		want    string
	}{
		{name: "falls back to request host", baseURL: "", want: "http://example.com/"},
		{name: "uses base url", baseURL: "https://sho.rt", want: "https://sho.rt/"},
		{name: "trims trailing slash from base url", baseURL: "https://sho.rt/", want: "https://sho.rt/"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := newTestServerWithBaseURL(t, tt.baseURL)

			resp := shorten(t, mux, `{"url":"https://example.com"}`)

			if resp.ShortURL != tt.want+resp.Code {
				t.Errorf("short_url = %q, want %q", resp.ShortURL, tt.want+resp.Code)
			}
		})
	}
}

func TestShortenURL_ReturnsJSON(t *testing.T) {
	mux := newTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/shorten", strings.NewReader(`{"url":"https://example.com"}`))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want %q", ct, "application/json")
	}
}

func TestShortenURL_BadRequest(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "invalid json", body: `{"url":`},
		{name: "empty body", body: ``},
		{name: "empty url", body: `{"url":""}`},
		{name: "missing url", body: `{}`},
		{name: "invalid url", body: `{"url":"not a url"}`},
		{name: "javascript url", body: `{"url":"javascript:alert(1)"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := newTestServer(t)

			req := httptest.NewRequest(http.MethodPost, "/shorten", strings.NewReader(tt.body))
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
			}

			var body ErrorResponse
			if err := json.NewDecoder(rec.Body).Decode(&body); err != nil || body.Error == "" {
				t.Errorf("body is not a JSON error: %v", err)
			}
		})
	}
}

func TestResolveURL_Redirects(t *testing.T) {
	mux := newTestServer(t)
	original := "https://example.com/some/path?q=1"

	resp := shorten(t, mux, `{"url":"`+original+`"}`)

	req := httptest.NewRequest(http.MethodGet, "/"+resp.Code, nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusFound)
	}
	if loc := rec.Header().Get("Location"); loc != original {
		t.Errorf("Location = %q, want %q", loc, original)
	}
}

func TestResolveURL_NotFound(t *testing.T) {
	mux := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/missing", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func shorten(t *testing.T, mux *http.ServeMux, body string) ShortenResponse {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, "/shorten", strings.NewReader(body))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("shorten status = %d, want %d (body: %q)", rec.Code, http.StatusCreated, rec.Body.String())
	}

	var resp ShortenResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode shorten response: %v", err)
	}
	return resp
}

type failingStorage struct {
	storage.Storage
}

func (failingStorage) Load(context.Context, string) (entity.Link, error) {
	return entity.Link{}, errors.New("connection refused")
}

func (failingStorage) Visit(context.Context, string) (entity.Link, error) {
	return entity.Link{}, errors.New("connection refused")
}

func TestResolveURL_StorageError(t *testing.T) {
	handler := NewHandler(shortener.NewService(failingStorage{}), "")
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{code}", handler.ResolveURL)

	req := httptest.NewRequest(http.MethodGet, "/abc123", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}
