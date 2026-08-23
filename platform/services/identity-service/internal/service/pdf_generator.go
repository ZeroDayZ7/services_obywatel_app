package service

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"html/template"
	"io"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
)

//go:embed templates/agreement.html
var templateFS embed.FS

type AgreementTemplateData struct {
	AgreementID     string
	AgreementNumber string
	FirstName       string
	LastName        string
	PESEL           string
	Street          string
	HouseNumber     string
	FlatNumber      string
	PostalCode      string
	City            string
	PhoneNumber     string
	SignedAt        string
}

type PDFGenerator interface {
	GenerateAgreementPDF(ctx context.Context, data AgreementTemplateData) ([]byte, error)
}

type pdfGenerator struct {
	tmpl *template.Template
}

func NewPDFGenerator() (PDFGenerator, error) {
	tmpl, err := template.ParseFS(templateFS, "templates/agreement.html")
	if err != nil {
		return nil, fmt.Errorf("pdf_gen: failed to parse template: %w", err)
	}

	return &pdfGenerator{tmpl: tmpl}, nil
}

func (g *pdfGenerator) GenerateAgreementPDF(ctx context.Context, data AgreementTemplateData) ([]byte, error) {
	// 1. Renderowanie HTML w pamięci RAM
	var htmlBuf bytes.Buffer
	if err := g.tmpl.Execute(&htmlBuf, data); err != nil {
		return nil, fmt.Errorf("pdf_gen: failed to execute template: %w", err)
	}

	// 2. Uruchomienie lokalnego Chrome/Edge bez pobierania z sieci
	chromePath := `C:\Program Files\Google\Chrome\Application\chrome.exe`
	// Jeśli chcesz użyć Edge, zamień na:
	// chromePath := `C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe`

	u := launcher.New().
		Bin(chromePath).
		Headless(true).
		MustLaunch()

	browser := rod.New().ControlURL(u).Context(ctx).MustConnect()
	defer browser.MustClose()

	page := browser.MustPage()
	defer page.MustClose()

	page.MustSetDocumentContent(htmlBuf.String())
	page.MustWaitLoad()

	pdfStream, err := page.PDF(&proto.PagePrintToPDF{
		PrintBackground: true,
		MarginTop:       new(0.4),
		MarginBottom:    new(0.4),
		MarginLeft:      new(0.4),
		MarginRight:     new(0.4),
	})
	if err != nil {
		return nil, fmt.Errorf("pdf_gen: failed to print pdf: %w", err)
	}

	pdfBytes, err := io.ReadAll(pdfStream)
	if err != nil {
		return nil, fmt.Errorf("pdf_gen: failed to read pdf stream: %w", err)
	}

	return pdfBytes, nil
}
