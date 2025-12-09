package config

import (
	"github.com/gofiber/fiber/v2/middleware/cors"
)

func CorsConfig() cors.Config {
	allowOrigins := AppConfig.CORSAllow
	return cors.Config{
		AllowOrigins:     allowOrigins,
		AllowMethods:     "GET,POST,PUT,DELETE",
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization, X-CSRF-TOKEN",
		AllowCredentials: false,
	}
}
