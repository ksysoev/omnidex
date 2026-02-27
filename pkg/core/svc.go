// Package core provides core service logic and interfaces.
package core

import (
	"context"
	"fmt"
	"log/slog"
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
}

// markdownRenderer defines the interface for markdown processing.
type markdownRenderer interface {
	ToHTML(src []byte) ([]byte, error)
	ToHTMLWithHeadings(src []byte) ([]byte, []Heading, error)
	ExtractTitle(src []byte) string
	ExtractHeadings(src []byte) []Heading
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
// It returns the number of stale documents deleted.
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

		slog.InfoContext(ctx, "sync: removing stale document", "repo", req.Repo, "path", doc.Path)

		if err := s.deleteDocument(ctx, req.Repo, doc.Path); err != nil {
			return deleted, fmt.Errorf("failed to delete stale document %s: %w", doc.Path, err)
		}

		deleted++
	}

	return deleted, nil
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

	if err := s.store.Delete(ctx, repo, path); err != nil {
		return fmt.Errorf("failed to delete document: %w", err)
	}

	if err := s.search.Remove(ctx, docID); err != nil {
		return fmt.Errorf("failed to remove document from index: %w", err)
	}

	return nil
}
