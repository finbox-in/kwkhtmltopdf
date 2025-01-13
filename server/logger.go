package main

import (
	"context"
	"net/http"
	"os"

	"github.com/sirupsen/logrus"
)

type logger struct {
	*logrus.Entry
}

var (
	GlobalLogger *logger
)

type loggerContextKeyType string

const loggerContextKey loggerContextKeyType = "logger"

func init() {
	Log := logrus.New()
	Log.SetReportCaller(true)
	Log.SetOutput(os.Stdout)
	Log.SetFormatter(&logrus.JSONFormatter{})
	Log.SetLevel(logrus.DebugLevel)

	GlobalLogger = &logger{
		Entry: logrus.NewEntry(Log),
	}
}

// Middleware to extract trace ID and attach a logger with traceID to context
func withTraceID(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		logger := loggerFromContext(r.Context())

		traceID := r.Header.Get("X-Trace-ID")

		// Create a new entry with the fields and assign it back to the logger
		loggerWithFields := logger.WithFields(logrus.Fields{
			"trace-id": traceID,
		})

		if traceID != "" {
			// Only assign a new entry if traceID is present
			logger.Entry = loggerWithFields
		}

		ctx := context.WithValue(r.Context(), loggerContextKey, logger)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

func newLogger() *logger {
	newLogger := *GlobalLogger
	return &newLogger
}

// Helper to get logger from context
func loggerFromContext(ctx context.Context) *logger {
	if logger, ok := ctx.Value(loggerContextKey).(*logger); ok {
		return logger
	}
	return newLogger()
}

// TODO: Remove
func (logger *logger) WithX(key string, value interface{}) *logger {
	logger.Entry = logger.WithField(key, value)
	return logger
}
