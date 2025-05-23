package fetch_test

import (
	"github.com/luma-sys/go-fetch/fetch"
	"net/http"
	"reflect"
	"testing"
)

func TestWithBasicAuth(t *testing.T) {
	// Arrange
	request, _ := http.NewRequest(http.MethodGet, "http://example.com", nil)
	user := "admin"
	pass := "pass123"

	// Act
	opt := fetch.WithBasicAuth(user, pass)
	opt(request)

	// Assert
	receivedUser, receivedPass, ok := request.BasicAuth()
	if !ok {
		t.Error("Basic auth not configured")
	}

	if receivedUser != user {
		t.Errorf("Wrong user. Expected: %s, received: %s", user, receivedUser)
	}

	if receivedPass != pass {
		t.Errorf("Wrong password. Expected: %s, received: %s", pass, receivedPass)
	}
}

func TestWithHeader(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		value    string
		expected []string
		existent string
	}{
		{
			name:     "Header simples",
			key:      "Content-Type",
			value:    "application/json",
			expected: []string{"application/json"},
		},
		{
			name:     "Header com value existent",
			key:      "Accept",
			value:    "application/xml",
			existent: "application/json",
			expected: []string{"application/json", "application/xml"},
		},
		{
			name:     "Header personalizado",
			key:      "X-Custom-Header",
			value:    "value-teste",
			expected: []string{"value-teste"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			request, _ := http.NewRequest(http.MethodGet, "http://example.com", nil)

			if tt.existent != "" {
				request.Header.Add(tt.key, tt.existent)
			}

			// Act
			opt := fetch.WithHeader(tt.key, tt.value)
			opt(request)

			// Assert
			valreceivedUeess := request.Header.Values(tt.key)
			if !reflect.DeepEqual(valreceivedUeess, tt.expected) {
				t.Errorf("Invalid Header to %s.\nExpected: %v\nreceived: %v", tt.key, tt.expected, valreceivedUeess)
			}
		})
	}
}

func TestMultipleHeaders(t *testing.T) {
	// Arrange
	request, _ := http.NewRequest(http.MethodGet, "http://example.com", nil)
	headers := []struct {
		key   string
		value string
	}{
		{"Accept", "application/json"},
		{"X-API-Key", "key123"},
		{"User-Agent", "TestClient"},
	}

	// Act
	for _, h := range headers {
		opt := fetch.WithHeader(h.key, h.value)
		opt(request)
	}

	// Assert
	for _, h := range headers {
		value := request.Header.Get(h.key)
		if value != h.value {
			t.Errorf("Invalid Header to %s.\nExpected: %s\nreceived: %s", h.key, h.value, value)
		}
	}
}
