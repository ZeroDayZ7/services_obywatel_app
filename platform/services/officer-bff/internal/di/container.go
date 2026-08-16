package di

import (
	"github.com/zerodayz7/services/officer-bff/config"
	"github.com/zerodayz7/services/officer-bff/internal/handler"
	"github.com/zerodayz7/services/officer-bff/internal/service"
)

type Container struct {
	Config          *config.Config
	HealthHandler   *handler.HealthHandler
	OfficialHandler *handler.OfficialHandler
}

func BuildContainer(app *config.App) *Container {
	healthSvc := service.NewHealthService()
	officialSvc := service.NewOfficialService()

	healthHdl := handler.NewHealthHandler(healthSvc)
	officialHdl := handler.NewOfficialHandler(officialSvc)

	return &Container{
		Config:          app.Config,
		HealthHandler:   healthHdl,
		OfficialHandler: officialHdl,
	}
}
