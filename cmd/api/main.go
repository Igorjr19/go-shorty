package main

import (
	"fmt"
	"net/http"

	"github.com/Igorjr19/go-shorty/internal/api"
	"github.com/Igorjr19/go-shorty/internal/shortener"
	"github.com/Igorjr19/go-shorty/internal/storage"
)

func main() {
	storage := storage.NewMemoryStorage()

	service := shortener.NewService(storage)

	handler := api.NewHandler(service)

	http.HandleFunc("POST /shorten", handler.ShortenURL)
	http.HandleFunc("GET /{code}", handler.ResolveURL)

	fmt.Println("Server started on :8080")
	http.ListenAndServe(":8080", nil)
}
