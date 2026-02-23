# Getting Started with Omnidex

Omnidex is a centralized documentation portal that aggregates and serves markdown documentation from your GitHub repositories. It provides full-text search, a browsable web portal, and a simple ingest API.

## How It Works

1. **Publish** documentation from your repositories using the GitHub Action or the ingest API
2. **Search** across all indexed documentation using the built-in full-text search engine
3. **Browse** documentation through the web portal with navigation and syntax highlighting

## Quick Start with Docker

The fastest way to get Omnidex running locally:

```bash
# Clone the repository
git clone https://github.com/ksysoev/omnidex.git
cd omnidex

# Start Omnidex (automatically seeds sample documentation)
make up
```

After startup, visit [http://localhost:8080](http://localhost:8080) to access the portal.

## Publishing Your First Document

Once Omnidex is running, you can publish documentation using the ingest API:

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
        "content": "# Hello World\n\nThis is my first document.",
        "action": "upsert"
      }
    ]
  }'
```

## Using the GitHub Action

Add the Omnidex publish action to your repository's workflow:

```yaml
- uses: ksysoev/omnidex/action@main
  with:
    omnidex_url: https://docs.example.com
    api_key: ${{ secrets.OMNIDEX_API_KEY }}
    docs_path: docs
```

This will automatically publish all markdown files from the `docs` directory to your Omnidex instance on every push.
