package errors

import "github.com/zerodayz7/platform/pkg/errors"

var (
	// autoryzacja i tokeny
	ErrUnauthorized = &errors.AppError{
		Code:    "UNAUTHORIZED",
		Type:    errors.Unauthorized,
		Message: "Unauthorized access",
	}
	ErrInvalidToken = &errors.AppError{
		Code:    "INVALID_TOKEN",
		Type:    errors.Unauthorized,
		Message: "Invalid token",
	}

	// walidacja requestów przychodzących do gateway
	ErrValidationFailed = &errors.AppError{
		Code:    "VALIDATION_FAILED",
		Type:    errors.Validation,
		Message: "Request validation failed",
	}

	ErrBadRequest = &errors.AppError{
		Code:    "BAD_REQUEST",
		Type:    errors.BadRequest,
		Message: "Bad request",
	}

	// rate limit
	ErrTooManyRequests = &errors.AppError{
		Code:    "TOO_MANY_REQUESTS",
		Type:    errors.BadRequest,
		Message: "Too many requests",
	}

	// serwerowe
	ErrInternal = &errors.AppError{
		Code:    "SERVER_ERROR",
		Type:    errors.Internal,
		Message: "Internal server error",
	}
)
