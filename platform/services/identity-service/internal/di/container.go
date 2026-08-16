package di

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/zerodayz7/services/identity-service/config"
	"github.com/zerodayz7/services/identity-service/internal/handler"
	"github.com/zerodayz7/services/identity-service/internal/repository"
	"github.com/zerodayz7/services/identity-service/internal/service"
)

type Container struct {
	Config         *config.Config
	DB             *pgxpool.Pool
	HealthHandler  *handler.HealthHandler
	CitizenHandler *handler.CitizenHandler
}

func BuildContainer(app *config.App) *Container {
	// 1. Repositories
	healthRepo := repository.NewHealthRepository(app.DB)
	citizenRepo := repository.NewCitizenRepository(app.DB)

	// 2. Services
	healthSvc := service.NewHealthService(healthRepo)
	citizenSvc := service.NewCitizenService(citizenRepo)

	// 3. Handlers
	healthHdl := handler.NewHealthHandler(healthSvc)
	citizenHdl := handler.NewCitizenHandler(citizenSvc)

	return &Container{
		Config:         app.Config,
		DB:             app.DB,
		HealthHandler:  healthHdl,
		CitizenHandler: citizenHdl,
	}
}
