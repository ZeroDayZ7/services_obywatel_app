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
	CitizenHandler *handler.CitizenHandler
}

func BuildContainer(app *config.App) *Container {
	// 1. Repositories
	citizenRepo := repository.NewCitizenRepository(app.DB)

	// 2. Services
	citizenSvc := service.NewCitizenService(citizenRepo)

	// 3. Handlers
	citizenHdl := handler.NewCitizenHandler(citizenSvc)

	return &Container{
		Config:         app.Config,
		DB:             app.DB,
		CitizenHandler: citizenHdl,
	}
}
