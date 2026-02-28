package core

import (
	"encoding/json"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// openAPIExtensions lists file extensions commonly used for OpenAPI specs.
var openAPIExtensions = map[string]bool{
	".yaml": true,
	".yml":  true,
	".json": true,
}

// DetectContentType determines the content type of a document based on its
// file path and content. It uses file extension as a fast pre-filter and then
// inspects the content for OpenAPI-specific markers (the "openapi" or "swagger"
// top-level keys). Files with non-YAML/JSON extensions are treated as markdown.
// YAML/JSON files that do not match OpenAPI heuristics return an empty ContentType
// to signal that they should be skipped (not treated as documentation).
func DetectContentType(path string, content []byte) ContentType {
	ext := strings.ToLower(filepath.Ext(path))

	// Only YAML/JSON files can be OpenAPI specs.
	if !openAPIExtensions[ext] {
		return ContentTypeMarkdown
	}

	if looksLikeOpenAPI(content, ext) {
		return ContentTypeOpenAPI
	}

	// Arbitrary YAML/JSON files that are not OpenAPI specs should not be
	// treated as documentation. Return empty to signal the caller to skip.
	return ""
}

// looksLikeOpenAPI checks whether the content contains an "openapi" (OAS 3.x)
// or "swagger" (OAS 2.0) top-level key. It supports both JSON and YAML formats.
func looksLikeOpenAPI(content []byte, ext string) bool {
	// Try JSON first if the extension suggests it or the content starts with '{'.
	if ext == ".json" || (len(content) > 0 && content[0] == '{') {
		return looksLikeOpenAPIJSON(content)
	}

	return looksLikeOpenAPIYAML(content)
}

// looksLikeOpenAPIJSON performs a lightweight check for the "openapi" or "swagger" key in JSON content.
func looksLikeOpenAPIJSON(content []byte) bool {
	var doc map[string]json.RawMessage

	if err := json.Unmarshal(content, &doc); err != nil {
		return false
	}

	_, hasOpenAPI := doc["openapi"]
	_, hasSwagger := doc["swagger"]

	return hasOpenAPI || hasSwagger
}

// looksLikeOpenAPIYAML performs a lightweight check for the "openapi" or "swagger" key in YAML content.
func looksLikeOpenAPIYAML(content []byte) bool {
	var doc map[string]any

	if err := yaml.Unmarshal(content, &doc); err != nil {
		return false
	}

	_, hasOpenAPI := doc["openapi"]
	_, hasSwagger := doc["swagger"]

	return hasOpenAPI || hasSwagger
}
