package domain

import "errors"

var (
	ErrNotFound  = errors.New("not found")
	ErrInvalidID = errors.New("invalid id")
)
