package fetch_test

import (
	"bytes"
	"encoding/json"
	"github.com/luma-sys/go-fetch/fetch"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
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

func TestDecodeJson(t *testing.T) {
	// Arrange
	expectedData := TestStruct{
		Name: "Mary",
		Age:  25,
	}

	jsonData, _ := json.Marshal(expectedData)
	response := &http.Response{
		Body: io.NopCloser(bytes.NewBuffer(jsonData)),
	}

	// Act
	result, err := fetch.DecodeJson[TestStruct](response)

	// Assert
	if err != nil {
		t.Errorf("DecodeJson returned unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("DecodeJson returned nil")
	}

	if !reflect.DeepEqual(*result, expectedData) {
		t.Errorf("Content decoded does not match expected.\nExpected: %+v\nreceived: %+v", expectedData, *result)
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

func TestDecodeJson_Erro(t *testing.T) {
	// Arrange
	invalidJson := `{"name": "John", "age": "invalid"}`
	response := &http.Response{
		Body: io.NopCloser(strings.NewReader(invalidJson)),
	}

	// Act
	_, err := fetch.DecodeJson[TestStruct](response)

	// Assert
	if err == nil {
		t.Error("DecodeJson must return an error for invalid JSON data")
	}
}
