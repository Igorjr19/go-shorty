package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/Igorjr19/go-shorty/internal/api"
	"github.com/Igorjr19/go-shorty/internal/config"
	"github.com/Igorjr19/go-shorty/internal/middleware"
	"github.com/Igorjr19/go-shorty/internal/shortener"
	"github.com/Igorjr19/go-shorty/internal/storage"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	storage := storage.NewPostgresStorage(config.ConnectDB())

	service := shortener.NewService(storage)

	handler := api.NewHandler(service)

	var readRateLimiter middleware.RateLimiter = middleware.NewInMemoryRateLimiter(1000, time.Minute)
	var writeRateLimiter middleware.RateLimiter = middleware.NewInMemoryRateLimiter(10, time.Minute)

	http.HandleFunc("POST /shorten", writeRateLimiter.Limit(handler.ShortenURL))
	http.HandleFunc("GET /{code}", readRateLimiter.Limit(handler.ResolveURL))

	fmt.Println("Server started on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
