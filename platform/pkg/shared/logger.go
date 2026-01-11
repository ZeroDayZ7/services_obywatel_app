package shared

import (
	"fmt"
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

	SensitiveKeys = []string{
		"password",
		"token",
		"secret",
		"code",
		"authorization",
		"credential",
		"apikey",
	}
)

const (
	colorGreen  = "\x1b[32m"
	colorYellow = "\x1b[33m"
	colorRed    = "\x1b[31m"
)

func InitBootstrapLogger(env string) *Logger {
	var level zapcore.Level
	if strings.ToLower(env) == "production" {
		level = zapcore.InfoLevel
	} else {
		level = zapcore.DebugLevel
	}

	encoderConfig := zapcore.EncoderConfig{
		MessageKey: "msg", LevelKey: "level", TimeKey: "time",
		CallerKey: "caller", EncodeLevel: zapcore.CapitalColorLevelEncoder,
		EncodeTime: zapcore.ISO8601TimeEncoder, EncodeCaller: zapcore.ShortCallerEncoder,
	}

	// Konsola
	consoleCore := zapcore.NewCore(zapcore.NewConsoleEncoder(encoderConfig), zapcore.AddSync(os.Stdout), level)

	// Plik (Lumberjack)
	logFile := &lumberjack.Logger{
		Filename: "logs/bootstrap.log",
		MaxSize:  2, MaxBackups: 1, Compress: false,
	}
	fileCore := zapcore.NewCore(zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()), zapcore.AddSync(logFile), zapcore.InfoLevel)

	core := zapcore.NewTee(consoleCore, fileCore)
	return &Logger{zap.New(core, zap.AddCaller())}
}

// InitLogger inicjalizuje singleton loggera z dynamicznym poziomem logowania
func InitLogger(env string) *Logger {
	once.Do(func() {
		isProd := strings.ToLower(env) == "production"

		var consoleLevel zapcore.Level
		if isProd {
			consoleLevel = zapcore.InfoLevel
		} else {
			consoleLevel = zapcore.DebugLevel
		}

		// --- 1. Konfiguracja dla Konsoli ---
		var consoleEncoder zapcore.Encoder

		if isProd {
			// Na PRODUKCJI
			consoleEncoder = zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())
		} else {
			// W DEVIE
			consoleEncoderConfig := zapcore.EncoderConfig{
				MessageKey:     "msg",
				LevelKey:       "level",
				TimeKey:        "",
				NameKey:        "logger",
				CallerKey:      "",
				LineEnding:     zapcore.DefaultLineEnding,
				EncodeLevel:    zapcore.CapitalColorLevelEncoder,
				EncodeTime:     zapcore.ISO8601TimeEncoder,
				EncodeDuration: zapcore.StringDurationEncoder,
				EncodeCaller:   zapcore.ShortCallerEncoder,
			}
			consoleEncoder = zapcore.NewConsoleEncoder(consoleEncoderConfig)
		}

		consoleCore := zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stdout), consoleLevel)

		// --- 2. Konfiguracja dla Pliku (Zawsze JSON) ---
		logFile := &lumberjack.Logger{
			Filename:   "logs/app.log",
			MaxSize:    10,
			MaxBackups: 5,
			Compress:   true,
		}
		fileCore := zapcore.NewCore(
			zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
			zapcore.AddSync(logFile),
			zap.InfoLevel,
		)

		// --- 3. Połączenie ---
		core := zapcore.NewTee(consoleCore, fileCore)

		// AddCallerSkip(1) jest ważne, żeby w logach widzieć miejsce wywołania log.Info,
		// a nie wnętrze Twojej paczki shared/logger
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

// --- Metody logowania ---
// region METODY
func (l *Logger) Info(msg string, args ...any)  { l.Logger.Info(msg, l.parseArgs(args...)...) }
func (l *Logger) Debug(msg string, args ...any) { l.Logger.Debug(msg, l.parseArgs(args...)...) }
func (l *Logger) Warn(msg string, args ...any)  { l.Logger.Warn(msg, l.parseArgs(args...)...) }
func (l *Logger) Error(msg string, args ...any) { l.Logger.Error(msg, l.parseArgs(args...)...) }

func (l *Logger) InfoMap(msg string, m map[string]any)  { l.Logger.Info(msg, toFields(m)...) }
func (l *Logger) DebugMap(msg string, m map[string]any) { l.Logger.Debug(msg, toFields(m)...) }
func (l *Logger) WarnMap(msg string, m map[string]any)  { l.Logger.Warn(msg, toFields(m)...) }
func (l *Logger) ErrorMap(msg string, m map[string]any) { l.Logger.Error(msg, toFields(m)...) }

func (l *Logger) InfoObj(msg string, obj any)  { l.Logger.Info(msg, convertStructToFields(obj)...) }
func (l *Logger) DebugObj(msg string, obj any) { l.Logger.Debug(msg, convertStructToFields(obj)...) }
func (l *Logger) WarnObj(msg string, obj any)  { l.Logger.Warn(msg, convertStructToFields(obj)...) }
func (l *Logger) ErrorObj(msg string, obj any) { l.Logger.Error(msg, convertStructToFields(obj)...) }

// region DEBUG
func (l *Logger) DebugPretty(msg string, m map[string]any) {
	l.Logger.Debug(msg, toFields(m)...)
}

func (l *Logger) DebugEmpty(msg string, key string) {
	l.Logger.Debug(msg, zap.String(key, "NULL/EMPTY ∅"))
}

// Dodaj to do shared/logger.go
func (l *Logger) DebugResponse(msg string, res any) {
	if !l.Core().Enabled(zap.DebugLevel) {
		return
	}

	fmt.Printf("\n\x1b[35m--- [DEBUG] Outgoing Response ---\x1b[0m\n")
	fmt.Printf("Message: %s\n", msg)

	fields := convertStructToFields(res)
	for _, f := range fields {
		fmt.Printf("  \x1b[32m%s\x1b[0m:", f.Key)
		l.printValue(f, 1) // 1 to poziom wcięcia
	}
	fmt.Printf("\x1b[35m---------------------------------\x1b[0m\n\n")
}

// DebugRequest ładnie formatuje przetworzone żądanie w konsoli (kolory ANSI)
func (l *Logger) DebugRequest(msg string, method, path string, status int, latency string, body any) {
	if !l.Core().Enabled(zapcore.DebugLevel) {
		return
	}

	fmt.Printf("\n\x1b[34m--- [DEBUG] HTTP Request Processed ---\x1b[0m\n")
	fmt.Printf("Message: %s\n", msg)
	fmt.Printf("  \x1b[32mMethod:\x1b[0m   %s\n", method)
	fmt.Printf("  \x1b[32mPath:\x1b[0m     %s\n", path)
	fmt.Printf("  \x1b[32mStatus:\x1b[0m   %d\n", status)
	fmt.Printf("  \x1b[32mLatency:\x1b[0m  %s\n", latency)

	if body != nil {
		fmt.Printf("  \x1b[32mBody:\x1b[0m")
		fields := convertStructToFields(body)
		for _, f := range fields {
			fmt.Printf("\n    \x1b[36m%s\x1b[0m:", f.Key)
			l.printValue(f, 0)
		}
		fmt.Println()
	}

	fmt.Printf("\x1b[34m--------------------------------------\x1b[0m\n\n")
}

// DebugInfo - Zielona ramka (idealna do kodów 2FA, sukcesów w testach)
func (l *Logger) DebugInfo(msg string, data any) {
	l.printFramedLog("INFO", msg, data, colorGreen)
}

// DebugWarn - Żółta ramka (ostrzeżenia, ważne punkty w logice)
func (l *Logger) DebugWarn(msg string, data any) {
	l.printFramedLog("WARN", msg, data, colorYellow)
}

// DebugError - Czerwona ramka (błędy, które chcesz widzieć wizualnie)
func (l *Logger) DebugError(msg string, data any) {
	l.printFramedLog("ERROR", msg, data, colorRed)
}

// region FATAL
func (l *Logger) Fatal(msg string, args ...any) {
	l.Logger.Fatal(msg, l.parseArgs(args...)...)
}

func (l *Logger) FatalMap(msg string, m map[string]any) {
	l.Logger.Fatal(msg, toFields(m)...)
}

func (l *Logger) FatalObj(msg string, obj any) {
	l.Logger.Fatal(msg, convertStructToFields(obj)...)
}

// region = HELPERY =

// region printFramedLog
func (l *Logger) printFramedLog(label, msg string, data any, color string) {
	if !l.Core().Enabled(zapcore.DebugLevel) {
		return
	}

	fmt.Printf("\n%s--- [DEBUG %s] %s ---\x1b[0m\n", color, label, strings.Repeat("-", 10))
	fmt.Printf("Message: %s", msg)

	if data != nil {
		fields := convertStructToFields(data)
		for _, f := range fields {
			fmt.Printf("\n  \x1b[32m%s\x1b[0m:", f.Key)
			l.printValue(f, 1)
		}
		fmt.Println()
	}
	fmt.Printf("%s------------------------------------------\x1b[0m\n\n", color)
}

// region printValue
func (l *Logger) printValue(f zap.Field, indent int) {
	switch f.Type {
	case zapcore.StringType:
		fmt.Printf(" %v", f.String)
	case zapcore.BoolType:
		fmt.Printf(" %v", f.Integer == 1)
	case zapcore.InlineMarshalerType, zapcore.ObjectMarshalerType:
		if subFields, ok := f.Interface.([]zap.Field); ok {
			// Przy zagnieżdżeniu robimy nową linię i wcięcie
			for _, sf := range subFields {
				spaces := strings.Repeat("  ", indent+3)
				fmt.Printf("\n%s\x1b[36m%s\x1b[0m:", spaces, sf.Key)
				l.printValue(sf, indent+1)
			}
		} else {
			fmt.Print(" {}")
		}
	default:
		if f.Interface == nil {
			fmt.Print(" null")
		} else {
			fmt.Printf(" %v", f.Interface)
		}
	}
}

// region convertStructToFields
func convertStructToFields(obj any) []zap.Field {
	fields := []zap.Field{}

	// 🔹 Obsługa nil
	if obj == nil {
		fields = append(fields, zap.Any("value", nil))
		return fields
	}

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

			if isSensitive(k) {
				fields = append(fields, zap.String(k, "********"))
			} else if childMap, ok := v.(map[string]any); ok {
				childFields := convertStructToFields(childMap)
				fields = append(fields, zap.Any(k, childFields))
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
				continue
			}

			if isSensitive(fieldName) {
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

// region isSensitive
func isSensitive(name string) bool {
	n := strings.ToLower(name)

	for _, key := range SensitiveKeys {
		if strings.Contains(n, key) {
			return true
		}
	}

	return false
}

// region toFields
// --- Helper konwertujący mapy na zap.Fields (również z maskowaniem) ---
func toFields(m map[string]any) []zap.Field {
	fields := make([]zap.Field, 0, len(m))
	for k, v := range m {
		if isSensitive(k) {
			fields = append(fields, zap.String(k, "********"))
			continue
		}

		if v == nil {
			fields = append(fields, zap.String(k, "null"))
		} else {
			fields = append(fields, zap.Any(k, v))
		}
	}
	return fields
}

// region parseArgs
func (l *Logger) parseArgs(args ...any) []zap.Field {
	fields := []zap.Field{}
	for _, arg := range args {
		switch v := arg.(type) {
		case error:
			fields = append(fields, zap.Error(v))
		case zap.Field:
			fields = append(fields, v)
		case map[string]any:
			fields = append(fields, toFields(v)...)
		default:
			fields = append(fields, convertStructToFields(v)...)
		}
	}
	return fields
}
