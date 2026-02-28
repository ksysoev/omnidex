//go:build !compile

package core

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// newTestService creates a Service with fresh mocks for each test.
func newTestService(t *testing.T) (*Service, *MockdocStore, *MocksearchEngine, *MockContentProcessor) {
	t.Helper()

	store := NewMockdocStore(t)
	search := NewMocksearchEngine(t)
	processor := NewMockContentProcessor(t)
	svc := New(store, search, map[ContentType]ContentProcessor{
		ContentTypeMarkdown: processor,
	})

	return svc, store, search, processor
}

// newTestServiceOnly creates a Service with fresh mocks, returning only the Service.
func newTestServiceOnly(t *testing.T) *Service {
	t.Helper()

	store := NewMockdocStore(t)
	search := NewMocksearchEngine(t)
	processor := NewMockContentProcessor(t)

	return New(store, search, map[ContentType]ContentProcessor{
		ContentTypeMarkdown: processor,
	})
}

func TestIngestDocuments_UpsertSuccess(t *testing.T) {
	svc, store, search, renderer := newTestService(t)
	ctx := t.Context()

	content := "# Hello\nWorld"

	renderer.EXPECT().ExtractTitle([]byte(content)).Return("Hello")
	renderer.EXPECT().ToPlainText([]byte(content)).Return("Hello World")
	store.EXPECT().Save(mock.Anything, mock.Anything).Return(nil)
	search.EXPECT().Index(mock.Anything, mock.Anything, "Hello World").Return(nil)

	req := IngestRequest{
		Repo:      "owner/repo",
		CommitSHA: "abc123",
		Documents: []IngestDocument{
			{Path: "docs/hello.md", Content: content, Action: "upsert"},
		},
	}

	resp, err := svc.IngestDocuments(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, 1, resp.Indexed)
	assert.Equal(t, 0, resp.Deleted)
}

func TestIngestDocuments_UpsertVerifiesDocFields(t *testing.T) {
	svc, store, search, renderer := newTestService(t)
	ctx := t.Context()

	content := "# My Title\nSome body"

	renderer.EXPECT().ExtractTitle([]byte(content)).Return("My Title")
	renderer.EXPECT().ToPlainText([]byte(content)).Return("My Title Some body")

	store.EXPECT().Save(mock.Anything, mock.MatchedBy(func(doc Document) bool {
		return doc.ID == "owner/repo/docs/readme.md" &&
			doc.Repo == "owner/repo" &&
			doc.Path == "docs/readme.md" &&
			doc.Title == "My Title" &&
			doc.Content == content &&
			doc.CommitSHA == "sha256" &&
			!doc.UpdatedAt.IsZero()
	})).Return(nil)

	search.EXPECT().Index(mock.Anything, mock.Anything, "My Title Some body").Return(nil)

	req := IngestRequest{
		Repo:      "owner/repo",
		CommitSHA: "sha256",
		Documents: []IngestDocument{
			{Path: "docs/readme.md", Content: content, Action: "upsert"},
		},
	}

	resp, err := svc.IngestDocuments(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, 1, resp.Indexed)
}

func TestIngestDocuments_UpsertEmptyTitleFallsBackToPath(t *testing.T) {
	svc, store, search, renderer := newTestService(t)
	ctx := t.Context()

	content := "no heading here"

	renderer.EXPECT().ExtractTitle([]byte(content)).Return("")
	renderer.EXPECT().ToPlainText([]byte(content)).Return("no heading here")

	store.EXPECT().Save(mock.Anything, mock.MatchedBy(func(doc Document) bool {
		return doc.Title == "docs/untitled.md"
	})).Return(nil)

	search.EXPECT().Index(mock.Anything, mock.Anything, "no heading here").Return(nil)

	req := IngestRequest{
		Repo:      "owner/repo",
		CommitSHA: "abc",
		Documents: []IngestDocument{
			{Path: "docs/untitled.md", Content: content, Action: "upsert"},
		},
	}

	resp, err := svc.IngestDocuments(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, 1, resp.Indexed)
}

func TestIngestDocuments_DeleteSuccess(t *testing.T) {
	svc, store, search, _ := newTestService(t)
	ctx := t.Context()

	search.EXPECT().Remove(mock.Anything, "owner/repo/docs/old.md").Return(nil)
	store.EXPECT().Delete(mock.Anything, "owner/repo", "docs/old.md").Return(nil)

	req := IngestRequest{
		Repo:      "owner/repo",
		CommitSHA: "abc",
		Documents: []IngestDocument{
			{Path: "docs/old.md", Action: "delete"},
		},
	}

	resp, err := svc.IngestDocuments(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, 0, resp.Indexed)
	assert.Equal(t, 1, resp.Deleted)
}

func TestIngestDocuments_MixedActions(t *testing.T) {
	svc, store, search, renderer := newTestService(t)
	ctx := t.Context()

	content := "# Doc"

	renderer.EXPECT().ExtractTitle([]byte(content)).Return("Doc")
	renderer.EXPECT().ToPlainText([]byte(content)).Return("Doc")
	store.EXPECT().Save(mock.Anything, mock.Anything).Return(nil)
	search.EXPECT().Index(mock.Anything, mock.Anything, "Doc").Return(nil)

	search.EXPECT().Remove(mock.Anything, "owner/repo/old.md").Return(nil)
	store.EXPECT().Delete(mock.Anything, "owner/repo", "old.md").Return(nil)

	req := IngestRequest{
		Repo:      "owner/repo",
		CommitSHA: "abc",
		Documents: []IngestDocument{
			{Path: "new.md", Content: content, Action: "upsert"},
			{Path: "old.md", Action: "delete"},
		},
	}

	resp, err := svc.IngestDocuments(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, 1, resp.Indexed)
	assert.Equal(t, 1, resp.Deleted)
}

func TestIngestDocuments_UnknownActionIsSkipped(t *testing.T) {
	svc := newTestServiceOnly(t)
	ctx := t.Context()

	req := IngestRequest{
		Repo:      "owner/repo",
		CommitSHA: "abc",
		Documents: []IngestDocument{
			{Path: "docs/weird.md", Content: "content", Action: "archive"},
		},
	}

	resp, err := svc.IngestDocuments(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, 0, resp.Indexed)
	assert.Equal(t, 0, resp.Deleted)
}

func TestIngestDocuments_EmptyDocuments(t *testing.T) {
	svc := newTestServiceOnly(t)
	ctx := t.Context()

	req := IngestRequest{
		Repo:      "owner/repo",
		CommitSHA: "abc",
		Documents: nil,
	}

	resp, err := svc.IngestDocuments(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, 0, resp.Indexed)
	assert.Equal(t, 0, resp.Deleted)
}

func TestIngestDocuments_UpsertErrors(t *testing.T) {
	tests := []struct {
		name       string
		setupMocks func(*MockdocStore, *MocksearchEngine, *MockContentProcessor)
		wantErrMsg string
	}{
		{
			name: "store save error propagates",
			setupMocks: func(store *MockdocStore, _ *MocksearchEngine, renderer *MockContentProcessor) {
				renderer.EXPECT().ExtractTitle(mock.Anything).Return("Title")
				store.EXPECT().Save(mock.Anything, mock.Anything).Return(errors.New("db connection lost"))
			},
			wantErrMsg: "db connection lost",
		},
		{
			name: "search index error propagates",
			setupMocks: func(store *MockdocStore, search *MocksearchEngine, renderer *MockContentProcessor) {
				renderer.EXPECT().ExtractTitle(mock.Anything).Return("Title")
				renderer.EXPECT().ToPlainText(mock.Anything).Return("plain")
				store.EXPECT().Save(mock.Anything, mock.Anything).Return(nil)
				search.EXPECT().Index(mock.Anything, mock.Anything, "plain").Return(errors.New("index unavailable"))
			},
			wantErrMsg: "index unavailable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, store, search, renderer := newTestService(t)
			tt.setupMocks(store, search, renderer)

			req := IngestRequest{
				Repo:      "owner/repo",
				CommitSHA: "abc",
				Documents: []IngestDocument{
					{Path: "docs/fail.md", Content: "# Title\nbody", Action: "upsert"},
				},
			}

			resp, err := svc.IngestDocuments(t.Context(), req)
			require.Error(t, err)
			assert.Nil(t, resp)
			assert.ErrorContains(t, err, tt.wantErrMsg)
			assert.ErrorContains(t, err, "docs/fail.md")
		})
	}
}

func TestIngestDocuments_DeleteErrors(t *testing.T) {
	tests := []struct {
		name       string
		setupMocks func(*MockdocStore, *MocksearchEngine, *MockContentProcessor)
		wantErrMsg string
	}{
		{
			name: "search remove error propagates",
			setupMocks: func(_ *MockdocStore, search *MocksearchEngine, _ *MockContentProcessor) {
				search.EXPECT().Remove(mock.Anything, "owner/repo/docs/gone.md").Return(errors.New("remove failed"))
			},
			wantErrMsg: "remove failed",
		},
		{
			name: "store delete error propagates with compensating re-index",
			setupMocks: func(store *MockdocStore, search *MocksearchEngine, renderer *MockContentProcessor) {
				search.EXPECT().Remove(mock.Anything, "owner/repo/docs/gone.md").Return(nil)
				store.EXPECT().Delete(mock.Anything, "owner/repo", "docs/gone.md").Return(errors.New("delete failed"))
				// Compensating action: re-index the document that's still in the store.
				store.EXPECT().Get(mock.Anything, "owner/repo", "docs/gone.md").Return(Document{
					ID: "owner/repo/docs/gone.md", Repo: "owner/repo", Path: "docs/gone.md",
					Content: "# Gone", Title: "Gone",
				}, nil)
				renderer.EXPECT().ToPlainText([]byte("# Gone")).Return("Gone")
				search.EXPECT().Index(mock.Anything, mock.Anything, "Gone").Return(nil)
			},
			wantErrMsg: "delete failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, store, search, renderer := newTestService(t)
			tt.setupMocks(store, search, renderer)

			req := IngestRequest{
				Repo:      "owner/repo",
				CommitSHA: "abc",
				Documents: []IngestDocument{
					{Path: "docs/gone.md", Action: "delete"},
				},
			}

			resp, err := svc.IngestDocuments(t.Context(), req)
			require.Error(t, err)
			assert.Nil(t, resp)
			assert.ErrorContains(t, err, tt.wantErrMsg)
			assert.ErrorContains(t, err, "docs/gone.md")
		})
	}
}

func TestIngestDocuments_SyncDeletesStaleDocuments(t *testing.T) {
	svc, store, search, renderer := newTestService(t)
	ctx := t.Context()

	content := "# Keep"

	// Mock the upsert for the document in the request.
	renderer.EXPECT().ExtractTitle([]byte(content)).Return("Keep")
	renderer.EXPECT().ToPlainText([]byte(content)).Return("Keep")
	store.EXPECT().Save(mock.Anything, mock.Anything).Return(nil)
	search.EXPECT().Index(mock.Anything, mock.Anything, "Keep").Return(nil)

	// Mock store.List returning both the kept doc and a stale doc.
	now := time.Now()
	store.EXPECT().List(mock.Anything, "owner/repo").Return([]DocumentMeta{
		{ID: "owner/repo/keep.md", Repo: "owner/repo", Path: "keep.md", Title: "Keep", UpdatedAt: now},
		{ID: "owner/repo/stale.md", Repo: "owner/repo", Path: "stale.md", Title: "Stale", UpdatedAt: now},
	}, nil)

	// Mock deletion of the stale document (search first, then store).
	search.EXPECT().Remove(mock.Anything, "owner/repo/stale.md").Return(nil)
	store.EXPECT().Delete(mock.Anything, "owner/repo", "stale.md").Return(nil)

	// Mock ListByRepo for orphan cleanup — no orphans remain after deletion.
	search.EXPECT().ListByRepo(mock.Anything, "owner/repo").Return([]string{"owner/repo/keep.md"}, nil)

	req := IngestRequest{
		Repo:      "owner/repo",
		CommitSHA: "abc",
		Sync:      true,
		Documents: []IngestDocument{
			{Path: "keep.md", Content: content, Action: "upsert"},
		},
	}

	resp, err := svc.IngestDocuments(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, 1, resp.Indexed)
	assert.Equal(t, 1, resp.Deleted)
}

func TestIngestDocuments_SyncNoStaleDocuments(t *testing.T) {
	svc, store, search, renderer := newTestService(t)
	ctx := t.Context()

	content := "# Doc"

	renderer.EXPECT().ExtractTitle([]byte(content)).Return("Doc")
	renderer.EXPECT().ToPlainText([]byte(content)).Return("Doc")
	store.EXPECT().Save(mock.Anything, mock.Anything).Return(nil)
	search.EXPECT().Index(mock.Anything, mock.Anything, "Doc").Return(nil)

	// All stored documents match the request — nothing to delete.
	now := time.Now()
	store.EXPECT().List(mock.Anything, "owner/repo").Return([]DocumentMeta{
		{ID: "owner/repo/doc.md", Repo: "owner/repo", Path: "doc.md", Title: "Doc", UpdatedAt: now},
	}, nil)

	// No orphans in search index either.
	search.EXPECT().ListByRepo(mock.Anything, "owner/repo").Return([]string{"owner/repo/doc.md"}, nil)

	req := IngestRequest{
		Repo:      "owner/repo",
		CommitSHA: "abc",
		Sync:      true,
		Documents: []IngestDocument{
			{Path: "doc.md", Content: content, Action: "upsert"},
		},
	}

	resp, err := svc.IngestDocuments(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, 1, resp.Indexed)
	assert.Equal(t, 0, resp.Deleted)
}

func TestIngestDocuments_SyncDisabledDoesNotDelete(t *testing.T) {
	svc, store, search, renderer := newTestService(t)
	ctx := t.Context()

	content := "# Doc"

	renderer.EXPECT().ExtractTitle([]byte(content)).Return("Doc")
	renderer.EXPECT().ToPlainText([]byte(content)).Return("Doc")
	store.EXPECT().Save(mock.Anything, mock.Anything).Return(nil)
	search.EXPECT().Index(mock.Anything, mock.Anything, "Doc").Return(nil)

	// store.List should NOT be called when sync is disabled.

	req := IngestRequest{
		Repo:      "owner/repo",
		CommitSHA: "abc",
		Sync:      false,
		Documents: []IngestDocument{
			{Path: "doc.md", Content: content, Action: "upsert"},
		},
	}

	resp, err := svc.IngestDocuments(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, 1, resp.Indexed)
	assert.Equal(t, 0, resp.Deleted)
}

func TestIngestDocuments_SyncErrors(t *testing.T) {
	tests := []struct {
		name       string
		setupMocks func(*MockdocStore, *MocksearchEngine, *MockContentProcessor)
		wantErrMsg string
	}{
		{
			name: "store list error propagates",
			setupMocks: func(store *MockdocStore, _ *MocksearchEngine, _ *MockContentProcessor) {
				store.EXPECT().List(mock.Anything, "owner/repo").Return(nil, errors.New("list failed"))
			},
			wantErrMsg: "list failed",
		},
		{
			name: "sync delete search remove error propagates",
			setupMocks: func(store *MockdocStore, search *MocksearchEngine, _ *MockContentProcessor) {
				now := time.Now()
				store.EXPECT().List(mock.Anything, "owner/repo").Return([]DocumentMeta{
					{ID: "owner/repo/stale.md", Repo: "owner/repo", Path: "stale.md", Title: "Stale", UpdatedAt: now},
				}, nil)
				search.EXPECT().Remove(mock.Anything, "owner/repo/stale.md").Return(errors.New("remove failed"))
			},
			wantErrMsg: "remove failed",
		},
		{
			name: "sync delete store error propagates",
			setupMocks: func(store *MockdocStore, search *MocksearchEngine, renderer *MockContentProcessor) {
				now := time.Now()
				store.EXPECT().List(mock.Anything, "owner/repo").Return([]DocumentMeta{
					{ID: "owner/repo/stale.md", Repo: "owner/repo", Path: "stale.md", Title: "Stale", UpdatedAt: now},
				}, nil)
				search.EXPECT().Remove(mock.Anything, "owner/repo/stale.md").Return(nil)
				store.EXPECT().Delete(mock.Anything, "owner/repo", "stale.md").Return(errors.New("delete failed"))
				// Compensating action: re-index the document that's still in the store.
				store.EXPECT().Get(mock.Anything, "owner/repo", "stale.md").Return(Document{
					ID: "owner/repo/stale.md", Repo: "owner/repo", Path: "stale.md",
					Content: "# Stale", Title: "Stale",
				}, nil)
				renderer.EXPECT().ToPlainText([]byte("# Stale")).Return("Stale")
				search.EXPECT().Index(mock.Anything, mock.Anything, "Stale").Return(nil)
			},
			wantErrMsg: "delete failed",
		},
		{
			name: "search ListByRepo error propagates",
			setupMocks: func(store *MockdocStore, search *MocksearchEngine, _ *MockContentProcessor) {
				store.EXPECT().List(mock.Anything, "owner/repo").Return(nil, nil)
				search.EXPECT().ListByRepo(mock.Anything, "owner/repo").Return(nil, errors.New("list by repo failed"))
			},
			wantErrMsg: "list by repo failed",
		},
		{
			name: "orphan search remove error propagates",
			setupMocks: func(store *MockdocStore, search *MocksearchEngine, _ *MockContentProcessor) {
				store.EXPECT().List(mock.Anything, "owner/repo").Return(nil, nil)
				search.EXPECT().ListByRepo(mock.Anything, "owner/repo").Return([]string{"owner/repo/orphan.md"}, nil)
				search.EXPECT().Remove(mock.Anything, "owner/repo/orphan.md").Return(errors.New("orphan remove failed"))
			},
			wantErrMsg: "orphan remove failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, store, search, renderer := newTestService(t)
			tt.setupMocks(store, search, renderer)

			req := IngestRequest{
				Repo:      "owner/repo",
				CommitSHA: "abc",
				Sync:      true,
				Documents: nil,
			}

			resp, err := svc.IngestDocuments(t.Context(), req)
			require.Error(t, err)
			assert.Nil(t, resp)
			assert.ErrorContains(t, err, tt.wantErrMsg)
		})
	}
}

func TestIngestDocuments_SyncCleansOrphanedSearchEntries(t *testing.T) {
	svc, store, search, _ := newTestService(t)
	ctx := t.Context()

	// No documents in the docstore — everything was already deleted.
	store.EXPECT().List(mock.Anything, "owner/repo").Return(nil, nil)

	// But the search index still has an orphaned entry from a previous partial failure.
	search.EXPECT().ListByRepo(mock.Anything, "owner/repo").Return([]string{"owner/repo/orphan.md"}, nil)

	// Expect the orphaned entry to be removed from the search index.
	search.EXPECT().Remove(mock.Anything, "owner/repo/orphan.md").Return(nil)

	req := IngestRequest{
		Repo:      "owner/repo",
		CommitSHA: "abc",
		Sync:      true,
		Documents: nil,
	}

	resp, err := svc.IngestDocuments(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, 0, resp.Indexed)
	assert.Equal(t, 1, resp.Deleted)
}

func TestIngestDocuments_SyncOrphanCleanupSkipsValidDocs(t *testing.T) {
	svc, store, search, renderer := newTestService(t)
	ctx := t.Context()

	content := "# Keep"

	renderer.EXPECT().ExtractTitle([]byte(content)).Return("Keep")
	renderer.EXPECT().ToPlainText([]byte(content)).Return("Keep")
	store.EXPECT().Save(mock.Anything, mock.Anything).Return(nil)
	search.EXPECT().Index(mock.Anything, mock.Anything, "Keep").Return(nil)

	now := time.Now()
	store.EXPECT().List(mock.Anything, "owner/repo").Return([]DocumentMeta{
		{ID: "owner/repo/keep.md", Repo: "owner/repo", Path: "keep.md", Title: "Keep", UpdatedAt: now},
	}, nil)

	// Search index has the valid doc plus an orphan.
	search.EXPECT().ListByRepo(mock.Anything, "owner/repo").Return(
		[]string{"owner/repo/keep.md", "owner/repo/orphan.md"}, nil,
	)

	// Only the orphan should be removed.
	search.EXPECT().Remove(mock.Anything, "owner/repo/orphan.md").Return(nil)

	req := IngestRequest{
		Repo:      "owner/repo",
		CommitSHA: "abc",
		Sync:      true,
		Documents: []IngestDocument{
			{Path: "keep.md", Content: content, Action: "upsert"},
		},
	}

	resp, err := svc.IngestDocuments(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, 1, resp.Indexed)
	assert.Equal(t, 1, resp.Deleted) // 1 orphan cleaned
}

func TestIngestDocuments_DeleteSearchFailurePreventStoreDelete(t *testing.T) {
	svc, _, search, _ := newTestService(t)
	ctx := t.Context()

	// search.Remove fails — store.Delete should NOT be called.
	search.EXPECT().Remove(mock.Anything, "owner/repo/docs/fail.md").Return(errors.New("search unavailable"))

	req := IngestRequest{
		Repo:      "owner/repo",
		CommitSHA: "abc",
		Documents: []IngestDocument{
			{Path: "docs/fail.md", Action: "delete"},
		},
	}

	resp, err := svc.IngestDocuments(ctx, req)
	require.Error(t, err)
	assert.Nil(t, resp)
	assert.ErrorContains(t, err, "search unavailable")
	// store.Delete was never called — verified by testify mock expectations.
}

func TestDeleteDocument_CompensatingReindexOnStoreFailure(t *testing.T) {
	tests := []struct {
		setupMocks func(*MockdocStore, *MocksearchEngine, *MockContentProcessor)
		name       string
	}{
		{
			name: "successful compensating re-index",
			setupMocks: func(store *MockdocStore, search *MocksearchEngine, renderer *MockContentProcessor) {
				search.EXPECT().Remove(mock.Anything, "owner/repo/docs/doc.md").Return(nil)
				store.EXPECT().Delete(mock.Anything, "owner/repo", "docs/doc.md").Return(errors.New("disk full"))
				store.EXPECT().Get(mock.Anything, "owner/repo", "docs/doc.md").Return(Document{
					ID: "owner/repo/docs/doc.md", Repo: "owner/repo", Path: "docs/doc.md",
					Content: "# Doc", Title: "Doc",
				}, nil)
				renderer.EXPECT().ToPlainText([]byte("# Doc")).Return("Doc")
				search.EXPECT().Index(mock.Anything, mock.Anything, "Doc").Return(nil)
			},
		},
		{
			name: "compensating re-index fails on store.Get",
			setupMocks: func(store *MockdocStore, search *MocksearchEngine, _ *MockContentProcessor) {
				search.EXPECT().Remove(mock.Anything, "owner/repo/docs/doc.md").Return(nil)
				store.EXPECT().Delete(mock.Anything, "owner/repo", "docs/doc.md").Return(errors.New("disk full"))
				store.EXPECT().Get(mock.Anything, "owner/repo", "docs/doc.md").Return(Document{}, errors.New("also broken"))
			},
		},
		{
			name: "compensating re-index fails on search.Index",
			setupMocks: func(store *MockdocStore, search *MocksearchEngine, renderer *MockContentProcessor) {
				search.EXPECT().Remove(mock.Anything, "owner/repo/docs/doc.md").Return(nil)
				store.EXPECT().Delete(mock.Anything, "owner/repo", "docs/doc.md").Return(errors.New("disk full"))
				store.EXPECT().Get(mock.Anything, "owner/repo", "docs/doc.md").Return(Document{
					ID: "owner/repo/docs/doc.md", Repo: "owner/repo", Path: "docs/doc.md",
					Content: "# Doc", Title: "Doc",
				}, nil)
				renderer.EXPECT().ToPlainText([]byte("# Doc")).Return("Doc")
				search.EXPECT().Index(mock.Anything, mock.Anything, "Doc").Return(errors.New("index broken"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, store, search, renderer := newTestService(t)
			tt.setupMocks(store, search, renderer)

			req := IngestRequest{
				Repo:      "owner/repo",
				CommitSHA: "abc",
				Documents: []IngestDocument{
					{Path: "docs/doc.md", Action: "delete"},
				},
			}

			resp, err := svc.IngestDocuments(t.Context(), req)
			require.Error(t, err)
			assert.Nil(t, resp)
			// The original delete error is always returned regardless of
			// compensating action outcome.
			assert.ErrorContains(t, err, "disk full")
		})
	}
}

func TestSyncDeleteStale_PartialOrphanCleanupPreservesCount(t *testing.T) {
	svc, store, search, _ := newTestService(t)
	ctx := t.Context()

	// No stale documents in the docstore.
	store.EXPECT().List(mock.Anything, "owner/repo").Return(nil, nil)

	// Search index has two orphaned entries.
	search.EXPECT().ListByRepo(mock.Anything, "owner/repo").Return(
		[]string{"owner/repo/orphan1.md", "owner/repo/orphan2.md"}, nil,
	)

	// First orphan removal succeeds, second fails.
	search.EXPECT().Remove(mock.Anything, "owner/repo/orphan1.md").Return(nil)
	search.EXPECT().Remove(mock.Anything, "owner/repo/orphan2.md").Return(errors.New("remove failed"))

	req := IngestRequest{
		Repo:      "owner/repo",
		CommitSHA: "abc",
		Sync:      true,
		Documents: nil,
	}

	deleted, err := svc.syncDeleteStale(ctx, req)
	require.Error(t, err)
	assert.ErrorContains(t, err, "remove failed")
	// The one successful orphan removal must be reflected in the count.
	assert.Equal(t, 1, deleted)
}

func TestGetDocument(t *testing.T) {
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)

	tests := []struct {
		wantDoc      Document
		setupMocks   func(*MockdocStore, *MockContentProcessor)
		name         string
		wantErr      string
		wantHTML     []byte
		wantHeadings []Heading
	}{
		{
			name: "success",
			setupMocks: func(store *MockdocStore, renderer *MockContentProcessor) {
				doc := Document{
					ID:        "owner/repo/docs/guide.md",
					Repo:      "owner/repo",
					Path:      "docs/guide.md",
					Title:     "Guide",
					Content:   "# Guide\nContent here",
					CommitSHA: "abc",
					UpdatedAt: now,
				}
				store.EXPECT().Get(mock.Anything, "owner/repo", "docs/guide.md").Return(doc, nil)
				renderer.EXPECT().RenderHTML([]byte("# Guide\nContent here")).Return(
					[]byte("<h1>Guide</h1><p>Content here</p>"),
					[]Heading{{Level: 1, ID: "guide", Text: "Guide"}},
					nil,
				)
			},
			wantDoc: Document{
				ID:        "owner/repo/docs/guide.md",
				Repo:      "owner/repo",
				Path:      "docs/guide.md",
				Title:     "Guide",
				Content:   "# Guide\nContent here",
				CommitSHA: "abc",
				UpdatedAt: now,
			},
			wantHTML:     []byte("<h1>Guide</h1><p>Content here</p>"),
			wantHeadings: []Heading{{Level: 1, ID: "guide", Text: "Guide"}},
		},
		{
			name: "store get error propagates",
			setupMocks: func(store *MockdocStore, _ *MockContentProcessor) {
				store.EXPECT().Get(mock.Anything, "owner/repo", "docs/missing.md").Return(Document{}, errors.New("not found"))
			},
			wantErr: "not found",
		},
		{
			name: "renderer toHTML error propagates",
			setupMocks: func(store *MockdocStore, renderer *MockContentProcessor) {
				doc := Document{
					ID:      "owner/repo/docs/bad.md",
					Content: "bad content",
				}
				store.EXPECT().Get(mock.Anything, "owner/repo", "docs/bad.md").Return(doc, nil)
				renderer.EXPECT().RenderHTML([]byte("bad content")).Return(nil, nil, errors.New("render error"))
			},
			wantErr: "render error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, store, _, renderer := newTestService(t)
			tt.setupMocks(store, renderer)

			repo := "owner/repo"

			path := "docs/guide.md"

			switch tt.name {
			case "store get error propagates":
				path = "docs/missing.md"
			case "renderer toHTML error propagates":
				path = "docs/bad.md"
			}

			doc, html, headings, err := svc.GetDocument(t.Context(), repo, path)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.ErrorContains(t, err, tt.wantErr)
				assert.Equal(t, Document{}, doc)
				assert.Nil(t, html)
				assert.Nil(t, headings)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantDoc, doc)
				assert.Equal(t, tt.wantHTML, html)
				assert.Equal(t, tt.wantHeadings, headings)
			}
		})
	}
}

func TestNew_PanicsOnNilProcessors(t *testing.T) {
	store := NewMockdocStore(t)
	search := NewMocksearchEngine(t)

	assert.PanicsWithValue(t, "processors map must not be nil", func() {
		New(store, search, nil)
	})
}

func TestNew_PanicsOnMissingMarkdownProcessor(t *testing.T) {
	store := NewMockdocStore(t)
	search := NewMocksearchEngine(t)

	assert.PanicsWithValue(t, "processors map must contain a ContentTypeMarkdown entry", func() {
		New(store, search, map[ContentType]ContentProcessor{})
	})
}

func TestSearchDocs(t *testing.T) {
	tests := []struct {
		setupMocks  func(*MocksearchEngine)
		wantResults *SearchResults
		name        string
		query       string
		wantErr     string
		opts        SearchOpts
	}{
		{
			name:  "success",
			query: "hello world",
			opts:  SearchOpts{Limit: 10, Offset: 0},
			setupMocks: func(search *MocksearchEngine) {
				results := &SearchResults{
					Hits: []SearchResult{
						{
							ID:        "owner/repo/docs/hello.md",
							Repo:      "owner/repo",
							Path:      "docs/hello.md",
							Title:     "Hello",
							Fragments: []string{"<b>hello</b> <b>world</b>"},
							Score:     1.5,
						},
					},
					Total:    1,
					Duration: 5 * time.Millisecond,
				}
				search.EXPECT().Search(mock.Anything, "hello world", SearchOpts{Limit: 10, Offset: 0}).Return(results, nil)
			},
			wantResults: &SearchResults{
				Hits: []SearchResult{
					{
						ID:        "owner/repo/docs/hello.md",
						Repo:      "owner/repo",
						Path:      "docs/hello.md",
						Title:     "Hello",
						Fragments: []string{"<b>hello</b> <b>world</b>"},
						Score:     1.5,
					},
				},
				Total:    1,
				Duration: 5 * time.Millisecond,
			},
		},
		{
			name:  "error propagates",
			query: "broken query",
			opts:  SearchOpts{Limit: 10},
			setupMocks: func(search *MocksearchEngine) {
				search.EXPECT().Search(mock.Anything, "broken query", SearchOpts{Limit: 10}).Return(nil, errors.New("search engine down"))
			},
			wantErr: "search engine down",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, _, search, _ := newTestService(t)
			tt.setupMocks(search)

			results, err := svc.SearchDocs(t.Context(), tt.query, tt.opts)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.ErrorContains(t, err, tt.wantErr)
				assert.Nil(t, results)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantResults, results)
			}
		})
	}
}

func TestListRepos(t *testing.T) {
	now := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		setupMocks func(*MockdocStore)
		name       string
		wantErr    string
		wantRepos  []RepoInfo
	}{
		{
			name: "success",
			setupMocks: func(store *MockdocStore) {
				repos := []RepoInfo{
					{Name: "owner/repo-a", DocCount: 10, LastUpdated: now},
					{Name: "owner/repo-b", DocCount: 3, LastUpdated: now.Add(-24 * time.Hour)},
				}
				store.EXPECT().ListRepos(mock.Anything).Return(repos, nil)
			},
			wantRepos: []RepoInfo{
				{Name: "owner/repo-a", DocCount: 10, LastUpdated: now},
				{Name: "owner/repo-b", DocCount: 3, LastUpdated: now.Add(-24 * time.Hour)},
			},
		},
		{
			name: "error propagates",
			setupMocks: func(store *MockdocStore) {
				store.EXPECT().ListRepos(mock.Anything).Return(nil, errors.New("db error"))
			},
			wantErr: "db error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, store, _, _ := newTestService(t)
			tt.setupMocks(store)

			repos, err := svc.ListRepos(t.Context())

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.ErrorContains(t, err, tt.wantErr)
				assert.Nil(t, repos)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantRepos, repos)
			}
		})
	}
}

func TestListDocuments(t *testing.T) {
	now := time.Date(2025, 3, 10, 8, 30, 0, 0, time.UTC)

	tests := []struct {
		setupMocks func(*MockdocStore)
		name       string
		repo       string
		wantErr    string
		wantDocs   []DocumentMeta
	}{
		{
			name: "success",
			repo: "owner/repo",
			setupMocks: func(store *MockdocStore) {
				docs := []DocumentMeta{
					{ID: "owner/repo/readme.md", Repo: "owner/repo", Path: "readme.md", Title: "README", UpdatedAt: now},
					{ID: "owner/repo/guide.md", Repo: "owner/repo", Path: "guide.md", Title: "Guide", UpdatedAt: now},
				}
				store.EXPECT().List(mock.Anything, "owner/repo").Return(docs, nil)
			},
			wantDocs: []DocumentMeta{
				{ID: "owner/repo/readme.md", Repo: "owner/repo", Path: "readme.md", Title: "README", UpdatedAt: now},
				{ID: "owner/repo/guide.md", Repo: "owner/repo", Path: "guide.md", Title: "Guide", UpdatedAt: now},
			},
		},
		{
			name: "error propagates",
			repo: "owner/missing",
			setupMocks: func(store *MockdocStore) {
				store.EXPECT().List(mock.Anything, "owner/missing").Return(nil, errors.New("repo not found"))
			},
			wantErr: "repo not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, store, _, _ := newTestService(t)
			tt.setupMocks(store)

			docs, err := svc.ListDocuments(t.Context(), tt.repo)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.ErrorContains(t, err, tt.wantErr)
				assert.Nil(t, docs)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantDocs, docs)
			}
		})
	}
}
