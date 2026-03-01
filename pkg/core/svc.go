// Package core provides core service logic and interfaces.
package core

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
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

// ContentProcessor handles rendering and indexing for a specific content type.
type ContentProcessor interface {
	// RenderHTML converts raw content into bytes consumed by the view layer
	// (e.g., HTML for markdown, JSON for OpenAPI) and returns any extracted headings.
	RenderHTML(src []byte) ([]byte, []Heading, error)
	// ExtractTitle returns a human-readable title from the content.
	ExtractTitle(src []byte) string
	// ToPlainText converts content to plain text for search indexing.
	ToPlainText(src []byte) string
	// ExtractHeadings returns the H1-H3 headings from the content with their
	// anchor IDs, used to resolve search result deep-links. Returns nil when
	// the content type does not support heading-based navigation.
	ExtractHeadings(src []byte) []Heading
}

// Service encapsulates core business logic and dependencies.
type Service struct {
	store      docStore
	search     searchEngine
	processors map[ContentType]ContentProcessor
}

// New creates a new Service instance with the provided dependencies.
// The processors map must contain at least a ContentTypeMarkdown entry.
// It panics if processors is nil or does not contain a markdown processor,
// since markdown is the default fallback for unknown content types.
func New(store docStore, search searchEngine, processors map[ContentType]ContentProcessor) *Service {
	if processors == nil {
		panic("processors map must not be nil")
	}

	if _, ok := processors[ContentTypeMarkdown]; !ok {
		panic("processors map must contain a ContentTypeMarkdown entry")
	}

	return &Service{
		store:      store,
		search:     search,
		processors: processors,
	}
}

// getProcessor returns the ContentProcessor for the given content type.
// It falls back to the markdown processor when the content type is empty or unknown.
func (s *Service) getProcessor(ct ContentType) ContentProcessor {
	if ct == "" {
		ct = ContentTypeMarkdown
	}

	if p, ok := s.processors[ct]; ok {
		return p
	}

	return s.processors[ContentTypeMarkdown]
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

// GetDocument retrieves a document and renders its content to HTML using the
// appropriate content processor. It also extracts headings for table of contents navigation.
func (s *Service) GetDocument(ctx context.Context, repo, path string) (Document, []byte, []Heading, error) {
	doc, err := s.store.Get(ctx, repo, path)
	if err != nil {
		return Document{}, nil, nil, fmt.Errorf("failed to get document: %w", err)
	}

	processor := s.getProcessor(doc.ContentType)

	html, headings, err := processor.RenderHTML([]byte(doc.Content))
	if err != nil {
		return Document{}, nil, nil, fmt.Errorf("failed to render document: %w", err)
	}

	return doc, html, headings, nil
}

// SearchDocs performs a full-text search across all indexed documents.
// After retrieving results from the search engine it attempts to resolve a
// heading anchor for each hit so that the result link can scroll directly to
// the matching section. Anchor resolution is best-effort; failures are logged
// and do not prevent results from being returned.
func (s *Service) SearchDocs(ctx context.Context, query string, opts SearchOpts) (*SearchResults, error) {
	results, err := s.search.Search(ctx, query, opts)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	s.resolveAnchors(ctx, results)

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
	ct := ingestDoc.ContentType
	if ct == "" {
		ct = ContentTypeMarkdown
	}

	// Normalize unknown content types to markdown so the persisted value always
	// matches a registered processor and remains consistent with how it will be
	// rendered and indexed.
	if _, known := s.processors[ct]; !known {
		ct = ContentTypeMarkdown
	}

	processor := s.getProcessor(ct)

	title := processor.ExtractTitle([]byte(ingestDoc.Content))
	if title == "" {
		title = ingestDoc.Path
	}

	doc := Document{
		ID:          repo + "/" + ingestDoc.Path,
		Repo:        repo,
		Path:        ingestDoc.Path,
		Title:       title,
		Content:     ingestDoc.Content,
		CommitSHA:   commitSHA,
		UpdatedAt:   time.Now(),
		ContentType: ct,
	}

	if err := s.store.Save(ctx, doc); err != nil {
		return fmt.Errorf("failed to save document: %w", err)
	}

	plainText := processor.ToPlainText([]byte(ingestDoc.Content))

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

	processor := s.getProcessor(doc.ContentType)
	plainText := processor.ToPlainText([]byte(doc.Content))

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

// markTagRE matches HTML <mark> and </mark> tags produced by Bleve highlighting
// so they can be stripped before comparing fragments against plain text.
var markTagRE = regexp.MustCompile(`</?mark>`)

// stripMarkTags removes <mark> and </mark> tags from a Bleve highlight fragment,
// returning the plain text that was actually indexed.
func stripMarkTags(fragment string) string {
	return markTagRE.ReplaceAllString(fragment, "")
}

// resolveAnchors enriches each SearchResult with a heading Anchor so that
// result links can deep-link directly to the matching section of a document.
// It works by:
//  1. Fetching the source document from the store.
//  2. Extracting headings (anchor IDs + text) via the content processor.
//  3. Converting the document to the same plain text that was indexed.
//  4. Stripping <mark> tags from the first content fragment to get raw text.
//  5. Finding that text in the plain text and determining which heading section
//     it falls under.
//
// Resolution is best-effort: failures are logged and do not affect other hits.
// Results with no content fragments (title-only matches) are skipped.
func (s *Service) resolveAnchors(ctx context.Context, results *SearchResults) {
	if results == nil {
		return
	}

	for i := range results.Hits {
		hit := &results.Hits[i]

		if len(hit.ContentFragments) == 0 {
			// Title-only match -- no content position to map; link to page top.
			continue
		}

		anchor, err := s.resolveAnchor(ctx, hit)
		if err != nil {
			slog.DebugContext(ctx, "anchor resolution skipped",
				"docID", hit.ID,
				"err", err,
			)

			continue
		}

		hit.Anchor = anchor
	}
}

// resolveAnchor resolves the heading anchor for a single SearchResult.
// It returns the heading ID of the section that contains the first content
// fragment, or an empty string when the match falls before the first heading.
func (s *Service) resolveAnchor(ctx context.Context, hit *SearchResult) (string, error) {
	doc, err := s.store.Get(ctx, hit.Repo, hit.Path)
	if err != nil {
		return "", fmt.Errorf("get document: %w", err)
	}

	processor := s.getProcessor(doc.ContentType)

	headings := processor.ExtractHeadings([]byte(doc.Content))
	if len(headings) == 0 {
		// Content type does not support heading navigation (e.g. OpenAPI).
		return "", nil
	}

	plainText := processor.ToPlainText([]byte(doc.Content))

	// Strip highlight markers from the first fragment to get comparable plain text.
	fragment := stripMarkTags(hit.ContentFragments[0])
	fragment = strings.TrimSpace(fragment)

	if fragment == "" {
		return "", nil
	}

	return findAnchorForFragment(plainText, headings, fragment), nil
}

// findAnchorForFragment returns the ID of the heading whose section contains
// the given text fragment. It works by locating each heading's text in the
// plain text document to establish section boundaries, then checking which
// section the fragment falls into.
//
// Returns an empty string when the fragment is found before the first heading
// (i.e. in the document preamble) or when it is not found at all.
func findAnchorForFragment(plainText string, headings []Heading, fragment string) string {
	// Locate the fragment in the plain text.
	fragIdx := strings.Index(plainText, fragment)
	if fragIdx < 0 {
		// Try a case-insensitive fallback for robustness.
		lowerPlain := strings.ToLower(plainText)
		lowerFrag := strings.ToLower(fragment)
		fragIdx = strings.Index(lowerPlain, lowerFrag)
	}

	if fragIdx < 0 {
		return ""
	}

	// Build a slice of (offset, headingID) pairs by finding each heading's
	// text in the plain text. We iterate in document order and search only in
	// the portion of the text that follows the previous heading to handle
	// duplicate heading texts correctly.
	type sectionBoundary struct {
		id     string
		offset int
	}

	boundaries := make([]sectionBoundary, 0, len(headings))
	searchFrom := 0

	for _, h := range headings {
		if h.Text == "" || h.ID == "" {
			continue
		}

		idx := strings.Index(plainText[searchFrom:], h.Text)
		if idx < 0 {
			// Heading not found in plain text (can happen when heading text
			// contains characters stripped during plain-text conversion).
			continue
		}

		abs := searchFrom + idx
		boundaries = append(boundaries, sectionBoundary{offset: abs, id: h.ID})
		searchFrom = abs + len(h.Text)
	}

	if len(boundaries) == 0 {
		return ""
	}

	// Find the last section boundary that starts at or before the fragment.
	anchor := ""

	for _, b := range boundaries {
		if b.offset > fragIdx {
			break
		}

		anchor = b.id
	}

	return anchor
}
