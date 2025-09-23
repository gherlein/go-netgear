package netgear

import "fmt"

// ErrorType represents the category of error
type ErrorType string

const (
	ErrorTypeAuth      ErrorType = "authentication"
	ErrorTypeNetwork   ErrorType = "network"
	ErrorTypeParsing   ErrorType = "parsing"
	ErrorTypeModel     ErrorType = "model"
	ErrorTypeOperation ErrorType = "operation"
)

// Error represents a netgear client error
type Error struct {
	Type    ErrorType
	Message string
	Cause   error
}

func (e *Error) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s error: %s: %v", e.Type, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s error: %s", e.Type, e.Message)
}

func (e *Error) Unwrap() error {
	return e.Cause
}

// Sentinel errors
var (
	ErrNotAuthenticated   = &Error{Type: ErrorTypeAuth, Message: "not authenticated"}
	ErrSessionExpired     = &Error{Type: ErrorTypeAuth, Message: "session expired"}
	ErrModelNotSupported  = &Error{Type: ErrorTypeModel, Message: "model not supported"}
	ErrModelNotDetected   = &Error{Type: ErrorTypeModel, Message: "could not detect switch model"}
	ErrInvalidCredentials = &Error{Type: ErrorTypeAuth, Message: "invalid credentials"}
	ErrNetworkTimeout     = &Error{Type: ErrorTypeNetwork, Message: "network timeout"}
	ErrInvalidResponse    = &Error{Type: ErrorTypeParsing, Message: "invalid response format"}
)

// NewError creates a new netgear error
func NewError(errorType ErrorType, message string, cause error) *Error {
	return &Error{
		Type:    errorType,
		Message: message,
		Cause:   cause,
	}
}

// NewAuthError creates a new authentication error
func NewAuthError(message string, cause error) *Error {
	return NewError(ErrorTypeAuth, message, cause)
}

// NewNetworkError creates a new network error
func NewNetworkError(message string, cause error) *Error {
	return NewError(ErrorTypeNetwork, message, cause)
}

// NewParsingError creates a new parsing error
func NewParsingError(message string, cause error) *Error {
	return NewError(ErrorTypeParsing, message, cause)
}

// NewModelError creates a new model error
func NewModelError(message string, cause error) *Error {
	return NewError(ErrorTypeModel, message, cause)
}

// NewOperationError creates a new operation error
func NewOperationError(message string, cause error) *Error {
	return NewError(ErrorTypeOperation, message, cause)
}