package fetch

import (
	"context"
	"errors"
	"io"
	"log"
	"net/http"
	"time"
)

type FetchAPI interface {
	Delete(path string, opts ...requestOpt) (*http.Response, error)
	Patch(path string, body io.Reader, opts ...requestOpt) (*http.Response, error)
	Put(path string, body io.Reader, opts ...requestOpt) (*http.Response, error)
	Post(path string, body io.Reader, opts ...requestOpt) (*http.Response, error)
	Get(path string, opts ...requestOpt) (*http.Response, error)
	SetOptions(opts ...fetchOpt) FetchAPI
	GetWithContext(ctx context.Context, path string, opts ...requestOpt) (*http.Response, error)
}

type fetch struct {
	baseURL     string
	attempts    int
	timeout     time.Duration
	requestOpts []requestOpt
}

type fetchOpt func(f *fetch)

func WithRetry(retry int) fetchOpt {
	return func(f *fetch) {
		f.attempts = retry + 1
	}
}

func WithTimeout(timeout time.Duration) fetchOpt {
	return func(f *fetch) {
		f.timeout = timeout
	}
}

func WithDefaultRequestOpts(opts ...requestOpt) fetchOpt {
	return func(f *fetch) {
		f.requestOpts = opts
	}
}

func New(baseURL string, opts ...fetchOpt) FetchAPI {
	fetch := &fetch{baseURL, 1, 0, []requestOpt{}}
	for _, opt := range opts {
		opt(fetch)
	}

	return fetch
}

func (e *fetch) SetOptions(opts ...fetchOpt) FetchAPI {
	fetch := &fetch{
		baseURL:     e.baseURL,
		attempts:    e.attempts,
		timeout:     e.timeout,
		requestOpts: e.requestOpts,
	}
	for _, opt := range opts {
		opt(fetch)
	}

	return fetch
}

func checkStatusCodeSuccess(code int) bool {
	return code >= 200 && code < 300
}

func (e *fetch) request(ctx context.Context, method, path string, body io.Reader, opts ...requestOpt) (*http.Response, error) {
	var r *http.Response
	var err error
	var cancel context.CancelFunc

	url := e.baseURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}

	for _, opt := range e.requestOpts {
		opt(req)
	}

	for _, opt := range opts {
		opt(req)
	}

	for i := range e.attempts {
		if cancel != nil {
			cancel()
		}

		if e.timeout > 0 {
			ctx, cancel = context.WithTimeout(ctx, e.timeout)
			req = req.WithContext(ctx)
		}

		r, err = http.DefaultClient.Do(req)
		if err != nil {
			log.Printf("Attempt %d failed: %v", i+1, err)
			continue
		}

		if checkStatusCodeSuccess(r.StatusCode) {
			err = nil
			break
		}
	}

	if cancel != nil {
		defer cancel()
	}

	if err != nil {
		return r, err
	}
	if !checkStatusCodeSuccess(r.StatusCode) {
		return r, errors.New(r.Status)
	}

	return r, err
}

func (e *fetch) Delete(path string, opts ...requestOpt) (*http.Response, error) {
	return e.request(context.Background(), http.MethodDelete, path, nil, opts...)
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

func (e *fetch) Get(path string, opts ...requestOpt) (*http.Response, error) {
	return e.request(context.Background(), http.MethodGet, path, nil, opts...)
}

func (e *fetch) GetWithContext(ctx context.Context, path string, opts ...requestOpt) (*http.Response, error) {
	return e.request(ctx, http.MethodGet, path, nil, opts...)
}
