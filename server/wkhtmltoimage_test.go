package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// minimalPNGBase64 is a 1×1 transparent PNG (valid magic and structure).
const minimalPNGBase64 = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg=="

func init() {
	// *_test.go is not linked into the server binary; init runs for `go test` so
	// loggerFromContext never dereferences a nil GlobalLogger.
	NewProductionLogger()
}

func writeFakeWkhtmltoimage(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "fake-wkhtmltoimage.sh")
	script := fmt.Sprintf(`#!/bin/sh
# Fake wkhtmltoimage: last CLI arg is output path; write a minimal PNG there.
while [ $# -gt 1 ]; do
  shift
done
OUT="$1"
printf '%%s' '%s' | base64 -d > "$OUT"
`, minimalPNGBase64)
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestImageHandler_success(t *testing.T) {
	t.Setenv("KWKHTMLTOIMAGE_BIN", writeFakeWkhtmltoimage(t))

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, err := mw.CreateFormFile("file", "index.html")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fw.Write([]byte("<!DOCTYPE html><html><body>x</body></html>")); err != nil {
		t.Fatal(err)
	}
	_ = mw.WriteField("format", "png")
	_ = mw.WriteField("width", "640")
	if err := mw.Close(); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/image", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("X-Trace-ID", "test-image-success")
	rec := httptest.NewRecorder()
	withTraceID(imageHandler)(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.Bytes()
	if len(body) < 8 || string(body[:8]) != "\x89PNG\r\n\x1a\n" {
		t.Fatalf("not PNG, len=%d", len(body))
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "image/png") {
		t.Fatalf("Content-Type: got %q want image/png", ct)
	}
}

func TestImageHandler_missingIndexHtml(t *testing.T) {
	t.Setenv("KWKHTMLTOIMAGE_BIN", writeFakeWkhtmltoimage(t))

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, err := mw.CreateFormFile("file", "other.html")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fw.Write([]byte("<html></html>")); err != nil {
		t.Fatal(err)
	}
	if err := mw.Close(); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/image", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	rec := httptest.NewRecorder()
	withTraceID(imageHandler)(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status %d want 400 body %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "index.html") {
		t.Fatalf("body: %s", rec.Body.String())
	}
}

func TestImageHandler_methodNotAllowed(t *testing.T) {
	t.Setenv("KWKHTMLTOIMAGE_BIN", writeFakeWkhtmltoimage(t))
	req := httptest.NewRequest(http.MethodGet, "/image", nil)
	rec := httptest.NewRecorder()
	withTraceID(imageHandler)(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status %d want 405", rec.Code)
	}
}

func TestImageHandler_integrationRealBinary(t *testing.T) {
	if os.Getenv("WKHTMLTOIMAGE_INTEGRATION") != "1" {
		t.Skip("set WKHTMLTOIMAGE_INTEGRATION=1 and install wkhtmltoimage on PATH")
	}
	// `go test` runs with the module root as the working directory.
	htmlPath := filepath.Join("samples", "hello-image.html")
	html, err := os.ReadFile(htmlPath)
	if err != nil {
		t.Skip("samples/hello-image.html not found:", err)
	}

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, err := mw.CreateFormFile("file", "index.html")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fw.Write(html); err != nil {
		t.Fatal(err)
	}
	_ = mw.WriteField("format", "png")
	_ = mw.WriteField("width", "320")
	if err := mw.Close(); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/image", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	rec := httptest.NewRecorder()
	withTraceID(imageHandler)(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.Bytes()
	if len(body) < 8 || string(body[:8]) != "\x89PNG\r\n\x1a\n" {
		t.Fatalf("not PNG, len=%d", len(body))
	}
}

func TestImageHandler_fakeBinaryWritesValidPNG(t *testing.T) {
	// Sanity-check the base64 stub decodes to a PNG header.
	b, err := base64.StdEncoding.DecodeString(minimalPNGBase64)
	if err != nil {
		t.Fatal(err)
	}
	if string(b[:8]) != "\x89PNG\r\n\x1a\n" {
		t.Fatalf("stub is not PNG")
	}
}
