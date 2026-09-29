package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"path/filepath"
	"testing"
	"time"
)

type logEntry struct {
	Msg       string `json:"msg"`
	RequestID string `json:"request_id"`
	Source    struct {
		File string `json:"file"`
	} `json:"source"`
}

func captureLogs(t *testing.T, level slog.Level) *bytes.Buffer {
	t.Helper()

	var buf bytes.Buffer
	old := log
	log = slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{AddSource: true, Level: level}))
	t.Cleanup(func() { log = old })
	return &buf
}

func decodeEntry(t *testing.T, buf *bytes.Buffer) logEntry {
	t.Helper()

	var entry logEntry
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("decode log entry %q: %v", buf.String(), err)
	}
	return entry
}

func TestLog_SourcePointsToCaller(t *testing.T) {
	tests := []struct {
		name string
		log  func(ctx context.Context)
	}{
		{name: "Info", log: func(ctx context.Context) { Info(ctx, "msg") }},
		{name: "Debug", log: func(ctx context.Context) { Debug(ctx, "msg") }},
		{name: "Warn", log: func(ctx context.Context) { Warn(ctx, "msg") }},
		{name: "Error", log: func(ctx context.Context) { Error(ctx, "msg") }},
		{name: "HTTPRequest", log: func(ctx context.Context) { HTTPRequest(ctx, "GET", "/", "::1", 200, time.Millisecond, nil) }},
		{name: "RateLimitExceeded", log: func(ctx context.Context) { RateLimitExceeded(ctx, "::1", 10, time.Minute) }},
		{name: "DatabaseQuery", log: func(ctx context.Context) { DatabaseQuery(ctx, "SELECT 1", time.Millisecond, nil) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := captureLogs(t, slog.LevelDebug)

			tt.log(context.Background())

			entry := decodeEntry(t, buf)
			if got := filepath.Base(entry.Source.File); got != "logger_test.go" {
				t.Errorf("source file = %q, want %q", got, "logger_test.go")
			}
		})
	}
}

func TestLog_AddsRequestID(t *testing.T) {
	buf := captureLogs(t, slog.LevelInfo)

	Info(WithRequestID(context.Background(), "req-1"), "msg")

	if entry := decodeEntry(t, buf); entry.RequestID != "req-1" {
		t.Errorf("request_id = %q, want %q", entry.RequestID, "req-1")
	}
}

func TestLog_RespectsLevel(t *testing.T) {
	buf := captureLogs(t, slog.LevelInfo)

	Debug(context.Background(), "msg")

	if buf.Len() != 0 {
		t.Errorf("debug log written at info level: %q", buf.String())
	}
}
