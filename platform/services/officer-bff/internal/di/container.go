package di

import (
	"github.com/zerodayz7/platform/pkg/httpserver"
	"github.com/zerodayz7/services/officer-bff/config"
	"github.com/zerodayz7/services/officer-bff/internal/handler"
	"github.com/zerodayz7/services/officer-bff/internal/service"
)

type Container struct {
	Config          *config.Config
	KeyStore        *httpserver.KeyStore
	OfficialHandler *handler.OfficialHandler
}

func BuildContainer(cfg *config.Config, keyStore *httpserver.KeyStore) *Container {
	officialSvc := service.NewOfficialService()
	officialHdl := handler.NewOfficialHandler(officialSvc)

	return &Container{
		Config:          cfg,
		KeyStore:        keyStore,
		OfficialHandler: officialHdl,
	}
}