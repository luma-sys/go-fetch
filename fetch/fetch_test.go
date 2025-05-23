package fetch_test

import (
	"context"
	"github.com/luma-sys/go-fetch/fetch"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	t.Run("Criação sem opções de retry", func(t *testing.T) {
		// Arrange & Act
		servidor := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer servidor.Close()

		cliente := fetch.New(servidor.URL)

		// Act
		_, err := cliente.Get("/teste", nil)

		// Assert
		if err == nil {
			t.Error("Esperava erro com status 500, mas não recebeu erro")
		}
	})

	t.Run("Criação com retry", func(t *testing.T) {
		// Arrange
		tentativas := 0
		servidor := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tentativas++
			if tentativas < 3 {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusOK)
		}))
		defer servidor.Close()

		cliente := fetch.New(servidor.URL, fetch.WithRetry(2))

		// Act
		resp, err := cliente.Get("/teste", nil)

		// Assert
		if err != nil {
			t.Errorf("Não esperava erro, mas recebeu: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Status code incorreto. Esperado: %d, Obtido: %d", http.StatusOK, resp.StatusCode)
		}
		if tentativas != 3 {
			t.Errorf("Número incorreto de tentativas. Esperado: 3, Obtido: %d", tentativas)
		}
	})
}

func TestMetodosHTTP(t *testing.T) {
	type testCase struct {
		nome          string
		método        func(cliente fetch.FetchAPI, path string, body io.Reader) (*http.Response, error)
		metodoChamado string
		statusCode    int
		corpo         string
	}

	testes := []testCase{
		{
			nome: "GET sucesso",
			método: func(c fetch.FetchAPI, p string, b io.Reader) (*http.Response, error) {
				return c.Get(p, b)
			},
			metodoChamado: http.MethodGet,
			statusCode:    http.StatusOK,
			corpo:         "teste get",
		},
		{
			nome: "POST sucesso",
			método: func(c fetch.FetchAPI, p string, b io.Reader) (*http.Response, error) {
				return c.Post(p, b)
			},
			metodoChamado: http.MethodPost,
			statusCode:    http.StatusCreated,
			corpo:         "teste post",
		},
		{
			nome: "PUT sucesso",
			método: func(c fetch.FetchAPI, p string, b io.Reader) (*http.Response, error) {
				return c.Put(p, b)
			},
			metodoChamado: http.MethodPut,
			statusCode:    http.StatusOK,
			corpo:         "teste put",
		},
		{
			nome: "PATCH sucesso",
			método: func(c fetch.FetchAPI, p string, b io.Reader) (*http.Response, error) {
				return c.Patch(p, b)
			},
			metodoChamado: http.MethodPatch,
			statusCode:    http.StatusOK,
			corpo:         "teste patch",
		},
		{
			nome: "DELETE sucesso",
			método: func(c fetch.FetchAPI, p string, b io.Reader) (*http.Response, error) {
				return c.Delete(p, b)
			},
			metodoChamado: http.MethodDelete,
			statusCode:    http.StatusNoContent,
			corpo:         "",
		},
	}

	for _, tt := range testes {
		t.Run(tt.nome, func(t *testing.T) {
			// Arrange
			servidor := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != tt.metodoChamado {
					t.Errorf("Método HTTP incorreto. Esperado: %s, Obtido: %s", tt.metodoChamado, r.Method)
				}
				w.WriteHeader(tt.statusCode)
				if tt.corpo != "" {
					w.Write([]byte(tt.corpo))
				}
			}))
			defer servidor.Close()

			cliente := fetch.New(servidor.URL)
			body := strings.NewReader("request body")

			// Act
			resp, err := tt.método(cliente, "/teste", body)

			// Assert
			if err != nil {
				t.Errorf("Não esperava erro, mas recebeu: %v", err)
			}
			if resp.StatusCode != tt.statusCode {
				t.Errorf("Status code incorreto. Esperado: %d, Obtido: %d", tt.statusCode, resp.StatusCode)
			}
		})
	}
}

func TestGetWithContext(t *testing.T) {
	t.Run("Contexto cancelado", func(t *testing.T) {
		// Arrange
		servidor := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(200 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
		}))
		defer servidor.Close()

		cliente := fetch.New(servidor.URL)
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		// Act
		_, err := cliente.GetWithContext(ctx, "/teste", nil)

		// Assert
		if err == nil {
			t.Error("Esperava erro de contexto cancelado")
		}
	})

	t.Run("Contexto válido", func(t *testing.T) {
		// Arrange
		servidor := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer servidor.Close()

		cliente := fetch.New(servidor.URL)
		ctx := context.Background()

		// Act
		resp, err := cliente.GetWithContext(ctx, "/teste", nil)

		// Assert
		if err != nil {
			t.Errorf("Não esperava erro, mas recebeu: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Status code incorreto. Esperado: %d, Obtido: %d", http.StatusOK, resp.StatusCode)
		}
	})
}

func TestErros(t *testing.T) {
	t.Run("URL inválida", func(t *testing.T) {
		// Arrange
		cliente := fetch.New("http://invalid-url")

		// Act
		_, err := cliente.Get("/teste", nil)

		// Assert
		if err == nil {
			t.Error("Esperava erro com URL inválida")
		}
	})

	t.Run("Status code não esperado", func(t *testing.T) {
		// Arrange
		servidor := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
		}))
		defer servidor.Close()

		cliente := fetch.New(servidor.URL)

		// Act
		_, err := cliente.Get("/teste", nil)

		// Assert
		if err == nil {
			t.Error("Esperava erro com status code 400")
		}
	})
}
