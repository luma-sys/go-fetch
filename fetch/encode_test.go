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
	Nome  string `json:"nome"`
	Idade int    `json:"idade"`
}

func TestEncodeData(t *testing.T) {
	// Arrange
	dados := TestStruct{
		Nome:  "João",
		Idade: 30,
	}

	// Act
	reader, err := fetch.EncodeData(dados)

	// Assert
	if err != nil {
		t.Errorf("EncodeData retornou erro inesperado: %v", err)
	}

	// Verificar se o conteúdo codificado está correto
	conteudo, err := io.ReadAll(reader)
	if err != nil {
		t.Errorf("Erro ao ler conteúdo codificado: %v", err)
	}

	var decodificado TestStruct
	err = json.Unmarshal(bytes.TrimSpace(conteudo), &decodificado)
	if err != nil {
		t.Errorf("Erro ao decodificar conteúdo: %v", err)
	}

	if !reflect.DeepEqual(dados, decodificado) {
		t.Errorf("Conteúdo codificado não corresponde ao esperado.\nEsperado: %+v\nObtido: %+v", dados, decodificado)
	}
}

func TestDecodeJson(t *testing.T) {
	// Arrange
	dadosEsperados := TestStruct{
		Nome:  "Maria",
		Idade: 25,
	}

	jsonData, _ := json.Marshal(dadosEsperados)
	resposta := &http.Response{
		Body: io.NopCloser(bytes.NewBuffer(jsonData)),
	}

	// Act
	resultado, err := fetch.DecodeJson[TestStruct](resposta)

	// Assert
	if err != nil {
		t.Errorf("DecodeJson retornou erro inesperado: %v", err)
	}

	if resultado == nil {
		t.Fatal("DecodeJson retornou nil")
	}

	if !reflect.DeepEqual(*resultado, dadosEsperados) {
		t.Errorf("Conteúdo decodificado não corresponde ao esperado.\nEsperado: %+v\nObtido: %+v", dadosEsperados, *resultado)
	}
}

func TestEncodeData_Erro(t *testing.T) {
	// Teste com um canal, que não pode ser codificado em JSON
	ch := make(chan int)
	_, err := fetch.EncodeData(ch)
	if err == nil {
		t.Error("EncodeData deveria retornar erro para tipos não codificáveis")
	}
}

func TestDecodeJson_Erro(t *testing.T) {
	// Arrange
	jsonInvalido := `{"nome": "João", "idade": "invalid"}`
	resposta := &http.Response{
		Body: io.NopCloser(strings.NewReader(jsonInvalido)),
	}

	// Act
	_, err := fetch.DecodeJson[TestStruct](resposta)

	// Assert
	if err == nil {
		t.Error("DecodeJson deveria retornar erro para JSON inválido")
	}
}
