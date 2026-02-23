# omnidex

[![Tests](https://github.com/ksysoev/omnidex/actions/workflows/tests.yml/badge.svg)](https://github.com/ksysoev/omnidex/actions/workflows/tests.yml)
[![codecov](https://codecov.io/gh/ksysoev/omnidex/graph/badge.svg?token=9PJI30S0XX)](https://codecov.io/gh/ksysoev/omnidex)
[![Go Report Card](https://goreportcard.com/badge/github.com/ksysoev/omnidex)](https://goreportcard.com/report/github.com/ksysoev/omnidex)
[![Go Reference](https://pkg.go.dev/badge/github.com/ksysoev/omnidex.svg)](https://pkg.go.dev/github.com/ksysoev/omnidex)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)

Centralized documentation portal for your repos. Omnidex aggregates markdown documentation from GitHub repositories and serves it through a searchable web portal.

## Prerequisites

- **Go 1.24+** — for building from source
- **Docker** and **Docker Compose** — for the containerized development environment
- **Tailwind CSS CLI** — for frontend development (`npm install -g tailwindcss` or [standalone binary](https://tailwindcss.com/blog/standalone-cli))

## Quick Start

The fastest way to get Omnidex running locally is with Docker Compose:

```bash
# Clone the repository
git clone https://github.com/ksysoev/omnidex.git
cd omnidex

# Copy the example environment file
cp .env.example .env

# Start Omnidex
make up

# (Optional) Seed with sample documentation
make seed
```

After startup, visit [http://localhost:8080](http://localhost:8080) to access the portal.

To stop the environment:

```bash
make down
```

## Configuration

Omnidex is configured via a YAML file and/or environment variables. Environment variables take precedence over the config file.

| YAML Key | Environment Variable | Default | Description |
|----------|---------------------|---------|-------------|
| `api.listen` | `API_LISTEN` | `:8080` | Address and port for the HTTP server |
| `api.api_keys` | `API_API_KEYS` | `changeme` | Comma-separated list of API keys for authentication |
| `storage.path` | `STORAGE_PATH` | `./data/repos` | Filesystem path for document storage |
| `search.index_path` | `SEARCH_INDEX_PATH` | `./data/search.bleve` | Path for the Bleve search index |
| — | `LOG_LEVEL` | `info` | Log level (`debug`, `info`, `warn`, `error`) |
| — | `LOG_TEXT` | `true` | Use text format for logs (`true`) or JSON (`false`) |

See [`.env.example`](.env.example) for a complete example.

## Development

### Building from Source

```bash
# Build the binary
make build

# Or manually with ldflags
CGO_ENABLED=0 go build -o omnidex -ldflags "-X main.version=dev -X main.name=omnidex" ./cmd/omnidex/main.go
```

### Running Locally (without Docker)

```bash
# Create data directories
mkdir -p data/repos data

# Build Tailwind CSS
make tailwind

# Run the server
make run
```

This uses the development config at `runtime/config.yml`.

### Installing via Go

```bash
go install github.com/ksysoev/omnidex/cmd/omnidex@latest
```

### Frontend Development

To watch and rebuild Tailwind CSS on changes:

```bash
make dev-css
```

## Publishing Docs

### Using the Ingest API

Send documentation to a running Omnidex instance via the REST API:

```bash
curl -X POST http://localhost:8080/api/v1/docs \
  -H "Authorization: Bearer changeme" \
  -H "Content-Type: application/json" \
  -d '{
    "repo": "myorg/myrepo",
    "commit_sha": "abc123",
    "documents": [
      {
        "path": "getting-started.md",
        "content": "# Hello\n\nYour markdown content here.",
        "action": "upsert"
      }
    ]
  }'
```

### Using the GitHub Action

Add the Omnidex publish action to your repository's CI workflow:

```yaml
- uses: ksysoev/omnidex/action@main
  with:
    omnidex_url: https://docs.example.com
    api_key: ${{ secrets.OMNIDEX_API_KEY }}
    docs_path: docs
```

This will publish all markdown files from the `docs` directory on every push.

## Testing

```bash
# Run unit tests with race detector
make test

# Run linter
make lint

# Generate mocks
make mocks
```

## Project Structure

```
cmd/omnidex/          Application entrypoint (main.go)
pkg/
  cmd/                CLI initialization, config loading, dependency wiring
  api/                HTTP server, routing, handlers
    middleware/        Authentication, request ID middleware
  core/               Business logic, domain types, service layer
  repo/
    docstore/         Filesystem-based document storage
    search/           Full-text search engine (Bleve)
  prov/
    markdown/         Markdown rendering and processing (goldmark)
  views/              HTML template rendering (Go templates + HTMX)
action/               GitHub Action for publishing docs
docs/sample/          Sample documentation for local development
scripts/              Development utility scripts
static/               Static assets (CSS, JavaScript)
runtime/              Development configuration files
```

## License

omnidex is licensed under the MIT License. See the [LICENSE](LICENSE) file for more details.
