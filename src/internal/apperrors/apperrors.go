package apperrors

import "errors"

type Kind string

const (
	KindBadRequest  Kind = "bad_request"
	KindUnauthorized     = "unauthorized"
	KindNotFound         = "not_found"
	KindConflict         = "conflict"
	KindInternal         = "internal"
)

type AppError struct {
	Kind    Kind
	Message string
	Err     error
}

func (e *AppError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	return string(e.Kind)
}

func (e *AppError) Unwrap() error {
	return e.Err
}

func From(err error) *AppError {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr
	}
	return nil
}

func NewNotFound(message string, err error) *AppError {
	return &AppError{Kind: KindNotFound, Message: message, Err: err}
}

func NewUnauthorized(message string, err error) *AppError {
	return &AppError{Kind: KindUnauthorized, Message: message, Err: err}
}

func NewBadRequest(message string, err error) *AppError {
	return &AppError{Kind: KindBadRequest, Message: message, Err: err}
}

func NewConflict(message string, err error) *AppError {
	return &AppError{Kind: KindConflict, Message: message, Err: err}
}

func NewInternal(message string, err error) *AppError {
	return &AppError{Kind: KindInternal, Message: message, Err: err}
}
