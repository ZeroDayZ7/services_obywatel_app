package di

import (
	"document-renderer/internal/handler"
	"document-renderer/internal/service"
)

func NewHandlerContainer(renderService service.RenderService) handler.RenderHandler {
	return handler.NewRenderHandler(renderService)
}
