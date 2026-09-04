package router

import (
	"document-renderer/internal/handler"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func NewRenderRouter(renderHandler handler.RenderHandler) http.Handler {
	r := chi.NewRouter()

	r.Post("/render", renderHandler.RenderPDF)

	return r
}
