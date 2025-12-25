package server

import (
	"time"

	"github.com/gofiber/fiber/v2"
)

func NewFiberApp(cfg FiberConfig) *fiber.App {
	return fiber.New(fiber.Config{
		AppName:       cfg.AppName,
		ServerHeader:  cfg.ServerHeader,
		Prefork:       cfg.Prefork,
		CaseSensitive: cfg.CaseSensitive,
		StrictRouting: cfg.StrictRouting,
		IdleTimeout:   cfg.IdleTimeout,
		ReadTimeout:   cfg.ReadTimeout,
		WriteTimeout:  cfg.WriteTimeout,
		ErrorHandler:  ErrorHandler(),
	})
}

type FiberConfig struct {
	AppName       string
	ServerHeader  string
	Prefork       bool
	CaseSensitive bool
	StrictRouting bool
	IdleTimeout   time.Duration
	ReadTimeout   time.Duration
	WriteTimeout  time.Duration
}
