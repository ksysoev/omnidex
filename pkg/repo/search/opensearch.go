package search

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/ksysoev/omnidex/pkg/core"
	"github.com/opensearch-project/opensearch-go/v4"
	"github.com/opensearch-project/opensearch-go/v4/opensearchapi"
)

// OpenSearchConfig holds configuration for the OpenSearch backend.
type OpenSearchConfig struct {
	Index     string   `mapstructure:"index"`
	Username  string   `mapstructure:"username"`
	Password  string   `mapstructure:"password"`
	CACert    string   `mapstructure:"ca_cert"`
	Addresses []string `mapstructure:"addresses"`
}

// OpenSearchEngine implements full-text search using OpenSearch.
type OpenSearchEngine struct {
	client *opensearchapi.Client
	index  string
}

// NewOpenSearch creates a new OpenSearch search engine.
// It configures the client and ensures the index exists with the correct mapping.
func NewOpenSearch(ctx context.Context, cfg *OpenSearchConfig) (*OpenSearchEngine, error) {
	index := cfg.Index
	if index == "" {
		index = "omnidex"
	}

	osCfg := opensearchapi.Config{
		Client: buildOSClientConfig(cfg),
	}

	client, err := opensearchapi.NewClient(osCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create opensearch client: %w", err)
	}

	engine := &OpenSearchEngine{
		client: client,
		index:  index,
	}

	if err := engine.ensureIndex(ctx); err != nil {
		return nil, fmt.Errorf("failed to ensure opensearch index: %w", err)
	}

	return engine, nil
}

// buildOSClientConfig converts OpenSearchConfig to the opensearch transport config.
func buildOSClientConfig(cfg *OpenSearchConfig) opensearch.Config {
	c := opensearch.Config{
		Addresses: cfg.Addresses,
		Username:  cfg.Username,
		Password:  cfg.Password,
	}

	if cfg.CACert != "" {
		cert, err := readFile(cfg.CACert)
		if err == nil {
			c.CACert = cert
		}
	}

	return c
}

// Index adds or updates a document in the OpenSearch index.
func (e *OpenSearchEngine) Index(ctx context.Context, doc core.Document, plainText string) error { //nolint:gocritic // Document is passed by value for immutability
	body := map[string]string{
		"title":   doc.Title,
		"content": plainText,
		"repo":    doc.Repo,
		"path":    doc.Path,
	}

	data, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to marshal document %s: %w", doc.ID, err)
	}

	resp, err := e.client.Index(ctx, opensearchapi.IndexReq{
		Index:      e.index,
		DocumentID: doc.ID,
		Body:       bytes.NewReader(data),
		Params:     opensearchapi.IndexParams{Refresh: "false"},
	})
	if err != nil {
		return fmt.Errorf("failed to index document %s: %w", doc.ID, err)
	}

	if resp.Inspect().Response.IsError() {
		return fmt.Errorf("opensearch index error for %s: %s", doc.ID, resp.Inspect().Response.String())
	}

	return nil
}

// Remove deletes a document from the OpenSearch index.
func (e *OpenSearchEngine) Remove(ctx context.Context, docID string) error {
	resp, err := e.client.Document.Delete(ctx, opensearchapi.DocumentDeleteReq{
		Index:      e.index,
		DocumentID: docID,
	})

	// Document.Delete returns an error for non-2xx responses including 404.
	// A 404 is acceptable — the document may already be gone.
	if err != nil {
		if resp != nil && resp.Inspect().Response.StatusCode == http.StatusNotFound {
			return nil
		}

		return fmt.Errorf("failed to remove document %s: %w", docID, err)
	}

	return nil
}

// Search performs a full-text search query against OpenSearch and returns matching results.
func (e *OpenSearchEngine) Search(ctx context.Context, query string, opts core.SearchOpts) (*core.SearchResults, error) {
	if opts.Limit <= 0 {
		opts.Limit = 20
	}

	esQuery := e.buildSearchQuery(query)

	body := map[string]any{
		"query":   esQuery,
		"size":    opts.Limit,
		"from":    opts.Offset,
		"_source": []string{"repo", "path", "title"},
		"highlight": map[string]any{
			"fields": map[string]any{
				"title":   map[string]any{"number_of_fragments": 3},
				"content": map[string]any{"fragment_size": 200, "number_of_fragments": 3},
			},
			"pre_tags":  []string{"<mark>"},
			"post_tags": []string{"</mark>"},
		},
	}

	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal search query: %w", err)
	}

	start := time.Now()

	resp, err := e.client.Search(ctx, &opensearchapi.SearchReq{
		Indices: []string{e.index},
		Body:    bytes.NewReader(data),
		Params:  opensearchapi.SearchParams{TrackTotalHits: true},
	})
	if err != nil {
		return nil, fmt.Errorf("search request failed: %w", err)
	}

	if resp.Inspect().Response.IsError() {
		return nil, fmt.Errorf("opensearch search error: %s", resp.Inspect().Response.String())
	}

	duration := time.Since(start)

	hits := make([]core.SearchResult, 0, len(resp.Hits.Hits))

	for i := range resp.Hits.Hits {
		hit := &resp.Hits.Hits[i]

		var src esSource
		if err := json.Unmarshal(hit.Source, &src); err != nil {
			return nil, fmt.Errorf("failed to decode hit source: %w", err)
		}

		sr := core.SearchResult{
			ID:               hit.ID,
			Score:            float64(hit.Score),
			Repo:             src.Repo,
			Path:             src.Path,
			Title:            src.Title,
			TitleFragments:   hit.Highlight["title"],
			ContentFragments: hit.Highlight["content"],
		}
		hits = append(hits, sr)
	}

	total := uint64(0)
	if resp.Hits.Total.Value > 0 {
		total = uint64(resp.Hits.Total.Value)
	}

	return &core.SearchResults{
		Hits:     hits,
		Total:    total,
		Duration: duration,
	}, nil
}

// ListByRepo returns the IDs of all documents in the index that belong to the given repository.
func (e *OpenSearchEngine) ListByRepo(ctx context.Context, repo string) ([]string, error) {
	ids := make([]string, 0, esListByRepoPageSize)

	var searchAfter []any

	for {
		body := map[string]any{
			"query": map[string]any{
				"term": map[string]any{
					"repo": repo,
				},
			},
			"size":    esListByRepoPageSize,
			"_source": false,
			"sort":    []any{"_doc"},
		}

		if searchAfter != nil {
			body["search_after"] = searchAfter
		}

		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal list query: %w", err)
		}

		resp, err := e.client.Search(ctx, &opensearchapi.SearchReq{
			Indices: []string{e.index},
			Body:    bytes.NewReader(data),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to list documents for repo %s: %w", repo, err)
		}

		if resp.Inspect().Response.IsError() {
			return nil, fmt.Errorf("opensearch list error for repo %s: %s", repo, resp.Inspect().Response.String())
		}

		if len(resp.Hits.Hits) == 0 {
			break
		}

		for i := range resp.Hits.Hits {
			ids = append(ids, resp.Hits.Hits[i].ID)
		}

		lastHit := resp.Hits.Hits[len(resp.Hits.Hits)-1]
		searchAfter = lastHit.Sort
	}

	return ids, nil
}

// ensureIndex creates the OpenSearch index with correct mappings if it does not already exist.
func (e *OpenSearchEngine) ensureIndex(ctx context.Context) error {
	existsResp, err := e.client.Indices.Exists(ctx, opensearchapi.IndicesExistsReq{
		Indices: []string{e.index},
	})

	// Indices.Exists returns an error for any non-2xx response.
	// We need to distinguish 404 (index missing) from other errors.
	if err != nil {
		switch {
		case existsResp != nil && existsResp.StatusCode == http.StatusNotFound:
			// Index does not exist — fall through to create it.
		case existsResp != nil:
			return fmt.Errorf("unexpected status checking index existence: %s", existsResp.String())
		default:
			return fmt.Errorf("failed to check index existence: %w", err)
		}
	} else {
		// 2xx — index already exists.
		return nil
	}

	mapping := map[string]any{
		"mappings": map[string]any{
			"properties": map[string]any{
				"title": map[string]any{
					"type":        "text",
					"analyzer":    "standard",
					"term_vector": "with_positions_offsets",
				},
				"content": map[string]any{
					"type":        "text",
					"analyzer":    "standard",
					"term_vector": "with_positions_offsets",
				},
				"repo": map[string]any{
					"type": "keyword",
				},
				"path": map[string]any{
					"type": "keyword",
				},
			},
		},
	}

	data, err := json.Marshal(mapping)
	if err != nil {
		return fmt.Errorf("failed to marshal index mapping: %w", err)
	}

	createResp, err := e.client.Indices.Create(ctx, opensearchapi.IndicesCreateReq{
		Index: e.index,
		Body:  bytes.NewReader(data),
	})
	if err != nil {
		return fmt.Errorf("failed to create index: %w", err)
	}

	if createResp.Inspect().Response.IsError() {
		return fmt.Errorf("opensearch create index error: %s", createResp.Inspect().Response.String())
	}

	return nil
}

// buildSearchQuery constructs an OpenSearch query DSL from user input.
// It mirrors the hybrid query logic from ElasticEngine.buildSearchQuery.
func (e *OpenSearchEngine) buildSearchQuery(userQuery string) map[string]any {
	return buildQueryDSL(userQuery)
}
