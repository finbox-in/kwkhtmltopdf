package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
)

// TraceLogger wraps the standard logger with trace ID context
type TraceLogger struct {
	traceID string
}

func (t *TraceLogger) Printf(format string, v ...interface{}) {
	log.Printf("[TraceID: %s] %s", t.traceID, fmt.Sprintf(format, v...))
}

func (t *TraceLogger) Println(v ...interface{}) {
	args := append([]interface{}{fmt.Sprintf("[TraceID: %s]", t.traceID)}, v...)
	log.Println(args...)
}

// Context key for trace ID
type contextKey string

const traceIDKey contextKey = "traceID"

// Middleware to extract trace ID and add it to context
func withTraceID(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		traceID := r.Header.Get("X-Trace-ID")
		if traceID == "" {
			traceID = "unknown"
		}

		logger := &TraceLogger{traceID: traceID}
		ctx := context.WithValue(r.Context(), traceIDKey, logger)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

// Helper to get logger from context
func loggerFromContext(ctx context.Context) *TraceLogger {
	if logger, ok := ctx.Value(traceIDKey).(*TraceLogger); ok {
		return logger
	}
	return &TraceLogger{traceID: "unknown"}
}
