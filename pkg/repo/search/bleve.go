// Package search provides full-text search functionality for documentation.
package search

import (
	"context"
	"fmt"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/mapping"
	bleveSearch "github.com/blevesearch/bleve/v2/search"
	"github.com/ksysoev/omnidex/pkg/core"
)

// searchDocument is the internal representation of a document stored in the Bleve index.
type searchDocument struct {
	ID      string `json:"id"`
	Repo    string `json:"repo"`
	Path    string `json:"path"`
	Title   string `json:"title"`
	Content string `json:"content"`
}

// BleveEngine implements full-text search using Bleve embedded search library.
type BleveEngine struct {
	index bleve.Index
}

// NewBleve creates a new Bleve search engine. It opens an existing index at indexPath,
// or creates a new one if it does not exist.
func NewBleve(indexPath string) (*BleveEngine, error) {
	index, err := bleve.Open(indexPath)
	if err != nil {
		index, err = bleve.New(indexPath, buildIndexMapping())
		if err != nil {
			return nil, fmt.Errorf("failed to create bleve index: %w", err)
		}
	}

	return &BleveEngine{index: index}, nil
}

// Index adds or updates a document in the search index.
func (e *BleveEngine) Index(_ context.Context, doc core.Document, plainText string) error { //nolint:gocritic // Document is passed by value for immutability
	searchDoc := searchDocument{
		ID:      doc.ID,
		Repo:    doc.Repo,
		Path:    doc.Path,
		Title:   doc.Title,
		Content: plainText,
	}

	if err := e.index.Index(doc.ID, searchDoc); err != nil {
		return fmt.Errorf("failed to index document %s: %w", doc.ID, err)
	}

	return nil
}

// Remove deletes a document from the search index.
func (e *BleveEngine) Remove(_ context.Context, docID string) error {
	if err := e.index.Delete(docID); err != nil {
		return fmt.Errorf("failed to remove document %s from index: %w", docID, err)
	}

	return nil
}

// Search performs a full-text search query and returns matching results with highlighted fragments.
func (e *BleveEngine) Search(_ context.Context, query string, opts core.SearchOpts) (*core.SearchResults, error) {
	if opts.Limit <= 0 {
		opts.Limit = 20
	}

	q := bleve.NewQueryStringQuery(query)
	req := bleve.NewSearchRequestOptions(q, opts.Limit, opts.Offset, false)
	req.Highlight = bleve.NewHighlight()
	req.Fields = []string{"repo", "path", "title"}

	result, err := e.index.Search(req)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	hits := make([]core.SearchResult, 0, len(result.Hits))

	for _, hit := range result.Hits {
		sr := core.SearchResult{
			ID:        hit.ID,
			Score:     hit.Score,
			Fragments: extractFragments(hit.Fragments),
		}

		if repo, ok := hit.Fields["repo"].(string); ok {
			sr.Repo = repo
		}

		if path, ok := hit.Fields["path"].(string); ok {
			sr.Path = path
		}

		if title, ok := hit.Fields["title"].(string); ok {
			sr.Title = title
		}

		hits = append(hits, sr)
	}

	return &core.SearchResults{
		Hits:     hits,
		Total:    result.Total,
		Duration: result.Took,
	}, nil
}

// Close closes the Bleve index.
func (e *BleveEngine) Close() error {
	if err := e.index.Close(); err != nil {
		return fmt.Errorf("failed to close bleve index: %w", err)
	}

	return nil
}

// DocCount returns the number of documents in the index.
func (e *BleveEngine) DocCount() (uint64, error) {
	count, err := e.index.DocCount()
	if err != nil {
		return 0, fmt.Errorf("failed to get doc count: %w", err)
	}

	return count, nil
}

func buildIndexMapping() mapping.IndexMapping {
	docMapping := bleve.NewDocumentMapping()

	textFieldMapping := bleve.NewTextFieldMapping()
	textFieldMapping.Store = true
	textFieldMapping.IncludeTermVectors = true

	keywordFieldMapping := bleve.NewKeywordFieldMapping()
	keywordFieldMapping.Store = true

	docMapping.AddFieldMappingsAt("title", textFieldMapping)
	docMapping.AddFieldMappingsAt("content", textFieldMapping)
	docMapping.AddFieldMappingsAt("repo", keywordFieldMapping)
	docMapping.AddFieldMappingsAt("path", keywordFieldMapping)
	docMapping.AddFieldMappingsAt("id", keywordFieldMapping)

	indexMapping := bleve.NewIndexMapping()
	indexMapping.DefaultMapping = docMapping

	return indexMapping
}

func extractFragments(fragments bleveSearch.FieldFragmentMap) []string {
	result := make([]string, 0, len(fragments))

	for _, frags := range fragments {
		result = append(result, frags...)
	}

	return result
}
