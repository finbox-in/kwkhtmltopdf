package server

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"

	models "github.com/finbox-in/internal/models/wkhtmltopdf"
	"github.com/finbox-in/internal/pkg/logger"
	"github.com/gin-gonic/gin"
)

func (s *ServerHandler) ConvertHTMLToPDF(c *gin.Context) {
	logger := logger.LoggerFromContext(c)

	form, err := c.MultipartForm()
	if err != nil {
		logger.Errorf("Failed to get multipart form: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Failed to parse multipart form: %v", err),
		})
		return
	}

	// Parse the request
	pdfReq, err := s.parseMultipartForm(c, form)
	if err != nil {
		logger.Errorf("Failed to parse request: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Failed to parse request: %v", err),
		})
		return
	}
	// Call the service
	pdfResponse, err := s.WkHTMLtoPDFService.Convert(c, pdfReq)
	if err != nil {
		logger.Errorf("Failed to convert PDF: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to convert PDF: %v", err),
		})
		return
	}

	// Set response headers
	c.Set("Content-Type", "application/pdf")
	c.Set("Content-Disposition", "attachment; filename=output.pdf")

	c.Data(http.StatusOK, "application/pdf", pdfResponse.PDF)
}

func (s *ServerHandler) parseMultipartForm(ctx context.Context, form *multipart.Form) (models.PDFRequest, error) {
	logger := logger.LoggerFromContext(ctx)

	pdfReq := models.PDFRequest{
		ExtraArgs: make(map[string]string),
	}

	// Handle file uploads
	for fileName, fileHeaders := range form.File {
		if len(fileHeaders) == 0 {
			continue
		}

		// Get first file from array
		file, err := fileHeaders[0].Open()
		if err != nil {
			logger.Errorf("Failed to open file %s: %v", fileName, err)
			return models.PDFRequest{}, fmt.Errorf("failed to open file %s: %v", fileName, err)
		}
		defer file.Close()

		// Read file content
		var buf bytes.Buffer
		if _, err := io.Copy(&buf, file); err != nil {
			logger.Errorf("Failed to read file %s: %v", fileName, err)
			return models.PDFRequest{}, fmt.Errorf("failed to read file %s: %v", fileName, err)
		}

		// Assign content based on filename
		switch fileHeaders[0].Filename {
		case "index.html":
			pdfReq.IndexHTML = buf.Bytes()
		case "header.html":
			pdfReq.HeaderHTML = buf.Bytes()
		case "footer.html":
			pdfReq.FooterHTML = buf.Bytes()
		}
	}

	// Handle other form values
	for key, values := range form.Value {
		if len(values) > 0 {
			pdfReq.ExtraArgs[key] = values[0]
		}
	}

	// Validate required fields
	if len(pdfReq.IndexHTML) == 0 {
		logger.Error("index.html file is required")
		return models.PDFRequest{}, fmt.Errorf("index.html file is required")
	}

	return pdfReq, nil
}
