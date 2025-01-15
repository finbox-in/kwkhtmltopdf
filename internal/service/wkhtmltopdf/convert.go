package wkhtmltopdf

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	models "github.com/finbox-in/internal/models/wkhtmltopdf"
	"github.com/finbox-in/internal/pkg/logger"
)

func (s *WkHTMLtoPDFService) Convert(ctx context.Context, req models.PDFRequest) (resp models.PDFResponse, err error) {
	logger := logger.LoggerFromContext(ctx)

	tmpdir, err := os.MkdirTemp("", "kwk")
	if err != nil {
		logger.Errorf("Failed to create temp directory: %v", err)
		s.metrics.IncreaseError("tempdir_creation_failed", err.Error())
		return models.PDFResponse{}, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tmpdir)

	logger.Infof("Temporary directory created: %s", tmpdir)

	// Prepare files and arguments
	args, err := s.prepareFilesAndArgs(ctx, tmpdir, req)
	if err != nil {
		s.metrics.IncreaseError("prepare_files_failed", err.Error())
		return models.PDFResponse{}, fmt.Errorf("failed to prepare files: %w", err)
	}

	logger.Info("Starting wkhtmltopdf conversion")

	// Execute wkhtmltopdf
	pdfBytes, err := s.executeWkHTMLtoPDF(ctx, args)
	if err != nil {
		s.metrics.IncreaseError("pdf_generation_failed", err.Error())
		return models.PDFResponse{}, fmt.Errorf("failed to generate PDF: %w", err)
	}

	s.metrics.ObservePDFSize(float64(len(pdfBytes)))
	logger.Info("PDF generation completed successfully",
		"size_bytes", len(pdfBytes),
		"has_header", len(req.HeaderHTML) > 0,
		"has_footer", len(req.FooterHTML) > 0)

	return models.PDFResponse{PDF: pdfBytes}, nil
}

func (s *WkHTMLtoPDFService) prepareFilesAndArgs(ctx context.Context, tmpdir string, req models.PDFRequest) ([]string, error) {
	logger := logger.LoggerFromContext(ctx)

	var args []string

	// Write index.html
	indexPath := filepath.Join(tmpdir, "index.html")
	if err := os.WriteFile(indexPath, req.IndexHTML, 0644); err != nil {
		return nil, fmt.Errorf("failed to write index.html: %w", err)
	}

	// Write header.html if provided
	if len(req.HeaderHTML) > 0 {
		headerPath := filepath.Join(tmpdir, "header.html")
		if err := os.WriteFile(headerPath, req.HeaderHTML, 0644); err != nil {
			return nil, fmt.Errorf("failed to write header.html: %w", err)
		}
		args = append(args, "--header-html", headerPath)
	}

	// Write footer.html if provided
	if len(req.FooterHTML) > 0 {
		footerPath := filepath.Join(tmpdir, "footer.html")
		if err := os.WriteFile(footerPath, req.FooterHTML, 0644); err != nil {
			return nil, fmt.Errorf("failed to write footer.html: %w", err)
		}
		args = append(args, "--footer-html", footerPath)
	}

	// Add extra arguments
	for key, value := range req.ExtraArgs {
		if value == "" {
			args = append(args, "--"+key)
		} else {
			args = append(args, "--"+key, value)
		}
	}

	// Add required arguments
	args = append(args,
		"--enable-local-file-access", // Allow loading local files -  https://github.com/wkhtmltopdf/wkhtmltopdf/issues/4460#issuecomment-661345113
		indexPath,                    // Input file
		"-",                          // Output to stdout
	)

	logger.Info("Prepared wkhtmltopdf arguments", "args", args)
	return args, nil
}

func (s *WkHTMLtoPDFService) executeWkHTMLtoPDF(ctx context.Context, args []string) ([]byte, error) {
	logger := logger.LoggerFromContext(ctx)

	logger.Info("Starting wkhtmltopdf process", "args", args)

	// ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	// defer cancel()

	// Create command
	cmd := exec.Command("wkhtmltopdf", args...)

	// Get stdout pipe
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		logger.Error("Failed to create stdout pipe", "error", err)
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	// Create buffer for stderr
	var stderrBuf bytes.Buffer
	cmd.Stderr = &stderrBuf

	// Start command
	if err := cmd.Start(); err != nil {
		logger.Error("Failed to start wkhtmltopdf", "error", err)
		return nil, fmt.Errorf("failed to start wkhtmltopdf: %w", err)
	}

	// Read output
	var outputBuf bytes.Buffer
	done := make(chan error, 1)

	go func() {
		_, err := io.Copy(&outputBuf, stdout)
		if err != nil {
			logger.Error("Failed to read stdout", "error", err)
			done <- fmt.Errorf("failed to read stdout: %w", err)
			return
		}
		done <- cmd.Wait()
	}()

	// Wait for completion or context cancellation
	select {
	case <-ctx.Done():
		logger.Warn("Context cancelled, killing wkhtmltopdf process", "pid", cmd.Process.Pid)

		// Context was canceled, kill the process
		if err := cmd.Process.Kill(); err != nil {
			logger.Error("failed to kill process after context cancellation", "error", err)
		}
		return nil, ctx.Err()

	case err := <-done:
		if err != nil {
			stderr := stderrBuf.String()
			logger.Error("wkhtmltopdf process failed",
				"error", err,
				"stderr", stderr,
				"pid", cmd.Process.Pid)

			if stderrBuf.Len() > 0 {
				return nil, fmt.Errorf("wkhtmltopdf failed: %w, stderr: %s", err, stderr)
			}
			return nil, fmt.Errorf("wkhtmltopdf failed: %w", err)
		}
	}

	outputSize := outputBuf.Len()
	logger.Info("wkhtmltopdf process completed",
		"output_size", outputSize,
		"pid", cmd.Process.Pid)

	// Check if we actually got any output
	if outputSize == 0 {
		logger.Error("wkhtmltopdf produced no output", "pid", cmd.Process.Pid)
		return nil, fmt.Errorf("wkhtmltopdf produced no output")
	}

	return outputBuf.Bytes(), nil
}
