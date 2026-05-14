package fetch

import (
	"io"
	"net/http"
	"time"
)

// FetchOpt configures a fetch client instance.
type FetchOpt func(f *fetch)

func WithRetry(retry int) FetchOpt {
	return func(f *fetch) {
		f.attempts = max(1, retry+1)
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
