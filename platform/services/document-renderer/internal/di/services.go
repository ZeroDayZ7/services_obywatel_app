package di

import (
	"document-renderer/internal/renderer"
	"document-renderer/internal/service"
)

func NewServiceContainer(pdfRenderer renderer.PDFRenderer, templatesDir string) service.RenderService {
	return service.NewRenderService(pdfRenderer, templatesDir)
}
