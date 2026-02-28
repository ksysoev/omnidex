package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDetectContentType(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		content  string
		expected ContentType
	}{
		{
			name:     "markdown file by extension",
			path:     "docs/readme.md",
			content:  "# Hello World",
			expected: ContentTypeMarkdown,
		},
		{
			name:     "markdown file without extension",
			path:     "README",
			content:  "# Hello",
			expected: ContentTypeMarkdown,
		},
		{
			name: "OpenAPI YAML spec",
			path: "api/petstore.yaml",
			content: `openapi: "3.0.3"
info:
  title: Test
  version: "1.0.0"
paths: {}`,
			expected: ContentTypeOpenAPI,
		},
		{
			name: "OpenAPI YML spec",
			path: "api/petstore.yml",
			content: `openapi: "3.0.3"
info:
  title: Test
  version: "1.0.0"
paths: {}`,
			expected: ContentTypeOpenAPI,
		},
		{
			name:     "OpenAPI JSON spec",
			path:     "api/petstore.json",
			content:  `{"openapi": "3.0.3", "info": {"title": "Test", "version": "1.0.0"}, "paths": {}}`,
			expected: ContentTypeOpenAPI,
		},
		{
			name: "YAML file without openapi key defaults to markdown",
			path: "config.yaml",
			content: `name: my-app
version: 1.0.0`,
			expected: ContentTypeMarkdown,
		},
		{
			name:     "JSON file without openapi key defaults to markdown",
			path:     "config.json",
			content:  `{"name": "my-app", "version": "1.0.0"}`,
			expected: ContentTypeMarkdown,
		},
		{
			name:     "YAML file with invalid YAML defaults to markdown",
			path:     "broken.yaml",
			content:  `: invalid yaml [[[`,
			expected: ContentTypeMarkdown,
		},
		{
			name:     "JSON file with invalid JSON defaults to markdown",
			path:     "broken.json",
			content:  `{not valid json}`,
			expected: ContentTypeMarkdown,
		},
		{
			name:     "uppercase extension handled",
			path:     "api/spec.YAML",
			content:  `openapi: "3.0.3"` + "\n" + `info:` + "\n" + `  title: Test` + "\n" + `  version: "1.0"` + "\n" + `paths: {}`,
			expected: ContentTypeOpenAPI,
		},
		{
			name:     "txt file is markdown regardless of content",
			path:     "notes.txt",
			content:  `openapi: "3.0.3"`,
			expected: ContentTypeMarkdown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetectContentType(tt.path, []byte(tt.content))
			assert.Equal(t, tt.expected, result)
		})
	}
}
