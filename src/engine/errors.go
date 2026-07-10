package engine

import "fmt"

type ErrorCode string

const (
	ErrInvalidRequest ErrorCode = "invalid_request"
	ErrNotFound       ErrorCode = "not_found"
	ErrPolicy         ErrorCode = "policy"
	ErrState          ErrorCode = "state"
	ErrSolvency       ErrorCode = "solvency"
)

type Error struct {
	Code    ErrorCode
	Message string
}

func (e *Error) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func invalid(message string, args ...any) error {
	return &Error{Code: ErrInvalidRequest, Message: fmt.Sprintf(message, args...)}
}

func notFound(message string, args ...any) error {
	return &Error{Code: ErrNotFound, Message: fmt.Sprintf(message, args...)}
}

func policyError(message string, args ...any) error {
	return &Error{Code: ErrPolicy, Message: fmt.Sprintf(message, args...)}
}

func stateError(message string, args ...any) error {
	return &Error{Code: ErrState, Message: fmt.Sprintf(message, args...)}
}

func solvencyError(message string, args ...any) error {
	return &Error{Code: ErrSolvency, Message: fmt.Sprintf(message, args...)}
}
