package main

import (
	"context"
	"net/http"
	"os"

	"github.com/sirupsen/logrus"
)

type Logger struct {
	*logrus.Entry
}

var (
	GlobalLogger *Logger
)

// Define a custom type for the context key
type contextKey string

const LoggerContextKey = contextKey("logger")

func NewProductionLogger() *Logger {
	Log := logrus.New()
	Log.SetReportCaller(true)
	Log.SetOutput(os.Stdout)
	Log.SetFormatter(&logrus.JSONFormatter{})
	Log.SetLevel(logrus.DebugLevel)

	GlobalLogger = &Logger{
		Entry: logrus.NewEntry(Log),
	}

	return GlobalLogger
}

func newLogger() *Logger {
	newLogger := *GlobalLogger
	return &newLogger
}

// Helper to get logger from context
func loggerFromContext(ctx context.Context) *Logger {
	if logger, ok := ctx.Value(LoggerContextKey).(*Logger); ok {
		return logger
	}

	return newLogger()
}

func (logger *Logger) WithTraceID(traceID string) *Logger {
	logger.Entry = logger.WithField("trace-id", traceID)
	return logger
}

// Middleware to extract trace ID and attach a logger with traceID to context
func withTraceID(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		traceID := r.Header.Get("X-Trace-ID")

		log := loggerFromContext(r.Context())

		// Create a new entry with the fields and assign it back to the logger

		if traceID != "" {
			// Only assign a new entry if traceID is present
			log = log.WithTraceID(traceID)
		}

		ctx := context.WithValue(r.Context(), LoggerContextKey, log)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}
