package service

import (
	"context"
	"document-renderer/internal/model"
	"document-renderer/internal/renderer"
	"document-renderer/internal/templates"
	"fmt"

	"github.com/zerodayz7/platform/pkg/shared"
)

type RenderService interface {
	RenderPDF(ctx context.Context, templateName string, data map[string]any, opts *model.PDFOptions) ([]byte, error)
}

type renderService struct {
	pdfRenderer    renderer.PDFRenderer
	templateLoader templates.TemplateRenderer
}

func NewRenderService(pdfRenderer renderer.PDFRenderer, templateLoader templates.TemplateRenderer) RenderService {
	return &renderService{
		pdfRenderer:    pdfRenderer,
		templateLoader: templateLoader,
	}
}

func (s *renderService) RenderPDF(ctx context.Context, templateName string, data map[string]any, opts *model.PDFOptions) ([]byte, error) {
	log := shared.GetLogger()

	htmlBytes, err := s.templateLoader.Render(templateName, data)
	if err != nil {
		return nil, fmt.Errorf("template render failed: %w", err)
	}

	log.Info("Template rendered successfully to HTML", "template", templateName, "bytes", len(htmlBytes))

	return s.pdfRenderer.RenderHTMLToPDF(ctx, string(htmlBytes), opts)
}
