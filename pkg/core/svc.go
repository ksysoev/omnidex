// Package core provides core service logic and interfaces.
package core

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"
)

// docStore defines the interface for document persistence operations.
type docStore interface {
	Save(ctx context.Context, doc Document) error
	Get(ctx context.Context, repo, path string) (Document, error)
	Delete(ctx context.Context, repo, path string) error
	List(ctx context.Context, repo string) ([]DocumentMeta, error)
	ListRepos(ctx context.Context) ([]RepoInfo, error)
}

// searchEngine defines the interface for full-text search operations.
type searchEngine interface {
	Index(ctx context.Context, doc Document, plainText string) error
	Remove(ctx context.Context, docID string) error
	Search(ctx context.Context, query string, opts SearchOpts) (*SearchResults, error)
	ListByRepo(ctx context.Context, repo string) ([]string, error)
}

// markdownRenderer defines the interface for markdown processing used by Service.
type markdownRenderer interface {
	ToHTMLWithHeadings(src []byte) ([]byte, []Heading, error)
	ExtractTitle(src []byte) string
	ToPlainText(src []byte) string
}

// Service encapsulates core business logic and dependencies.
type Service struct {
	store    docStore
	search   searchEngine
	renderer markdownRenderer
}

// New creates a new Service instance with the provided dependencies.
func New(store docStore, search searchEngine, renderer markdownRenderer) *Service {
	return &Service{
		store:    store,
		search:   search,
		renderer: renderer,
	}
}

// IngestDocuments processes a batch of document upserts and deletes from a repository.
// When req.Sync is true, after processing all documents the server treats the incoming
// document set as the complete truth for the repo and removes any stored documents
// whose paths are not present in the request.
func (s *Service) IngestDocuments(ctx context.Context, req IngestRequest) (*IngestResponse, error) {
	var indexed, deleted int

	for _, ingestDoc := range req.Documents {
		switch ingestDoc.Action {
		case "upsert":
			if err := s.upsertDocument(ctx, req.Repo, req.CommitSHA, ingestDoc); err != nil {
				return nil, fmt.Errorf("failed to upsert document %s: %w", ingestDoc.Path, err)
			}

			indexed++
		case "delete":
			if err := s.deleteDocument(ctx, req.Repo, ingestDoc.Path); err != nil {
				return nil, fmt.Errorf("failed to delete document %s: %w", ingestDoc.Path, err)
			}

			deleted++
		default:
			slog.WarnContext(ctx, "unknown document action", "action", ingestDoc.Action, "path", ingestDoc.Path)
		}
	}

	if req.Sync {
		syncDeleted, err := s.syncDeleteStale(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("failed to sync stale documents: %w", err)
		}

		deleted += syncDeleted
	}

	return &IngestResponse{
		Indexed: indexed,
		Deleted: deleted,
	}, nil
}

// syncDeleteStale removes stored documents that are not present in the ingest request.
// It also cleans up orphaned entries in the search index that may have been left behind
// by previous partial failures. It returns the total number of documents removed.
func (s *Service) syncDeleteStale(ctx context.Context, req IngestRequest) (int, error) {
	stored, err := s.store.List(ctx, req.Repo)
	if err != nil {
		return 0, fmt.Errorf("failed to list stored documents for repo %s: %w", req.Repo, err)
	}

	// Build a set of upserted document paths from the request.
	// Only upsert actions matter here because explicit deletes have already been
	// processed and removed from the store before sync runs.
	requestPaths := make(map[string]struct{}, len(req.Documents))
	for _, doc := range req.Documents {
		if doc.Action == "upsert" {
			requestPaths[doc.Path] = struct{}{}
		}
	}

	var deleted int

	for _, doc := range stored {
		if _, exists := requestPaths[doc.Path]; exists {
			continue
		}

		slog.DebugContext(ctx, "sync: removing stale document", "repo", req.Repo, "path", doc.Path)

		if err := s.deleteDocument(ctx, req.Repo, doc.Path); err != nil {
			return deleted, fmt.Errorf("failed to delete stale document %s: %w", doc.Path, err)
		}

		deleted++
	}

	if deleted > 0 {
		slog.InfoContext(ctx, "sync: stale document cleanup complete", "repo", req.Repo, "deleted", deleted)
	}

	// Clean up orphaned entries in the search index. These can exist when a
	// previous deletion removed a document from the docstore but failed to
	// remove it from the search index.
	orphaned, err := s.cleanOrphanedSearchEntries(ctx, req.Repo, requestPaths)
	deleted += orphaned

	if err != nil {
		return deleted, err
	}

	return deleted, nil
}

// cleanOrphanedSearchEntries removes search index entries for the given repo
// that do not correspond to any path in validPaths. It returns the number of
// orphaned entries removed.
func (s *Service) cleanOrphanedSearchEntries(ctx context.Context, repo string, validPaths map[string]struct{}) (int, error) {
	indexed, err := s.search.ListByRepo(ctx, repo)
	if err != nil {
		return 0, fmt.Errorf("failed to list search index entries for repo %s: %w", repo, err)
	}

	prefix := repo + "/"

	var cleaned int

	for _, docID := range indexed {
		path := strings.TrimPrefix(docID, prefix)

		if _, exists := validPaths[path]; exists {
			continue
		}

		slog.DebugContext(ctx, "sync: removing orphaned search entry", "repo", repo, "docID", docID)

		if err := s.search.Remove(ctx, docID); err != nil {
			return cleaned, fmt.Errorf("failed to remove orphaned search entry %s: %w", docID, err)
		}

		cleaned++
	}

	if cleaned > 0 {
		slog.InfoContext(ctx, "sync: orphan cleanup complete", "repo", repo, "cleaned", cleaned)
	}

	return cleaned, nil
}

// GetDocument retrieves a document and renders its markdown content to HTML.
// It also extracts headings from the markdown source for table of contents navigation.
func (s *Service) GetDocument(ctx context.Context, repo, path string) (Document, []byte, []Heading, error) {
	doc, err := s.store.Get(ctx, repo, path)
	if err != nil {
		return Document{}, nil, nil, fmt.Errorf("failed to get document: %w", err)
	}

	html, headings, err := s.renderer.ToHTMLWithHeadings([]byte(doc.Content))
	if err != nil {
		return Document{}, nil, nil, fmt.Errorf("failed to render document: %w", err)
	}

	return doc, html, headings, nil
}

// SearchDocs performs a full-text search across all indexed documents.
func (s *Service) SearchDocs(ctx context.Context, query string, opts SearchOpts) (*SearchResults, error) {
	results, err := s.search.Search(ctx, query, opts)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	return results, nil
}

// ListRepos returns metadata for all indexed repositories.
func (s *Service) ListRepos(ctx context.Context) ([]RepoInfo, error) {
	repos, err := s.store.ListRepos(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list repos: %w", err)
	}

	return repos, nil
}

// ListDocuments returns metadata for all documents in a repository.
func (s *Service) ListDocuments(ctx context.Context, repo string) ([]DocumentMeta, error) {
	docs, err := s.store.List(ctx, repo)
	if err != nil {
		return nil, fmt.Errorf("failed to list documents: %w", err)
	}

	return docs, nil
}

func (s *Service) upsertDocument(ctx context.Context, repo, commitSHA string, ingestDoc IngestDocument) error {
	title := s.renderer.ExtractTitle([]byte(ingestDoc.Content))
	if title == "" {
		title = ingestDoc.Path
	}

	doc := Document{
		ID:        repo + "/" + ingestDoc.Path,
		Repo:      repo,
		Path:      ingestDoc.Path,
		Title:     title,
		Content:   ingestDoc.Content,
		CommitSHA: commitSHA,
		UpdatedAt: time.Now(),
	}

	if err := s.store.Save(ctx, doc); err != nil {
		return fmt.Errorf("failed to save document: %w", err)
	}

	plainText := s.renderer.ToPlainText([]byte(ingestDoc.Content))

	if err := s.search.Index(ctx, doc, plainText); err != nil {
		return fmt.Errorf("failed to index document: %w", err)
	}

	return nil
}

func (s *Service) deleteDocument(ctx context.Context, repo, path string) error {
	docID := repo + "/" + path

	// Remove from search index first. If this fails the document remains in the
	// docstore, so syncDeleteStale can discover and retry on the next sync run.
	if err := s.search.Remove(ctx, docID); err != nil {
		return fmt.Errorf("failed to remove document from index: %w", err)
	}

	if err := s.store.Delete(ctx, repo, path); err != nil {
		// Best-effort compensating action: re-index the document so the search
		// index stays consistent with the docstore that still holds the document.
		s.reindexForCompensation(ctx, repo, path, err)

		return fmt.Errorf("failed to delete document: %w", err)
	}

	return nil
}

// reindexForCompensation attempts to re-add a document to the search index
// after a docstore delete failure left the document in the store but missing
// from the index. Errors are logged but not propagated because this is a
// best-effort repair; the next sync run will correct any remaining
// inconsistency.
func (s *Service) reindexForCompensation(ctx context.Context, repo, path string, deleteErr error) {
	doc, err := s.store.Get(ctx, repo, path)
	if err != nil {
		slog.Warn("compensating re-index: failed to fetch document from store",
			"repo", repo,
			"path", path,
			"deleteErr", deleteErr,
			"getErr", err,
		)

		return
	}

	plainText := s.renderer.ToPlainText([]byte(doc.Content))

	if err := s.search.Index(ctx, doc, plainText); err != nil {
		slog.Warn("compensating re-index: failed to re-index document",
			"repo", repo,
			"path", path,
			"deleteErr", deleteErr,
			"indexErr", err,
		)

		return
	}

	slog.Warn("compensating re-index: document re-indexed after docstore delete failure",
		"repo", repo,
		"path", path,
		"deleteErr", deleteErr,
	)
}
