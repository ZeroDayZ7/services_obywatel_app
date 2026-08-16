package di

import (
	"github.com/zerodayz7/services/officer-bff/config"
	"github.com/zerodayz7/services/officer-bff/internal/handler"
	"github.com/zerodayz7/services/officer-bff/internal/service"
)

type Container struct {
	Config          *config.Config
	OfficialHandler *handler.OfficialHandler
}

func BuildContainer(app *config.App) *Container {
	officialSvc := service.NewOfficialService()

	officialHdl := handler.NewOfficialHandler(officialSvc)

	return &Container{
		Config:          app.Config,
		OfficialHandler: officialHdl,
	}
}
