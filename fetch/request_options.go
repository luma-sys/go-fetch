package fetch

import "net/http"

type RequestOpt func(r *http.Request)

func WithBasicAuth(user, pass string) RequestOpt {
	return func(r *http.Request) {
		r.SetBasicAuth(user, pass)
	}
}

func WithHeader(key, value string) RequestOpt {
	return func(r *http.Request) {
		r.Header.Add(key, value)
	}
}

func WithBearerToken(token string) RequestOpt {
	return WithHeader("Authorization", "Bearer "+token)
}

func WithJsonBody() RequestOpt {
	return WithHeader("Content-Type", "application/json")
}
