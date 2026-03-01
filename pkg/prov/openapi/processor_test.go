package openapi

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const minimalSpecYAML = `openapi: "3.0.3"
info:
  title: Petstore API
  description: A sample API for pets
  version: "1.0.0"
paths:
  /pets:
    get:
      summary: List all pets
      description: Returns a list of all pets in the store
      operationId: listPets
      tags:
        - pets
      responses:
        "200":
          description: A list of pets
    post:
      summary: Create a pet
      operationId: createPet
      tags:
        - pets
      responses:
        "201":
          description: Pet created
  /pets/{petId}:
    get:
      summary: Get a pet by ID
      description: Returns a single pet
      operationId: showPetById
      tags:
        - pets
      parameters:
        - name: petId
          in: path
          required: true
          schema:
            type: string
      responses:
        "200":
          description: A pet
tags:
  - name: pets
    description: Everything about your Pets
`

const minimalSpecJSON = `{
  "openapi": "3.0.3",
  "info": {
    "title": "Petstore API",
    "description": "A sample API for pets",
    "version": "1.0.0"
  },
  "paths": {
    "/pets": {
      "get": {
        "summary": "List all pets",
        "responses": {
          "200": {
            "description": "A list of pets"
          }
        }
      }
    }
  }
}`

func TestProcessor_RenderHTML(t *testing.T) {
	t.Run("valid YAML spec returns JSON", func(t *testing.T) {
		p := New()
		html, headings, err := p.RenderHTML([]byte(minimalSpecYAML))

		require.NoError(t, err)
		assert.Empty(t, headings, "OpenAPI specs should not produce headings")

		// The output should be valid JSON.
		assert.True(t, json.Valid(html), "output should be valid JSON")

		// The JSON should contain key spec fields.
		var doc map[string]any
		require.NoError(t, json.Unmarshal(html, &doc))
		assert.Equal(t, "3.0.3", doc["openapi"])
	})

	t.Run("valid JSON spec returns JSON", func(t *testing.T) {
		p := New()
		html, headings, err := p.RenderHTML([]byte(minimalSpecJSON))

		require.NoError(t, err)
		assert.Empty(t, headings)
		assert.True(t, json.Valid(html))
	})

	t.Run("invalid spec returns error", func(t *testing.T) {
		p := New()
		_, _, err := p.RenderHTML([]byte("not a valid spec at all"))

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse OpenAPI spec")
	})

	t.Run("semantically invalid spec is accepted", func(t *testing.T) {
		p := New()
		// Path has {petId} parameter but operation doesn't define it — previously
		// failed semantic validation, but now passes because we rely on Scalar API Reference
		// to surface validation warnings to the user.
		specWithMissingParam := []byte(`openapi: "3.0.3"
info:
  title: Bad API
  version: "1.0.0"
paths:
  /items/{itemId}:
    get:
      summary: Get item
      responses:
        "200":
          description: OK`)

		html, headings, err := p.RenderHTML(specWithMissingParam)

		require.NoError(t, err)
		assert.Empty(t, headings)
		assert.True(t, json.Valid(html), "output should be valid JSON")
	})
}

func TestProcessor_ExtractTitle(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "YAML spec with title",
			input:    minimalSpecYAML,
			expected: "Petstore API",
		},
		{
			name:     "JSON spec with title",
			input:    minimalSpecJSON,
			expected: "Petstore API",
		},
		{
			name: "spec without title",
			input: `openapi: "3.0.3"
info:
  version: "1.0.0"
paths: {}`,
			expected: "",
		},
		{
			name:     "invalid content",
			input:    "not a spec",
			expected: "",
		},
		{
			name: "semantically invalid spec still extracts title",
			input: `openapi: "3.0.3"
info:
  title: Bad API
  version: "1.0.0"
paths:
  /items/{itemId}:
    get:
      summary: Get item
      responses:
        "200":
          description: OK`,
			expected: "Bad API",
		},
	}

	p := New()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := p.ExtractTitle([]byte(tt.input))
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGitHubSlug(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "lowercase", input: "pets", expected: "pets"},
		{name: "capitalized", input: "Pets", expected: "pets"},
		{name: "with-space", input: "Pet Store", expected: "pet-store"},
		{name: "multi-word", input: "user authentication", expected: "user-authentication"},
		{name: "contains-slash", input: "user/v2", expected: "user-v2"},
		{name: "with-punctuation", input: "My Tag!", expected: "my-tag"},
		{name: "empty-string", input: "", expected: ""},
		{name: "leading-dashes", input: "--leading--", expected: "leading"},
		{name: "surrounding-spaces", input: "  spaces  ", expected: "spaces"},
		{name: "starting-with-numbers", input: "123numbers", expected: "123numbers"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, githubSlug(tt.input))
		})
	}
}

func TestProcessor_ExtractHeadings(t *testing.T) {
	t.Run("spec with tags and operations produces Scalar-compatible anchors", func(t *testing.T) {
		p := New()
		headings := p.ExtractHeadings([]byte(minimalSpecYAML))

		require.NotEmpty(t, headings)

		// Tag heading.
		assert.Equal(t, "pets", headings[0].Text)
		assert.Equal(t, "tag/pets", headings[0].ID)

		// Operation headings (paths sorted: /pets before /pets/{petId}).
		// GET /pets
		assert.Equal(t, "GET /pets", headings[1].Text)
		assert.Equal(t, "tag/pets/GET/pets", headings[1].ID)
		// POST /pets
		assert.Equal(t, "POST /pets", headings[2].Text)
		assert.Equal(t, "tag/pets/POST/pets", headings[2].ID)
		// GET /pets/{petId}
		assert.Equal(t, "GET /pets/{petId}", headings[3].Text)
		assert.Equal(t, "tag/pets/GET/pets/{petId}", headings[3].ID)
	})

	t.Run("spec without tags creates untagged operation anchors", func(t *testing.T) {
		p := New()
		spec := []byte(`openapi: "3.0.3"
info:
  title: No Tags API
  version: "1.0.0"
paths:
  /items:
    get:
      summary: List items
      responses:
        "200":
          description: OK
    post:
      summary: Create item
      responses:
        "201":
          description: Created
`)
		headings := p.ExtractHeadings(spec)

		require.Len(t, headings, 2)
		assert.Equal(t, "GET /items", headings[0].Text)
		assert.Equal(t, "GET/items", headings[0].ID)
		assert.Equal(t, "POST /items", headings[1].Text)
		assert.Equal(t, "POST/items", headings[1].ID)
	})

	t.Run("paths are sorted alphabetically", func(t *testing.T) {
		p := New()
		spec := []byte(`openapi: "3.0.3"
info:
  title: Sorted API
  version: "1.0.0"
paths:
  /zebra:
    get:
      summary: Zebra
      responses:
        "200":
          description: OK
  /apple:
    get:
      summary: Apple
      responses:
        "200":
          description: OK
  /mango:
    get:
      summary: Mango
      responses:
        "200":
          description: OK
`)
		headings := p.ExtractHeadings(spec)

		require.Len(t, headings, 3)
		assert.Equal(t, "GET /apple", headings[0].Text)
		assert.Equal(t, "GET /mango", headings[1].Text)
		assert.Equal(t, "GET /zebra", headings[2].Text)
	})

	t.Run("tag with spaces gets slugged correctly", func(t *testing.T) {
		p := New()
		spec := []byte(`openapi: "3.0.3"
info:
  title: Multi Word Tags
  version: "1.0.0"
tags:
  - name: Pet Store
paths:
  /pets:
    get:
      summary: List pets
      tags:
        - Pet Store
      responses:
        "200":
          description: OK
`)
		headings := p.ExtractHeadings(spec)

		require.NotEmpty(t, headings)
		assert.Equal(t, "Pet Store", headings[0].Text)
		assert.Equal(t, "tag/pet-store", headings[0].ID)
		assert.Equal(t, "GET /pets", headings[1].Text)
		assert.Equal(t, "tag/pet-store/GET/pets", headings[1].ID)
	})

	t.Run("spec with no paths returns tag headings only", func(t *testing.T) {
		p := New()
		spec := []byte(`openapi: "3.0.3"
info:
  title: Tags Only API
  version: "1.0.0"
tags:
  - name: admin
paths: {}
`)
		headings := p.ExtractHeadings(spec)

		require.Len(t, headings, 1)
		assert.Equal(t, "admin", headings[0].Text)
		assert.Equal(t, "tag/admin", headings[0].ID)
	})

	t.Run("invalid spec returns nil", func(t *testing.T) {
		p := New()
		headings := p.ExtractHeadings([]byte("not a spec"))
		assert.Nil(t, headings)
	})

	t.Run("empty-name tags are skipped to match ToPlainText alignment", func(t *testing.T) {
		p := New()
		spec := []byte(`openapi: "3.0.3"
info:
  title: Mixed Tags API
  version: "1.0.0"
tags:
  - name: ""
  - name: visible
paths:
  /items:
    get:
      summary: List items
      tags:
        - visible
      responses:
        "200":
          description: OK
`)
		headings := p.ExtractHeadings(spec)
		plainText := p.ToPlainText(spec)

		// Empty-name tag must not appear in headings.
		require.Len(t, headings, 2)
		assert.Equal(t, "visible", headings[0].Text)
		assert.Equal(t, "tag/visible", headings[0].ID)

		// Empty-name tag must not produce a line in plain text,
		// so both functions iterate the same entries.
		assert.NotContains(t, plainText, "\n\n")
		assert.Contains(t, plainText, "visible")
	})
}

func TestProcessor_ToPlainText(t *testing.T) {
	t.Run("extracts searchable text from YAML spec", func(t *testing.T) {
		p := New()
		text := p.ToPlainText([]byte(minimalSpecYAML))

		assert.Contains(t, text, "Petstore API")
		assert.Contains(t, text, "A sample API for pets")
		assert.Contains(t, text, "/pets")
		assert.Contains(t, text, "List all pets")
		assert.Contains(t, text, "Create a pet")
		assert.Contains(t, text, "Get a pet by ID")
		assert.Contains(t, text, "pets")
		assert.Contains(t, text, "Everything about your Pets")
		// Operations are emitted as "METHOD path" lines.
		assert.Contains(t, text, "GET /pets")
		assert.Contains(t, text, "POST /pets")
		assert.Contains(t, text, "GET /pets/{petId}")
	})

	t.Run("extracts searchable text from JSON spec", func(t *testing.T) {
		p := New()
		text := p.ToPlainText([]byte(minimalSpecJSON))

		assert.Contains(t, text, "Petstore API")
		assert.Contains(t, text, "List all pets")
	})

	t.Run("invalid content returns empty string", func(t *testing.T) {
		p := New()
		text := p.ToPlainText([]byte("not a spec"))

		assert.Empty(t, text)
	})

	t.Run("semantically invalid spec still extracts text", func(t *testing.T) {
		p := New()
		text := p.ToPlainText([]byte(`openapi: "3.0.3"
info:
  title: Bad API
  version: "1.0.0"
paths:
  /items/{itemId}:
    get:
      summary: Get item
      responses:
        "200":
          description: OK`))

		assert.Contains(t, text, "Bad API")
		assert.Contains(t, text, "/items/{itemId}")
		assert.Contains(t, text, "Get item")
	})

	t.Run("spec with no paths", func(t *testing.T) {
		p := New()
		text := p.ToPlainText([]byte(`openapi: "3.0.3"
info:
  title: Empty API
  version: "1.0.0"
paths: {}`))

		assert.Contains(t, text, "Empty API")
	})
}
