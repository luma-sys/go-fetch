package fetch_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/luma-sys/go-fetch/fetch"
)

func TestNew(t *testing.T) {
	t.Run("Create options without retry", func(t *testing.T) {
		// Arrange & Act
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		client := fetch.New(server.URL)

		// Act
		_, err := client.Get("/test")

		// Assert
		if err == nil {
			t.Error("Expected status 500, but did not receive error.")
		}
	})

	t.Run("Create options with retry", func(t *testing.T) {
		// Arrange
		tries := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tries++
			if tries < 3 {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client := fetch.New(server.URL, fetch.WithRetry(2))

		// Act
		resp, err := client.Get("/test")
		// Assert
		if err != nil {
			t.Errorf("Not expected error, but received: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Wrong status code. Expected: %d, received: %d", http.StatusOK, resp.StatusCode)
		}
		if tries != 3 {
			t.Errorf("Number of tries not expected. Expected: 3, received: %d", tries)
		}
	})
}

func TestMetodosHTTP(t *testing.T) {
	type testCase struct {
		name         string
		method       func(client fetch.FetchAPI, path string, body io.Reader) (*fetch.FetchResponse, error)
		calledMethod string
		statusCode   int
		body         string
	}

	tests := []testCase{
		{
			name: "GET success",
			method: func(c fetch.FetchAPI, p string, b io.Reader) (*fetch.FetchResponse, error) {
				return c.Get(p)
			},
			calledMethod: http.MethodGet,
			statusCode:   http.StatusOK,
			body:         "test get",
		},
		{
			name: "POST success",
			method: func(c fetch.FetchAPI, p string, b io.Reader) (*fetch.FetchResponse, error) {
				return c.Post(p, b)
			},
			calledMethod: http.MethodPost,
			statusCode:   http.StatusCreated,
			body:         "test post",
		},
		{
			name: "PUT success",
			method: func(c fetch.FetchAPI, p string, b io.Reader) (*fetch.FetchResponse, error) {
				return c.Put(p, b)
			},
			calledMethod: http.MethodPut,
			statusCode:   http.StatusOK,
			body:         "test put",
		},
		{
			name: "PATCH success",
			method: func(c fetch.FetchAPI, p string, b io.Reader) (*fetch.FetchResponse, error) {
				return c.Patch(p, b)
			},
			calledMethod: http.MethodPatch,
			statusCode:   http.StatusOK,
			body:         "test patch",
		},
		{
			name: "DELETE success",
			method: func(c fetch.FetchAPI, p string, b io.Reader) (*fetch.FetchResponse, error) {
				return c.Delete(p)
			},
			calledMethod: http.MethodDelete,
			statusCode:   http.StatusNoContent,
			body:         "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != tt.calledMethod {
					t.Errorf("Wrong HTTP method. Expected: %s, received: %s", tt.calledMethod, r.Method)
				}
				w.WriteHeader(tt.statusCode)
				if tt.body != "" {
					w.Write([]byte(tt.body))
				}
			}))
			defer server.Close()

			client := fetch.New(server.URL)
			body := strings.NewReader("request body")

			// Act
			resp, err := tt.method(client, "/test", body)
			// Assert
			if err != nil {
				t.Errorf("Not expected error, but received: %v", err)
			}
			if resp.StatusCode != tt.statusCode {
				t.Errorf("Wrong status code. Expected: %d, received: %d", tt.statusCode, resp.StatusCode)
			}
		})
	}
}

func TestErros(t *testing.T) {
	t.Run("Invalid URL", func(t *testing.T) {
		// Arrange
		client := fetch.New("http://invalid-url")

		// Act
		_, err := client.Get("/test")

		// Assert
		if err == nil {
			t.Error("Expected invalid URL error")
		}
	})

	t.Run("Status code not expected", func(t *testing.T) {
		// Arrange
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
		}))
		defer server.Close()

		client := fetch.New(server.URL)

		// Act
		_, err := client.Get("/test")

		// Assert
		if err == nil {
			t.Error("Expected status code 400 error")
		}
	})
}
