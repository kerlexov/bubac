package logger

import (
	"fmt"
)

type ErrType string

const (
	ErrTypeInvalidConfig ErrType = "INVALID_CONFIG"
	ErrTypeNetworkError  ErrType = "NETWORK_ERROR"
	ErrTypeBufferFull    ErrType = "BUFFER_FULL"
	ErrTypeTimeout       ErrType = "TIMEOUT"
	ErrTypeServerError   ErrType = "SERVER_ERROR"
)

type Error struct {
	Type    ErrType `json:"type"`
	Message string  `json:"message"`
	Err     error   `json:"error,omitempty"`
}

func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s - %v", e.Type, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

func (e *Error) Unwrap() error {
	return e.Err
}

func ErrInvalidConfig(message string) *Error {
	return &Error{
		Type:    ErrTypeInvalidConfig,
		Message: message,
	}
}

func ErrNetworkError(message string, err error) *Error {
	return &Error{
		Type:    ErrTypeNetworkError,
		Message: message,
		Err:     err,
	}
}

func ErrBufferFull(message string) *Error {
	return &Error{
		Type:    ErrTypeBufferFull,
		Message: message,
	}
}

func ErrTimeout(message string, err error) *Error {
	return &Error{
		Type:    ErrTypeTimeout,
		Message: message,
		Err:     err,
	}
}

func ErrServerError(message string, err error) *Error {
	return &Error{
		Type:    ErrTypeServerError,
		Message: message,
		Err:     err,
	}
}
