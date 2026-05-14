package fetch

import (
	"errors"
	"fmt"
)

// ErrFetchRequestTimeout is returned when the per-request context timeout fires before the server responds.
var ErrFetchRequestTimeout = errors.New("request timed out")

// ErrFetchTransportTimeout is returned when the HTTP transport fires a network-level timeout
// (e.g. dial, TLS handshake, response header) independently of the request context.
var ErrFetchTransportTimeout = errors.New("transport timed out")

// ErrFetchDecodeJSON is returned when the response body cannot be decoded as JSON.
var ErrFetchDecodeJSON = errors.New("failed to decode JSON response")

// ErrFetchBodyTooLarge is returned when the request body exceeds the 15MB retry buffer limit.
var ErrFetchBodyTooLarge = errors.New("request body exceeds max size")

// ErrFetchHTTPStatus is returned on non-2xx responses.
// Use errors.As to access StatusCode and inspect the HTTP status.
type ErrFetchHTTPStatus struct {
	StatusCode int
	Status     string
}

func (e *ErrFetchHTTPStatus) Error() string {
	return fmt.Sprintf("request failed with status: %s", e.Status)
}
