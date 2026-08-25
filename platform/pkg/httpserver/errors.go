package httpserver

import (
	"net/http"

	apperr "github.com/zerodayz7/platform/pkg/errors" // zmień ścieżkę na swój moduł błędów
)

// SendAppError mapuje AppError na kod HTTP i wysyła czysty JSON przez net/http
//#region SendError
func SendError(w http.ResponseWriter, r *http.Request, err error) {
	appErr, ok := err.(*apperr.AppError)
	if !ok {
		// Jeśli to nie jest nasz AppError (np. błąd z bazy), traktujemy jako Internal
		appErr = apperr.ErrInternal
	}

	statusMap := map[apperr.ErrorType]int{
		apperr.Validation:   http.StatusBadRequest,
		apperr.Unauthorized: http.StatusUnauthorized,
		apperr.Forbidden:    http.StatusForbidden,
		apperr.NotFound:     http.StatusNotFound,
		apperr.Internal:     http.StatusInternalServerError,
		apperr.BadRequest:   http.StatusBadRequest,
		apperr.Timeout:      http.StatusGatewayTimeout,
		apperr.Conflict:     http.StatusConflict,
	}

	status, exists := statusMap[appErr.Type]
	if !exists {
		status = http.StatusInternalServerError
	}

	// Struktura odpowiedzi zgodna z tym, czego oczekujesz
	responseBody := map[string]any{
		"code":    appErr.Code,
		"message": appErr.Message,
	}

	if len(appErr.Meta) > 0 {
		responseBody["meta"] = appErr.Meta
	}

	JSON(w, status, responseBody)
}
