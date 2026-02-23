# omnidex

[![Tests](https://github.com/ksysoev/omnidex/actions/workflows/tests.yml/badge.svg)](https://github.com/ksysoev/omnidex/actions/workflows/tests.yml)
[![codecov](https://codecov.io/gh/ksysoev/omnidex/graph/badge.svg?token=9PJI30S0XX)](https://codecov.io/gh/ksysoev/omnidex)
[![Go Report Card](https://goreportcard.com/badge/github.com/ksysoev/omnidex)](https://goreportcard.com/report/github.com/ksysoev/omnidex)
[![Go Reference](https://pkg.go.dev/badge/github.com/ksysoev/omnidex.svg)](https://pkg.go.dev/github.com/ksysoev/omnidex)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)

Centralized documentation portal for your repos

## Installation

## Building from Source

```sh
CGO_ENABLED=0 go build -o omnidex -ldflags "-X main.version=dev -X main.name=omnidex" ./cmd/omnidex/main.go
```

### Using Go

If you have Go installed, you can install omnidex directly:

```sh
go install github.com/ksysoev/omnidex/cmd/omnidex@latest
```


## Using

```sh
omnidex --log-level=debug --log-text=true --config=runtime/config.yml
```

## License

omnidex is licensed under the MIT License. See the LICENSE file for more details.
