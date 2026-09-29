package api

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/Igorjr19/go-shorty/internal/logger"
	"github.com/Igorjr19/go-shorty/internal/shortener"
	"github.com/Igorjr19/go-shorty/internal/storage"
)

type ShortenRequest struct {
	URL string `json:"url"`
}

type ShortenResponse struct {
	Code     string `json:"code"`
	ShortURL string `json:"short_url"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

type Handler struct {
	service *shortener.Service
	baseURL string
}

func NewHandler(service *shortener.Service, baseURL string) *Handler {
	return &Handler{
		service: service,
		baseURL: strings.TrimRight(baseURL, "/"),
	}
}

func (h *Handler) ShortenURL(w http.ResponseWriter, r *http.Request) {
	var req ShortenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Warn(r.Context(), "Invalid request body", slog.String("error", err.Error()))
		writeError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.URL == "" {
		logger.Warn(r.Context(), "URL is required but not provided")
		writeError(w, "URL is required", http.StatusBadRequest)
		return
	}

	logger.Debug(r.Context(), "Creating short URL", slog.String("original_url", req.URL))

	code, err := h.service.Shorten(r.Context(), req.URL)
	if errors.Is(err, shortener.ErrInvalidURL) {
		logger.Warn(r.Context(), "Invalid URL provided", slog.String("original_url", req.URL))
		writeError(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err != nil {
		logger.Error(r.Context(), "Failed to create short URL",
			slog.String("original_url", req.URL),
			slog.String("error", err.Error()),
		)
		writeError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	logger.Info(r.Context(), "Short URL created successfully",
		slog.String("code", code),
		slog.String("original_url", req.URL),
	)

	writeJSON(w, http.StatusCreated, ShortenResponse{
		Code:     code,
		ShortURL: h.shortURL(r, code),
	})
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

func (h *Handler) shortURL(r *http.Request, code string) string {
	if h.baseURL != "" {
		return h.baseURL + "/" + code
	}

	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	return scheme + "://" + r.Host + "/" + code
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, message string, status int) {
	writeJSON(w, status, ErrorResponse{Error: message})
}
