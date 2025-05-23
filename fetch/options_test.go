package fetch_test

import (
	"github.com/luma-sys/go-fetch/fetch"
	"net/http"
	"reflect"
	"testing"
)

func TestWithBasicAuth(t *testing.T) {
	// Arrange
	requisicao, _ := http.NewRequest(http.MethodGet, "http://exemplo.com", nil)
	usuario := "admin"
	senha := "senha123"

	// Act
	opt := fetch.WithBasicAuth(usuario, senha)
	opt(requisicao)

	// Assert
	usuarioObtido, senhaObtida, ok := requisicao.BasicAuth()
	if !ok {
		t.Error("Basic auth não foi configurado corretamente")
	}

	if usuarioObtido != usuario {
		t.Errorf("Usuário incorreto. Esperado: %s, Obtido: %s", usuario, usuarioObtido)
	}

	if senhaObtida != senha {
		t.Errorf("Senha incorreta. Esperado: %s, Obtido: %s", senha, senhaObtida)
	}
}

func TestWithHeader(t *testing.T) {
	testes := []struct {
		nome      string
		chave     string
		valor     string
		esperado  []string
		existente string
	}{
		{
			nome:     "Header simples",
			chave:    "Content-Type",
			valor:    "application/json",
			esperado: []string{"application/json"},
		},
		{
			nome:      "Header com valor existente",
			chave:     "Accept",
			valor:     "application/xml",
			existente: "application/json",
			esperado:  []string{"application/json", "application/xml"},
		},
		{
			nome:     "Header personalizado",
			chave:    "X-Custom-Header",
			valor:    "valor-teste",
			esperado: []string{"valor-teste"},
		},
	}

	for _, tt := range testes {
		t.Run(tt.nome, func(t *testing.T) {
			// Arrange
			requisicao, _ := http.NewRequest(http.MethodGet, "http://exemplo.com", nil)

			// Adiciona header existente, se especificado
			if tt.existente != "" {
				requisicao.Header.Add(tt.chave, tt.existente)
			}

			// Act
			opt := fetch.WithHeader(tt.chave, tt.valor)
			opt(requisicao)

			// Assert
			valoresObtidos := requisicao.Header.Values(tt.chave)
			if !reflect.DeepEqual(valoresObtidos, tt.esperado) {
				t.Errorf("Header incorreto para %s.\nEsperado: %v\nObtido: %v",
					tt.chave, tt.esperado, valoresObtidos)
			}

		})
	}
}

func TestHeadersMultiplos(t *testing.T) {
	// Arrange
	requisicao, _ := http.NewRequest(http.MethodGet, "http://exemplo.com", nil)
	headers := []struct {
		chave string
		valor string
	}{
		{"Accept", "application/json"},
		{"X-API-Key", "chave123"},
		{"User-Agent", "TestClient"},
	}

	// Act
	for _, h := range headers {
		opt := fetch.WithHeader(h.chave, h.valor)
		opt(requisicao)
	}

	// Assert
	for _, h := range headers {
		valor := requisicao.Header.Get(h.chave)
		if valor != h.valor {
			t.Errorf("Header %s incorreto.\nEsperado: %s\nObtido: %s",
				h.chave, h.valor, valor)
		}
	}
}
