package search

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/ksysoev/omnidex/pkg/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockESHandler tracks requests and returns configurable responses.
type mockESHandler struct {
	handlers map[string]http.HandlerFunc
	requests []mockESRequest
	mu       sync.Mutex
}

type mockESRequest struct {
	Method string
	Path   string
	Body   string
}

func newMockESHandler() *mockESHandler {
	return &mockESHandler{
		handlers: make(map[string]http.HandlerFunc),
	}
}

func (h *mockESHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)

	h.mu.Lock()
	h.requests = append(h.requests, mockESRequest{
		Method: r.Method,
		Path:   r.URL.Path,
		Body:   string(body),
	})
	h.mu.Unlock()

	// The go-elasticsearch client performs product validation.
	// Set required headers to pass the check.
	w.Header().Set("X-Elastic-Product", "Elasticsearch")
	w.Header().Set("Content-Type", "application/json")

	// Try exact match first, then method-only match.
	for key, handler := range h.handlers {
		if key == r.Method+" "+r.URL.Path {
			handler(w, r)
			return
		}
	}

	for key, handler := range h.handlers {
		if key == r.Method {
			handler(w, r)
			return
		}
	}

	// Default: return 200 with empty JSON.
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"acknowledged":true}`))
}

func (h *mockESHandler) getRequests() []mockESRequest {
	h.mu.Lock()
	defer h.mu.Unlock()

	cp := make([]mockESRequest, len(h.requests))
	copy(cp, h.requests)

	return cp
}

// newTestElasticEngine creates an ElasticEngine pointing at a mock server.
// The mock handler responds to index-existence checks and index creation.
func newTestElasticEngine(t *testing.T, handler *mockESHandler) (*ElasticEngine, *httptest.Server) {
	t.Helper()

	// Register default handlers for index setup.
	handler.handlers["HEAD /omnidex"] = func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}

	srv := httptest.NewServer(handler)

	engine, err := NewElastic(&ElasticSearchConfig{
		Addresses: []string{srv.URL},
		Index:     "omnidex",
	})
	require.NoError(t, err)

	return engine, srv
}

func TestNewElastic_Success(t *testing.T) {
	handler := newMockESHandler()

	engine, srv := newTestElasticEngine(t, handler)
	defer srv.Close()

	assert.NotNil(t, engine)
	assert.Equal(t, "omnidex", engine.index)
}

func TestNewElastic_DefaultIndex(t *testing.T) {
	handler := newMockESHandler()
	handler.handlers["HEAD /omnidex"] = func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}

	srv := httptest.NewServer(handler)
	defer srv.Close()

	engine, err := NewElastic(&ElasticSearchConfig{
		Addresses: []string{srv.URL},
	})
	require.NoError(t, err)
	assert.Equal(t, "omnidex", engine.index)
}

func TestNewElastic_CreatesIndex(t *testing.T) {
	handler := newMockESHandler()

	// Override: index does NOT exist.
	handler.handlers["HEAD /omnidex"] = func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}
	handler.handlers["PUT /omnidex"] = func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"acknowledged":true}`))
	}

	srv := httptest.NewServer(handler)
	defer srv.Close()

	engine, err := NewElastic(&ElasticSearchConfig{
		Addresses: []string{srv.URL},
		Index:     "omnidex",
	})
	require.NoError(t, err)
	assert.NotNil(t, engine)

	// Verify a PUT was issued to create the index.
	reqs := handler.getRequests()
	hasPut := false

	for _, r := range reqs {
		if r.Method == "PUT" && r.Path == "/omnidex" {
			hasPut = true

			break
		}
	}

	assert.True(t, hasPut, "expected PUT /omnidex to create the index")
}

func TestElasticEngine_Index(t *testing.T) {
	handler := newMockESHandler()

	handler.handlers["PUT"] = func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"result":"created","_id":"test/doc.md"}`))
	}

	engine, srv := newTestElasticEngine(t, handler)
	defer srv.Close()

	doc := core.Document{
		ID:    "owner/repo/doc.md",
		Repo:  "owner/repo",
		Path:  "doc.md",
		Title: "Test Document",
	}

	err := engine.Index(t.Context(), doc, "plain text content")
	require.NoError(t, err)

	// Verify the indexed document body from recorded requests.
	reqs := handler.getRequests()

	var indexedBody string

	for _, r := range reqs {
		if r.Method == "PUT" && r.Body != "" {
			indexedBody = r.Body

			break
		}
	}

	var m map[string]string
	require.NoError(t, json.Unmarshal([]byte(indexedBody), &m))
	assert.Equal(t, "Test Document", m["title"])
	assert.Equal(t, "plain text content", m["content"])
	assert.Equal(t, "owner/repo", m["repo"])
	assert.Equal(t, "doc.md", m["path"])
}

func TestElasticEngine_Remove(t *testing.T) {
	handler := newMockESHandler()
	handler.handlers["DELETE"] = func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"result":"deleted"}`))
	}

	engine, srv := newTestElasticEngine(t, handler)
	defer srv.Close()

	err := engine.Remove(t.Context(), "owner/repo/doc.md")
	require.NoError(t, err)
}

func TestElasticEngine_Remove_NotFound(t *testing.T) {
	handler := newMockESHandler()
	handler.handlers["DELETE"] = func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"result":"not_found"}`))
	}

	engine, srv := newTestElasticEngine(t, handler)
	defer srv.Close()

	// 404 on delete should not return an error.
	err := engine.Remove(t.Context(), "owner/repo/nonexistent.md")
	require.NoError(t, err)
}

func TestElasticEngine_Search(t *testing.T) {
	handler := newMockESHandler()
	handler.handlers["POST"] = func(w http.ResponseWriter, _ *http.Request) {
		resp := map[string]any{
			"hits": map[string]any{
				"total": map[string]any{"value": 1},
				"hits": []any{
					map[string]any{
						"_id":    "owner/repo/doc.md",
						"_score": 5.5,
						"_source": map[string]any{
							"repo":  "owner/repo",
							"path":  "doc.md",
							"title": "Getting Started",
						},
						"highlight": map[string]any{
							"title":   []any{"<mark>Getting</mark> Started"},
							"content": []any{"Welcome to the <mark>getting</mark> started guide"},
						},
					},
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}

	engine, srv := newTestElasticEngine(t, handler)
	defer srv.Close()

	results, err := engine.Search(t.Context(), "getting started", core.SearchOpts{Limit: 10})
	require.NoError(t, err)

	assert.Equal(t, uint64(1), results.Total)
	require.Len(t, results.Hits, 1)
	assert.Equal(t, "owner/repo/doc.md", results.Hits[0].ID)
	assert.Equal(t, "owner/repo", results.Hits[0].Repo)
	assert.Equal(t, "doc.md", results.Hits[0].Path)
	assert.Equal(t, "Getting Started", results.Hits[0].Title)
	assert.InDelta(t, 5.5, results.Hits[0].Score, 0.01)
	assert.Len(t, results.Hits[0].TitleFragments, 1)
	assert.Len(t, results.Hits[0].ContentFragments, 1)
}

func TestElasticEngine_Search_DefaultLimit(t *testing.T) {
	handler := newMockESHandler()

	handler.handlers["POST"] = func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"hits":{"total":{"value":0},"hits":[]}}`))
	}

	engine, srv := newTestElasticEngine(t, handler)
	defer srv.Close()

	_, err := engine.Search(t.Context(), "test", core.SearchOpts{})
	require.NoError(t, err)

	// Verify request body contains default limit of 20.
	reqs := handler.getRequests()

	var requestBody map[string]any

	for _, r := range reqs {
		if r.Method == "POST" && r.Body != "" {
			require.NoError(t, json.Unmarshal([]byte(r.Body), &requestBody))

			break
		}
	}

	assert.Equal(t, float64(20), requestBody["size"])
}

func TestElasticEngine_ListByRepo(t *testing.T) {
	handler := newMockESHandler()

	callCount := 0
	handler.handlers["POST"] = func(w http.ResponseWriter, _ *http.Request) {
		callCount++

		var resp map[string]any
		if callCount == 1 {
			resp = map[string]any{
				"hits": map[string]any{
					"total": map[string]any{"value": 2},
					"hits": []any{
						map[string]any{"_id": "owner/repo/a.md", "sort": []any{0}},
						map[string]any{"_id": "owner/repo/b.md", "sort": []any{1}},
					},
				},
			}
		} else {
			resp = map[string]any{
				"hits": map[string]any{
					"total": map[string]any{"value": 2},
					"hits":  []any{},
				},
			}
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}

	engine, srv := newTestElasticEngine(t, handler)
	defer srv.Close()

	ids, err := engine.ListByRepo(t.Context(), "owner/repo")
	require.NoError(t, err)
	assert.Equal(t, []string{"owner/repo/a.md", "owner/repo/b.md"}, ids)
}

func TestBuildESTermQuery(t *testing.T) {
	q := buildESTermQuery("hello")
	boolQ, ok := q["bool"].(map[string]any)
	require.True(t, ok)

	should, ok := boolQ["should"].([]any)
	require.True(t, ok)

	// Short term (5 chars) — should have match + prefix + fuzzy = 3.
	assert.Len(t, should, 3)
}

func TestBuildESTermQuery_ShortTerm(t *testing.T) {
	q := buildESTermQuery("hi")
	boolQ, ok := q["bool"].(map[string]any)
	require.True(t, ok)

	should, ok := boolQ["should"].([]any)
	require.True(t, ok)

	// Short term (2 chars) — no fuzzy = match + prefix = 2.
	assert.Len(t, should, 2)
}

func TestBuildESPhraseQuery(t *testing.T) {
	q := buildESPhraseQuery("hello world")
	boolQ, ok := q["bool"].(map[string]any)
	require.True(t, ok)

	should, ok := boolQ["should"].([]any)
	require.True(t, ok)
	assert.Len(t, should, 2)
}

func TestElasticEngine_BuildSearchQuery_EmptyQuery(t *testing.T) {
	engine := &ElasticEngine{index: "test"}
	q := engine.buildSearchQuery("")
	_, hasMatchNone := q["match_none"]
	assert.True(t, hasMatchNone)
}

func TestElasticEngine_BuildSearchQuery_SingleTerm(t *testing.T) {
	engine := &ElasticEngine{index: "test"}
	q := engine.buildSearchQuery("kubernetes")
	_, hasBool := q["bool"]
	assert.True(t, hasBool, "single term should produce a bool query")
}

func TestElasticEngine_BuildSearchQuery_MultiWord(t *testing.T) {
	engine := &ElasticEngine{index: "test"}
	q := engine.buildSearchQuery("getting started guide")

	// Multi-word should have a top-level bool with should (per-word + full-phrase fallback).
	boolQ, ok := q["bool"].(map[string]any)
	require.True(t, ok)

	should, ok := boolQ["should"].([]any)
	require.True(t, ok)
	assert.Len(t, should, 2, "multi-word should have per-word + full-phrase paths")
}

func TestElasticEngine_BuildSearchQuery_QuotedPhrase(t *testing.T) {
	engine := &ElasticEngine{index: "test"}
	q := engine.buildSearchQuery(`"exact match"`)

	// Quoted phrase should produce a bool with match_phrase queries.
	boolQ, ok := q["bool"].(map[string]any)
	require.True(t, ok)
	assert.NotNil(t, boolQ["should"])
}

func TestNewElastic_OpenSearchHeaders(t *testing.T) {
	handler := newMockESHandler()
	handler.handlers["HEAD /omnidex"] = func(w http.ResponseWriter, r *http.Request) {
		// Verify compatibility headers are sent.
		ct := r.Header.Get("Content-Type")
		assert.Contains(t, ct, "compatible-with=7")
		w.WriteHeader(http.StatusOK)
	}

	srv := httptest.NewServer(handler)
	defer srv.Close()

	engine, err := NewElastic(&ElasticSearchConfig{
		Addresses:  []string{srv.URL},
		Index:      "omnidex",
		OpenSearch: true,
	})
	require.NoError(t, err)
	assert.NotNil(t, engine)
}
