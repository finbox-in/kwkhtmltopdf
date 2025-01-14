package wkhtmltopdf

import (
	"context"

	models "github.com/finbox-in/internal/models/wkhtmltopdf"
)

type WkHTMLtoPDFServiceProvider interface {
	Convert(ctx context.Context, req models.PDFRequest) (resp models.PDFResponse, err error)
}
