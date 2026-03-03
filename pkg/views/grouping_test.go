package views

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ksysoev/omnidex/pkg/core"
)

func meta(path string) core.DocumentMeta {
	return core.DocumentMeta{
		ID:        "repo/" + path,
		Repo:      "owner/repo",
		Path:      path,
		Title:     path,
		UpdatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}
}

func TestBuildDocTree_Empty(t *testing.T) {
	result := BuildDocTree(nil)
	assert.Nil(t, result)

	result = BuildDocTree([]core.DocumentMeta{})
	assert.Nil(t, result)
}

func TestBuildDocTree_AllRootLevel(t *testing.T) {
	docs := []core.DocumentMeta{
		meta("readme.md"),
		meta("changelog.md"),
		meta("contributing.md"),
	}

	result := BuildDocTree(docs)

	require.Len(t, result, 3)

	// Root docs are sorted alphabetically by path.
	assert.Equal(t, "changelog.md", result[0].Name)
	assert.NotNil(t, result[0].Doc)
	assert.Empty(t, result[0].Children)

	assert.Equal(t, "contributing.md", result[1].Name)
	assert.NotNil(t, result[1].Doc)

	assert.Equal(t, "readme.md", result[2].Name)
	assert.NotNil(t, result[2].Doc)
}

func TestBuildDocTree_SingleFolderMultipleDocs(t *testing.T) {
	docs := []core.DocumentMeta{
		meta("guides/setup.md"),
		meta("guides/deployment.md"),
	}

	result := BuildDocTree(docs)

	require.Len(t, result, 1)

	folder := result[0]
	assert.Equal(t, "guides", folder.Name)
	assert.Nil(t, folder.Doc, "folder node must have nil Doc")
	require.Len(t, folder.Children, 2)

	// Children inside the folder are also sorted.
	assert.Equal(t, "deployment.md", folder.Children[0].Name)
	assert.NotNil(t, folder.Children[0].Doc)
	assert.Equal(t, "deployment.md", folder.Children[0].Doc.Path)

	assert.Equal(t, "setup.md", folder.Children[1].Name)
	assert.NotNil(t, folder.Children[1].Doc)
	assert.Equal(t, "setup.md", folder.Children[1].Doc.Path)
}

func TestBuildDocTree_MultipleFoldersSortedAlphabetically(t *testing.T) {
	docs := []core.DocumentMeta{
		meta("tutorials/intro.md"),
		meta("api/overview.md"),
		meta("guides/setup.md"),
	}

	result := BuildDocTree(docs)

	require.Len(t, result, 3)

	// Folders sorted: api, guides, tutorials.
	assert.Equal(t, "api", result[0].Name)
	assert.Equal(t, "guides", result[1].Name)
	assert.Equal(t, "tutorials", result[2].Name)
}

func TestBuildDocTree_RootDocsAppearBeforeFolders(t *testing.T) {
	docs := []core.DocumentMeta{
		meta("guides/setup.md"),
		meta("readme.md"),
		meta("api/overview.md"),
		meta("changelog.md"),
	}

	result := BuildDocTree(docs)

	require.Len(t, result, 4)

	// Root docs first, sorted.
	assert.Equal(t, "changelog.md", result[0].Name)
	assert.NotNil(t, result[0].Doc)

	assert.Equal(t, "readme.md", result[1].Name)
	assert.NotNil(t, result[1].Doc)

	// Folders after root docs, sorted.
	assert.Equal(t, "api", result[2].Name)
	assert.Nil(t, result[2].Doc)

	assert.Equal(t, "guides", result[3].Name)
	assert.Nil(t, result[3].Doc)
}

func TestBuildDocTree_ArbitraryNesting(t *testing.T) {
	docs := []core.DocumentMeta{
		meta("api/reference/endpoints.md"),
		meta("api/overview.md"),
	}

	result := BuildDocTree(docs)

	require.Len(t, result, 1)

	apiFolder := result[0]
	assert.Equal(t, "api", apiFolder.Name)
	assert.Nil(t, apiFolder.Doc)
	require.Len(t, apiFolder.Children, 2)

	// overview.md is root-level within api/ so appears first.
	assert.Equal(t, "overview.md", apiFolder.Children[0].Name)
	assert.NotNil(t, apiFolder.Children[0].Doc)

	// reference/ is a sub-folder.
	refFolder := apiFolder.Children[1]
	assert.Equal(t, "reference", refFolder.Name)
	assert.Nil(t, refFolder.Doc)
	require.Len(t, refFolder.Children, 1)
	assert.Equal(t, "endpoints.md", refFolder.Children[0].Name)
	assert.NotNil(t, refFolder.Children[0].Doc)
	assert.Equal(t, "endpoints.md", refFolder.Children[0].Doc.Path)
}

func TestBuildDocTree_OriginalPathNotMutated(t *testing.T) {
	docs := []core.DocumentMeta{
		meta("api/overview.md"),
		meta("guides/setup.md"),
	}

	// Original paths before calling BuildDocTree.
	originalPaths := make([]string, len(docs))
	for i, d := range docs {
		originalPaths[i] = d.Path
	}

	BuildDocTree(docs)

	// Original slice must not be mutated.
	for i, d := range docs {
		assert.Equal(t, originalPaths[i], d.Path)
	}
}

func TestBuildDocTree_DocPointerPreservesOriginalData(t *testing.T) {
	updatedAt := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	docs := []core.DocumentMeta{
		{
			ID:        "owner/repo/readme.md",
			Repo:      "owner/repo",
			Path:      "readme.md",
			Title:     "README",
			UpdatedAt: updatedAt,
		},
	}

	result := BuildDocTree(docs)

	require.Len(t, result, 1)
	require.NotNil(t, result[0].Doc)
	assert.Equal(t, "README", result[0].Doc.Title)
	assert.Equal(t, "owner/repo", result[0].Doc.Repo)
	assert.Equal(t, updatedAt, result[0].Doc.UpdatedAt)
}

func TestBuildDocTree_SingleDocInFolder(t *testing.T) {
	docs := []core.DocumentMeta{
		meta("guides/only.md"),
	}

	result := BuildDocTree(docs)

	require.Len(t, result, 1)
	assert.Equal(t, "guides", result[0].Name)
	assert.Nil(t, result[0].Doc)
	require.Len(t, result[0].Children, 1)
	assert.Equal(t, "only.md", result[0].Children[0].Name)
}

func TestBuildDocTree_DeepNestingThreeLevels(t *testing.T) {
	docs := []core.DocumentMeta{
		meta("a/b/c/deep.md"),
	}

	result := BuildDocTree(docs)

	require.Len(t, result, 1)
	assert.Equal(t, "a", result[0].Name)

	require.Len(t, result[0].Children, 1)
	assert.Equal(t, "b", result[0].Children[0].Name)

	require.Len(t, result[0].Children[0].Children, 1)
	assert.Equal(t, "c", result[0].Children[0].Children[0].Name)

	require.Len(t, result[0].Children[0].Children[0].Children, 1)
	assert.Equal(t, "deep.md", result[0].Children[0].Children[0].Children[0].Name)
	assert.NotNil(t, result[0].Children[0].Children[0].Children[0].Doc)
}
