package fetch

import (
	"context"
	"errors"
	"io"
	"net/http"
)

type FetchAPI interface {
	Delete(path string, body io.Reader, opts ...requestOpt) (*http.Response, error)
	Patch(path string, body io.Reader, opts ...requestOpt) (*http.Response, error)
	Put(path string, body io.Reader, opts ...requestOpt) (*http.Response, error)
	Post(path string, body io.Reader, opts ...requestOpt) (*http.Response, error)
	Get(path string, body io.Reader, opts ...requestOpt) (*http.Response, error)
	GetWithContext(ctx context.Context, path string, body io.Reader, opts ...requestOpt) (*http.Response, error)
}

type fetch struct {
	baseURL  string
	attempts int
}

type fetchOpt func(f *fetch)

func WithRetry(retry int) fetchOpt {
	return func(f *fetch) {
		f.attempts = retry + 1
	}
}

func New(baseURL string, opts ...fetchOpt) FetchAPI {
	fetch := &fetch{baseURL, 1}
	for _, opt := range opts {
		opt(fetch)
	}

	return fetch
}

func (e *fetch) request(ctx context.Context, method, path string, body io.Reader, opts ...requestOpt) (*http.Response, error) {
	var r *http.Response
	var err error

	url := e.baseURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}

	for _, opt := range opts {
		opt(req)
	}

	for range e.attempts {
		r, err = http.DefaultClient.Do(req)
		if err != nil {
			continue
		}
		if r.StatusCode == http.StatusOK || r.StatusCode == http.StatusCreated || r.StatusCode == http.StatusNoContent {
			err = nil
			break
		}
	}

	if err != nil {
		return nil, err
	}
	if r.StatusCode != http.StatusOK && r.StatusCode != http.StatusCreated && r.StatusCode != http.StatusNoContent {
		return r, errors.New(r.Status)
	}

	return r, err
}

func (e *fetch) Delete(path string, body io.Reader, opts ...requestOpt) (*http.Response, error) {
	return e.request(context.Background(), http.MethodDelete, path, body, opts...)
}

func (e *fetch) Patch(path string, body io.Reader, opts ...requestOpt) (*http.Response, error) {
	return e.request(context.Background(), http.MethodPatch, path, body, opts...)
}

func (e *fetch) Put(path string, body io.Reader, opts ...requestOpt) (*http.Response, error) {
	return e.request(context.Background(), http.MethodPut, path, body, opts...)
}

func (e *fetch) Post(path string, body io.Reader, opts ...requestOpt) (*http.Response, error) {
	return e.request(context.Background(), http.MethodPost, path, body, opts...)
}

func (e *fetch) Get(path string, body io.Reader, opts ...requestOpt) (*http.Response, error) {
	return e.request(context.Background(), http.MethodGet, path, body, opts...)
}

func (e *fetch) GetWithContext(ctx context.Context, path string, body io.Reader, opts ...requestOpt) (*http.Response, error) {
	return e.request(ctx, http.MethodGet, path, body, opts...)
}
