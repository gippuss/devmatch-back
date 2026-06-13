package domain

import (
	"errors"
	"fmt"
	"net/http"
)

var (
	ErrNotFound         = NewError(http.StatusNotFound, "not_found", "resource not found", nil)
	ErrForbidden        = NewError(http.StatusForbidden, "forbidden", "access denied", nil)
	ErrConflict         = NewError(http.StatusConflict, "conflict", "conflict", nil)
	ErrValidation       = NewError(http.StatusBadRequest, "validation_error", "invalid input", nil)
	ErrUnauthorized     = NewError(http.StatusUnauthorized, "unauthorized", "unauthorized", nil)
	ErrInternal         = NewError(http.StatusInternalServerError, "internal_error", "internal server error", nil)
	ErrNoFreeSlots      = NewError(http.StatusConflict, "no_slots", "no free slots for this role", nil)
	ErrInvalidAppStatus = NewError(http.StatusBadRequest, "invalid_application_status", "invalid application status transition", nil)
)

type AppError struct {
	Status  int
	Code    string
	Message string
	Cause   error
}

func (e *AppError) Error() string {
	if e.Cause == nil {
		return e.Message
	}
	return fmt.Sprintf("%s: %v", e.Message, e.Cause)
}

func (e *AppError) Unwrap() error {
	return e.Cause
}

func NewError(status int, code, message string, cause error) *AppError {
	return &AppError{Status: status, Code: code, Message: message, Cause: cause}
}

func Wrap(base *AppError, cause error, message string) *AppError {
	return &AppError{Status: base.Status, Code: base.Code, Message: message, Cause: cause}
}

func AsAppError(err error) *AppError {
	if err == nil {
		return nil
	}

	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr
	}

	return Wrap(ErrInternal, err, ErrInternal.Message)
}
