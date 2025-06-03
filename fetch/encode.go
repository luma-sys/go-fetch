package fetch

import (
	"bytes"
	"encoding/json"
	"io"
)

func EncodeData[T any](data T) (io.Reader, error) {
	var body bytes.Buffer
	err := json.NewEncoder(&body).Encode(data)
	if err != nil {
		return nil, err
	}

	return &body, nil
}

// deprecated: prefer fetchResponse.DecodeJson
func DecodeJson[T any](r *FetchResponse) (*T, error) {
	defer r.Body.Close()
	if r.cancel != nil {
		defer r.cancel()
	}

	var body T
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		return nil, err
	}

	return &body, nil
}
