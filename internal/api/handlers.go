package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/Igorjr19/go-shorty/internal/logger"
	"github.com/Igorjr19/go-shorty/internal/shortener"
	"github.com/Igorjr19/go-shorty/internal/storage"
)

type ShortenRequest struct {
	URL string `json:"url"`
}

type ShortenResponse struct {
	Code string `json:"code"`
}

type Handler struct {
	service *shortener.Service
}

func NewHandler(service *shortener.Service) *Handler {
	return &Handler{
		service: service,
	}
}

func (h *Handler) ShortenURL(w http.ResponseWriter, r *http.Request) {
	var req ShortenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Warn(r.Context(), "Invalid request body", slog.String("error", err.Error()))
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.URL == "" {
		logger.Warn(r.Context(), "URL is required but not provided")
		http.Error(w, "URL is required", http.StatusBadRequest)
		return
	}

	logger.Debug(r.Context(), "Creating short URL", slog.String("original_url", req.URL))

	code, err := h.service.Shorten(r.Context(), req.URL)
	if errors.Is(err, shortener.ErrInvalidURL) {
		logger.Warn(r.Context(), "Invalid URL provided", slog.String("original_url", req.URL))
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err != nil {
		logger.Error(r.Context(), "Failed to create short URL",
			slog.String("original_url", req.URL),
			slog.String("error", err.Error()),
		)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	logger.Info(r.Context(), "Short URL created successfully",
		slog.String("code", code),
		slog.String("original_url", req.URL),
	)

	fullURL := fmt.Sprintf("http://%s/%s\n", r.Host, code)

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(fullURL))
}

func (h *Handler) ResolveURL(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")

	logger.Debug(r.Context(), "Resolving short URL", slog.String("code", code))

	url, err := h.service.Resolve(r.Context(), code)
	if errors.Is(err, storage.ErrNotFound) {
		logger.Warn(r.Context(), "Short URL not found", slog.String("code", code))
		http.NotFound(w, r)
		return
	}
	if err != nil {
		logger.Error(r.Context(), "Failed to resolve short URL",
			slog.String("code", code),
			slog.String("error", err.Error()),
		)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	logger.Info(r.Context(), "Short URL resolved successfully",
		slog.String("code", code),
		slog.String("original_url", url),
	)

	http.Redirect(w, r, url, http.StatusFound)
}
