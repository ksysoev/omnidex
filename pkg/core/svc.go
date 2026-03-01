// Package core provides core service logic and interfaces.
package core

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
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
		// Content type does not support heading navigation.
		return "", nil
	}

	plainText := processor.ToPlainText([]byte(doc.Content))

	// Locate the matched term's byte offset in the plain text.
	// fragmentMatchIndex handles Bleve's ellipsis padding and mid-word cuts.
	fragIdx := fragmentMatchIndex(hit.ContentFragments[0], plainText)
	if fragIdx < 0 {
		return "", nil
	}

	return findAnchorAtPosition(plainText, headings, fragIdx), nil
}

// findAnchorAtPosition returns the ID of the heading whose section contains
// the character at fragIdx in plainText. It builds section boundaries by
// locating each heading's text in document order, then returns the last
// boundary whose offset is ≤ fragIdx.
//
// Returns an empty string when fragIdx falls before the first heading or no
// valid boundaries can be established.
func findAnchorAtPosition(plainText string, headings []Heading, fragIdx int) string {
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

// bleveEllipsis is the Unicode ellipsis character (U+2026) that Bleve's
// SimpleFragmenter prepends/appends when a fragment window does not align
// with the document start or end.
const bleveEllipsis = "…"

// caseInsensitiveIndex returns the byte offset of the first case-insensitive
// occurrence of substr in s. It advances rune by rune through s and compares
// each window using strings.EqualFold, so the returned offset is always a
// valid byte position in the original string regardless of Unicode case folding.
// Returns -1 if substr is not found or substr is empty.
func caseInsensitiveIndex(s, substr string) int {
	if substr == "" {
		return -1
	}

	n := len(substr)

	for i := 0; i+n <= len(s); {
		if strings.EqualFold(s[i:i+n], substr) {
			return i
		}

		_, size := utf8.DecodeRuneInString(s[i:])
		i += size
	}

	return -1
}

// fragmentMatchIndex locates the first <mark>-ed term from a Bleve highlight
// fragment within plainText, returning its byte offset. Returns -1 if not found.
//
// Bleve's SimpleFragmenter may:
//   - Prefix the fragment with "…" (U+2026) when the window doesn't start at
//     the document beginning.
//   - Cut the content mid-word right after the "…" (e.g. "…ntroduction").
//
// This function strips the ellipsis and any resulting partial leading word,
// builds a locator string of (cleaned context before mark) + (marked term),
// finds that locator in plainText, and returns the offset pointing AT the
// marked term — not the start of the surrounding context window.
func fragmentMatchIndex(rawFrag, plainText string) int {
	markOpen := strings.Index(rawFrag, "<mark>")
	if markOpen < 0 {
		// No marks: fall back to stripping everything and trimming ellipsis.
		s := strings.TrimLeft(stripMarkTags(rawFrag), bleveEllipsis)
		s = skipPartialLeadingWord(s)
		s = strings.TrimSpace(s)

		if s == "" {
			return -1
		}

		idx := strings.Index(plainText, s)
		if idx < 0 {
			idx = caseInsensitiveIndex(plainText, s)
		}

		return idx
	}

	// Extract the marked (matched) term.
	afterOpen := rawFrag[markOpen+len("<mark>"):]

	closeIdx := strings.Index(afterOpen, "</mark>")
	if closeIdx < 0 {
		return -1
	}

	markedTerm := afterOpen[:closeIdx]

	// Build cleaned context before the mark.
	// The pre-mark text may start with "…" and a partial word; strip both.
	preMark := rawFrag[:markOpen]
	hadEllipsis := strings.HasPrefix(preMark, bleveEllipsis)
	preMark = strings.TrimLeft(preMark, bleveEllipsis)

	if hadEllipsis {
		// After stripping "…" the first "word" may be a partial word fragment.
		preMark = skipPartialLeadingWord(preMark)
	}

	// Limit context length to avoid very long locators that might fail due
	// to subtle whitespace differences.
	const maxContextBytes = 120
	if len(preMark) > maxContextBytes {
		preMark = preMark[len(preMark)-maxContextBytes:]
	}

	locator := preMark + markedTerm

	idx := strings.Index(plainText, locator)
	if idx < 0 {
		idx = caseInsensitiveIndex(plainText, locator)
		if idx >= 0 {
			return idx + len(preMark)
		}

		// Context didn't match; fall back to the marked term alone.
		idx = strings.Index(plainText, markedTerm)
		if idx < 0 {
			idx = caseInsensitiveIndex(plainText, markedTerm)
		}

		return idx
	}

	// Return the position of the marked term within the plain text.
	return idx + len(preMark)
}

// skipPartialLeadingWord advances s past the first line when s starts with a
// lowercase letter, indicating that Bleve cut the content mid-word immediately
// after "…". Uppercase, digit, or whitespace at the start means the content
// already begins at a word boundary and the string is returned unchanged.
//
// When skipping, the function advances to the character after the first newline
// so that a partial trailing line such as "ome content.\nSetup\n…" is consumed
// as a unit rather than leaving "content.\n…" as a misleading prefix.
// If no newline is present it falls back to the first space or tab.
func skipPartialLeadingWord(s string) string {
	if s == "" {
		return s
	}

	// If s starts with whitespace it is already at a word boundary.
	if s[0] == ' ' || s[0] == '\t' || s[0] == '\n' || s[0] == '\r' {
		return s
	}

	// Only skip when the first character is a lowercase ASCII letter, which is
	// the tell-tale sign of a Bleve mid-word cut (e.g. "…ntroduction").
	if s[0] < 'a' || s[0] > 'z' {
		return s
	}

	// Advance past the first newline to discard the entire partial line.
	if idx := strings.IndexByte(s, '\n'); idx >= 0 {
		return s[idx+1:]
	}

	// No newline — fall back to the first horizontal whitespace character.
	if idx := strings.IndexAny(s, " \t\r"); idx > 0 {
		return s[idx+1:]
	}

	// No boundary found — the entire string might be a single partial word;
	// return as-is so callers can still attempt a lookup.
	return s
}
