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
	"github.com/zerodayz7/services/identity-service/config"
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
	PhoneNumber     string
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
	tmpl        *template.Template
	chromeWSURL string
}

//#region NewPDFGenerator
func NewPDFGenerator(cfg config.PDFConfig) (PDFGenerator, error) {
	tmpl, err := template.ParseFS(templateFS, "templates/agreement.html")
	if err != nil {
		return nil, fmt.Errorf("pdf_gen: failed to parse template: %w", err)
	}

	return &pdfGenerator{
		tmpl:        tmpl,
		chromeWSURL: cfg.ChromeWSURL,
	}, nil
}

//#region GenerateAgreementPDF
func (g *pdfGenerator) GenerateAgreementPDF(ctx context.Context, data AgreementTemplateData) ([]byte, error) {
	var htmlBuf bytes.Buffer
	if err := g.tmpl.Execute(&htmlBuf, data); err != nil {
		return nil, fmt.Errorf("pdf_gen: failed to execute template: %w", err)
	}

	var browser *rod.Browser
	if g.chromeWSURL != "" {
		browser = rod.New().ControlURL(g.chromeWSURL).Context(ctx).MustConnect()
	} else {
		u := launcher.New().Headless(true).MustLaunch()
		browser = rod.New().ControlURL(u).Context(ctx).MustConnect()
	}
	defer browser.MustClose()

	page := browser.MustPage()
	defer page.MustClose()

	page.MustSetDocumentContent(htmlBuf.String())
	page.MustWaitLoad()

	margin := 0.4
	pdfStream, err := page.PDF(&proto.PagePrintToPDF{
		PrintBackground: true,
		MarginTop:       &margin,
		MarginBottom:    &margin,
		MarginLeft:      &margin,
		MarginRight:     &margin,
	})
	if err != nil {
		return nil, fmt.Errorf("pdf_gen: failed to print pdf: %w", err)
	}

	return io.ReadAll(pdfStream)
}
