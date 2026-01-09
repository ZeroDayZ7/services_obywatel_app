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

		// consoleEncoderConfig := zapcore.EncoderConfig{
		// 	MessageKey:  "msg",
		// 	LevelKey:    "level",
		// 	TimeKey:     "time",
		// 	EncodeTime:  zapcore.ISO8601TimeEncoder,
		// 	EncodeLevel: zapcore.CapitalColorLevelEncoder,
		// }

		consoleEncoderConfig := zapcore.EncoderConfig{
			MessageKey:     "msg",
			LevelKey:       "level",
			TimeKey:        "time",
			NameKey:        "logger",
			CallerKey:      "caller",
			FunctionKey:    zapcore.OmitKey,
			StacktraceKey:  "stacktrace",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.CapitalColorLevelEncoder,
			EncodeTime:     zapcore.ISO8601TimeEncoder,
			EncodeDuration: zapcore.StringDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
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

// isSensitive sprawdza, czy nazwa pola sugeruje dane wrażliwe
func isSensitive(name string) bool {
	n := strings.ToLower(name)

	// Lista słów, które oznaczają dane wrażliwe
	sensitiveKeys := []string{
		"password",
		"token",
		"secret",
		"code",
		"authorization",
		"credential",
		"apikey",
	}

	for _, key := range sensitiveKeys {
		if strings.Contains(n, key) {
			return true
		}
	}

	return false
}

// --- Helper konwertujący mapy na zap.Fields (również z maskowaniem) ---
func toFields(m map[string]any) []zap.Field {
	fields := make([]zap.Field, 0, len(m))
	for k, v := range m {
		if isSensitive(k) {
			fields = append(fields, zap.String(k, "********"))
			continue
		}

		// Jeśli wartością jest kolejna mapa, spłaszczamy ją lub logujemy jako Any
		if v == nil {
			fields = append(fields, zap.String(k, "null"))
		} else {
			fields = append(fields, zap.Any(k, v))
		}
	}
	return fields
}

func (l *Logger) parseArgs(args ...any) []zap.Field {
	fields := []zap.Field{}
	for _, arg := range args {
		switch v := arg.(type) {
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

// --- Metody logowania ---

func (l *Logger) Info(msg string, args ...any) {
	l.Logger.Info(msg, l.parseArgs(args...)...)
}

func (l *Logger) Debug(msg string, args ...any) {
	l.Logger.Debug(msg, l.parseArgs(args...)...)
}

func (l *Logger) Warn(msg string, args ...any) {
	l.Logger.Warn(msg, l.parseArgs(args...)...)
}

func (l *Logger) Error(msg string, args ...any) {
	l.Logger.Error(msg, l.parseArgs(args...)...)
}

func (l *Logger) InfoMap(msg string, m map[string]any)  { l.Logger.Info(msg, toFields(m)...) }
func (l *Logger) DebugMap(msg string, m map[string]any) { l.Logger.Debug(msg, toFields(m)...) }
func (l *Logger) WarnMap(msg string, m map[string]any)  { l.Logger.Warn(msg, toFields(m)...) }
func (l *Logger) ErrorMap(msg string, m map[string]any) { l.Logger.Error(msg, toFields(m)...) }

func (l *Logger) InfoObj(msg string, obj any)  { l.Logger.Info(msg, convertStructToFields(obj)...) }
func (l *Logger) DebugObj(msg string, obj any) { l.Logger.Debug(msg, convertStructToFields(obj)...) }
func (l *Logger) WarnObj(msg string, obj any)  { l.Logger.Warn(msg, convertStructToFields(obj)...) }
func (l *Logger) ErrorObj(msg string, obj any) { l.Logger.Error(msg, convertStructToFields(obj)...) }

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

// Pomocnicza metoda do ładnego wypisywania zagnieżdżonych danych
func (l *Logger) printValue(f zap.Field, indent int) {
	spaces := strings.Repeat("  ", indent)

	switch f.Type {
	case zapcore.StringType:
		fmt.Printf(" %v\n", f.String)
	case zapcore.BoolType:
		fmt.Printf(" %v\n", f.Integer == 1)
	case zapcore.InlineMarshalerType, zapcore.ObjectMarshalerType:
		// Jeśli to zagnieżdżony obiekt
		fmt.Printf("\n")
		if subFields, ok := f.Interface.([]zap.Field); ok {
			for _, sf := range subFields {
				fmt.Printf("%s\x1b[36m%s\x1b[0m:", spaces+"  ", sf.Key)
				l.printValue(sf, indent+1)
			}
		}
	default:
		// Dla map i innych
		if f.Interface == nil {
			fmt.Printf(" null\n")
		} else {
			fmt.Printf(" %v\n", f.Interface)
		}
	}
}
