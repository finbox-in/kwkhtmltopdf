package models

type PDFRequest struct {
	IndexHTML  []byte
	HeaderHTML []byte
	FooterHTML []byte
	ExtraArgs  map[string]string
}
