package middleware

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/Igorjr19/go-shorty/internal/logger"
)

func TestMain(m *testing.M) {
	logger.Init("test")
	os.Exit(m.Run())
}

func TestInMemoryRateLimiter_AllowRequest(t *testing.T) {
	rl := NewInMemoryRateLimiter(2, time.Minute)

	for i := range 2 {
		if !rl.AllowRequest("1.2.3.4") {
			t.Fatalf("request %d was blocked, want allowed", i+1)
		}
	}
	if rl.AllowRequest("1.2.3.4") {
		t.Error("request 3 was allowed, want blocked")
	}
}

func TestInMemoryRateLimiter_SeparateVisitors(t *testing.T) {
	rl := NewInMemoryRateLimiter(1, time.Minute)

	if !rl.AllowRequest("1.1.1.1") {
		t.Fatal("first request from 1.1.1.1 was blocked")
	}
	if rl.AllowRequest("1.1.1.1") {
		t.Error("second request from 1.1.1.1 was allowed, want blocked")
	}
	if !rl.AllowRequest("2.2.2.2") {
		t.Error("first request from 2.2.2.2 was blocked, want allowed")
	}
}

func TestInMemoryRateLimiter_ResetsAfterWindow(t *testing.T) {
	window := 20 * time.Millisecond
	rl := NewInMemoryRateLimiter(1, window)

	rl.AllowRequest("1.1.1.1")
	if rl.AllowRequest("1.1.1.1") {
		t.Fatal("second request inside the window was allowed")
	}

	time.Sleep(window + 10*time.Millisecond)

	if !rl.AllowRequest("1.1.1.1") {
		t.Error("request after the window was blocked, want allowed")
	}
}

func TestInMemoryRateLimiter_LimitReturns429(t *testing.T) {
	rl := NewInMemoryRateLimiter(1, time.Minute)
	handler := rl.Limit(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wantStatuses := []int{http.StatusOK, http.StatusTooManyRequests}
	for i, want := range wantStatuses {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "10.0.0.1:1234"
		rec := httptest.NewRecorder()
		handler(rec, req)

		if rec.Code != want {
			t.Errorf("request %d status = %d, want %d", i+1, rec.Code, want)
		}
	}
}

func TestGetIP(t *testing.T) {
	tests := []struct {
		name         string
		trustProxy   bool
		remoteAddr   string
		forwardedFor string
		realIP       string
		want         string
	}{
		{name: "ignores X-Forwarded-For by default", remoteAddr: "10.0.0.1:1234", forwardedFor: "6.6.6.6", want: "10.0.0.1"},
		{name: "ignores X-Real-IP by default", remoteAddr: "10.0.0.1:1234", realIP: "6.6.6.6", want: "10.0.0.1"},
		{name: "trusted X-Forwarded-For", trustProxy: true, remoteAddr: "10.0.0.1:1234", forwardedFor: "1.2.3.4", want: "1.2.3.4"},
		{name: "trusted X-Forwarded-For with multiple IPs", trustProxy: true, remoteAddr: "10.0.0.1:1234", forwardedFor: "1.2.3.4, 10.0.0.2", want: "1.2.3.4"},
		{name: "trusted X-Real-IP", trustProxy: true, remoteAddr: "10.0.0.1:1234", realIP: "5.6.7.8", want: "5.6.7.8"},
		{name: "X-Forwarded-For takes precedence over X-Real-IP", trustProxy: true, remoteAddr: "10.0.0.1:1234", forwardedFor: "1.2.3.4", realIP: "5.6.7.8", want: "1.2.3.4"},
		{name: "trusted without headers falls back to RemoteAddr", trustProxy: true, remoteAddr: "10.0.0.1:1234", want: "10.0.0.1"},
		{name: "RemoteAddr without port", remoteAddr: "10.0.0.1", want: "10.0.0.1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			old := TrustProxyHeaders
			TrustProxyHeaders = tt.trustProxy
			t.Cleanup(func() { TrustProxyHeaders = old })

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = tt.remoteAddr
			if tt.forwardedFor != "" {
				req.Header.Set("X-Forwarded-For", tt.forwardedFor)
			}
			if tt.realIP != "" {
				req.Header.Set("X-Real-IP", tt.realIP)
			}

			if got := getIP(req); got != tt.want {
				t.Errorf("getIP() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRecoverMiddleware_Returns500OnPanic(t *testing.T) {
	handler := RecoverMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("boom")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}

func TestLoggingMiddleware_SetsRequestID(t *testing.T) {
	var ctxRequestID string
	handler := LoggingMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctxRequestID = logger.GetRequestID(r.Context())
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	headerRequestID := rec.Header().Get("X-Request-ID")
	if headerRequestID == "" {
		t.Fatal("X-Request-ID header is empty")
	}
	if ctxRequestID != headerRequestID {
		t.Errorf("context request ID = %q, want %q", ctxRequestID, headerRequestID)
	}
}

func TestLoggingMiddleware_RecordsStatusCode(t *testing.T) {
	var wrapped *responseWriter
	handler := LoggingMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wrapped = w.(*responseWriter)
		w.WriteHeader(http.StatusTeapot)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusTeapot {
		t.Errorf("response status = %d, want %d", rec.Code, http.StatusTeapot)
	}
	if wrapped.statusCode != http.StatusTeapot {
		t.Errorf("recorded status = %d, want %d", wrapped.statusCode, http.StatusTeapot)
	}
}

func TestLoggingAndRecover_PanicReturns500WithRequestID(t *testing.T) {
	handler := LoggingMiddleware(RecoverMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("boom")
	})))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
	if rec.Header().Get("X-Request-ID") == "" {
		t.Error("X-Request-ID header is empty")
	}
}
