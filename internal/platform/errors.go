package platform

import (
	"errors"
	"fmt"
	"net/http"
)

// Standard error codes.
const (
	CodeBadRequest     = "BAD_REQUEST"
	CodeUnauthorized   = "UNAUTHORIZED"
	CodeForbidden      = "FORBIDDEN"
	CodeNotFound       = "NOT_FOUND"
	CodeConflict       = "CONFLICT"
	CodeInternal       = "INTERNAL_ERROR"
	CodeValidation     = "VALIDATION_ERROR"
	CodeRateLimited    = "RATE_LIMITED"
)

// AppError is a structured application error.
type AppError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Status  int    `json:"-"`
	Err     error  `json:"-"`
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *AppError) Unwrap() error {
	return e.Err
}

func NewBadRequest(msg string) *AppError {
	return &AppError{Code: CodeBadRequest, Message: msg, Status: http.StatusBadRequest}
}

func NewUnauthorized(msg string) *AppError {
	return &AppError{Code: CodeUnauthorized, Message: msg, Status: http.StatusUnauthorized}
}

func NewForbidden(msg string) *AppError {
	return &AppError{Code: CodeForbidden, Message: msg, Status: http.StatusForbidden}
}

func NewNotFound(msg string) *AppError {
	return &AppError{Code: CodeNotFound, Message: msg, Status: http.StatusNotFound}
}

func NewConflict(msg string) *AppError {
	return &AppError{Code: CodeConflict, Message: msg, Status: http.StatusConflict}
}

func NewInternal(msg string, err error) *AppError {
	return &AppError{Code: CodeInternal, Message: msg, Status: http.StatusInternalServerError, Err: err}
}

func NewValidation(msg string) *AppError {
	return &AppError{Code: CodeValidation, Message: msg, Status: http.StatusUnprocessableEntity}
}

// AsAppError extracts an AppError from an error chain.
func AsAppError(err error) (*AppError, bool) {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr, true
	}
	return nil, false
}
