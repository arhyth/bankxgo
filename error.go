package bankxgo

import (
	"errors"
	"fmt"
)

var (
	ErrInternalServer = errors.New("internal server error")
)

type ErrBadRequest struct {
	Fields map[string]string
}

func (e ErrBadRequest) Error() string {
	return fmt.Sprintf("missing/invalid params: %v", e.Fields)
}

type ErrNotFound struct {
	ID int `json:"id"`
}

func (e ErrNotFound) Error() string {
	return "record not found"
}
