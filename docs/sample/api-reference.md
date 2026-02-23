# API Reference

Omnidex exposes a REST API for document ingestion and repository listing. All API endpoints require Bearer token authentication.

## Authentication

All `/api/v1/*` endpoints require a Bearer token in the `Authorization` header:

```
Authorization: Bearer <your-api-key>
```

API keys are configured via the `API_API_KEYS` environment variable or the `api.api_keys` config field.

## Endpoints

### Health Check

```
GET /livez
```

Returns `200 OK` if the server is running. No authentication required.

**Response:**
```
Ok
```

### Ingest Documents

```
POST /api/v1/docs
```

Batch upsert or delete documents for a repository.

**Request Body:**
```json
{
  "repo": "owner/repo-name",
  "commit_sha": "abc123def456",
  "documents": [
    {
      "path": "docs/getting-started.md",
      "content": "# Getting Started\n\nYour markdown content here.",
      "action": "upsert"
    },
    {
      "path": "docs/old-page.md",
      "action": "delete"
    }
  ]
}
```

**Fields:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `repo` | string | yes | Repository identifier in `owner/name` format |
| `commit_sha` | string | yes | Git commit SHA for tracking |
| `documents` | array | yes | List of document operations |
| `documents[].path` | string | yes | File path relative to the docs root |
| `documents[].content` | string | for upsert | Markdown content of the document |
| `documents[].action` | string | yes | Either `"upsert"` or `"delete"` |

**Response (200 OK):**
```json
{
  "indexed": 2,
  "deleted": 0
}
```

### List Repositories

```
GET /api/v1/repos
```

Returns a list of all indexed repositories with document counts.

**Response (200 OK):**
```json
{
  "repos": [
    {
      "name": "owner/repo-name",
      "doc_count": 15,
      "last_updated": "2025-01-15T10:30:00Z"
    }
  ]
}
```

## Portal Routes

These routes serve HTML pages and do not require authentication:

| Route | Description |
|-------|-------------|
| `GET /` | Home page showing all indexed repositories |
| `GET /docs/{owner}/{repo}/{path...}` | Rendered documentation page |
| `GET /search?q={query}` | Search results page |
| `GET /static/*` | Static assets (CSS, JavaScript) |
