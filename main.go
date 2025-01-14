package main

import (
	"net/http"

	"github.com/finbox-in/http/routes"
	"github.com/finbox-in/http/server"
	logger "github.com/finbox-in/internal/pkg/logger"
	"github.com/finbox-in/internal/pkg/metrics"
	"github.com/finbox-in/internal/service/wkhtmltopdf"
	"github.com/finbox-in/middleware"
	"github.com/gin-gonic/gin"
)

func main() {
	l := logger.NewProductionLogger()

	srv := initServer(l, *metrics.NewMetricsRecorder())

	if err := start(srv, l); err != nil {
		l.Fatalf("Failed to start server: %v", err)
	}
}

// func start()

func initServer(l *logger.Logger, metrics metrics.MetricsRecorder) *server.ServerHandler {

	wkhtmltopdfService := wkhtmltopdf.InitWKHTMLtoPDFService(l, metrics)

	server := server.ServerHandler{
		WkHTMLtoPDFService: wkhtmltopdfService,
	}

	return &server
}

func start(s *server.ServerHandler, logger *logger.Logger) error {
	// Create a Gin router instance
	router := gin.Default()

	router.MaxMultipartMemory = 30 * 1024 * 1024

	router.Use(middleware.TraceIDMiddleware())

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, map[string]interface{}{
			"status": "ok",
		})
	})
	v1 := router.Group("/v1")
	{
		routes.AddHTMLToPDFRoutesV1(v1, s)
	}

	logger.Info("Starting server on port 8080")

	return router.Run(":8080")
}
