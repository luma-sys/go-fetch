package fetch

import (
	"context"
	"encoding/json"
	"net/http"
)

type FetchResponse struct {
	*http.Response
	cancel context.CancelFunc
}

func (r *FetchResponse) DecodeJson(body *any) error {
	defer r.Body.Close()
	if r.cancel != nil {
		defer r.cancel()
	}

	err := json.NewDecoder(r.Body).Decode(body)
	return err
}
