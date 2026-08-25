package shared

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"
	"sync"
	"time"

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
		"secret",
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

//#region InitBootstrapLogger
func InitBootstrapLogger(env string, saveToFile bool) *Logger {
	env = strings.ToLower(env)
	isProd := env == "production" || env == "staging"

	var level zapcore.Level
	if isProd {
		level = zapcore.InfoLevel
	} else {
		level = zapcore.DebugLevel
	}

	var encoder zapcore.Encoder
	if isProd {
		encoderConfig := zap.NewProductionEncoderConfig()
		encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	} else {
		encoderConfig := zapcore.EncoderConfig{
			MessageKey:   "msg",
			LevelKey:     "level",
			TimeKey:      "time",
			CallerKey:    "caller",
			EncodeLevel:  zapcore.CapitalColorLevelEncoder,
			EncodeTime:   zapcore.ISO8601TimeEncoder,
			EncodeCaller: zapcore.ShortCallerEncoder,
		}
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	}

	consoleCore := zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), level)
	cores := []zapcore.Core{consoleCore}

	if saveToFile && !isProd {
		if _, err := os.Stat("logs"); os.IsNotExist(err) {
			_ = os.Mkdir("logs", 0755)
		}

		logFile := &lumberjack.Logger{
			Filename:   "logs/bootstrap.log",
			MaxSize:    2,
			MaxBackups: 1,
			Compress:   false,
		}
		fileCore := zapcore.NewCore(
			zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
			zapcore.AddSync(logFile),
			zapcore.InfoLevel,
		)
		cores = append(cores, fileCore)
	}

	core := zapcore.NewTee(cores...)
	return &Logger{zap.New(core, zap.AddCaller())}
}

//#region InitLogger
func InitLogger(env string, saveToFile bool) *Logger {
	once.Do(func() {
		env = strings.ToLower(env)
		isProd := env == "production" || env == "staging"

		var consoleLevel zapcore.Level
		if isProd {
			consoleLevel = zapcore.InfoLevel
		} else {
			consoleLevel = zapcore.DebugLevel
		}

		var consoleEncoder zapcore.Encoder
		if isProd {
			encoderConfig := zap.NewProductionEncoderConfig()
			encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
			consoleEncoder = zapcore.NewJSONEncoder(encoderConfig)
		} else {
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
		cores := []zapcore.Core{consoleCore}
		if saveToFile && !isProd {
			if _, err := os.Stat("logs"); os.IsNotExist(err) {
				_ = os.Mkdir("logs", 0755)
			}

			logFile := &lumberjack.Logger{
				Filename:   "logs/app.log",
				MaxSize:    10, // MB
				MaxBackups: 5,
				Compress:   true,
			}

			fileCore := zapcore.NewCore(
				zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
				zapcore.AddSync(logFile),
				zap.InfoLevel,
			)
			cores = append(cores, fileCore)
		}

		core := zapcore.NewTee(cores...)

		zapLogger := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
		instance = &Logger{zapLogger}
	})
	return instance
}

//#region GetLogger
func GetLogger() *Logger {
	if instance == nil {
		InitLogger("development", false)
	}
	return instance
}

// --- Metody logowania ---
// region METODY
//#region Info
func (l *Logger) Info(msg string, args ...any)  { l.Logger.Info(msg, l.parseArgs(args...)...) }
//#region Debug
func (l *Logger) Debug(msg string, args ...any) { l.Logger.Debug(msg, l.parseArgs(args...)...) }
//#region Warn
func (l *Logger) Warn(msg string, args ...any)  { l.Logger.Warn(msg, l.parseArgs(args...)...) }
//#region Error
func (l *Logger) Error(msg string, args ...any) { l.Logger.Error(msg, l.parseArgs(args...)...) }

//#region InfoMap
func (l *Logger) InfoMap(msg string, m map[string]any)  { l.Logger.Info(msg, toFields(m)...) }
//#region DebugMap
func (l *Logger) DebugMap(msg string, m map[string]any) { l.Logger.Debug(msg, toFields(m)...) }
//#region WarnMap
func (l *Logger) WarnMap(msg string, m map[string]any)  { l.Logger.Warn(msg, toFields(m)...) }
//#region ErrorMap
func (l *Logger) ErrorMap(msg string, m map[string]any) { l.Logger.Error(msg, toFields(m)...) }

//#region InfoObj
func (l *Logger) InfoObj(msg string, obj any)  { l.Logger.Info(msg, convertStructToFields(obj)...) }
//#region DebugObj
func (l *Logger) DebugObj(msg string, obj any) { l.Logger.Debug(msg, convertStructToFields(obj)...) }
//#region WarnObj
func (l *Logger) WarnObj(msg string, obj any)  { l.Logger.Warn(msg, convertStructToFields(obj)...) }
//#region ErrorObj
func (l *Logger) ErrorObj(msg string, obj any) { l.Logger.Error(msg, convertStructToFields(obj)...) }

// region DEBUG
const colorCyan = "\x1b[36m"

// DebugDB - Niebieska ramka (idealna do logowania modeli z bazy, query result itp.)
//#region DebugDB
func (l *Logger) DebugDB(msg string, data any) {
	if !l.Core().Enabled(zapcore.DebugLevel) {
		return
	}

	fmt.Printf("\n%s--- [DATABASE] %s ---\x1b[0m\n", colorCyan, strings.Repeat("-", 10))
	fmt.Printf("Action: %s", msg)

	if data != nil {
		fields := convertStructToFields(data)
		for _, f := range fields {
			fmt.Printf("\n  \x1b[32m%s\x1b[0m:", f.Key)
			l.printValue(f, 1)
		}
		fmt.Println()
	}
	fmt.Printf("%s------------------------------------------\x1b[0m\n\n", colorCyan)
}

//#region DebugPretty
func (l *Logger) DebugPretty(msg string, m map[string]any) {
	l.Logger.Debug(msg, toFields(m)...)
}

//#region DebugEmpty
func (l *Logger) DebugEmpty(msg string, key string) {
	l.Logger.Debug(msg, zap.String(key, "NULL/EMPTY ∅"))
}

// Dodaj to do shared/logger.go
//#region DebugResponse
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
//#region DebugRequest
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
//#region DebugInfo
func (l *Logger) DebugInfo(msg string, data any) {
	l.printFramedLog("INFO", msg, data, colorGreen)
}

// DebugWarn - Żółta ramka (ostrzeżenia, ważne punkty w logice)
//#region DebugWarn
func (l *Logger) DebugWarn(msg string, data any) {
	l.printFramedLog("WARN", msg, data, colorYellow)
}

// DebugError - Czerwona ramka (błędy, które chcesz widzieć wizualnie)
//#region DebugError
func (l *Logger) DebugError(msg string, data any) {
	l.printFramedLog("ERROR", msg, data, colorRed)
}

//#region DebugJSON
func (l *Logger) DebugJSON(msg string, obj any) {
	if !l.Core().Enabled(zapcore.DebugLevel) {
		return
	}

	// Marshalling do JSONa z wcięciami
	jsonBytes, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		fmt.Printf("\n\x1b[31m--- [DEBUG JSON ERROR] ---\x1b[0m\n")
		fmt.Printf("Message: %s\nError: %v\n", msg, err)
		return
	}

	fmt.Printf("\n\x1b[35m--- [DEBUG RAW JSON] ---\x1b[0m\n")
	fmt.Printf("Message: \x1b[33m%s\x1b[0m\n", msg)
	fmt.Printf("\x1b[32m%s\x1b[0m\n", string(jsonBytes)) // Zielony kolor dla treści JSON
	fmt.Printf("\x1b[35m------------------------\x1b[0m\n\n")
}

// ===========================================
// region FATAL
// ===========================================
//#region Fatal
func (l *Logger) Fatal(msg string, args ...any) {
	l.Logger.Fatal(msg, l.parseArgs(args...)...)
}

//#region FatalMap
func (l *Logger) FatalMap(msg string, m map[string]any) {
	l.Logger.Fatal(msg, toFields(m)...)
}

//#region FatalObj
func (l *Logger) FatalObj(msg string, obj any) {
	l.Logger.Fatal(msg, convertStructToFields(obj)...)
}

// region = HELPERY =

// region printFramedLog
//#region printFramedLog
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
//#region printValue
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
//#region convertStructToFields
func convertStructToFields(obj any) []zap.Field {
	fields := []zap.Field{}

	if obj == nil {
		fields = append(fields, zap.Any("value", nil))
		return fields
	}

	val := reflect.ValueOf(obj)
	// Obsługa wskaźników
	if val.Kind() == reflect.Pointer {
		if val.IsNil() {
			fields = append(fields, zap.Any("value", nil))
			return fields
		}
		val = val.Elem()
	}

	if !val.IsValid() {
		return fields
	}

	// 🔹 SPECJALNA OBSŁUGA CZASU (aby uniknąć napisu "Local")
	if t, ok := val.Interface().(time.Time); ok {
		return []zap.Field{zap.String("time", t.Format("2006-01-02 15:04:05"))}
	}

	typ := val.Type()

	switch val.Kind() {
	case reflect.Map:
		for _, key := range val.MapKeys() {
			k := key.String()
			v := val.MapIndex(key).Interface()
			if isSensitive(k) {
				fields = append(fields, zap.String(k, "********"))
			} else {
				fields = append(fields, zap.Any(k, v))
			}
		}
	case reflect.Struct:
		for i := 0; i < val.NumField(); i++ {
			field := typ.Field(i)
			fieldVal := val.Field(i)

			if !fieldVal.CanInterface() {
				continue
			}

			name := field.Name
			value := fieldVal.Interface()

			// 🔹 Obsługa dat wewnątrz struktur
			if t, ok := value.(time.Time); ok {
				fields = append(fields, zap.String(name, t.Format("2006-01-02 15:04:05")))
				continue
			}

			if isSensitive(name) {
				fields = append(fields, zap.String(name, "********"))
			} else {
				fields = append(fields, zap.Any(name, value))
			}
		}
	default:
		fields = append(fields, zap.Any("value", obj))
	}

	return fields
}

// region isSensitive
//#region isSensitive
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
//#region toFields
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
//#region parseArgs
func (l *Logger) parseArgs(args ...any) []zap.Field {
	var fields []zap.Field
	n := len(args)

	for i := 0; i < n; i++ {
		arg := args[i]

		switch v := arg.(type) {
		case error:
			fields = append(fields, zap.Error(v))
		case zap.Field:
			fields = append(fields, v)
		case map[string]any:
			fields = append(fields, toFields(v)...)
		case string:
			if i+1 < n {
				val := args[i+1]
				i++

				if isSensitive(v) {
					fields = append(fields, zap.String(v, "********"))
				} else {
					fields = append(fields, zap.Any(v, val))
				}
			} else {
				fields = append(fields, zap.String("extra", v))
			}
		default:
			fields = append(fields, convertStructToFields(v)...)
		}
	}

	return fields
}
