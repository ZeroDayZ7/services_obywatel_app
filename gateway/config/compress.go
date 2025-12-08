package config

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
)

func CompressConfig() compress.Config {
	return compress.Config{
		Level: compress.LevelBestSpeed,
		Next: func(c *fiber.Ctx) bool {
			ct := string(c.Response().Header.ContentType())
			return ct == "image/png" || ct == "image/jpeg" ||
				ct == "video/mp4" || ct == "application/pdf"
		},
	}
}
