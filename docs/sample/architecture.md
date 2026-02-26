# Architecture Overview

Omnidex follows a clean architecture with clear separation of concerns. The application is built as a single Go binary with no external database dependencies.

## System Architecture

```mermaid
graph TD
    GHA["GitHub Action (action/)"] -->|"POST /api/v1/docs (Bearer auth)"| API

    subgraph API["API Server (pkg/api)"]
        MW["Middleware: reqID, auth"]
        Ingest["Ingest API (JSON)<br/>POST /api/v1/docs<br/>GET /api/v1/repos"]
        Portal["Portal (HTML + HTMX)<br/>GET / (home)<br/>GET /docs/{owner}/{repo}/{path}<br/>GET /search?q=..."]
    end

    API --> SVC["Service (pkg/core)"]
    SVC --> DocStore["DocStore (repo/)"]
    SVC --> Search["Search (repo/)"]
    SVC --> Markdown["Markdown (prov/)"]
    DocStore --> FS["Filesystem"]
    Search --> Bleve["Bleve Index"]
```

## Package Layout

| Package | Path | Responsibility |
|---------|------|----------------|
| **cmd** | `cmd/omnidex/` | Application entrypoint |
| **cli** | `pkg/cmd/` | CLI initialization, config loading, dependency wiring |
| **api** | `pkg/api/` | HTTP server, routing, handlers, middleware |
| **core** | `pkg/core/` | Business logic, domain types, service orchestration |
| **docstore** | `pkg/repo/docstore/` | Filesystem-based document storage |
| **search** | `pkg/repo/search/` | Full-text search engine (Bleve) |
| **markdown** | `pkg/prov/markdown/` | Markdown rendering and processing |
| **views** | `pkg/views/` | HTML template rendering for the portal |
| **action** | `action/` | GitHub Action for publishing docs |

## Request Flows

### Document Ingestion

```mermaid
sequenceDiagram
    participant GHA as GitHub Action
    participant API as API Server
    participant Core as Service
    participant DS as DocStore
    participant SE as Search Engine
    participant MD as Markdown Renderer

    GHA->>API: POST /api/v1/docs (Bearer auth)
    API->>Core: IngestDocuments(request)
    loop For each document
        Core->>MD: ExtractTitle(content)
        MD-->>Core: title
        Core->>MD: ToPlainText(content)
        MD-->>Core: plain text
        Core->>DS: Upsert(document)
        Core->>SE: Index(document, plainText)
    end
    Core-->>API: indexed count
    API-->>GHA: 200 OK
```

### Document Viewing

```mermaid
sequenceDiagram
    participant Browser
    participant API as API Server
    participant Core as Service
    participant DS as DocStore
    participant MD as Markdown Renderer
    participant Views as View Renderer

    Browser->>API: GET /docs/owner/repo/path
    API->>Core: GetDocument(owner, repo, path)
    Core->>DS: Get(owner, repo, path)
    DS-->>Core: document (raw markdown)
    Core->>MD: ToHTML(content)
    MD-->>Core: sanitized HTML
    Core-->>API: document + HTML
    API->>Views: RenderDoc(doc, html, navDocs)
    Views-->>API: HTML page
    API-->>Browser: HTML response
```

## Key Design Decisions

### Embedded Search

Omnidex uses [Bleve](https://blevesearch.com/) as an embedded full-text search engine. This eliminates the need for external search infrastructure like Elasticsearch while still providing features like highlighted search results and relevance scoring.

### Filesystem Storage

Documents are stored directly on the filesystem with JSON metadata sidecars. This makes the system simple to deploy, backup, and debug. Each document is stored as two files:

- The markdown content file
- A `.meta.json` sidecar with title, commit SHA, and timestamp

### HTMX Portal

The web portal uses server-rendered HTML enhanced with [HTMX](https://htmx.org/) for SPA-like navigation without a JavaScript framework. Handlers detect HTMX requests via the `HX-Request` header and return either full pages or partial content fragments.

### Dependency Inversion

The core business logic defines its own interfaces (unexported) for storage, search, and rendering. Concrete implementations are injected during application startup in `pkg/cmd/server.go`. This keeps the core package free of infrastructure concerns and makes it easy to test with mocks.
