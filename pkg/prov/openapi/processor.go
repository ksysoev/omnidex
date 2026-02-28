// Package openapi provides an OpenAPI specification content processor.
// It implements the core.ContentProcessor interface for indexing, searching,
// and rendering OpenAPI specs (both YAML and JSON) using Swagger UI.
package openapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/ksysoev/omnidex/pkg/core"
)

// Processor implements core.ContentProcessor for OpenAPI specifications.
// It uses kin-openapi to parse specs and extract structured information for
// search indexing and title extraction. HTML rendering returns the parsed spec
// marshaled to JSON for consumption by Swagger UI.
type Processor struct{}

// New creates a new OpenAPI Processor.
func New() *Processor {
	return &Processor{}
}

// RenderHTML returns the raw OpenAPI spec as HTML-safe content for Swagger UI rendering.
// The view layer is responsible for embedding this into a Swagger UI container.
// Headings are not extracted for OpenAPI specs since Swagger UI provides its own navigation.
func (p *Processor) RenderHTML(src []byte) ([]byte, []core.Heading, error) {
	spec, err := parseSpec(src)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse OpenAPI spec: %w", err)
	}

	// Marshal the spec to JSON for Swagger UI consumption.
	specJSON, err := json.Marshal(spec)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal OpenAPI spec to JSON: %w", err)
	}

	return specJSON, nil, nil
}

// ExtractTitle returns the API title from the OpenAPI info section.
// Falls back to an empty string if the spec cannot be parsed or has no title.
func (p *Processor) ExtractTitle(src []byte) string {
	spec, err := parseSpec(src)
	if err != nil {
		return ""
	}

	if spec.Info != nil && spec.Info.Title != "" {
		return spec.Info.Title
	}

	return ""
}

// ToPlainText extracts searchable plain text from an OpenAPI spec.
// It collects the API title, description, endpoint paths, operation summaries,
// operation descriptions, and tag names to create a rich text representation
// for full-text search indexing.
func (p *Processor) ToPlainText(src []byte) string {
	spec, err := parseSpec(src)
	if err != nil {
		return ""
	}

	var buf bytes.Buffer

	// API-level metadata.
	if spec.Info != nil {
		if spec.Info.Title != "" {
			buf.WriteString(spec.Info.Title)
			buf.WriteByte('\n')
		}

		if spec.Info.Description != "" {
			buf.WriteString(spec.Info.Description)
			buf.WriteByte('\n')
		}
	}

	// Tag descriptions.
	for _, tag := range spec.Tags {
		if tag != nil {
			buf.WriteString(tag.Name)
			buf.WriteByte('\n')

			if tag.Description != "" {
				buf.WriteString(tag.Description)
				buf.WriteByte('\n')
			}
		}
	}

	// Paths and operations.
	if spec.Paths != nil {
		for path, pathItem := range spec.Paths.Map() {
			buf.WriteString(path)
			buf.WriteByte('\n')

			if pathItem == nil {
				continue
			}

			for _, op := range collectOperations(pathItem) {
				if op.Summary != "" {
					buf.WriteString(op.Summary)
					buf.WriteByte('\n')
				}

				if op.Description != "" {
					buf.WriteString(op.Description)
					buf.WriteByte('\n')
				}
			}
		}
	}

	return strings.TrimSpace(buf.String())
}

// parseSpec parses an OpenAPI spec from raw bytes (YAML or JSON).
// It uses a lenient loader that does not resolve external references.
// Semantic validation is intentionally skipped so that Swagger UI can render
// specs with minor compliance issues and provide its own user-facing feedback.
func parseSpec(src []byte) (*openapi3.T, error) {
	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = false

	spec, err := loader.LoadFromData(src)
	if err != nil {
		return nil, fmt.Errorf("failed to load OpenAPI spec: %w", err)
	}

	return spec, nil
}

// collectOperations returns all non-nil operations from a path item in a deterministic order.
func collectOperations(item *openapi3.PathItem) []*openapi3.Operation {
	ops := make([]*openapi3.Operation, 0, 8) //nolint:mnd // 8 HTTP methods

	for _, op := range []*openapi3.Operation{
		item.Get,
		item.Post,
		item.Put,
		item.Delete,
		item.Patch,
		item.Head,
		item.Options,
		item.Trace,
	} {
		if op != nil {
			ops = append(ops, op)
		}
	}

	return ops
}
