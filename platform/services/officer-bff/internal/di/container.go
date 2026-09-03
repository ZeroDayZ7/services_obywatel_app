package di

import (
	"github.com/zerodayz7/platform/pkg/httpserver"
	"github.com/zerodayz7/services/officer-bff/config"
)

type Container struct {
	Config   *config.Config
	KeyStore *httpserver.KeyStore
}

//#region BuildContainer
func BuildContainer(cfg *config.Config, keyStore *httpserver.KeyStore) *Container {
	return &Container{
		Config:   cfg,
		KeyStore: keyStore,
	}
}
