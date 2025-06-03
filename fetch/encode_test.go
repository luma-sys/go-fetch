package fetch_test

import (
	"bytes"
	"encoding/json"
	"io"
	"reflect"
	"testing"

	"github.com/luma-sys/go-fetch/fetch"
)

type TestStruct struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

func TestEncodeData(t *testing.T) {
	// Arrange
	data := TestStruct{
		Name: "John",
		Age:  30,
	}

	// Act
	reader, err := fetch.EncodeData(data)
	// Assert
	if err != nil {
		t.Errorf("EncodeData returned unexpected error: %v", err)
	}

	content, err := io.ReadAll(reader)
	if err != nil {
		t.Errorf("Error while reading encoded content: %v", err)
	}

	var decodedData TestStruct
	err = json.Unmarshal(bytes.TrimSpace(content), &decodedData)
	if err != nil {
		t.Errorf("Error to decode content: %v", err)
	}

	if !reflect.DeepEqual(data, decodedData) {
		t.Errorf("Encoded content does not match expected. \nExpected: %+v\nreceived: %+v", data, decodedData)
	}
}

func TestEncodeData_Erro(t *testing.T) {
	// Teste com um canal, que não pode ser codificado em JSON
	ch := make(chan int)
	_, err := fetch.EncodeData(ch)
	if err == nil {
		t.Error("EncodeData must return an error for non-encodable types")
	}
}
