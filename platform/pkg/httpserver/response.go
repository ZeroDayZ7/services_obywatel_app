// platform/pkg/httpserver/response.go
package httpserver

import (
	"encoding/json"
	"net/http"
)

// JSON wysyła odpowiedź w formacie JSON z podanym kodem HTTP
//#region JSON
func JSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		_ = json.NewEncoder(w).Encode(data)
	}
}

// ErrorResponse struktura odpowiedzi dla błędów
type ErrorResponse struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Meta    map[string]any `json:"meta,omitempty"`
}
