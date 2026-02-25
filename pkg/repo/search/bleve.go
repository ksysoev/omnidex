// Package search provides full-text search functionality for documentation.
package search

import (
	"context"
	"fmt"
	"strings"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/mapping"
	bleveQuery "github.com/blevesearch/bleve/v2/search/query"

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

	q := buildSearchQuery(query)
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

// minFuzzyTermLength is the minimum term length required to apply fuzzy matching.
// Shorter terms produce too many false-positive matches.
const minFuzzyTermLength = 4

// longTermThreshold is the term length at which fuzzy matching uses a higher edit distance.
const longTermThreshold = 7

// queryTerm represents a single parsed search term.
type queryTerm struct {
	text   string
	phrase bool // true when the term was enclosed in double quotes
}

// splitQueryTerms parses user input into individual search terms.
// Double-quoted substrings are treated as phrase terms; unquoted words are split on whitespace.
func splitQueryTerms(input string) []queryTerm {
	var terms []queryTerm

	input = strings.TrimSpace(input)
	if input == "" {
		return terms
	}

	i := 0
	for i < len(input) {
		// Skip whitespace.
		if input[i] == ' ' || input[i] == '\t' {
			i++
			continue
		}

		// Handle quoted phrase.
		if input[i] == '"' {
			end := strings.IndexByte(input[i+1:], '"')
			if end == -1 {
				// No closing quote -- treat the rest as a single phrase.
				phrase := strings.TrimSpace(input[i+1:])
				if phrase != "" {
					terms = append(terms, queryTerm{text: phrase, phrase: true})
				}

				break
			}

			phrase := strings.TrimSpace(input[i+1 : i+1+end])
			if phrase != "" {
				terms = append(terms, queryTerm{text: phrase, phrase: true})
			}

			i += end + 2 // skip past closing quote

			continue
		}

		// Handle unquoted word.
		end := strings.IndexAny(input[i:], " \t")
		if end == -1 {
			terms = append(terms, queryTerm{text: input[i:]})

			break
		}

		terms = append(terms, queryTerm{text: input[i : i+end]})
		i += end
	}

	return terms
}

// buildSearchQuery constructs a hybrid Bleve query from user input.
// For each term it creates a disjunction of match, prefix, and fuzzy queries
// targeting both title and content fields with appropriate boost values.
// Multiple terms are combined with a conjunction so all terms must match.
func buildSearchQuery(userQuery string) bleveQuery.Query {
	terms := splitQueryTerms(userQuery)
	if len(terms) == 0 {
		return bleve.NewMatchNoneQuery()
	}

	termQueries := make([]bleveQuery.Query, 0, len(terms))

	for _, term := range terms {
		var disj bleveQuery.Query
		if term.phrase {
			disj = buildPhraseQueries(term.text)
		} else {
			disj = buildTermQueries(term.text)
		}

		termQueries = append(termQueries, disj)
	}

	if len(termQueries) == 1 {
		return termQueries[0]
	}

	return bleve.NewConjunctionQuery(termQueries...)
}

// buildPhraseQueries creates a disjunction of MatchPhraseQuery for title and content fields.
func buildPhraseQueries(phrase string) bleveQuery.Query {
	titleQ := bleve.NewMatchPhraseQuery(phrase)
	titleQ.SetField("title")
	titleQ.SetBoost(10.0)

	contentQ := bleve.NewMatchPhraseQuery(phrase)
	contentQ.SetField("content")
	contentQ.SetBoost(5.0)

	return bleve.NewDisjunctionQuery(titleQ, contentQ)
}

// buildTermQueries creates a disjunction of match, prefix, and fuzzy queries
// for a single non-phrase term, targeting both title and content fields.
func buildTermQueries(term string) bleveQuery.Query {
	subQueries := make([]bleveQuery.Query, 0, 6) //nolint:mnd // up to 6 sub-queries: match, prefix, fuzzy for title and content

	// Exact/analyzed match -- highest priority.
	titleMatch := bleve.NewMatchQuery(term)
	titleMatch.SetField("title")
	titleMatch.SetBoost(6.0)

	contentMatch := bleve.NewMatchQuery(term)
	contentMatch.SetField("content")
	contentMatch.SetBoost(3.0)

	subQueries = append(subQueries, titleMatch, contentMatch)

	// Prefix match -- medium priority.
	lowered := strings.ToLower(term)

	titlePrefix := bleve.NewPrefixQuery(lowered)
	titlePrefix.SetField("title")
	titlePrefix.SetBoost(3.0)

	contentPrefix := bleve.NewPrefixQuery(lowered)
	contentPrefix.SetField("content")
	contentPrefix.SetBoost(1.5)

	subQueries = append(subQueries, titlePrefix, contentPrefix)

	// Fuzzy match -- lowest priority (only for terms long enough to avoid noise).
	if len(term) >= minFuzzyTermLength {
		fuzziness := 1
		if len(term) >= longTermThreshold {
			fuzziness = 2
		}

		titleFuzzy := bleve.NewFuzzyQuery(lowered)
		titleFuzzy.SetField("title")
		titleFuzzy.SetFuzziness(fuzziness)
		titleFuzzy.SetBoost(1.0)

		contentFuzzy := bleve.NewFuzzyQuery(lowered)
		contentFuzzy.SetField("content")
		contentFuzzy.SetFuzziness(fuzziness)
		contentFuzzy.SetBoost(0.5)

		subQueries = append(subQueries, titleFuzzy, contentFuzzy)
	}

	return bleve.NewDisjunctionQuery(subQueries...)
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
