package api

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"strings"
	"testing"

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

	handler := NewHandler(shortener.NewService(storage.NewMemoryStorage()))

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
	mux := newTestServer(t)

	rec := shorten(t, mux, `{"url":"https://example.com"}`)

	body := strings.TrimSpace(rec.Body.String())
	if !strings.HasPrefix(body, "http://example.com/") {
		t.Errorf("body = %q, want prefix %q", body, "http://example.com/")
	}
	if ct := rec.Header().Get("Content-Type"); ct != "text/plain" {
		t.Errorf("Content-Type = %q, want %q", ct, "text/plain")
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
		})
	}
}

func TestResolveURL_Redirects(t *testing.T) {
	mux := newTestServer(t)
	original := "https://example.com/some/path?q=1"

	rec := shorten(t, mux, `{"url":"`+original+`"}`)
	code := path.Base(strings.TrimSpace(rec.Body.String()))

	req := httptest.NewRequest(http.MethodGet, "/"+code, nil)
	rec = httptest.NewRecorder()
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

func shorten(t *testing.T, mux *http.ServeMux, body string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, "/shorten", strings.NewReader(body))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("shorten status = %d, want %d (body: %q)", rec.Code, http.StatusCreated, rec.Body.String())
	}
	return rec
}
