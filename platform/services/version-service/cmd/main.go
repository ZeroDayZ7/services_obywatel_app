package main

import (
	"log"
	"net/http"

	"github.com/zerodayz7/platform/pkg/server"
	"github.com/zerodayz7/platform/services/version-service/config"
	"github.com/zerodayz7/platform/services/version-service/internal/router"
)

func main() {
	cfg := config.Get()

	mux := http.NewServeMux()
	router.RegisterRoutes(mux)

	srv := server.New(mux)
	log.Printf("version-service running on :%s\n", cfg.Port)

	if err := srv.Start(":" + cfg.Port); err != nil {
		log.Fatal(err)
	}
}
