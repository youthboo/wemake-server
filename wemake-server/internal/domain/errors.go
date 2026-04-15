package domain

import "errors"

var (
	ErrForbidden          = errors.New("forbidden")
	ErrImageLimitExceeded = errors.New("image limit exceeded: max 10 images per showcase")
)
