package middleware

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/Igorjr19/go-shorty/internal/logger"
	"github.com/google/uuid"
)

func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		requestID := uuid.New().String()
		ctx := logger.WithRequestID(r.Context(), requestID)
		r = r.WithContext(ctx)

		wrapped := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		wrapped.Header().Set("X-Request-ID", requestID)

		logger.Debug(ctx, "Incoming request",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.String("ip", getIP(r)),
			slog.String("user_agent", r.UserAgent()),
		)

		next.ServeHTTP(wrapped, r)

		latency := time.Since(start)
		logger.HTTPRequest(ctx, r.Method, r.URL.Path, getIP(r), wrapped.statusCode, latency, nil)
	})
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

func (rw *responseWriter) WriteHeader(code int) {
	if !rw.written {
		rw.statusCode = code
		rw.written = true
		rw.ResponseWriter.WriteHeader(code)
	}
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.written {
		rw.WriteHeader(http.StatusOK)
	}
	return rw.ResponseWriter.Write(b)
}

func RecoverMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				logger.Error(r.Context(), "Panic recovered",
					slog.Any("panic", err),
					slog.String("path", r.URL.Path),
					slog.String("method", r.Method),
				)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}
