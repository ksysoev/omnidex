//go:build !compile

package api

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ksysoev/omnidex/pkg/core"
	"github.com/ksysoev/omnidex/pkg/repo/docstore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestHomePage_Success(t *testing.T) {
	svc := NewMockService(t)
	views := NewMockViewRenderer(t)

	repos := []core.RepoInfo{
		{Name: "owner/repo", DocCount: 10, LastUpdated: time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)},
	}

	svc.EXPECT().ListRepos(mock.Anything).Return(repos, nil)
	views.EXPECT().RenderHome(mock.Anything, repos, false).Return(nil)

	api := &API{svc: svc, views: views}

	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	rec := httptest.NewRecorder()

	api.homePage(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "text/html; charset=utf-8", rec.Header().Get("Content-Type"))
}

func TestHomePage_HTMXPartial(t *testing.T) {
	svc := NewMockService(t)
	views := NewMockViewRenderer(t)

	repos := []core.RepoInfo{
		{Name: "owner/repo", DocCount: 10, LastUpdated: time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)},
	}

	svc.EXPECT().ListRepos(mock.Anything).Return(repos, nil)
	views.EXPECT().RenderHome(mock.Anything, repos, true).Return(nil)

	api := &API{svc: svc, views: views}

	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	req.Header.Set("HX-Request", "true")

	rec := httptest.NewRecorder()

	api.homePage(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "text/html; charset=utf-8", rec.Header().Get("Content-Type"))
}

func TestHomePage_ServiceError(t *testing.T) {
	svc := NewMockService(t)
	views := NewMockViewRenderer(t)

	svc.EXPECT().ListRepos(mock.Anything).Return(nil, fmt.Errorf("database error"))

	api := &API{svc: svc, views: views}

	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	rec := httptest.NewRecorder()

	api.homePage(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "Internal Server Error")
}

func TestRepoIndexPage_Success(t *testing.T) {
	svc := NewMockService(t)
	views := NewMockViewRenderer(t)

	docs := []core.DocumentMeta{
		{ID: "owner/repo/docs/readme.md", Repo: "owner/repo", Path: "docs/readme.md", Title: "README", UpdatedAt: time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)},
		{ID: "owner/repo/docs/guide.md", Repo: "owner/repo", Path: "docs/guide.md", Title: "Guide", UpdatedAt: time.Date(2025, 6, 2, 0, 0, 0, 0, time.UTC)},
	}

	svc.EXPECT().ListDocuments(mock.Anything, "owner/repo").Return(docs, nil)
	views.EXPECT().RenderRepoIndex(mock.Anything, "owner/repo", docs, false).Return(nil)

	api := &API{svc: svc, views: views}

	req := httptest.NewRequest(http.MethodGet, "/docs/owner/repo/", http.NoBody)
	req.SetPathValue("owner", "owner")
	req.SetPathValue("repo", "repo")

	rec := httptest.NewRecorder()

	api.repoIndexPage(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "text/html; charset=utf-8", rec.Header().Get("Content-Type"))
}

func TestRepoIndexPage_HTMXPartial(t *testing.T) {
	svc := NewMockService(t)
	views := NewMockViewRenderer(t)

	docs := []core.DocumentMeta{
		{ID: "owner/repo/docs/readme.md", Repo: "owner/repo", Path: "docs/readme.md", Title: "README", UpdatedAt: time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)},
	}

	svc.EXPECT().ListDocuments(mock.Anything, "owner/repo").Return(docs, nil)
	views.EXPECT().RenderRepoIndex(mock.Anything, "owner/repo", docs, true).Return(nil)

	api := &API{svc: svc, views: views}

	req := httptest.NewRequest(http.MethodGet, "/docs/owner/repo/", http.NoBody)
	req.Header.Set("HX-Request", "true")
	req.SetPathValue("owner", "owner")
	req.SetPathValue("repo", "repo")

	rec := httptest.NewRecorder()

	api.repoIndexPage(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "text/html; charset=utf-8", rec.Header().Get("Content-Type"))
}

func TestRepoIndexPage_ServiceError(t *testing.T) {
	svc := NewMockService(t)
	views := NewMockViewRenderer(t)

	svc.EXPECT().ListDocuments(mock.Anything, "owner/repo").Return(nil, fmt.Errorf("storage error"))

	api := &API{svc: svc, views: views}

	req := httptest.NewRequest(http.MethodGet, "/docs/owner/repo/", http.NoBody)
	req.SetPathValue("owner", "owner")
	req.SetPathValue("repo", "repo")

	rec := httptest.NewRecorder()

	api.repoIndexPage(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "Internal Server Error")
}

func TestRepoIndexPage_MissingValues(t *testing.T) {
	svc := NewMockService(t)
	views := NewMockViewRenderer(t)

	api := &API{svc: svc, views: views}

	tests := []struct {
		name  string
		owner string
		repo  string
	}{
		{name: "missing owner", owner: "", repo: "repo"},
		{name: "missing repo", owner: "owner", repo: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/docs/x/y/", http.NoBody)
			req.SetPathValue("owner", tt.owner)
			req.SetPathValue("repo", tt.repo)

			rec := httptest.NewRecorder()

			api.repoIndexPage(rec, req)

			assert.Equal(t, http.StatusNotFound, rec.Code)
		})
	}
}

func TestRepoIndexPage_EmptyRepo(t *testing.T) {
	svc := NewMockService(t)
	views := NewMockViewRenderer(t)

	svc.EXPECT().ListDocuments(mock.Anything, "owner/repo").Return([]core.DocumentMeta{}, nil)
	views.EXPECT().RenderRepoIndex(mock.Anything, "owner/repo", []core.DocumentMeta{}, false).Return(nil)

	api := &API{svc: svc, views: views}

	req := httptest.NewRequest(http.MethodGet, "/docs/owner/repo/", http.NoBody)
	req.SetPathValue("owner", "owner")
	req.SetPathValue("repo", "repo")

	rec := httptest.NewRecorder()

	api.repoIndexPage(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "text/html; charset=utf-8", rec.Header().Get("Content-Type"))
}

func TestDocPage_EmptyPathDelegatesToRepoIndex(t *testing.T) {
	svc := NewMockService(t)
	views := NewMockViewRenderer(t)

	docs := []core.DocumentMeta{
		{ID: "owner/repo/docs/readme.md", Repo: "owner/repo", Path: "docs/readme.md", Title: "README", UpdatedAt: time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)},
	}

	svc.EXPECT().ListDocuments(mock.Anything, "owner/repo").Return(docs, nil)
	views.EXPECT().RenderRepoIndex(mock.Anything, "owner/repo", docs, false).Return(nil)

	api := &API{svc: svc, views: views}

	req := httptest.NewRequest(http.MethodGet, "/docs/owner/repo/", http.NoBody)
	req.SetPathValue("owner", "owner")
	req.SetPathValue("repo", "repo")
	req.SetPathValue("path", "")

	rec := httptest.NewRecorder()

	api.docPage(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "text/html; charset=utf-8", rec.Header().Get("Content-Type"))
}

func TestDocPage_Success(t *testing.T) {
	svc := NewMockService(t)
	views := NewMockViewRenderer(t)

	doc := core.Document{
		ID:        "owner/repo/docs/readme.md",
		Repo:      "owner/repo",
		Path:      "docs/readme.md",
		Title:     "README",
		Content:   "# README",
		CommitSHA: "abc123",
		UpdatedAt: time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC),
	}
	htmlContent := []byte("<h1>README</h1>")
	navDocs := []core.DocumentMeta{
		{ID: "owner/repo/docs/readme.md", Repo: "owner/repo", Path: "docs/readme.md", Title: "README"},
	}

	svc.EXPECT().GetDocument(mock.Anything, "owner/repo", "docs/readme.md").Return(doc, htmlContent, nil)
	svc.EXPECT().ListDocuments(mock.Anything, "owner/repo").Return(navDocs, nil)
	views.EXPECT().RenderDoc(mock.Anything, doc, htmlContent, navDocs, false).Return(nil)

	api := &API{svc: svc, views: views}

	req := httptest.NewRequest(http.MethodGet, "/docs/owner/repo/docs/readme.md", http.NoBody)
	req.SetPathValue("owner", "owner")
	req.SetPathValue("repo", "repo")
	req.SetPathValue("path", "docs/readme.md")

	rec := httptest.NewRecorder()

	api.docPage(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "text/html; charset=utf-8", rec.Header().Get("Content-Type"))
}

func TestDocPage_NotFound(t *testing.T) {
	svc := NewMockService(t)
	views := NewMockViewRenderer(t)

	svc.EXPECT().GetDocument(mock.Anything, "owner/repo", "docs/missing.md").
		Return(core.Document{}, nil, fmt.Errorf("failed to get document: %w", docstore.ErrNotFound))

	api := &API{svc: svc, views: views}

	req := httptest.NewRequest(http.MethodGet, "/docs/owner/repo/docs/missing.md", http.NoBody)
	req.SetPathValue("owner", "owner")
	req.SetPathValue("repo", "repo")
	req.SetPathValue("path", "docs/missing.md")

	rec := httptest.NewRecorder()

	api.docPage(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestDocPage_MissingPathValues(t *testing.T) {
	svc := NewMockService(t)
	views := NewMockViewRenderer(t)

	api := &API{svc: svc, views: views}

	tests := []struct {
		name  string
		owner string
		repo  string
		path  string
	}{
		{name: "missing owner", owner: "", repo: "repo", path: "docs/readme.md"},
		{name: "missing repo", owner: "owner", repo: "", path: "docs/readme.md"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/docs/x/y/z", http.NoBody)
			req.SetPathValue("owner", tt.owner)
			req.SetPathValue("repo", tt.repo)
			req.SetPathValue("path", tt.path)

			rec := httptest.NewRecorder()

			api.docPage(rec, req)

			assert.Equal(t, http.StatusNotFound, rec.Code)
		})
	}
}

func TestSearchPage_WithQuery(t *testing.T) {
	svc := NewMockService(t)
	views := NewMockViewRenderer(t)

	results := &core.SearchResults{
		Hits: []core.SearchResult{
			{
				ID:        "owner/repo/docs/readme.md",
				Repo:      "owner/repo",
				Path:      "docs/readme.md",
				Title:     "README",
				Fragments: []string{"matching <em>content</em>"},
				Score:     1.5,
			},
		},
		Total:    1,
		Duration: 10 * time.Millisecond,
	}

	svc.EXPECT().SearchDocs(mock.Anything, "test query", core.SearchOpts{Limit: 20}).Return(results, nil)
	views.EXPECT().RenderSearch(mock.Anything, "test query", results, false).Return(nil)

	api := &API{svc: svc, views: views}

	req := httptest.NewRequest(http.MethodGet, "/search?q=test+query", http.NoBody)
	rec := httptest.NewRecorder()

	api.searchPage(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "text/html; charset=utf-8", rec.Header().Get("Content-Type"))
}

func TestSearchPage_EmptyQuery(t *testing.T) {
	svc := NewMockService(t)
	views := NewMockViewRenderer(t)

	views.EXPECT().RenderSearch(mock.Anything, "", (*core.SearchResults)(nil), false).Return(nil)

	api := &API{svc: svc, views: views}

	req := httptest.NewRequest(http.MethodGet, "/search", http.NoBody)
	rec := httptest.NewRecorder()

	api.searchPage(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "text/html; charset=utf-8", rec.Header().Get("Content-Type"))
}

func TestSearchPage_SearchError(t *testing.T) {
	svc := NewMockService(t)
	views := NewMockViewRenderer(t)

	svc.EXPECT().SearchDocs(mock.Anything, "broken query", core.SearchOpts{Limit: 20}).
		Return(nil, fmt.Errorf("search engine unavailable"))

	api := &API{svc: svc, views: views}

	req := httptest.NewRequest(http.MethodGet, "/search?q=broken+query", http.NoBody)
	rec := httptest.NewRecorder()

	api.searchPage(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "Search failed")
}

func TestDocPage_ServiceInternalError(t *testing.T) {
	svc := NewMockService(t)
	views := NewMockViewRenderer(t)

	svc.EXPECT().GetDocument(mock.Anything, "owner/repo", "docs/readme.md").
		Return(core.Document{}, nil, fmt.Errorf("database connection lost"))

	api := &API{svc: svc, views: views}

	req := httptest.NewRequest(http.MethodGet, "/docs/owner/repo/docs/readme.md", http.NoBody)
	req.SetPathValue("owner", "owner")
	req.SetPathValue("repo", "repo")
	req.SetPathValue("path", "docs/readme.md")

	rec := httptest.NewRecorder()

	api.docPage(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "Internal Server Error")
}

func TestDocPage_ListDocumentsError(t *testing.T) {
	svc := NewMockService(t)
	views := NewMockViewRenderer(t)

	doc := core.Document{
		ID:        "owner/repo/docs/readme.md",
		Repo:      "owner/repo",
		Path:      "docs/readme.md",
		Title:     "README",
		Content:   "# README",
		CommitSHA: "abc123",
		UpdatedAt: time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC),
	}
	htmlContent := []byte("<h1>README</h1>")

	svc.EXPECT().GetDocument(mock.Anything, "owner/repo", "docs/readme.md").Return(doc, htmlContent, nil)
	svc.EXPECT().ListDocuments(mock.Anything, "owner/repo").Return(nil, fmt.Errorf("nav list error"))
	// When ListDocuments fails, docs will be nil but page still renders.
	views.EXPECT().RenderDoc(mock.Anything, doc, htmlContent, []core.DocumentMeta(nil), false).Return(nil)

	api := &API{svc: svc, views: views}

	req := httptest.NewRequest(http.MethodGet, "/docs/owner/repo/docs/readme.md", http.NoBody)
	req.SetPathValue("owner", "owner")
	req.SetPathValue("repo", "repo")
	req.SetPathValue("path", "docs/readme.md")

	rec := httptest.NewRecorder()

	api.docPage(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "text/html; charset=utf-8", rec.Header().Get("Content-Type"))
}

func TestDocPage_HTMXPartial(t *testing.T) {
	svc := NewMockService(t)
	views := NewMockViewRenderer(t)

	doc := core.Document{
		ID:        "owner/repo/docs/readme.md",
		Repo:      "owner/repo",
		Path:      "docs/readme.md",
		Title:     "README",
		Content:   "# README",
		CommitSHA: "abc123",
		UpdatedAt: time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC),
	}
	htmlContent := []byte("<h1>README</h1>")
	navDocs := []core.DocumentMeta{
		{ID: "owner/repo/docs/readme.md", Repo: "owner/repo", Path: "docs/readme.md", Title: "README"},
	}

	svc.EXPECT().GetDocument(mock.Anything, "owner/repo", "docs/readme.md").Return(doc, htmlContent, nil)
	svc.EXPECT().ListDocuments(mock.Anything, "owner/repo").Return(navDocs, nil)
	views.EXPECT().RenderDoc(mock.Anything, doc, htmlContent, navDocs, true).Return(nil)

	api := &API{svc: svc, views: views}

	req := httptest.NewRequest(http.MethodGet, "/docs/owner/repo/docs/readme.md", http.NoBody)
	req.Header.Set("HX-Request", "true")
	req.SetPathValue("owner", "owner")
	req.SetPathValue("repo", "repo")
	req.SetPathValue("path", "docs/readme.md")

	rec := httptest.NewRecorder()

	api.docPage(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "text/html; charset=utf-8", rec.Header().Get("Content-Type"))
}
