package repository

import "errors"

var (
	ErrNotFound      = errors.New("job not found")
	ErrAlreadyExists = errors.New("job already exists")
)
