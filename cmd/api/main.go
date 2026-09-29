package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
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

	db := config.ConnectDB()
	defer db.Close()

	store := storage.NewPostgresStorage(db)

	service := shortener.NewService(store)

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
	server := &http.Server{
		Addr:              ":" + port,
		Handler:           finalHandler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	stopCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	serverErr := make(chan error, 1)
	go func() {
		logger.Info(ctx, "Server started", slog.String("port", port))
		serverErr <- server.ListenAndServe()
	}()

	select {
	case err := <-serverErr:
		logger.Error(ctx, "Server failed to start", slog.String("error", err.Error()))
		db.Close()
		os.Exit(1)
	case <-stopCtx.Done():
		logger.Info(ctx, "Shutdown signal received, draining connections")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error(ctx, "Graceful shutdown failed", slog.String("error", err.Error()))
		return
	}

	logger.Info(ctx, "Server stopped")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
