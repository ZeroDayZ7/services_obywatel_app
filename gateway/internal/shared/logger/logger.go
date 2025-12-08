package logger

import (
	"fmt"
	"os"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

type Logger struct {
	*zap.Logger
}

var (
	instance *Logger
	once     sync.Once
)

func InitLogger(env string) (*Logger, error) {
	var err error
	once.Do(func() {
		consoleEncoderConfig := zapcore.EncoderConfig{
			MessageKey:  "msg",
			LevelKey:    "level",
			TimeKey:     "",
			NameKey:     "",
			CallerKey:   "",
			EncodeLevel: zapcore.CapitalColorLevelEncoder,
		}

		consoleCore := zapcore.NewCore(
			zapcore.NewConsoleEncoder(consoleEncoderConfig),
			zapcore.AddSync(os.Stdout),
			zapcore.DebugLevel,
		)

		logFile := &lumberjack.Logger{
			Filename:   "logs/app.log",
			MaxSize:    10,
			MaxBackups: 5,
			MaxAge:     7,
			Compress:   true,
		}

		fileCore := zapcore.NewCore(
			zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
			zapcore.AddSync(logFile),
			zapcore.InfoLevel,
		)

		core := zapcore.NewTee(consoleCore, fileCore)

		zapLogger := zap.New(core)
		instance = &Logger{zapLogger}
	})
	return instance, err
}

func GetLogger() *Logger {
	if instance == nil {
		InitLogger("development")
	}
	return instance
}

func (l *Logger) Info(msg string, fields ...zap.Field) {
	l.Logger.WithOptions(zap.AddCallerSkip(1)).Info(msg, fields...)
}

func (l *Logger) Debug(msg string, fields ...zap.Field) {
	l.Logger.WithOptions(zap.AddCallerSkip(1)).Debug(msg, fields...)
}

func (l *Logger) Warn(msg string, fields ...zap.Field) {
	l.Logger.WithOptions(zap.AddCallerSkip(1)).Warn(msg, fields...)
}

func (l *Logger) Error(msg string, fields ...zap.Field) {
	l.Logger.WithOptions(zap.AddCallerSkip(1)).Error(msg, fields...)
}

func (l *Logger) InfoObj(msg string, obj any) {
	l.Logger.WithOptions(zap.AddCallerSkip(1)).Info(msg, zap.Any("data", obj))
}

func (l *Logger) WarnObj(msg string, obj any) {
	l.Logger.WithOptions(zap.AddCallerSkip(1)).Warn(msg, zap.Any("data", obj))
}

func (l *Logger) ErrorObj(msg string, obj any) {
	l.Logger.WithOptions(zap.AddCallerSkip(1)).Error(msg, zap.Any("data", obj))
}

func (l *Logger) DebugObj(msg string, obj any) {
	l.Logger.WithOptions(zap.AddCallerSkip(1)).Debug(msg, zap.Any("data", obj))
}

func (l *Logger) InfoMap(msg string, m map[string]any) {
	l.Logger.WithOptions(zap.AddCallerSkip(1)).Info(msg)
	for k, v := range m {
		l.Logger.WithOptions(zap.AddCallerSkip(1)).Info(fmt.Sprintf("  %s: %v", k, v))
	}
}

func (l *Logger) DebugMap(msg string, fields map[string]string) {
	zapFields := make([]zap.Field, 0, len(fields))
	for k, v := range fields {
		zapFields = append(zapFields, zap.String(k, v))
	}
	l.Logger.WithOptions(zap.AddCallerSkip(1)).Debug(msg, zapFields...)
}

func (l *Logger) WarnMap(msg string, fields map[string]string) {
	zapFields := make([]zap.Field, 0, len(fields))
	for k, v := range fields {
		zapFields = append(zapFields, zap.String(k, v))
	}
	l.Logger.WithOptions(zap.AddCallerSkip(1)).Warn(msg, zapFields...)
}

func (l *Logger) ErrorMap(msg string, fields map[string]string) {
	zapFields := make([]zap.Field, 0, len(fields))
	for k, v := range fields {
		zapFields = append(zapFields, zap.String(k, v))
	}
	l.Logger.WithOptions(zap.AddCallerSkip(1)).Error(msg, zapFields...)
}
