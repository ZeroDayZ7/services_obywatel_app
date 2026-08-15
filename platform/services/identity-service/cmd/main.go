package main

import (
	"log"
	"net/http"
	"time"

	"github.com/zerodayz7/services/identity-service/config"
	"github.com/zerodayz7/services/identity-service/internal/router"
)

func main() {
	cfg := config.Load()

	r := router.NewRouter()

	server := &http.Server{
		Addr: ":" + cfg.Port,

		Handler: r,

		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Printf(
		"%s running on port %s",
		cfg.AppName,
		cfg.Port,
	)

	err := server.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}
}
