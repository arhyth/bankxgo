package bankxgo

import (
	"errors"
	"fmt"
)

var (
	ErrInternalServer     = errors.New("internal server error")
	ErrServiceUnavailable = errors.New("service unavailable")
)

type ErrBadRequest struct {
	Fields map[string]string `json:"fields"`
}

func (e ErrBadRequest) Error() string {
	return fmt.Sprintf("missing/invalid params: %v", e.Fields)
}

type ErrNotFound struct {
	ID int64 `json:"id"`
}

func (e ErrNotFound) Error() string {
	return "record not found"
}
