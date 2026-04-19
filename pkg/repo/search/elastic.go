package search

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/ksysoev/omnidex/pkg/core"
)

// ElasticSearchConfig holds configuration for the Elasticsearch/OpenSearch backend.
type ElasticSearchConfig struct {
	Index      string   `mapstructure:"index"`
	Username   string   `mapstructure:"username"`
	Password   string   `mapstructure:"password"`
	APIKey     string   `mapstructure:"api_key"`
	CACert     string   `mapstructure:"ca_cert"`
	Addresses  []string `mapstructure:"addresses"`
	OpenSearch bool     `mapstructure:"opensearch"`
}

// ElasticEngine implements full-text search using Elasticsearch or OpenSearch.
type ElasticEngine struct {
	client *elasticsearch.Client
	index  string
}

// NewElastic creates a new Elasticsearch search engine.
// It configures the client, verifies connectivity, and ensures the index exists with the correct mapping.
func NewElastic(cfg *ElasticSearchConfig) (*ElasticEngine, error) {
	index := cfg.Index
	if index == "" {
		index = "omnidex"
	}

	esCfg := elasticsearch.Config{
		Addresses: cfg.Addresses,
		Username:  cfg.Username,
		Password:  cfg.Password,
	}

	if cfg.APIKey != "" {
		esCfg.APIKey = cfg.APIKey
	}

	if cfg.CACert != "" {
		cert, err := readCACert(cfg.CACert)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA cert: %w", err)
		}

		esCfg.CACert = cert
	}

	if cfg.OpenSearch {
		// Enable compatibility mode for OpenSearch.
		esCfg.Header = http.Header{
			"Content-Type": []string{"application/json; compatible-with=7"},
			"Accept":       []string{"application/json; compatible-with=7"},
		}
	}

	client, err := elasticsearch.NewClient(esCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create elasticsearch client: %w", err)
	}

	engine := &ElasticEngine{
		client: client,
		index:  index,
	}

	if err := engine.ensureIndex(); err != nil {
		return nil, fmt.Errorf("failed to ensure elasticsearch index: %w", err)
	}

	return engine, nil
}

// readCACert reads a CA certificate file and returns its contents.
func readCACert(path string) ([]byte, error) {
	// Use os.ReadFile via the standard approach; imported at top level.
	return readFile(path)
}

// Index adds or updates a document in the Elasticsearch index.
func (e *ElasticEngine) Index(ctx context.Context, doc core.Document, plainText string) error { //nolint:gocritic // Document is passed by value for immutability
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

	resp, err := e.client.Index(
		e.index,
		bytes.NewReader(data),
		e.client.Index.WithContext(ctx),
		e.client.Index.WithDocumentID(doc.ID),
		e.client.Index.WithRefresh("false"),
	)
	if err != nil {
		return fmt.Errorf("failed to index document %s: %w", doc.ID, err)
	}
	defer resp.Body.Close()

	if resp.IsError() {
		return fmt.Errorf("elasticsearch index error for %s: %s", doc.ID, resp.String())
	}

	return nil
}

// Remove deletes a document from the Elasticsearch index.
func (e *ElasticEngine) Remove(ctx context.Context, docID string) error {
	resp, err := e.client.Delete(
		e.index,
		docID,
		e.client.Delete.WithContext(ctx),
		e.client.Delete.WithRefresh("false"),
	)
	if err != nil {
		return fmt.Errorf("failed to remove document %s: %w", docID, err)
	}
	defer resp.Body.Close()

	// 404 is acceptable — the document may already be gone.
	if resp.IsError() && resp.StatusCode != http.StatusNotFound {
		return fmt.Errorf("elasticsearch delete error for %s: %s", docID, resp.String())
	}

	return nil
}

// Search performs a full-text search query against Elasticsearch and returns matching results.
func (e *ElasticEngine) Search(ctx context.Context, query string, opts core.SearchOpts) (*core.SearchResults, error) {
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

	resp, err := e.client.Search(
		e.client.Search.WithContext(ctx),
		e.client.Search.WithIndex(e.index),
		e.client.Search.WithBody(bytes.NewReader(data)),
		e.client.Search.WithTrackTotalHits(true),
	)
	if err != nil {
		return nil, fmt.Errorf("search request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.IsError() {
		return nil, fmt.Errorf("elasticsearch search error: %s", resp.String())
	}

	duration := time.Since(start)

	var result esSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode search response: %w", err)
	}

	hits := make([]core.SearchResult, 0, len(result.Hits.Hits))

	for _, hit := range result.Hits.Hits {
		sr := core.SearchResult{
			ID:               hit.ID,
			Score:            hit.Score,
			Repo:             hit.Source.Repo,
			Path:             hit.Source.Path,
			Title:            hit.Source.Title,
			TitleFragments:   hit.Highlight["title"],
			ContentFragments: hit.Highlight["content"],
		}
		hits = append(hits, sr)
	}

	return &core.SearchResults{
		Hits:     hits,
		Total:    result.Hits.Total.Value,
		Duration: duration,
	}, nil
}

// esListByRepoPageSize is the page size used when collecting all document IDs for a repository.
const esListByRepoPageSize = 10000

// ListByRepo returns the IDs of all documents in the index that belong to the given repository.
func (e *ElasticEngine) ListByRepo(ctx context.Context, repo string) ([]string, error) {
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

		resp, err := e.client.Search(
			e.client.Search.WithContext(ctx),
			e.client.Search.WithIndex(e.index),
			e.client.Search.WithBody(bytes.NewReader(data)),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to list documents for repo %s: %w", repo, err)
		}

		if resp.IsError() {
			resp.Body.Close()
			return nil, fmt.Errorf("elasticsearch list error for repo %s: %s", repo, resp.String())
		}

		var result esSearchResponse
		if err := decodeAndClose(resp.Body, &result); err != nil {
			return nil, fmt.Errorf("failed to decode list response: %w", err)
		}

		if len(result.Hits.Hits) == 0 {
			break
		}

		for _, hit := range result.Hits.Hits {
			ids = append(ids, hit.ID)
		}

		lastHit := result.Hits.Hits[len(result.Hits.Hits)-1]
		searchAfter = lastHit.Sort

		if uint64(len(ids)) >= result.Hits.Total.Value {
			break
		}
	}

	return ids, nil
}

// ensureIndex creates the Elasticsearch index with the correct mappings if it does not already exist.
func (e *ElasticEngine) ensureIndex() error {
	resp, err := e.client.Indices.Exists([]string{e.index})
	if err != nil {
		return fmt.Errorf("failed to check index existence: %w", err)
	}
	defer resp.Body.Close()

	if !resp.IsError() {
		// Index already exists.
		return nil
	}

	if resp.StatusCode != http.StatusNotFound {
		return fmt.Errorf("unexpected status checking index existence: %s", resp.String())
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

	createResp, err := e.client.Indices.Create(
		e.index,
		e.client.Indices.Create.WithBody(bytes.NewReader(data)),
	)
	if err != nil {
		return fmt.Errorf("failed to create index: %w", err)
	}
	defer createResp.Body.Close()

	if createResp.IsError() {
		return fmt.Errorf("elasticsearch create index error: %s", createResp.String())
	}

	return nil
}

// buildSearchQuery constructs an Elasticsearch query DSL from user input.
// It mirrors the hybrid query logic from BleveEngine.buildSearchQuery.
func (e *ElasticEngine) buildSearchQuery(userQuery string) map[string]any {
	terms := splitQueryTerms(userQuery)
	if len(terms) == 0 {
		return map[string]any{"match_none": map[string]any{}}
	}

	mustClauses := make([]map[string]any, 0, len(terms))
	wordTermCount := 0

	for _, term := range terms {
		if term.phrase {
			mustClauses = append(mustClauses, buildESPhraseQuery(term.text))
		} else {
			mustClauses = append(mustClauses, buildESTermQuery(term.text))
			wordTermCount++
		}
	}

	var perWordQuery map[string]any
	if len(mustClauses) == 1 {
		perWordQuery = mustClauses[0]
	} else {
		must := make([]any, len(mustClauses))
		for i, c := range mustClauses {
			must[i] = c
		}

		perWordQuery = map[string]any{
			"bool": map[string]any{
				"must": must,
			},
		}
	}

	// For multi-word unquoted queries, add a full-phrase fallback (same as Bleve logic).
	if wordTermCount > 1 && wordTermCount == len(terms) {
		return map[string]any{
			"bool": map[string]any{
				"should": []any{
					perWordQuery,
					buildESFullPhraseQuery(userQuery),
				},
				"minimum_should_match": 1,
			},
		}
	}

	return perWordQuery
}

// buildESTermQuery creates an ES query for a single non-phrase term with match, prefix, and fuzzy variants.
func buildESTermQuery(term string) map[string]any {
	should := []any{
		// Exact/analyzed match — highest priority.
		map[string]any{
			"multi_match": map[string]any{
				"query":  term,
				"fields": []string{"title^6", "content^3"},
				"type":   "best_fields",
			},
		},
		// Prefix match — medium priority.
		map[string]any{
			"multi_match": map[string]any{
				"query":  term,
				"fields": []string{"title^3", "content^1.5"},
				"type":   "phrase_prefix",
			},
		},
	}

	// Fuzzy match — only for terms long enough.
	if len(term) >= minFuzzyTermLength {
		fuzziness := "1"
		if len(term) >= longTermThreshold {
			fuzziness = "2"
		}

		should = append(should, map[string]any{
			"multi_match": map[string]any{
				"query":     term,
				"fields":    []string{"title^1", "content^0.5"},
				"fuzziness": fuzziness,
			},
		})
	}

	return map[string]any{
		"bool": map[string]any{
			"should":               should,
			"minimum_should_match": 1,
		},
	}
}

// buildESPhraseQuery creates an ES phrase query for quoted terms.
func buildESPhraseQuery(phrase string) map[string]any {
	return map[string]any{
		"bool": map[string]any{
			"should": []any{
				map[string]any{
					"match_phrase": map[string]any{
						"title": map[string]any{
							"query": phrase,
							"boost": 10.0,
						},
					},
				},
				map[string]any{
					"match_phrase": map[string]any{
						"content": map[string]any{
							"query": phrase,
							"boost": 5.0,
						},
					},
				},
			},
			"minimum_should_match": 1,
		},
	}
}

// buildESFullPhraseQuery creates a multi_match AND query for stopword-tolerant full-phrase matching.
func buildESFullPhraseQuery(phrase string) map[string]any {
	return map[string]any{
		"bool": map[string]any{
			"should": []any{
				map[string]any{
					"match": map[string]any{
						"title": map[string]any{
							"query":    phrase,
							"operator": "and",
							"boost":    8.0,
						},
					},
				},
				map[string]any{
					"match": map[string]any{
						"content": map[string]any{
							"query":    phrase,
							"operator": "and",
							"boost":    4.0,
						},
					},
				},
			},
			"minimum_should_match": 1,
		},
	}
}

// esSearchResponse represents the Elasticsearch search response structure.
type esSearchResponse struct {
	Hits esHits `json:"hits"`
}

// esHits represents the hits section of an ES search response.
type esHits struct {
	Hits  []esHit `json:"hits"`
	Total esTotal `json:"total"`
}

// esTotal represents the total count in an ES search response.
type esTotal struct {
	Value uint64 `json:"value"`
}

// esHit represents a single hit in an ES search response.
type esHit struct {
	ID        string              `json:"_id"`
	Source    esSource            `json:"_source"`
	Highlight map[string][]string `json:"highlight"`
	Sort      []any               `json:"sort"`
	Score     float64             `json:"_score"`
}

// esSource represents the _source fields of an ES hit.
type esSource struct {
	Repo  string `json:"repo"`
	Path  string `json:"path"`
	Title string `json:"title"`
}

// decodeAndClose decodes a JSON response body and closes it.
func decodeAndClose(body io.ReadCloser, v any) error {
	defer body.Close()
	return json.NewDecoder(body).Decode(v)
}

// readFile reads a file from disk.
var readFile = os.ReadFile
