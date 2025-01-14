package middleware

import (
	"github.com/finbox-in/internal/pkg/logger"
	"github.com/gin-gonic/gin"
)

// TraceIDMiddleware extracts X-Trace-ID from the request header or generates one if not present.
func TraceIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		traceID := c.GetHeader("X-Trace-ID")

		log := logger.LoggerFromContext(c)

		// Create a new entry with the fields and assign it back to the logger

		if traceID != "" {
			// Only assign a new entry if traceID is present
			log = log.WithTraceID(traceID)
		}

		// Set the Trace ID in the context for access in handlers
		c.Set(logger.LoggerContextKey, log)

		c.Next() // Continue to the next middleware/handler
	}
}
