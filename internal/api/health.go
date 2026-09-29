package api

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/Igorjr19/go-shorty/internal/logger"
)

type Pinger interface {
	PingContext(ctx context.Context) error
}

type HealthResponse struct {
	Status string `json:"status"`
}

func HealthHandler(db Pinger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		status, resp := http.StatusOK, HealthResponse{Status: "ok"}
		if err := db.PingContext(ctx); err != nil {
			logger.Error(r.Context(), "Health check failed", slog.String("error", err.Error()))
			status, resp = http.StatusServiceUnavailable, HealthResponse{Status: "unavailable"}
		}

		writeJSON(w, status, resp)
	}
}
