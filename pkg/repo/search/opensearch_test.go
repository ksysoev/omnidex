package search

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ksysoev/omnidex/pkg/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestOpenSearchEngine creates a test OpenSearchEngine backed by a mock HTTP server.
func newTestOpenSearchEngine(t *testing.T, handler *mockESHandler) (*OpenSearchEngine, *httptest.Server) {
	t.Helper()

	handler.handlers["HEAD /omnidex"] = func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}

	srv := httptest.NewServer(handler)

	engine, err := NewOpenSearch(context.Background(), &OpenSearchConfig{
		Addresses: []string{srv.URL},
		Index:     "omnidex",
	})
	require.NoError(t, err)

	return engine, srv
}

func TestNewOpenSearch_Success(t *testing.T) {
	handler := newMockESHandler()

	engine, srv := newTestOpenSearchEngine(t, handler)
	defer srv.Close()

	assert.NotNil(t, engine)
	assert.Equal(t, "omnidex", engine.index)
}

func TestNewOpenSearch_DefaultIndex(t *testing.T) {
	handler := newMockESHandler()
	handler.handlers["HEAD /omnidex"] = func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}

	srv := httptest.NewServer(handler)
	defer srv.Close()

	engine, err := NewOpenSearch(context.Background(), &OpenSearchConfig{
		Addresses: []string{srv.URL},
	})
	require.NoError(t, err)
	assert.Equal(t, "omnidex", engine.index)
}

func TestNewOpenSearch_CreatesIndex(t *testing.T) { //nolint:dupl // Parallel to TestNewElastic_CreatesIndex but tests a different type
	handler := newMockESHandler()

	handler.handlers["HEAD /omnidex"] = func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}
	handler.handlers["PUT /omnidex"] = func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"acknowledged":true,"shards_acknowledged":true,"index":"omnidex"}`))
	}

	srv := httptest.NewServer(handler)
	defer srv.Close()

	engine, err := NewOpenSearch(context.Background(), &OpenSearchConfig{
		Addresses: []string{srv.URL},
		Index:     "omnidex",
	})
	require.NoError(t, err)
	assert.NotNil(t, engine)

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

func TestNewOpenSearch_EnsureIndexError(t *testing.T) {
	handler := newMockESHandler()
	handler.handlers["HEAD /omnidex"] = func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}

	srv := httptest.NewServer(handler)
	defer srv.Close()

	_, err := NewOpenSearch(context.Background(), &OpenSearchConfig{
		Addresses: []string{srv.URL},
		Index:     "omnidex",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected status checking index existence")
}

func TestOpenSearchEngine_Index(t *testing.T) {
	handler := newMockESHandler()

	handler.handlers["PUT /omnidex/_doc/owner/repo/doc.md"] = func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"_index":"omnidex","_id":"owner/repo/doc.md","result":"created","_version":1,"_shards":{"total":1,"successful":1,"failed":0},"_seq_no":0,"_primary_term":1}`))
	}

	engine, srv := newTestOpenSearchEngine(t, handler)
	defer srv.Close()

	doc := core.Document{
		ID:    "owner/repo/doc.md",
		Title: "Test Doc",
		Repo:  "owner/repo",
		Path:  "doc.md",
	}

	err := engine.Index(context.Background(), doc, "plain text content")
	require.NoError(t, err)
}

func TestOpenSearchEngine_Remove(t *testing.T) {
	handler := newMockESHandler()

	handler.handlers["DELETE /omnidex/_doc/owner/repo/doc.md"] = func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"_index":"omnidex","_id":"owner/repo/doc.md","result":"deleted","_version":2,"_shards":{"total":1,"successful":1,"failed":0},"_seq_no":1,"_primary_term":1}`))
	}

	engine, srv := newTestOpenSearchEngine(t, handler)
	defer srv.Close()

	err := engine.Remove(context.Background(), "owner/repo/doc.md")
	require.NoError(t, err)
}

func TestOpenSearchEngine_Remove_NotFound(t *testing.T) {
	handler := newMockESHandler()

	handler.handlers["DELETE /omnidex/_doc/missing/doc"] = func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"_index":"omnidex","_id":"missing/doc","result":"not_found","_version":0,"_shards":{"total":1,"successful":1,"failed":0},"_seq_no":-2,"_primary_term":0}`))
	}

	engine, srv := newTestOpenSearchEngine(t, handler)
	defer srv.Close()

	err := engine.Remove(context.Background(), "missing/doc")
	require.NoError(t, err)
}

func TestOpenSearchEngine_Search(t *testing.T) {
	handler := newMockESHandler()

	handler.handlers["POST"] = func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"hits": {
				"total": {"value": 1, "relation": "eq"},
				"max_score": 1.5,
				"hits": [{
					"_id": "owner/repo/doc.md",
					"_score": 1.5,
					"_source": {"repo": "owner/repo", "path": "doc.md", "title": "Test Doc"},
					"highlight": {"content": ["some <mark>match</mark>"]}
				}]
			}
		}`))
	}

	engine, srv := newTestOpenSearchEngine(t, handler)
	defer srv.Close()

	results, err := engine.Search(context.Background(), "match", core.SearchOpts{Limit: 10})
	require.NoError(t, err)
	require.Len(t, results.Hits, 1)
	assert.Equal(t, "owner/repo/doc.md", results.Hits[0].ID)
	assert.Equal(t, "owner/repo", results.Hits[0].Repo)
	assert.Equal(t, "doc.md", results.Hits[0].Path)
	assert.Equal(t, "Test Doc", results.Hits[0].Title)
	assert.Equal(t, []string{"some <mark>match</mark>"}, results.Hits[0].ContentFragments)
}

func TestOpenSearchEngine_Search_DefaultLimit(t *testing.T) {
	handler := newMockESHandler()

	handler.handlers["POST"] = func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"hits":{"total":{"value":0,"relation":"eq"},"hits":[]}}`))
	}

	engine, srv := newTestOpenSearchEngine(t, handler)
	defer srv.Close()

	_, err := engine.Search(context.Background(), "query", core.SearchOpts{})
	require.NoError(t, err)

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

func TestOpenSearchEngine_ListByRepo(t *testing.T) {
	handler := newMockESHandler()

	callCount := 0
	handler.handlers["POST"] = func(w http.ResponseWriter, _ *http.Request) {
		callCount++

		var resp map[string]any
		if callCount == 1 {
			resp = map[string]any{
				"hits": map[string]any{
					"total": map[string]any{"value": 2, "relation": "eq"},
					"hits": []any{
						map[string]any{"_id": "owner/repo/a.md", "sort": []any{0}},
						map[string]any{"_id": "owner/repo/b.md", "sort": []any{1}},
					},
				},
			}
		} else {
			resp = map[string]any{
				"hits": map[string]any{
					"total": map[string]any{"value": 2, "relation": "eq"},
					"hits":  []any{},
				},
			}
		}

		w.Header().Set("Content-Type", "application/json")

		data, _ := json.Marshal(resp)
		_, _ = w.Write(data)
	}

	engine, srv := newTestOpenSearchEngine(t, handler)
	defer srv.Close()

	ids, err := engine.ListByRepo(context.Background(), "owner/repo")
	require.NoError(t, err)
	assert.Equal(t, []string{"owner/repo/a.md", "owner/repo/b.md"}, ids)
}

func TestOpenSearchEngine_BuildSearchQuery_SingleTerm(t *testing.T) {
	engine := &OpenSearchEngine{index: "test"}
	q := engine.buildSearchQuery("hello")

	boolQ, ok := q["bool"].(map[string]any)
	require.True(t, ok)
	assert.NotNil(t, boolQ["should"])
}

func TestOpenSearchEngine_BuildSearchQuery_MultiWord(t *testing.T) {
	engine := &OpenSearchEngine{index: "test"}
	q := engine.buildSearchQuery("hello world")

	boolQ, ok := q["bool"].(map[string]any)
	require.True(t, ok)
	assert.NotNil(t, boolQ["should"])
}

func TestOpenSearchEngine_BuildSearchQuery_Empty(t *testing.T) {
	engine := &OpenSearchEngine{index: "test"}
	q := engine.buildSearchQuery("")

	_, hasMatchNone := q["match_none"]
	assert.True(t, hasMatchNone)
}
