package di

import (
	"document-renderer/internal/renderer"
	"document-renderer/internal/service"
	"document-renderer/internal/templates"
)

func NewServiceContainer(pdfRenderer renderer.PDFRenderer, templatesDir string) service.RenderService {
	templateLoader := templates.NewLoader(templatesDir)
	return service.NewRenderService(pdfRenderer, templateLoader)
}
