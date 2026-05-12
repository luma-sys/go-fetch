# go-fetch

Simple HTTP client for Go with retry, timeout, and per-request options.

## Installation

```sh
go get github.com/luma-sys/go-fetch
```

## Quick start

```go
client := fetch.New("https://api.example.com",
    fetch.WithRetry(3),
    fetch.WithTimeout(10*time.Second),
    fetch.WithDefaultRequestOpts(fetch.WithBearerToken(token)),
)

type User struct {
    ID   string `json:"id"`
    Name string `json:"name"`
}

resp, err := client.Get(ctx, "/users/42")
if err != nil {
    return err
}

user, err := fetch.DecodeJSON[User](resp)
```

---

## Implementation example

Below is a complete example of a service client built on top of `go-fetch`.

```go
package orders

import (
    "context"
    "errors"
    "fmt"
    "net/http"
    "time"

    "github.com/luma-sys/go-fetch/fetch"
)

type Order struct {
    ID       string  `json:"id"`
    Product  string  `json:"product"`
    Quantity int     `json:"quantity"`
    Total    float64 `json:"total"`
}

type CreateOrderRequest struct {
    Product  string `json:"product"`
    Quantity int    `json:"quantity"`
}

type APIError struct {
    Code    string `json:"code"`
    Message string `json:"message"`
}

func (e *APIError) Error() string {
    return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

type Client struct {
    http fetch.FetchAPI
}

func NewClient(baseURL, token string) *Client {
    return &Client{
        http: fetch.New(baseURL,
            fetch.WithRetry(3),
            fetch.WithTimeout(15*time.Second),
            fetch.WithDefaultRequestOpts(
                fetch.WithBearerToken(token),
                fetch.WithJsonBody(),
            ),
        ),
    }
}

func (c *Client) GetOrder(ctx context.Context, id string) (*Order, error) {
    resp, err := c.http.Get(ctx, "/orders/"+id)
    if err != nil {
        return nil, c.wrapError(resp, err)
    }

    return fetch.DecodeJSON[Order](resp)
}

func (c *Client) ListOrders(ctx context.Context) ([]Order, error) {
    resp, err := c.http.Get(ctx, "/orders")
    if err != nil {
        return nil, c.wrapError(resp, err)
    }

    orders, err := fetch.DecodeJSON[[]Order](resp)
    if err != nil {
        return nil, err
    }
    return *orders, nil
}

func (c *Client) CreateOrder(ctx context.Context, req CreateOrderRequest) (*Order, error) {
    body, err := fetch.EncodeData(req)
    if err != nil {
        return nil, err
    }

    resp, err := c.http.Post(ctx, "/orders", body)
    if err != nil {
        return nil, c.wrapError(resp, err)
    }

    return fetch.DecodeJSON[Order](resp)
}

func (c *Client) CancelOrder(ctx context.Context, id string) error {
    resp, err := c.http.Delete(ctx, "/orders/"+id)
    if err != nil {
        return c.wrapError(resp, err)
    }
    return resp.Close()
}

// wrapError extracts the API error body when the server returned a structured error response.
// On network errors or when no response body is available, returns the original error.
func (c *Client) wrapError(resp *fetch.FetchResponse, err error) error {
    if resp == nil || resp.Response == nil {
        return err
    }

    if resp.StatusCode == http.StatusNotFound {
        defer resp.Close()
        return fmt.Errorf("not found: %w", err)
    }

    apiErr, decodeErr := fetch.DecodeJSON[APIError](resp)
    if decodeErr != nil || apiErr == nil {
        return err
    }
    return fmt.Errorf("%w: %s", err, apiErr)
}
```

### Usage

```go
client := orders.NewClient("https://api.example.com", os.Getenv("API_TOKEN"))

order, err := client.GetOrder(ctx, "ord_123")
if err != nil {
    if errors.Is(err, context.DeadlineExceeded) {
        log.Println("request timed out")
    }
    return err
}

fmt.Println(order.Product, order.Total)
```

### Debug mode

```go
client := fetch.New("https://api.example.com",
    fetch.WithDebugWriter(os.Stderr),
)
// Prints to stderr per request:
// curl -X POST 'https://api.example.com/orders' -d '{"product":"widget"}' -H 'Authorization: Bearer ...'
```

> Never use `WithDebugWriter` in production — output contains auth headers.

### Branching config per caller

`SetOptions` returns a new isolated client — the original is unchanged:

```go
base     := fetch.New("https://api.example.com", fetch.WithRetry(3))
noRetry  := base.SetOptions(fetch.WithRetry(0))   // one-shot, no retry
strict   := base.SetOptions(fetch.WithTimeout(2 * time.Second))
```

---

## Options reference

### Client options (`FetchOpt`)

| Function                                | Description                                                        |
| --------------------------------------- | ------------------------------------------------------------------ |
| `WithRetry(n int)`                      | Retry up to n times on 5xx + network errors. Negative treated as 0 |
| `WithTimeout(d time.Duration)`          | Per-attempt context timeout                                        |
| `WithHTTPClient(*http.Client)`          | Custom HTTP client (transport, TLS, proxy, mocks)                  |
| `WithDefaultRequestOpts(...RequestOpt)` | Default opts applied to every request (replaces prior defaults)    |
| `WithDebugWriter(io.Writer)`            | Print curl-equivalent per request. **Dev only** — exposes auth headers |

### Request options (`RequestOpt`)

| Function                           | Description                                |
| ---------------------------------- | ------------------------------------------ |
| `WithHeader(key, value string)`    | Add HTTP header (appends via `Header.Add`) |
| `WithBasicAuth(user, pass string)` | Basic Auth credentials                     |
| `WithBearerToken(token string)`    | `Authorization: Bearer <token>`            |
| `WithJsonBody()`                   | `Content-Type: application/json`           |

---

## Response handling

`*FetchResponse` embeds `*http.Response`. Always release it — either decode the body or close it:

```go
// decode JSON body
result, err := fetch.DecodeJSON[MyStruct](resp)

// discard body (DELETE, 204 No Content, etc.)
defer resp.Close()
```

On non-2xx responses, the library returns both `(*FetchResponse, error)`. This allows inspecting the error body from the API before returning to the caller.

---

## License

MIT — see [LICENSE](https://opensource.org/license/mit)

## Team

[Luma Sistemas](https://github.com/luma-sys) · Copyright 2025
