package config

import (
	"io"
	"os"

	"github.com/gofiber/fiber/v2"
	fiberlogger "github.com/gofiber/fiber/v2/middleware/logger"
	"gopkg.in/natefinch/lumberjack.v2"
)

func FiberLoggerMiddleware() fiber.Handler {

	logFile := &lumberjack.Logger{
		Filename:   "./logs/http.log",
		MaxSize:    10,   // MB
		MaxBackups: 5,    // number of backups
		MaxAge:     7,    // day
		Compress:   true, // compression
	}

	multiWriter := io.MultiWriter(os.Stdout, logFile)

	format := "${pid} | ${locals:requestid} | ${status} | ${latency} | ${ip} | ${method} | ${path}\n"
	timeFormat := "2006-01-02 15:04:05.000"

	loggerConfig := fiberlogger.Config{
		Output:     multiWriter,
		TimeFormat: timeFormat,
		Format:     format,
		TimeZone:   "Europe/Warsaw",
		// Callback after each log, such as sending 5xx to Slack
		Done: func(c *fiber.Ctx, logBytes []byte) {
			if c.Response().StatusCode() >= 500 {
				// reporter.SendToSlack(logBytes)
			}
		},
	}

	return fiberlogger.New(loggerConfig)
}
