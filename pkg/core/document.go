package core

import "time"

// Document represents a documentation file from a repository.
type Document struct {
	UpdatedAt time.Time
	ID        string
	Repo      string
	Path      string
	Title     string
	Content   string
	CommitSHA string
}

// DocumentMeta contains metadata about a document without its full content.
type DocumentMeta struct {
	UpdatedAt time.Time
	ID        string
	Repo      string
	Path      string
	Title     string
}

// RepoInfo contains metadata about an indexed repository.
type RepoInfo struct {
	LastUpdated time.Time
	Name        string
	DocCount    int
}

// SearchResult represents a single search result with highlighted snippets.
type SearchResult struct {
	ID        string
	Repo      string
	Path      string
	Title     string
	Fragments []string
	Score     float64
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
}

// IngestDocument represents a single document in an ingest request.
type IngestDocument struct {
	Path    string `json:"path"`
	Content string `json:"content,omitempty"`
	Action  string `json:"action"` // "upsert" or "delete"
}

// IngestResponse is returned after processing an ingest request.
type IngestResponse struct {
	Indexed int `json:"indexed"`
	Deleted int `json:"deleted"`
}
