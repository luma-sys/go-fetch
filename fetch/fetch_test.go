package fetch

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func mockClient(statusCode int, status string, body string) *http.Client {
	return &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: statusCode,
				Status:     status,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(strings.NewReader(body)),
				Request:    r,
			}, nil
		}),
	}
}

func TestPostReturnsResponseBodyOnNon2xx(t *testing.T) {
	type errorResponse struct {
		Message string `json:"message"`
	}

	payload, _ := json.Marshal(errorResponse{Message: "invalid movement payload"})

	client := New("https://protheus.example.com",
		WithHTTPClient(mockClient(http.StatusBadRequest, "400 Bad Request", string(payload))),
	)

	res, err := client.Post(context.Background(), "/movements", nil)
	if err == nil {
		t.Fatal("expected request error")
	}

	if err.Error() != "request failed with status: 400 Bad Request" {
		t.Fatalf("unexpected error: %v", err)
	}

	if res == nil || res.Response == nil {
		t.Fatal("expected response to be returned on non-2xx status")
	}

	body, decodeErr := DecodeJSON[errorResponse](res)
	if decodeErr != nil {
		t.Fatalf("expected error body to remain available, got decode error: %v", decodeErr)
	}

	if body.Message != "invalid movement payload" {
		t.Fatalf("unexpected response body: %#v", body)
	}
}

func TestRetryOnlyOn5xx(t *testing.T) {
	t.Run("does not retry on 4xx", func(t *testing.T) {
		calls := 0
		client := New("https://example.com",
			WithRetry(3),
			WithHTTPClient(&http.Client{
				Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
					calls++
					return &http.Response{
						StatusCode: http.StatusBadRequest,
						Status:     "400 Bad Request",
						Header:     http.Header{},
						Body:       io.NopCloser(strings.NewReader("")),
						Request:    r,
					}, nil
				}),
			}),
		)

		_, err := client.Get(context.Background(), "/test")
		if err == nil {
			t.Fatal("expected error")
		}
		if calls != 1 {
			t.Errorf("expected 1 call (no retry on 4xx), got %d", calls)
		}
	})

	t.Run("retries on 5xx", func(t *testing.T) {
		calls := 0
		client := New("https://example.com",
			WithRetry(2),
			WithHTTPClient(&http.Client{
				Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
					calls++
					if calls < 3 {
						return &http.Response{
							StatusCode: http.StatusInternalServerError,
							Status:     "500 Internal Server Error",
							Header:     http.Header{},
							Body:       io.NopCloser(strings.NewReader("")),
							Request:    r,
						}, nil
					}
					return &http.Response{
						StatusCode: http.StatusOK,
						Status:     "200 OK",
						Header:     http.Header{},
						Body:       io.NopCloser(strings.NewReader("")),
						Request:    r,
					}, nil
				}),
			}),
		)

		resp, err := client.Get(context.Background(), "/test")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected 200, got %d", resp.StatusCode)
		}
		if calls != 3 {
			t.Errorf("expected 3 calls, got %d", calls)
		}
	})
}

func TestWithRetryNegative(t *testing.T) {
	client := New("https://example.com",
		WithRetry(-1),
		WithHTTPClient(mockClient(http.StatusOK, "200 OK", "")),
	)

	resp, err := client.Get(context.Background(), "/test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	client := New("https://example.com",
		WithHTTPClient(mockClient(http.StatusOK, "200 OK", "")),
	)

	_, err := client.Get(ctx, "/test")
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

func TestSetOptionsDoesNotMutateOriginal(t *testing.T) {
	calls := 0
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		return &http.Response{
			StatusCode: http.StatusInternalServerError,
			Status:     "500 Internal Server Error",
			Header:     http.Header{},
			Body:       io.NopCloser(strings.NewReader("")),
			Request:    r,
		}, nil
	})

	base := New("https://example.com",
		WithRetry(2),
		WithHTTPClient(&http.Client{Transport: transport}),
	)
	noRetry := base.SetOptions(WithRetry(0))

	_, _ = noRetry.Get(context.Background(), "/test")
	if calls != 1 {
		t.Errorf("SetOptions(WithRetry(0)): expected 1 call, got %d", calls)
	}

	calls = 0
	_, _ = base.Get(context.Background(), "/test")
	if calls != 3 {
		t.Errorf("original still has retry=2: expected 3 calls, got %d", calls)
	}
}

func TestBodyExceedsMaxSize(t *testing.T) {
	client := New("https://example.com",
		WithHTTPClient(mockClient(http.StatusOK, "200 OK", "")),
	)

	huge := io.LimitReader(nopReader{}, maxRetryBodySize+1)
	_, err := client.Post(context.Background(), "/test", huge)
	if err == nil {
		t.Fatal("expected error for body exceeding max size")
	}
	if err.Error() != "request body exceeds max size" {
		t.Fatalf("unexpected error: %v", err)
	}
}

type nopReader struct{}

func (nopReader) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = 'x'
	}
	return len(p), nil
}

func TestWithTimeout(t *testing.T) {
	blocked := make(chan struct{})
	client := New("https://example.com",
		WithTimeout(50*time.Millisecond),
		WithHTTPClient(&http.Client{
			Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				select {
				case <-r.Context().Done():
					return nil, r.Context().Err()
				case <-blocked:
					return nil, nil
				}
			}),
		}),
	)

	_, err := client.Get(context.Background(), "/test")
	if err == nil {
		t.Fatal("expected timeout error")
	}
	close(blocked)
}

func TestWithDebugWriter(t *testing.T) {
	t.Run("writes curl command to writer", func(t *testing.T) {
		var buf bytes.Buffer
		client := New("https://example.com",
			WithDebugWriter(&buf),
			WithHTTPClient(mockClient(http.StatusOK, "200 OK", "")),
		)

		_, err := client.Get(context.Background(), "/ping")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		out := buf.String()
		if !strings.Contains(out, "curl") {
			t.Errorf("expected curl output, got: %q", out)
		}
		if !strings.Contains(out, "https://example.com/ping") {
			t.Errorf("expected URL in curl output, got: %q", out)
		}
	})

	t.Run("body is sent after debug read", func(t *testing.T) {
		var receivedBody string
		client := New("https://example.com",
			WithDebugWriter(io.Discard),
			WithHTTPClient(&http.Client{
				Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
					b, _ := io.ReadAll(r.Body)
					receivedBody = string(b)
					return &http.Response{
						StatusCode: http.StatusOK,
						Status:     "200 OK",
						Header:     http.Header{},
						Body:       io.NopCloser(strings.NewReader("")),
						Request:    r,
					}, nil
				}),
			}),
		)

		payload := strings.NewReader(`{"key":"value"}`)
		_, err := client.Post(context.Background(), "/test", payload)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if receivedBody != `{"key":"value"}` {
			t.Errorf("body was consumed by debug writer: got %q", receivedBody)
		}
	})
}

func TestRetryBodySentOnEveryAttempt(t *testing.T) {
	calls := 0
	var bodies []string

	client := New("https://example.com",
		WithRetry(2),
		WithHTTPClient(&http.Client{
			Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				calls++
				b, _ := io.ReadAll(r.Body)
				bodies = append(bodies, string(b))
				if calls < 3 {
					return &http.Response{
						StatusCode: http.StatusInternalServerError,
						Status:     "500 Internal Server Error",
						Header:     http.Header{},
						Body:       io.NopCloser(strings.NewReader("")),
						Request:    r,
					}, nil
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Status:     "200 OK",
					Header:     http.Header{},
					Body:       io.NopCloser(strings.NewReader("")),
					Request:    r,
				}, nil
			}),
		}),
	)

	body := strings.NewReader(`{"key":"value"}`)
	_, err := client.Post(context.Background(), "/test", body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 3 {
		t.Errorf("expected 3 attempts, got %d", calls)
	}
	for i, b := range bodies {
		if b != `{"key":"value"}` {
			t.Errorf("attempt %d sent wrong body: %q", i+1, b)
		}
	}
}
