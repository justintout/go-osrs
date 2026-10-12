package osrs

import (
	"errors"
	"fmt"
)

var (
	// ErrNotFound matches a StatusError for a 404 response.
	ErrNotFound = errors.New("osrs: not found")
	// ErrRateLimited matches a StatusError for a 429 response.
	ErrRateLimited = errors.New("osrs: rate limited")
)

// StatusError reports an unexpected HTTP status from an upstream API.
type StatusError struct {
	StatusCode int
	URL        string
}

func (e *StatusError) Error() string {
	return fmt.Sprintf("osrs: GET %s: status %d", e.URL, e.StatusCode)
}

// Is lets errors.Is match ErrNotFound and ErrRateLimited.
func (e *StatusError) Is(target error) bool {
	switch target {
	case ErrNotFound:
		return e.StatusCode == 404
	case ErrRateLimited:
		return e.StatusCode == 429
	}
	return false
}
