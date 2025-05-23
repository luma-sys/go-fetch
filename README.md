# Go Fetch

Fetch is a simple HTTP client for Go.

## Summary

- [Go Fetch](#go-fetch)
  - [Summary](#summary)
  - [Usage](#usage)
  - [License](#license)
  - [Team](#team)

## Usage

To use MongoDB store:

```go
package some_package

import (
    "github.com/luma-sys/go-fetch/fetch"
    ...
)

type MyStruct struct {
    ID   string `json:"id"`
    Name string `json:"name"`
}

func SomeService() {
    f := fetch.New("https://example.com")

    responseJson, err := f.Get("/contacts/1", nil)
    if err != nil {
      return nil, err
    }

    response, err := fetch.DecodeJson[MyStruct](responseJson)
    if err != nil {
      return nil, err
    }
    ...
}
```

## License

This project is licensed under the MIT License - see the [LICENSE](https://opensource.org/license/mit) file for details.

## Team

[Luma Sistemas](https://github.com/luma-sys)

Copyright 2025 - [Luma Sistemas](https://github.com/luma-sys)
