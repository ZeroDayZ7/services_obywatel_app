package httpserver

import (
	"bytes"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// statusResponseWriter to wrapper na http.ResponseWriter do przechwytywania statusu i treści odpowiedzi.
type responseWriterInterceptor struct {
	http.ResponseWriter
	statusCode int
	body       *bytes.Buffer
}

func newResponseWriterInterceptor(w http.ResponseWriter) *responseWriterInterceptor {
	return &responseWriterInterceptor{
		ResponseWriter: w,
		statusCode:     http.StatusOK, // domyślnie 200 OK
		body:           &bytes.Buffer{},
	}
}

func (rwi *responseWriterInterceptor) WriteHeader(statusCode int) {
	rwi.statusCode = statusCode
	rwi.ResponseWriter.WriteHeader(statusCode)
}

func (rwi *responseWriterInterceptor) Write(b []byte) (int, error) {
	rwi.body.Write(b) // zapisujemy kopię do debuggowania
	return rwi.ResponseWriter.Write(b)
}

// LoggerMiddleware zwraca handler HTTP logujący szczegóły nadchodzącego żądania i odpowiedzi.
func LoggerMiddleware(logger LoggerInterface) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Odczyt i odtworzenie Request Body (aby nie zablokować późniejszego czytania w handlerach)
			var reqBodyBytes []byte
			if r.Body != nil {
				reqBodyBytes, _ = io.ReadAll(r.Body)
				r.Body = io.NopCloser(bytes.NewBuffer(reqBodyBytes))
			}

			// Interceptor do przechwytywania odpowiedzi
			interceptor := newResponseWriterInterceptor(w)

			// Wykonanie właściwego handlera
			next.ServeHTTP(interceptor, r)

			// Dane po przetworzeniu żądania
			latency := time.Since(start)
			clientIP := getClientIP(r)

			// Logowanie w standardowym Zap Loggerze
			logger.Info("HTTP Request",
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.String("query", r.URL.RawQuery),
				zap.Int("status", interceptor.statusCode),
				zap.Duration("latency", latency),
				zap.String("ip", clientIP),
				zap.String("user_agent", r.UserAgent()),
			)

			// Rozszerzony Debugger w konsoli z użyciem Twojej metody DebugRequest
			var bodyToLog any
			if len(reqBodyBytes) > 0 {
				bodyToLog = string(reqBodyBytes)
			} else {
				bodyToLog = "empty body"
			}

			if method, ok := logger.(interface {
				DebugRequest(msg string, method, path string, status int, latency string, body any)
			}); ok {
				method.DebugRequest(
					"HTTP Trace Debugger",
					r.Method,
					r.URL.Path,
					interceptor.statusCode,
					latency.String(),
					map[string]any{
						"IP":      clientIP,
						"Headers": r.Header,
						"ReqBody": bodyToLog,
						"ResBody": interceptor.body.String(),
					},
				)
			}
		})
	}
}

// LoggerInterface definiuje zestaw podstawowych metod wymaganych przez middleware.
type LoggerInterface interface {
	Info(msg string, args ...any)
	Debug(msg string, args ...any)
}

// Helper pobierający adres IP klienta
func getClientIP(r *http.Request) string {
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		return ip
	}
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	return r.RemoteAddr
}
