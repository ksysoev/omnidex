//go:build !compile

package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ksysoev/omnidex/pkg/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestIngestDocs_Success(t *testing.T) {
	svc := NewMockService(t)
	views := NewMockViewRenderer(t)

	ingestReq := core.IngestRequest{
		Repo:      "owner/repo",
		CommitSHA: "abc123",
		Documents: []core.IngestDocument{
			{Path: "docs/readme.md", Content: "# Hello", Action: "upsert"},
		},
	}

	svc.EXPECT().IngestDocuments(mock.Anything, ingestReq).Return(&core.IngestResponse{
		Indexed: 1,
		Deleted: 0,
	}, nil)

	api := &API{svc: svc, views: views}

	body, err := json.Marshal(ingestReq)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/docs", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()

	api.ingestDocs(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var resp core.IngestResponse

	err = json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(t, err)

	assert.Equal(t, 1, resp.Indexed)
	assert.Equal(t, 0, resp.Deleted)
}

func TestIngestDocs_InvalidJSON(t *testing.T) {
	svc := NewMockService(t)
	views := NewMockViewRenderer(t)

	api := &API{svc: svc, views: views}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/docs", strings.NewReader("{invalid json"))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()

	api.ingestDocs(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "invalid request body")
}

func TestIngestDocs_EmptyRepo(t *testing.T) {
	svc := NewMockService(t)
	views := NewMockViewRenderer(t)

	api := &API{svc: svc, views: views}

	ingestReq := core.IngestRequest{
		Repo: "",
		Documents: []core.IngestDocument{
			{Path: "docs/readme.md", Content: "# Hello", Action: "upsert"},
		},
	}

	body, err := json.Marshal(ingestReq)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/docs", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()

	api.ingestDocs(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "repo field is required")
}

func TestIngestDocs_EmptyDocuments(t *testing.T) {
	svc := NewMockService(t)
	views := NewMockViewRenderer(t)

	api := &API{svc: svc, views: views}

	ingestReq := core.IngestRequest{
		Repo:      "owner/repo",
		CommitSHA: "abc123",
		Documents: []core.IngestDocument{},
	}

	body, err := json.Marshal(ingestReq)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/docs", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()

	api.ingestDocs(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "documents field is required")
}

func TestIngestDocs_ServiceError(t *testing.T) {
	svc := NewMockService(t)
	views := NewMockViewRenderer(t)

	ingestReq := core.IngestRequest{
		Repo:      "owner/repo",
		CommitSHA: "abc123",
		Documents: []core.IngestDocument{
			{Path: "docs/readme.md", Content: "# Hello", Action: "upsert"},
		},
	}

	svc.EXPECT().IngestDocuments(mock.Anything, ingestReq).Return(nil, fmt.Errorf("storage failure"))

	api := &API{svc: svc, views: views}

	body, err := json.Marshal(ingestReq)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/docs", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()

	api.ingestDocs(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "failed to process documents")
}

func TestListRepos_Success(t *testing.T) {
	svc := NewMockService(t)
	views := NewMockViewRenderer(t)

	repos := []core.RepoInfo{
		{Name: "owner/repo1", DocCount: 5, LastUpdated: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)},
		{Name: "owner/repo2", DocCount: 3, LastUpdated: time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC)},
	}

	svc.EXPECT().ListRepos(mock.Anything).Return(repos, nil)

	api := &API{svc: svc, views: views}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/repos", http.NoBody)
	rec := httptest.NewRecorder()

	api.listRepos(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var result map[string][]core.RepoInfo

	err := json.NewDecoder(rec.Body).Decode(&result)
	require.NoError(t, err)

	assert.Len(t, result["repos"], 2)
	assert.Equal(t, "owner/repo1", result["repos"][0].Name)
	assert.Equal(t, 5, result["repos"][0].DocCount)
}

func TestListRepos_Error(t *testing.T) {
	svc := NewMockService(t)
	views := NewMockViewRenderer(t)

	svc.EXPECT().ListRepos(mock.Anything).Return(nil, fmt.Errorf("database error"))

	api := &API{svc: svc, views: views}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/repos", http.NoBody)
	rec := httptest.NewRecorder()

	api.listRepos(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "failed to list repositories")
}
