package core

import "time"

// ContentType identifies the format of a document's content.
type ContentType string

const (
	// ContentTypeMarkdown represents standard markdown documents.
	ContentTypeMarkdown ContentType = "markdown"
	// ContentTypeOpenAPI represents OpenAPI specification documents.
	ContentTypeOpenAPI ContentType = "openapi"
)

// Document represents a documentation file from a repository.
type Document struct {
	UpdatedAt   time.Time
	ID          string
	Repo        string
	Path        string
	Title       string
	Content     string
	CommitSHA   string
	ContentType ContentType
}

// DocumentMeta contains metadata about a document without its full content.
type DocumentMeta struct {
	UpdatedAt   time.Time
	ID          string
	Repo        string
	Path        string
	Title       string
	ContentType ContentType
}

// RepoInfo contains metadata about an indexed repository.
type RepoInfo struct {
	LastUpdated time.Time `json:"last_updated"`
	Name        string    `json:"name"`
	DocCount    int       `json:"doc_count"`
}

// SearchResult represents a single search result with highlighted snippets.
type SearchResult struct {
	ID               string
	Repo             string
	Path             string
	Title            string
	TitleFragments   []string // highlighted fragments from the title field
	ContentFragments []string // highlighted fragments from the content field
	Score            float64
}

// SearchResults holds the response from a search query.
type SearchResults struct {
	Hits     []SearchResult
	Total    uint64
	Duration time.Duration
}

// SearchOpts configures search behavior.
type SearchOpts struct {
	Limit  int
	Offset int
}

// IngestRequest represents a batch document ingest request from a GitHub Action.
type IngestRequest struct {
	Repo      string           `json:"repo"`
	CommitSHA string           `json:"commit_sha"`
	Documents []IngestDocument `json:"documents"`
	Sync      bool             `json:"sync,omitempty"`
}

// IngestDocument represents a single document in an ingest request.
type IngestDocument struct {
	Path        string      `json:"path"`
	Content     string      `json:"content,omitempty"`
	Action      string      `json:"action"`                 // "upsert" or "delete"
	ContentType ContentType `json:"content_type,omitempty"` // defaults to "markdown" when empty
}

// IngestResponse is returned after processing an ingest request.
type IngestResponse struct {
	Indexed int `json:"indexed"`
	Deleted int `json:"deleted"`
}

// Heading represents a heading extracted from a document for table of contents navigation.
type Heading struct {
	ID    string
	Text  string
	Level int
}
