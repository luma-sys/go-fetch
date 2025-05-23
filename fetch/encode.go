package fetch

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
)

func EncodeData[T any](data T) (io.Reader, error) {
	var body bytes.Buffer
	err := json.NewEncoder(&body).Encode(data)
	if err != nil {
		return nil, err
	}

	return &body, nil
}

func DecodeJson[T any](r *http.Response) (*T, error) {
	defer r.Body.Close()

	var body T
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		return nil, err
	}

	return &body, nil
}
