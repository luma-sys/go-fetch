// Package fetch provides a configurable HTTP client with retry, timeout, and per-request options.
package fetch

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/moul/http2curl"
)

const maxRetryBodySize = 15 * 1024 * 1024 // 15MB

var defaultHTTPClient = &http.Client{
	Transport: func() *http.Transport {
		t := http.DefaultTransport.(*http.Transport).Clone()
		t.MaxIdleConns = 100
		t.MaxIdleConnsPerHost = 20
		t.IdleConnTimeout = 5 * time.Minute
		return t
	}(),
}

type FetchAPI interface {
	Delete(ctx context.Context, path string, opts ...RequestOpt) (*FetchResponse, error)
	Patch(ctx context.Context, path string, body io.Reader, opts ...RequestOpt) (*FetchResponse, error)
	Put(ctx context.Context, path string, body io.Reader, opts ...RequestOpt) (*FetchResponse, error)
	Post(ctx context.Context, path string, body io.Reader, opts ...RequestOpt) (*FetchResponse, error)
	Get(ctx context.Context, path string, opts ...RequestOpt) (*FetchResponse, error)
	SetOptions(opts ...FetchOpt) FetchAPI
}

type fetch struct {
	baseURL     string
	attempts    int
	timeout     time.Duration
	httpClient  *http.Client
	requestOpts []RequestOpt
	debugWriter io.Writer
}

// FetchOpt configures a fetch client instance.
type FetchOpt func(f *fetch)

func WithRetry(retry int) FetchOpt {
	return func(f *fetch) {
		if retry < 0 {
			retry = 0
		}
		f.attempts = retry + 1
	}
}

func WithTimeout(timeout time.Duration) FetchOpt {
	return func(f *fetch) {
		f.timeout = timeout
	}
}

// WithDebugWriter writes a curl-equivalent command for each request to w.
// Useful for local debugging. Do not use in production — output may contain auth headers.
func WithDebugWriter(w io.Writer) FetchOpt {
	return func(f *fetch) {
		f.debugWriter = w
	}
}

// WithHTTPClient overrides the HTTP client used for requests.
func WithHTTPClient(client *http.Client) FetchOpt {
	return func(f *fetch) {
		f.httpClient = client
	}
}

// WithDefaultRequestOpts sets default RequestOpts applied to every request.
// Replaces any previously configured defaults.
func WithDefaultRequestOpts(opts ...RequestOpt) FetchOpt {
	return func(f *fetch) {
		f.requestOpts = opts
	}
}

func New(baseURL string, opts ...FetchOpt) FetchAPI {
	f := &fetch{
		baseURL:    baseURL,
		attempts:   1,
		httpClient: defaultHTTPClient,
	}
	for _, opt := range opts {
		opt(f)
	}
	return f
}

func (e *fetch) SetOptions(opts ...FetchOpt) FetchAPI {
	newFetch := &fetch{
		baseURL:     e.baseURL,
		attempts:    e.attempts,
		timeout:     e.timeout,
		httpClient:  e.httpClient,
		requestOpts: e.requestOpts,
		debugWriter: e.debugWriter,
	}
	for _, opt := range opts {
		opt(newFetch)
	}
	return newFetch
}

func checkStatusCodeSuccess(code int) bool {
	return code >= 200 && code < 300
}

// isRetryable returns true only for 5xx responses and network errors.
// 4xx errors are permanent client errors and must not be retried.
func isRetryable(statusCode int) bool {
	return statusCode >= 500
}

func retryBackoff(attempt int) time.Duration {
	return time.Duration(attempt) * 100 * time.Millisecond
}

func (e *fetch) request(ctx context.Context, method, path string, body io.Reader, opts ...RequestOpt) (*FetchResponse, error) {
	var bodyBytes []byte
	if body != nil {
		limited := io.LimitReader(body, maxRetryBodySize+1)
		var err error
		bodyBytes, err = io.ReadAll(limited)
		if err != nil {
			return nil, err
		}
		if len(bodyBytes) > maxRetryBodySize {
			if c, ok := body.(io.Closer); ok {
				_ = c.Close()
			}
			return nil, errors.New("request body exceeds max size")
		}
		if c, ok := body.(io.Closer); ok {
			_ = c.Close()
		}
	}

	url := e.baseURL + path
	var r *http.Response
	var lastErr error
	var cancel context.CancelFunc

	for i := 0; i < e.attempts; i++ {
		if cancel != nil {
			cancel()
			cancel = nil
		}

		if err := ctx.Err(); err != nil {
			return nil, err
		}

		if i > 0 {
			select {
			case <-time.After(retryBackoff(i)):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}

		reqCtx := ctx
		if e.timeout > 0 {
			reqCtx, cancel = context.WithTimeout(ctx, e.timeout)
		}

		var currentBody io.Reader
		if bodyBytes != nil {
			currentBody = bytes.NewReader(bodyBytes)
		}

		req, err := http.NewRequestWithContext(reqCtx, method, url, currentBody)
		if err != nil {
			lastErr = err
			continue
		}

		for _, opt := range e.requestOpts {
			if opt != nil {
				opt(req)
			}
		}
		for _, opt := range opts {
			if opt != nil {
				opt(req)
			}
		}

		if e.debugWriter != nil {
			if cmd, curlErr := http2curl.GetCurlCommand(req); curlErr == nil {
				fmt.Fprintln(e.debugWriter, cmd)
			}
			// GetCurlCommand reads req.Body — restore it so Do sends the actual payload.
			if req.GetBody != nil {
				req.Body, _ = req.GetBody()
			}
		}

		r, lastErr = e.httpClient.Do(req)
		if lastErr != nil {
			r = nil
			continue
		}

		if checkStatusCodeSuccess(r.StatusCode) {
			return &FetchResponse{Response: r, cancel: cancel}, nil
		}

		// Preserve body so caller can inspect the error response.
		responseBody, readErr := io.ReadAll(r.Body)
		_ = r.Body.Close()
		if readErr != nil {
			r.Body = io.NopCloser(bytes.NewReader(nil))
		} else {
			r.Body = io.NopCloser(bytes.NewReader(responseBody))
		}
		lastErr = errors.New("request failed with status: " + r.Status)

		if !isRetryable(r.StatusCode) {
			break
		}
	}

	if lastErr != nil {
		if r != nil {
			return &FetchResponse{Response: r, cancel: cancel}, lastErr
		}
		if cancel != nil {
			cancel()
		}
		return nil, lastErr
	}

	return &FetchResponse{Response: r, cancel: cancel}, nil
}

func (e *fetch) Delete(ctx context.Context, path string, opts ...RequestOpt) (*FetchResponse, error) {
	return e.request(ctx, http.MethodDelete, path, nil, opts...)
}

func (e *fetch) Patch(ctx context.Context, path string, body io.Reader, opts ...RequestOpt) (*FetchResponse, error) {
	return e.request(ctx, http.MethodPatch, path, body, opts...)
}

func (e *fetch) Put(ctx context.Context, path string, body io.Reader, opts ...RequestOpt) (*FetchResponse, error) {
	return e.request(ctx, http.MethodPut, path, body, opts...)
}

func (e *fetch) Post(ctx context.Context, path string, body io.Reader, opts ...RequestOpt) (*FetchResponse, error) {
	return e.request(ctx, http.MethodPost, path, body, opts...)
}

func (e *fetch) Get(ctx context.Context, path string, opts ...RequestOpt) (*FetchResponse, error) {
	return e.request(ctx, http.MethodGet, path, nil, opts...)
}
