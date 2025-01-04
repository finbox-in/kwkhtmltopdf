package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// Counter for total requests
	requestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "pdf_requests_total",
			Help: "Total number of PDF generation requests",
		},
		[]string{"path", "status"},
	)

	// Histogram for request duration
	requestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "pdf_request_duration_seconds",
			Help:    "Time taken to process PDF generation requests",
			Buckets: []float64{.1, .5, 1, 2.5, 5, 10, 20, 30},
		},
		[]string{"path"},
	)

	// Gauge for current active requests
	activeRequests = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "pdf_active_requests",
			Help: "Number of currently active PDF generation requests",
		},
	)

	// Counter for errors
	errorTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "pdf_errors_total",
			Help: "Total number of PDF generation errors",
		},
		[]string{"type", "error"},
	)

	// Histogram for PDF file sizes
	pdfSize = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "pdf_size_bytes",
			Help:    "Size of generated PDFs in bytes",
			Buckets: prometheus.ExponentialBuckets(1024, 2, 10), // Starting from 1KB
		},
	)
)

type statusRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (sr *statusRecorder) WriteHeader(code int) {
	sr.statusCode = code
	sr.ResponseWriter.WriteHeader(code)
}

// TODO ignore opts?
// --log-level, -q, --quiet, --read-args-from-stdin, --dump-default-toc-xsl
// --dump-outline <file>, --allow <path>, --cache-dir <path>,
// --disable-local-file-access, --enable-local-file-access

// TODO sensitive opts to be hidden from log
// --cookie <name> <value>, --password <password>,
// --ssl-key-password <password>

func wkhtmltopdfBin() string {
	bin := os.Getenv("KWKHTMLTOPDF_BIN")
	if bin != "" {
		return bin
	}
	return "wkhtmltopdf"
}

func httpError(w http.ResponseWriter, err error, code int, logger *TraceLogger) {
	logger.Printf("HTTP error: %v", err)

	if sr, ok := w.(*statusRecorder); ok {
		sr.statusCode = code
	}

	http.Error(w, err.Error(), code)
}

func httpAbort(w http.ResponseWriter, err error, logger *TraceLogger) {
	logger.Printf("HTTP abort: %v", err)

	if sr, ok := w.(*statusRecorder); ok {
		sr.statusCode = http.StatusInternalServerError
	}

	wh, ok := w.(http.Hijacker)
	if !ok {
		errorTotal.WithLabelValues("hijack_unsupported", err.Error()).Inc()
		logger.Println("cannot abort connection, error not reported to client: http.Hijacker not supported")
		return
	}
	c, _, err := wh.Hijack()
	if err != nil {
		errorTotal.WithLabelValues("hijack_failed", err.Error()).Inc()
		logger.Println("cannot abort connection, error not reported to client: ", err)
		return
	}
	c.Close()
}
func pdfHandler(w http.ResponseWriter, r *http.Request) {
	logger := loggerFromContext(r.Context())

	if r.Method != http.MethodPost {
		errorTotal.WithLabelValues("method_not_allowed", r.Method).Inc()
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	start := time.Now()
	activeRequests.Inc()
	defer activeRequests.Dec()

	rec := &statusRecorder{ResponseWriter: w, statusCode: http.StatusOK}
	defer func() {
		duration := time.Since(start).Seconds()
		requestDuration.WithLabelValues(r.URL.Path).Observe(duration)
		requestsTotal.WithLabelValues(r.URL.Path, fmt.Sprintf("%d", rec.statusCode)).Inc()
	}()

	tmpdir, err := os.MkdirTemp("", "kwk")
	if err != nil {
		errorTotal.WithLabelValues("tempdir_creation_failed", err.Error()).Inc()
		httpError(w, err, http.StatusInternalServerError, logger)
		return
	}
	defer os.RemoveAll(tmpdir)

	logger.Printf("Temporary directory created: %s", tmpdir)

	reader, err := r.MultipartReader()
	if err != nil {
		errorTotal.WithLabelValues("multipart_reader_creation_failed", err.Error()).Inc()
		logger.Printf("Failed to create multipart reader: %v", err)
		httpError(w, err, http.StatusBadRequest, logger)
		return
	}

	args, endArgs, indexPath, err := parseMultipartForm(reader, tmpdir, logger)
	if err != nil {
		errorTotal.WithLabelValues("parse_multipart_form_failed", err.Error()).Inc()
		logger.Printf("Failed to parse multipart form: %v", err)
		httpError(w, err, http.StatusBadRequest, logger)
		return
	}

	if indexPath == "" {
		errorTotal.WithLabelValues("index_html_file_not_found", "").Inc()
		logger.Println("index.html file is required but not found")
		httpError(w, errors.New("index.html file is required"), http.StatusBadRequest, logger)
		return
	}

	endArgs = append(endArgs, indexPath)
	args = append(args, endArgs...)

	runWkhtmltopdf(rec, r.Context(), args, logger)
}

func parseMultipartForm(reader *multipart.Reader, tmpdir string, logger *TraceLogger) (args []string, endArgs []string, indexPath string, err error) {

	defer func() {
		// Track errors
		if err != nil {
			errorTotal.WithLabelValues("parse_multipart_form", err.Error()).Inc()
		}
	}()

	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, nil, "", err
		}

		if part.FormName() == "file" {
			path := filepath.Join(tmpdir, filepath.Base(part.FileName()))
			file, err := os.Create(path)
			if err != nil {
				return nil, nil, "", err
			}
			_, err = io.Copy(file, part)
			file.Close()
			if err != nil {
				return nil, nil, "", err
			}

			switch part.FileName() {
			case "header.html":
				endArgs = append(endArgs, "--header-html", path)
			case "footer.html":
				endArgs = append(endArgs, "--footer-html", path)
			case "index.html":
				indexPath = path
			}
		} else {
			buf := new(bytes.Buffer)
			buf.ReadFrom(part)
			arg := buf.String()
			if arg == "" {
				args = append(args, fmt.Sprintf("--%s", part.FormName()))
			} else {
				args = append(args, fmt.Sprintf("--%s", part.FormName()), arg)
			}
		}
	}

	return args, endArgs, indexPath, nil
}

func runWkhtmltopdf(w http.ResponseWriter, ctx context.Context, args []string, logger *TraceLogger) {
	args = append(args, "--enable-local-file-access") // https://github.com/wkhtmltopdf/wkhtmltopdf/issues/4460#issuecomment-661345113
	args = append(args, "-")
	logger.Println("Args", args)

	logger.Println("Starting wkhtmltopdf process")
	cmd := exec.Command(wkhtmltopdfBin(), args...)
	cmdStdout, err := cmd.StdoutPipe()
	if err != nil {
		errorTotal.WithLabelValues("stdout_pipe_failed", err.Error()).Inc()
		httpError(w, err, http.StatusInternalServerError, logger)
		return
	}
	cmd.Stderr = os.Stderr
	done := make(chan error, 1)

	err = cmd.Start()
	if err != nil {
		errorTotal.WithLabelValues("process_start_failed", err.Error()).Inc()
		httpError(w, err, http.StatusInternalServerError, logger)
		return
	}

	logger.Println("wkhtmltopdf process started")

	// Buffer the output
	var pdfBuffer bytes.Buffer

	go func() {
		_, err := io.Copy(&pdfBuffer, cmdStdout)
		if err != nil {
			errorTotal.WithLabelValues("copy_output_failed", err.Error()).Inc()
			logger.Printf("Error copying command output: %v", err)
			done <- err
			return
		}
		done <- cmd.Wait()
	}()

	select {
	case <-ctx.Done():
		if ctx.Err() != nil {
			errorTotal.WithLabelValues("context_cancelled", ctx.Err().Error()).Inc()
		}
		logger.Println("Context cancelled, killing wkhtmltopdf process")
		if err := cmd.Process.Kill(); err != nil {
			logger.Printf("Failed to kill process: %v", err)
		}
		httpError(w, ctx.Err(), http.StatusRequestTimeout, logger)
		return
	case err := <-done:
		if err != nil {
			logger.Printf("wkhtmltopdf process failed: %v", err)
			errorTotal.WithLabelValues("process_failed", err.Error()).Inc()
			httpError(w, err, http.StatusInternalServerError, logger)
			return
		}
	}

	// Only set the content type header when the process is successful
	w.Header().Set("Content-Type", "application/pdf")
	// Write the PDF to the client
	_, err = w.Write(pdfBuffer.Bytes())
	if err != nil {
		logger.Printf("Failed to write PDF to response: %v", err)
		httpAbort(w, err, logger)
		return
	}

	// Log and track the size of the generated PDF
	logger.Printf("Generated PDF size: %d bytes", pdfBuffer.Len())
	pdfSize.Observe(float64(pdfBuffer.Len()))
	logger.Println("wkhtmltopdf process completed successfully")
}

func statusHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func main() {
	router := http.NewServeMux()
	router.HandleFunc("/status", withTraceID(statusHandler))
	router.HandleFunc("/pdf", withTraceID(pdfHandler))
	router.Handle("/metrics", promhttp.Handler())

	log.Println("kwkhtmltopdf server listening on port 8080")

	server := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}
	log.Fatal(server.ListenAndServe())

}
