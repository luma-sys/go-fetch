package fetch

import (
	"bytes"
	"encoding/json"
	"io"
)

func EncodeData[T any](data T) (io.Reader, error) {
	var body bytes.Buffer
	if err := json.NewEncoder(&body).Encode(data); err != nil {
		return nil, err
	}

	return &body, nil
}
