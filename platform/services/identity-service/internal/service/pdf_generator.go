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
	SecondName      *string
	LastName        string
	PESEL           string
	Email           string
	PhoneNumber     string // <-- Dodane
	Street          string
	HouseNumber     string
	FlatNumber      *string
	PostalCode      string
	City            string
	SignedAt        string
	KeyVersion      int
	DocumentHash    string

	OfficerName   string
	OfficerID     string
	DepartmentID  string
	InstitutionID string
}

type PDFGenerator interface {
	GenerateAgreementPDF(ctx context.Context, data AgreementTemplateData) ([]byte, error)
}

type pdfGenerator struct {
	tmpl *template.Template
}

//#region NewPDFGenerator
func NewPDFGenerator() (PDFGenerator, error) {
	tmpl, err := template.ParseFS(templateFS, "templates/agreement.html")
	if err != nil {
		return nil, fmt.Errorf("pdf_gen: failed to parse template: %w", err)
	}

	return &pdfGenerator{tmpl: tmpl}, nil
}

//#region GenerateAgreementPDF
func (g *pdfGenerator) GenerateAgreementPDF(ctx context.Context, data AgreementTemplateData) ([]byte, error) {
	var htmlBuf bytes.Buffer
	if err := g.tmpl.Execute(&htmlBuf, data); err != nil {
		return nil, fmt.Errorf("pdf_gen: failed to execute template: %w", err)
	}

	chromePath := `C:\Program Files\Google\Chrome\Application\chrome.exe`

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
