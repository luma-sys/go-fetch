package fetch

import (
	"net/http"
)

type requestOpt func(r *http.Request)

func WithHeader(key, value string) requestOpt {
	return func(r *http.Request) {
		r.Header.Add(key, value)
	}
}

func WithBasicAuth(user, pass string) requestOpt {
	return func(r *http.Request) {
		r.SetBasicAuth(user, pass)
	}
}

func WithJsonBody() requestOpt {
	return func(r *http.Request) {
		r.Header.Add("Content-Type", "application/json")
	}
}
