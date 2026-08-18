package httpserver

import (
	"net/http"

	apperr "github.com/zerodayz7/platform/pkg/errors"
)

// SendError mapuje AppError na kod HTTP i wysyła JSON
func SendError(w http.ResponseWriter, r *http.Request, err error) {
	appErr, ok := err.(*apperr.AppError)
	if !ok {
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

	JSON(w, status, ErrorResponse{
		Code:    appErr.Code,
		Message: appErr.Message,
		Meta:    appErr.Meta,
	})
}
