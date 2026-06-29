package core

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"
)

// InitLogger sets up structured JSON logging for production or text for development.
func InitLogger() {
	env := os.Getenv("ENV")
	var handler slog.Handler
	if env == "production" {
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	} else {
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})
	}
	slog.SetDefault(slog.New(handler))
}

// RequestLogger is an HTTP middleware that emits a structured log line per request.
// It records method, path, status code, duration, and request ID.
func RequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		lrw := &loggingResponseWriter{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(lrw, r)

		slog.InfoContext(r.Context(), "http request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", lrw.status,
			"duration_ms", time.Since(start).Milliseconds(),
			"request_id", requestIDFromContext(r.Context()),
		)
	})
}

type loggingResponseWriter struct {
	http.ResponseWriter
	status int
}

func (lrw *loggingResponseWriter) WriteHeader(status int) {
	lrw.status = status
	lrw.ResponseWriter.WriteHeader(status)
}

type requestIDKey struct{}

func requestIDFromContext(ctx context.Context) string {
	if id, ok := ctx.Value(requestIDKey{}).(string); ok {
		return id
	}
	return ""
}

// LogPipelineMetrics emits structured log lines for pipeline cost monitoring (SC-002, SC-004).
func LogPipelineMetrics(ctx context.Context, userID string, durationMs int64, cacheReadTokens, inputTokens, outputTokens int) {
	slog.InfoContext(ctx, "soap_pipeline_metrics",
		"user_id", userID,
		"duration_ms", durationMs,
		"cache_read_tokens", cacheReadTokens,
		"input_tokens", inputTokens,
		"output_tokens", outputTokens,
	)
}
