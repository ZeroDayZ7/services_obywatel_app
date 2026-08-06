package config

import (
	"github.com/gofiber/fiber/v2/middleware/cors"
)

func CorsConfig() cors.Config {
	return cors.Config{
		AllowOrigins:     AppConfig.CORSAllowOrigins,
		AllowMethods:     "GET,POST,PUT,PATCH,DELETE,OPTIONS",
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization, X-CSRF-Token",
		AllowCredentials: false,
	}
}
