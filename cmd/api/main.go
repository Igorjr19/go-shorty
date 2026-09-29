package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/Igorjr19/go-shorty/internal/api"
	"github.com/Igorjr19/go-shorty/internal/config"
	"github.com/Igorjr19/go-shorty/internal/logger"
	"github.com/Igorjr19/go-shorty/internal/middleware"
	"github.com/Igorjr19/go-shorty/internal/shortener"
	"github.com/Igorjr19/go-shorty/internal/storage"
	"github.com/joho/godotenv"
)

func main() {

	envErr := godotenv.Load()

	env := getEnv("ENVIRONMENT", "development")
	logger.Init(env)

	ctx := logger.WithRequestID(context.Background(), "startup")

	if envErr != nil {
		logger.Info(ctx, "No .env file found, using environment variables")
	}
	logger.Info(ctx, "Starting go-shorty server",
		slog.String("environment", env),
		slog.String("version", "1.0.0"),
	)

	storage := storage.NewPostgresStorage(config.ConnectDB())

	service := shortener.NewService(storage)

	handler := api.NewHandler(service)

	middleware.TrustProxyHeaders = getEnv("TRUST_PROXY_HEADERS", "false") == "true"

	var readRateLimiter middleware.RateLimiter = middleware.NewInMemoryRateLimiter(1000, time.Minute)
	var writeRateLimiter middleware.RateLimiter = middleware.NewInMemoryRateLimiter(10, time.Minute)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /shorten", writeRateLimiter.Limit(handler.ShortenURL))
	mux.HandleFunc("GET /{code}", readRateLimiter.Limit(handler.ResolveURL))

	finalHandler := middleware.LoggingMiddleware(
		middleware.RecoverMiddleware(mux),
	)

	port := getEnv("PORT", "8080")
	logger.Info(ctx, "Server started", slog.String("port", port))

	if err := http.ListenAndServe(":"+port, finalHandler); err != nil {
		logger.Error(ctx, "Server failed to start", slog.String("error", err.Error()))
		os.Exit(1)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
