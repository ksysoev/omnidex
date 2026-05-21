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

// ElasticSearchConfig holds configuration for the Elasticsearch backend.
type ElasticSearchConfig struct {
	Index     string   `mapstructure:"index"`
	Username  string   `mapstructure:"username"`
	Password  string   `mapstructure:"password"`
	APIKey    string   `mapstructure:"api_key"`
	CACert    string   `mapstructure:"ca_cert"`
	Addresses []string `mapstructure:"addresses"`
}

// ElasticEngine implements full-text search using Elasticsearch.
type ElasticEngine struct {
	client *elasticsearch.Client
	index  string
}

// NewElastic creates a new Elasticsearch search engine.
// It configures the client and ensures the index exists with the correct mapping.
func NewElastic(ctx context.Context, cfg *ElasticSearchConfig) (*ElasticEngine, error) {
	index := cfg.Index
	if index == "" {
		index = defaultIndex
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

	client, err := elasticsearch.NewClient(esCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create elasticsearch client: %w", err)
	}

	engine := &ElasticEngine{
		client: client,
		index:  index,
	}

	if err := engine.ensureIndex(ctx); err != nil {
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
		fieldTitle:   doc.Title,
		fieldContent: plainText,
		fieldRepo:    doc.Repo,
		fieldPath:    doc.Path,
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
		dslQuery:  esQuery,
		dslSize:   opts.Limit,
		"from":    opts.Offset,
		dslSource: []string{fieldRepo, fieldPath, fieldTitle},
		dslHighlight: map[string]any{
			dslFields: map[string]any{
				fieldTitle:   map[string]any{dslNumberOfFragments: 3},
				fieldContent: map[string]any{"fragment_size": 200, dslNumberOfFragments: 3},
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
			TitleFragments:   hit.Highlight[fieldTitle],
			ContentFragments: hit.Highlight[fieldContent],
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

// defaultIndex is the default Elasticsearch/OpenSearch index name.
const defaultIndex = "omnidex"

// Query DSL field name constants shared by Elasticsearch and OpenSearch.
const (
	dslQuery              = "query"
	dslSize               = "size"
	dslSource             = "_source"
	dslHighlight          = "highlight"
	dslFields             = "fields"
	dslNumberOfFragments  = "number_of_fragments"
	dslSort               = "sort"
	dslBool               = "bool"
	dslShould             = "should"
	dslMinimumShouldMatch = "minimum_should_match"
	dslBoost              = "boost"
	dslMultiMatch         = "multi_match"
	dslType               = "type"

	mappingTypeText            = "text"
	mappingTypeKeyword         = "keyword"
	mappingAnalyzer            = "analyzer"
	mappingTermVector          = "term_vector"
	mappingAnalyzerStandard    = "standard"
	mappingTermVectorPositions = "with_positions_offsets"
)

// ListByRepo returns the IDs of all documents in the index that belong to the given repository.
func (e *ElasticEngine) ListByRepo(ctx context.Context, repo string) ([]string, error) {
	ids := make([]string, 0, esListByRepoPageSize)

	var searchAfter []any

	for {
		body := map[string]any{
			dslQuery: map[string]any{
				"term": map[string]any{
					fieldRepo: repo,
				},
			},
			dslSize:   esListByRepoPageSize,
			dslSource: false,
			dslSort:   []any{"_doc"},
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
	}

	return ids, nil
}

// ensureIndex creates the Elasticsearch index with the correct mappings if it does not already exist.
func (e *ElasticEngine) ensureIndex(ctx context.Context) error {
	resp, err := e.client.Indices.Exists([]string{e.index}, e.client.Indices.Exists.WithContext(ctx))
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
				fieldTitle: map[string]any{
					dslType:           mappingTypeText,
					mappingAnalyzer:   mappingAnalyzerStandard,
					mappingTermVector: mappingTermVectorPositions,
				},
				fieldContent: map[string]any{
					dslType:           mappingTypeText,
					mappingAnalyzer:   mappingAnalyzerStandard,
					mappingTermVector: mappingTermVectorPositions,
				},
				fieldRepo: map[string]any{
					dslType: mappingTypeKeyword,
				},
				fieldPath: map[string]any{
					dslType: mappingTypeKeyword,
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
		e.client.Indices.Create.WithContext(ctx),
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
	return buildQueryDSL(userQuery)
}

// buildESTermQuery creates an ES query for a single non-phrase term with match, prefix, and fuzzy variants.
func buildESTermQuery(term string) map[string]any {
	should := []any{
		// Exact/analyzed match — highest priority.
		map[string]any{
			dslMultiMatch: map[string]any{
				dslQuery:  term,
				dslFields: []string{"title^6", "content^3"},
				dslType:   "best_fields",
			},
		},
		// Prefix match — medium priority.
		map[string]any{
			dslMultiMatch: map[string]any{
				dslQuery:  term,
				dslFields: []string{"title^3", "content^1.5"},
				dslType:   "phrase_prefix",
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
			dslMultiMatch: map[string]any{
				dslQuery:    term,
				dslFields:   []string{"title^1", "content^0.5"},
				"fuzziness": fuzziness,
			},
		})
	}

	return map[string]any{
		dslBool: map[string]any{
			dslShould:             should,
			dslMinimumShouldMatch: 1,
		},
	}
}

// buildESPhraseQuery creates an ES phrase query for quoted terms.
func buildESPhraseQuery(phrase string) map[string]any {
	return map[string]any{
		dslBool: map[string]any{
			dslShould: []any{
				map[string]any{
					"match_phrase": map[string]any{
						fieldTitle: map[string]any{
							dslQuery: phrase,
							dslBoost: 10.0,
						},
					},
				},
				map[string]any{
					"match_phrase": map[string]any{
						fieldContent: map[string]any{
							dslQuery: phrase,
							dslBoost: 5.0,
						},
					},
				},
			},
			dslMinimumShouldMatch: 1,
		},
	}
}

// buildESFullPhraseQuery creates a multi_match AND query for stopword-tolerant full-phrase matching.
func buildESFullPhraseQuery(phrase string) map[string]any {
	return map[string]any{
		dslBool: map[string]any{
			dslShould: []any{
				map[string]any{
					"match": map[string]any{
						fieldTitle: map[string]any{
							dslQuery:   phrase,
							"operator": "and",
							dslBoost:   8.0,
						},
					},
				},
				map[string]any{
					"match": map[string]any{
						fieldContent: map[string]any{
							dslQuery:   phrase,
							"operator": "and",
							dslBoost:   4.0,
						},
					},
				},
			},
			dslMinimumShouldMatch: 1,
		},
	}
}

// buildQueryDSL constructs a query DSL map from user input.
// It is shared between ElasticEngine and OpenSearchEngine as both use compatible query DSL.
func buildQueryDSL(userQuery string) map[string]any {
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
			dslBool: map[string]any{
				"must": must,
			},
		}
	}

	// For multi-word unquoted queries, add a full-phrase fallback (mirrors Bleve logic).
	if wordTermCount > 1 && wordTermCount == len(terms) {
		return map[string]any{
			dslBool: map[string]any{
				dslShould: []any{
					perWordQuery,
					buildESFullPhraseQuery(userQuery),
				},
				dslMinimumShouldMatch: 1,
			},
		}
	}

	return perWordQuery
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
