package fetch

import "net/http"

type requestOpt func(r *http.Request)

func WithBasicAuth(user, pass string) requestOpt {
	return func(r *http.Request) {
		r.SetBasicAuth(user, pass)
	}
}

func WithHeader(key, value string) requestOpt {
	return func(r *http.Request) {
		r.Header.Add(key, value)
	}
}
