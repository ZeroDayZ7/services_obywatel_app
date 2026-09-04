package di

import (
	"document-renderer/internal/handler"
	"document-renderer/internal/service"
)

func NewHandlerContainer(renderService service.RenderService, maxRequestBodyBytes int64) handler.RenderHandler {
	return handler.NewRenderHandler(renderService, maxRequestBodyBytes)
}
