// Package fetch provides a configurable HTTP client with retry, timeout, and per-request options.
package fetch

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/moul/http2curl"
)

const (
	maxRetryBodySize = 15 * 1024 * 1024 // 15MB
	defaultTimeout   = 90 * time.Second
)

var defaultHTTPClient = &http.Client{
	Transport: func() *http.Transport {
		t := http.DefaultTransport.(*http.Transport).Clone()
		t.MaxIdleConns = 100
		t.MaxIdleConnsPerHost = 5
		t.IdleConnTimeout = 2 * time.Minute
		t.DisableKeepAlives = true
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

func New(baseURL string, opts ...FetchOpt) FetchAPI {
	f := &fetch{
		baseURL:    baseURL,
		attempts:   1,
		timeout:    defaultTimeout,
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
	return time.Duration(attempt) * 200 * time.Millisecond
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
			return nil, ErrFetchBodyTooLarge
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
				_, _ = fmt.Fprintln(e.debugWriter, cmd)
			}
			// GetCurlCommand reads req.Body — restore it so Do sends the actual payload.
			if req.GetBody != nil {
				req.Body, _ = req.GetBody()
			}
		}

		r, lastErr = e.httpClient.Do(req)
		if lastErr != nil {
			r = nil
			if e.timeout > 0 && errors.Is(reqCtx.Err(), context.DeadlineExceeded) && ctx.Err() == nil {
				lastErr = fmt.Errorf("%w: %w", ErrFetchRequestTimeout, lastErr)
			} else if ctx.Err() == nil {
				var netErr net.Error
				if errors.As(lastErr, &netErr) && netErr.Timeout() {
					lastErr = fmt.Errorf("%w: %w", ErrFetchTransportTimeout, lastErr)
				}
			}
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
		lastErr = &ErrFetchHTTPStatus{StatusCode: r.StatusCode, Status: r.Status}

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
