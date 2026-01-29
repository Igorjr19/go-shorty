package api

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/Igorjr19/go-shorty/internal/logger"
	"github.com/Igorjr19/go-shorty/internal/shortener"
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
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

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

	code, err := h.service.Shorten(req.URL)
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
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	code := r.PathValue("code")
	if code == "" {
		code = r.URL.Path[1:]
	}

	if code == "" {
		logger.Warn(r.Context(), "Code is required but not provided")
		http.Error(w, "Code is required", http.StatusBadRequest)
		return
	}

	logger.Debug(r.Context(), "Resolving short URL", slog.String("code", code))

	url, err := h.service.Resolve(code)
	if err != nil {
		logger.Warn(r.Context(), "Short URL not found", slog.String("code", code))
		http.NotFound(w, r)
		return
	}

	logger.Info(r.Context(), "Short URL resolved successfully",
		slog.String("code", code),
		slog.String("original_url", url),
	)

	http.Redirect(w, r, url, http.StatusFound)
}
