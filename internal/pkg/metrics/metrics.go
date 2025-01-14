package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type MetricsRecorder struct {
	errorTotal      *prometheus.CounterVec
	activeRequests  prometheus.Gauge
	requestDuration *prometheus.HistogramVec
	requestsTotal   *prometheus.CounterVec
	pdfSize         prometheus.Histogram
}

func NewMetricsRecorder() *MetricsRecorder {
	return &MetricsRecorder{
		errorTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "pdf_converter_errors_total",
				Help: "Total number of errors by type",
			},
			[]string{"error_type", "error_message"},
		),

		activeRequests: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "pdf_converter_active_requests",
				Help: "Number of requests currently being processed",
			},
		),

		requestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "pdf_converter_request_duration_seconds",
				Help:    "Time taken to process requests",
				Buckets: []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
			},
			[]string{"path"},
		),

		requestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "pdf_converter_requests_total",
				Help: "Total number of requests processed",
			},
			[]string{"path", "status_code"},
		),

		pdfSize: promauto.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "pdf_converter_pdf_size_bytes",
				Help:    "Size of generated PDFs in bytes",
				Buckets: []float64{1000, 10000, 100000, 1000000, 10000000},
			},
		),
	}
}

func (m *MetricsRecorder) IncreaseError(errorType, errorMsg string) {
	m.errorTotal.WithLabelValues(errorType, errorMsg).Inc()
}

func (m *MetricsRecorder) IncreaseActiveRequests() {
	m.activeRequests.Inc()
}

func (m *MetricsRecorder) DecreaseActiveRequests() {
	m.activeRequests.Dec()
}

func (m *MetricsRecorder) ObserveRequestDuration(path string, duration float64) {
	m.requestDuration.WithLabelValues(path).Observe(duration)
}

func (m *MetricsRecorder) ObservePDFSize(size float64) {
	m.pdfSize.Observe(size)
}

func (m *MetricsRecorder) IncreaseRequestTotal(path string, statusCode string) {
	m.requestsTotal.WithLabelValues(path, statusCode).Inc()
}

// // Add this to main.go
// func setupMetricsEndpoint() {
// 	http.Handle("/metrics", promhttp.Handler())
// }
