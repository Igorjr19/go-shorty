package logger

import (
	"context"
	"log/slog"
	"os"
	"time"
)

type contextKey string

const (
	RequestIDKey contextKey = "request_id"
)

var log *slog.Logger

func Init(env string) {
	var handler slog.Handler

	opts := &slog.HandlerOptions{
		Level:     getLogLevel(env),
		AddSource: env == "development",
	}

	switch env {
	case "production":

		handler = slog.NewJSONHandler(os.Stdout, opts)
	default:

		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	log = slog.New(handler)
	slog.SetDefault(log)
}

func getLogLevel(env string) slog.Level {
	switch env {
	case "production":
		return slog.LevelInfo
	case "development":
		return slog.LevelDebug
	default:
		return slog.LevelInfo
	}
}

func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, RequestIDKey, requestID)
}

func GetRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(RequestIDKey).(string); ok {
		return id
	}
	return ""
}

func Info(ctx context.Context, msg string, args ...any) {
	args = addContextAttrs(ctx, args)
	log.InfoContext(ctx, msg, args...)
}

func Debug(ctx context.Context, msg string, args ...any) {
	args = addContextAttrs(ctx, args)
	log.DebugContext(ctx, msg, args...)
}

func Warn(ctx context.Context, msg string, args ...any) {
	args = addContextAttrs(ctx, args)
	log.WarnContext(ctx, msg, args...)
}

func Error(ctx context.Context, msg string, args ...any) {
	args = addContextAttrs(ctx, args)
	log.ErrorContext(ctx, msg, args...)
}

func addContextAttrs(ctx context.Context, args []any) []any {
	if requestID := GetRequestID(ctx); requestID != "" {
		args = append(args, slog.String("request_id", requestID))
	}
	return args
}

func HTTPRequest(ctx context.Context, method, path, ip string, statusCode int, latency time.Duration, err error) {
	attrs := []any{
		slog.String("method", method),
		slog.String("path", path),
		slog.String("ip", ip),
		slog.Int("status", statusCode),
		slog.Duration("latency_ms", latency),
	}

	if err != nil {
		attrs = append(attrs, slog.String("error", err.Error()))
		Error(ctx, "HTTP request failed", attrs...)
	} else {
		Info(ctx, "HTTP request", attrs...)
	}
}

func RateLimitExceeded(ctx context.Context, ip string, limit int, window time.Duration) {
	Warn(ctx, "Rate limit exceeded",
		slog.String("ip", ip),
		slog.Int("limit", limit),
		slog.Duration("window", window),
	)
}

func DatabaseQuery(ctx context.Context, query string, duration time.Duration, err error) {
	attrs := []any{
		slog.String("query", query),
		slog.Duration("duration_ms", duration),
	}

	if err != nil {
		attrs = append(attrs, slog.String("error", err.Error()))
		Error(ctx, "Database query failed", attrs...)
	} else {
		Debug(ctx, "Database query executed", attrs...)
	}
}
