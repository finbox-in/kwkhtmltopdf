package logger

import (
	"context"
	"os"

	"github.com/sirupsen/logrus"
)

type Logger struct {
	*logrus.Entry
}

var (
	GlobalLogger *Logger
)

const LoggerContextKey = "logger"

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
func LoggerFromContext(ctx context.Context) *Logger {
	if logger, ok := ctx.Value(LoggerContextKey).(*Logger); ok {
		return logger
	}

	return newLogger()
}

// TODO: Remove
func (logger *Logger) WithX(key string, value interface{}) *Logger {
	logger.Entry = logger.WithField(key, value)
	return logger
}

func (logger *Logger) WithTraceID(traceID string) *Logger {
	logger.Entry = logger.WithField("trace-id", traceID)
	return logger
}
