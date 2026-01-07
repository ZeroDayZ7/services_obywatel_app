package shared

import (
	"os"
	"reflect"
	"strings"
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

// InitLogger inicjalizuje singleton loggera z dynamicznym poziomem logowania
func InitLogger(env string) *Logger {
	once.Do(func() {
		var consoleLevel zapcore.Level
		if strings.ToLower(env) == "production" {
			consoleLevel = zapcore.InfoLevel
		} else {
			consoleLevel = zapcore.DebugLevel
		}

		consoleEncoderConfig := zapcore.EncoderConfig{
			MessageKey:  "msg",
			LevelKey:    "level",
			TimeKey:     "time",
			EncodeTime:  zapcore.ISO8601TimeEncoder,
			EncodeLevel: zapcore.CapitalColorLevelEncoder,
		}

		// Rdzeń dla Konsoli
		consoleCore := zapcore.NewCore(
			zapcore.NewConsoleEncoder(consoleEncoderConfig),
			zapcore.AddSync(os.Stdout),
			consoleLevel,
		)

		// Konfiguracja rotacji plików (Lumberjack)
		logFile := &lumberjack.Logger{
			Filename:   "logs/app.log",
			MaxSize:    10, // megabytes
			MaxBackups: 5,
			MaxAge:     7, // days
			Compress:   true,
		}

		// Rdzeń dla Pliku (zawsze JSON, zawsze od Info wzwyż dla wydajności)
		fileCore := zapcore.NewCore(
			zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
			zapcore.AddSync(logFile),
			zapcore.InfoLevel,
		)

		// Połączenie rdzeni
		core := zapcore.NewTee(consoleCore, fileCore)

		// Dodajemy AddCaller, aby widzieć linię kodu, z której pochodzi log
		zapLogger := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
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

// convertStructToFields zamienia struct/map na []zap.Field z MASKOWANIEM SEKRETÓW
func convertStructToFields(obj any) []zap.Field {
	fields := []zap.Field{}

	// 🔹 Obsługa nil
	if obj == nil {
		fields = append(fields, zap.Any("value", nil))
		return fields
	}

	isDev := instance.Core().Enabled(zap.DebugLevel)

	val := reflect.ValueOf(obj)
	if !val.IsValid() {
		fields = append(fields, zap.Any("value", nil))
		return fields
	}

	// Obsługa wskaźników
	if val.Kind() == reflect.Pointer {
		if val.IsNil() {
			fields = append(fields, zap.Any("value", nil))
			return fields
		}
		val = val.Elem()
	}

	typ := val.Type()

	switch val.Kind() {
	case reflect.Map:
		for _, key := range val.MapKeys() {
			k := key.String()
			v := val.MapIndex(key).Interface()
			if !isDev && isSensitive(k) {
				fields = append(fields, zap.String(k, "********"))
			} else {
				fields = append(fields, zap.Any(k, v))
			}
		}
	case reflect.Struct:
		for i := 0; i < val.NumField(); i++ {
			field := typ.Field(i)
			fieldName := field.Name
			fieldVal := val.Field(i)
			if !fieldVal.CanInterface() {
				continue // ⚠️ Pomijamy pola, które nie da się interfejsować
			}

			if !isDev && isSensitive(fieldName) {
				fields = append(fields, zap.String(fieldName, "********"))
			} else {
				fields = append(fields, zap.Any(fieldName, fieldVal.Interface()))
			}
		}
	default:
		// Dla stringów, intów, itp.
		fields = append(fields, zap.Any("value", obj))
	}

	return fields
}

// isSensitive sprawdza, czy nazwa pola sugeruje dane wrażliwe
func isSensitive(name string) bool {
	n := strings.ToLower(name)
	return strings.Contains(n, "password") ||
		strings.Contains(n, "token") ||
		strings.Contains(n, "secret") ||
		strings.Contains(n, "code")
}

// --- Helper konwertujący mapy na zap.Fields (również z maskowaniem) ---
func toFields(m map[string]any) []zap.Field {
	isDev := instance.Core().Enabled(zap.DebugLevel)
	fields := make([]zap.Field, 0, len(m))
	for k, v := range m {
		if !isDev && isSensitive(k) {
			fields = append(fields, zap.String(k, "********"))
		} else {
			fields = append(fields, zap.Any(k, v))
		}
	}
	return fields
}

// --- Metody logowania ---

func (l *Logger) Info(msg string)  { l.Logger.Info(msg) }
func (l *Logger) Debug(msg string) { l.Logger.Debug(msg) }
func (l *Logger) Warn(msg string)  { l.Logger.Warn(msg) }
func (l *Logger) Error(msg string) { l.Logger.Error(msg) }

func (l *Logger) InfoMap(msg string, m map[string]any)  { l.Logger.Info(msg, toFields(m)...) }
func (l *Logger) DebugMap(msg string, m map[string]any) { l.Logger.Debug(msg, toFields(m)...) }
func (l *Logger) WarnMap(msg string, m map[string]any)  { l.Logger.Warn(msg, toFields(m)...) }
func (l *Logger) ErrorMap(msg string, m map[string]any) { l.Logger.Error(msg, toFields(m)...) }

func (l *Logger) InfoObj(msg string, obj any)  { l.Logger.Info(msg, convertStructToFields(obj)...) }
func (l *Logger) DebugObj(msg string, obj any) { l.Logger.Debug(msg, convertStructToFields(obj)...) }
func (l *Logger) WarnObj(msg string, obj any)  { l.Logger.Warn(msg, convertStructToFields(obj)...) }
func (l *Logger) ErrorObj(msg string, obj any) { l.Logger.Error(msg, convertStructToFields(obj)...) }
