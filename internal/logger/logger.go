package logger

import (
	"context"
	"log/slog"
	"os"
	"runtime"
	"time"
)

type contextKey string

const (
	RequestIDKey contextKey = "request_id"
)

var log = slog.Default()

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
	logAt(ctx, slog.LevelInfo, msg, args)
}

func Debug(ctx context.Context, msg string, args ...any) {
	logAt(ctx, slog.LevelDebug, msg, args)
}

func Warn(ctx context.Context, msg string, args ...any) {
	logAt(ctx, slog.LevelWarn, msg, args)
}

func Error(ctx context.Context, msg string, args ...any) {
	logAt(ctx, slog.LevelError, msg, args)
}

func logAt(ctx context.Context, level slog.Level, msg string, args []any) {
	if !log.Enabled(ctx, level) {
		return
	}

	var pcs [1]uintptr
	runtime.Callers(3, pcs[:])

	record := slog.NewRecord(time.Now(), level, msg, pcs[0])
	record.Add(addContextAttrs(ctx, args)...)
	_ = log.Handler().Handle(ctx, record)
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
		slog.Float64("latency_ms", float64(latency.Microseconds())/1000),
	}

	if err != nil {
		attrs = append(attrs, slog.String("error", err.Error()))
		logAt(ctx, slog.LevelError, "HTTP request failed", attrs)
	} else {
		logAt(ctx, slog.LevelInfo, "HTTP request", attrs)
	}
}

func RateLimitExceeded(ctx context.Context, ip string, limit int, window time.Duration) {
	logAt(ctx, slog.LevelWarn, "Rate limit exceeded", []any{
		slog.String("ip", ip),
		slog.Int("limit", limit),
		slog.Duration("window", window),
	})
}

func DatabaseQuery(ctx context.Context, query string, duration time.Duration, err error) {
	attrs := []any{
		slog.String("query", query),
		slog.Float64("duration_ms", float64(duration.Microseconds())/1000),
	}

	if err != nil {
		attrs = append(attrs, slog.String("error", err.Error()))
		logAt(ctx, slog.LevelError, "Database query failed", attrs)
	} else {
		logAt(ctx, slog.LevelDebug, "Database query executed", attrs)
	}
}
