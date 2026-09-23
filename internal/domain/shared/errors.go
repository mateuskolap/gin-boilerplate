package shared

import "fmt"

type ErrorType string

const (
	ErrTypeNotFound        ErrorType = "NOT_FOUND"
	ErrTypeConflict        ErrorType = "CONFLICT"
	ErrTypeUnauthorized    ErrorType = "UNAUTHORIZED"
	ErrTypeInternal        ErrorType = "INTERNAL"
	ErrTypeValidation      ErrorType = "VALIDATION"
	ErrTypeForbidden       ErrorType = "FORBIDDEN"
	ErrTypeTooManyRequests ErrorType = "TOO_MANY_REQUESTS"
	ErrTypeUnavailable     ErrorType = "SERVICE_UNAVAILABLE"
)

type AppError struct {
	Type    ErrorType
	Message string
	Err     error
}

func (e *AppError) Unwrap() error {
	return e.Err
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

// NewAppError constructs a new AppError instance.
func NewAppError(errType ErrorType, message string, err error) *AppError {
	return &AppError{
		Type:    errType,
		Message: message,
		Err:     err,
	}
}
