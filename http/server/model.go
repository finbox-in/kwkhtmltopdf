package server

import (
	"github.com/finbox-in/internal/service/wkhtmltopdf"
)

type ServerHandler struct {
	WkHTMLtoPDFService wkhtmltopdf.WkHTMLtoPDFServiceProvider
}
