package di

import (
	"document-renderer/config"
	"document-renderer/internal/renderer"
	"document-renderer/internal/router"
	"net/http"
)

type Container struct {
	Router http.Handler
}

func NewContainer(cfg *config.Config, pdfRenderer renderer.PDFRenderer) *Container {
	renderService := NewServiceContainer(pdfRenderer, cfg.TemplatesDir)
	renderHandler := NewHandlerContainer(renderService)
	r := router.NewRenderRouter(renderHandler)

	return &Container{
		Router: r,
	}
}
