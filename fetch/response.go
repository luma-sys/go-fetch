package fetch

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

type FetchResponse struct {
	*http.Response
	cancel context.CancelFunc
}

// Close drains and closes the response body and releases the associated context.
func (r *FetchResponse) Close() error {
	if r.cancel != nil {
		defer r.cancel()
	}
	_, _ = io.Copy(io.Discard, r.Body)
	return r.Body.Close()
}

// DecodeJSON decodes the JSON response body into a value of type T.
// Always closes the body and releases the associated context.
func DecodeJSON[T any](r *FetchResponse) (*T, error) {
	if r == nil || r.Response == nil {
		return nil, errors.New("response is nil")
	}

	defer r.Close()

	var body T
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFetchDecodeJSON, err)
	}

	return &body, nil
}
