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
			name: "YAML file without openapi key returns empty",
			path: "config.yaml",
			content: `name: my-app
version: 1.0.0`,
			expected: "",
		},
		{
			name:     "JSON file without openapi key returns empty",
			path:     "config.json",
			content:  `{"name": "my-app", "version": "1.0.0"}`,
			expected: "",
		},
		{
			name:     "YAML file with invalid YAML returns empty",
			path:     "broken.yaml",
			content:  `: invalid yaml [[[`,
			expected: "",
		},
		{
			name:     "JSON file with invalid JSON returns empty",
			path:     "broken.json",
			content:  `{not valid json}`,
			expected: "",
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
		{
			name: "Swagger 2.0 YAML spec detected as OpenAPI",
			path: "api/legacy.yaml",
			content: `swagger: "2.0"
info:
  title: Legacy API
  version: "1.0.0"
basePath: /v1
paths: {}`,
			expected: ContentTypeOpenAPI,
		},
		{
			name:     "Swagger 2.0 JSON spec detected as OpenAPI",
			path:     "api/legacy.json",
			content:  `{"swagger": "2.0", "info": {"title": "Legacy API", "version": "1.0.0"}, "basePath": "/v1", "paths": {}}`,
			expected: ContentTypeOpenAPI,
		},
		{
			name: "Swagger 2.0 YML spec detected as OpenAPI",
			path: "api/legacy.yml",
			content: `swagger: "2.0"
info:
  title: Legacy API
  version: "1.0.0"`,
			expected: ContentTypeOpenAPI,
		},
		{
			name:     "YAML flow mapping OpenAPI spec detected correctly",
			path:     "api/flow.yaml",
			content:  `{openapi: "3.0.3", info: {title: "Flow API", version: "1.0.0"}, paths: {}}`,
			expected: ContentTypeOpenAPI,
		},
		{
			name:     "YAML flow mapping non-OpenAPI returns empty",
			path:     "config.yml",
			content:  `{name: my-app, version: "1.0.0"}`,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetectContentType(tt.path, []byte(tt.content))
			assert.Equal(t, tt.expected, result)
		})
	}
}
