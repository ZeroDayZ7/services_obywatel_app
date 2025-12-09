package shared

import (
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

// InitLogger inicjalizuje singleton loggera
func InitLogger(env string) *Logger {
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
	return instance
}

// GetLogger zwraca singleton
func GetLogger() *Logger {
	if instance == nil {
		InitLogger("development")
	}
	return instance
}

// --- Logowanie prostych komunikatów ---
func (l *Logger) Info(msg string) {
	l.Logger.WithOptions(zap.AddCallerSkip(1)).Info(msg)
}

func (l *Logger) Debug(msg string) {
	l.Logger.WithOptions(zap.AddCallerSkip(1)).Debug(msg)
}

func (l *Logger) Warn(msg string) {
	l.Logger.WithOptions(zap.AddCallerSkip(1)).Warn(msg)
}

func (l *Logger) Error(msg string) {
	l.Logger.WithOptions(zap.AddCallerSkip(1)).Error(msg)
}

// --- Logowanie map ---
func (l *Logger) InfoMap(msg string, m map[string]any) {
	fields := toFields(m)
	l.Logger.WithOptions(zap.AddCallerSkip(1)).Info(msg, fields...)
}

func (l *Logger) DebugMap(msg string, m map[string]any) {
	fields := toFields(m)
	l.Logger.WithOptions(zap.AddCallerSkip(1)).Debug(msg, fields...)
}

func (l *Logger) WarnMap(msg string, m map[string]any) {
	fields := toFields(m)
	l.Logger.WithOptions(zap.AddCallerSkip(1)).Warn(msg, fields...)
}

func (l *Logger) ErrorMap(msg string, m map[string]any) {
	fields := toFields(m)
	l.Logger.WithOptions(zap.AddCallerSkip(1)).Error(msg, fields...)
}

// --- Logowanie obiektów ---
func (l *Logger) InfoObj(msg string, obj any) {
	l.Logger.WithOptions(zap.AddCallerSkip(1)).Info(msg, zap.Any("data", obj))
}

func (l *Logger) DebugObj(msg string, obj any) {
	l.Logger.WithOptions(zap.AddCallerSkip(1)).Debug(msg, zap.Any("data", obj))
}

func (l *Logger) WarnObj(msg string, obj any) {
	l.Logger.WithOptions(zap.AddCallerSkip(1)).Warn(msg, zap.Any("data", obj))
}

func (l *Logger) ErrorObj(msg string, obj any) {
	l.Logger.WithOptions(zap.AddCallerSkip(1)).Error(msg, zap.Any("data", obj))
}

// --- Helper konwertujący mapy na zap.Fields ---
func toFields(m map[string]any) []zap.Field {
	fields := make([]zap.Field, 0, len(m))
	for k, v := range m {
		fields = append(fields, zap.Any(k, v))
	}
	return fields
}
