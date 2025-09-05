// Package errors provides custom error types and error handling utilities
package errors

import (
	"fmt"
	"net/http"
)

// AppError represents an application error with context
type AppError struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	StatusCode int    `json:"-"`
	Details    string `json:"details,omitempty"`
	Cause      error  `json:"-"`
}

// Error implements the error interface
func (e *AppError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (caused by: %v)", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap implements the errors.Unwrap interface
func (e *AppError) Unwrap() error {
	return e.Cause
}

// Common error codes
const (
	ErrCodeValidation          = "VALIDATION_ERROR"
	ErrCodeNotFound            = "NOT_FOUND"
	ErrCodeConflict            = "CONFLICT"
	ErrCodeOutOfStock          = "OUT_OF_STOCK"
	ErrCodeInternalError       = "INTERNAL_ERROR"
	ErrCodeUnauthorized        = "UNAUTHORIZED"
	ErrCodeForbidden           = "FORBIDDEN"
	ErrCodeBadRequest          = "BAD_REQUEST"
	ErrCodeServiceUnavailable  = "SERVICE_UNAVAILABLE"
	ErrCodeJobNotFound         = "JOB_NOT_FOUND"
	ErrCodeJobAlreadyCancelled = "JOB_ALREADY_CANCELLED"
	ErrCodeConcurrencyConflict = "CONCURRENCY_CONFLICT"
)

// Pre-defined errors
var (
	ErrProductNotFound = &AppError{
		Code:       ErrCodeNotFound,
		Message:    "Product not found",
		StatusCode: http.StatusNotFound,
	}

	ErrOutOfStock = &AppError{
		Code:       ErrCodeOutOfStock,
		Message:    "Insufficient stock",
		StatusCode: http.StatusConflict,
	}

	ErrOrderNotFound = &AppError{
		Code:       ErrCodeNotFound,
		Message:    "Order not found",
		StatusCode: http.StatusNotFound,
	}

	ErrJobNotFound = &AppError{
		Code:       ErrCodeJobNotFound,
		Message:    "Job not found",
		StatusCode: http.StatusNotFound,
	}

	ErrJobAlreadyCancelled = &AppError{
		Code:       ErrCodeJobAlreadyCancelled,
		Message:    "Job is already cancelled",
		StatusCode: http.StatusConflict,
	}

	ErrInternalError = &AppError{
		Code:       ErrCodeInternalError,
		Message:    "Internal server error",
		StatusCode: http.StatusInternalServerError,
	}
)

// NewAppError creates a new application error
func NewAppError(code, message string, statusCode int) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		StatusCode: statusCode,
	}
}

// NewAppErrorWithCause creates a new application error with a cause
func NewAppErrorWithCause(code, message string, statusCode int, cause error) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		StatusCode: statusCode,
		Cause:      cause,
	}
}

// NewValidationError creates a validation error
func NewValidationError(message string) *AppError {
	return &AppError{
		Code:       ErrCodeValidation,
		Message:    message,
		StatusCode: http.StatusBadRequest,
	}
}

// NewConcurrencyError creates a concurrency conflict error
func NewConcurrencyError(message string) *AppError {
	return &AppError{
		Code:       ErrCodeConcurrencyConflict,
		Message:    message,
		StatusCode: http.StatusConflict,
	}
}

// IsAppError checks if an error is an AppError
func IsAppError(err error) (*AppError, bool) {
	if appErr, ok := err.(*AppError); ok {
		return appErr, true
	}
	return nil, false
}

// GetStatusCode returns the HTTP status code for an error
func GetStatusCode(err error) int {
	if appErr, ok := IsAppError(err); ok {
		return appErr.StatusCode
	}
	return http.StatusInternalServerError
}

// ErrorResponse represents an error response structure
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail represents error details in the response
type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// ToErrorResponse converts an error to an error response
func ToErrorResponse(err error) ErrorResponse {
	if appErr, ok := IsAppError(err); ok {
		return ErrorResponse{
			Error: ErrorDetail{
				Code:    appErr.Code,
				Message: appErr.Message,
				Details: appErr.Details,
			},
		}
	}

	return ErrorResponse{
		Error: ErrorDetail{
			Code:    ErrCodeInternalError,
			Message: "Internal server error",
		},
	}
}
