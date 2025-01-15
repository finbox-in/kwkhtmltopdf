package wkhtmltopdf

import (
	logger "github.com/finbox-in/internal/pkg/logger"
	"github.com/finbox-in/internal/pkg/metrics"
)

type WkHTMLtoPDFService struct {
	logger  *logger.Logger
	metrics metrics.MetricsRecorder
}

func InitWKHTMLtoPDFService(logger *logger.Logger, metrics metrics.MetricsRecorder) *WkHTMLtoPDFService {
	return &WkHTMLtoPDFService{
		logger:  logger,
		metrics: metrics,
	}
}
