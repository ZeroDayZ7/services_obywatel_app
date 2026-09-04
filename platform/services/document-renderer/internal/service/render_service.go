package service

import (
	"context"
	"document-renderer/internal/renderer"
	"document-renderer/internal/templates"
	"fmt"
	"log"
)

type RenderService interface {
	RenderPDF(ctx context.Context, templateName string, data map[string]any, opts *renderer.PDFOptions) ([]byte, error)
}

type renderService struct {
	pdfRenderer    renderer.PDFRenderer
	templateLoader *templates.Loader
}

func NewRenderService(pdfRenderer renderer.PDFRenderer, templatesDir string) RenderService {
	return &renderService{
		pdfRenderer:    pdfRenderer,
		templateLoader: templates.NewLoader(templatesDir),
	}
}

func (s *renderService) RenderPDF(ctx context.Context, templateName string, data map[string]any, opts *renderer.PDFOptions) ([]byte, error) {
	htmlBytes, err := s.templateLoader.Render(templateName, data)
	if err != nil {
		return nil, fmt.Errorf("template render failed: %w", err)
	}

	// PODGLĄD WYRENDEROWANEGO HTML (sprawdzasz czy klucze się wstawiły)
	log.Printf("[DEBUG] Rendered HTML output for %s:\n%s", templateName, string(htmlBytes))

	pdfOpts := renderer.DefaultPDFOptions()
	if opts != nil {
		pdfOpts = *opts
	}

	return s.pdfRenderer.RenderHTMLToPDF(ctx, string(htmlBytes), pdfOpts)
}
