package s3store

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
	"github.com/johannesboyne/gofakes3"
	"github.com/johannesboyne/gofakes3/backend/s3mem"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ksysoev/omnidex/pkg/core"
)

const (
	testBucket = "test-bucket"
	testRegion = "us-east-1"
)

// newTestStore creates a Store backed by an in-process gofakes3 server.
// It also creates the test bucket so the store is ready to use immediately.
func newTestStore(t *testing.T) *Store {
	t.Helper()

	backend := s3mem.New()

	// Create the bucket directly on the backend before starting the HTTP server.
	require.NoError(t, backend.CreateBucket(testBucket))

	fake := gofakes3.New(backend)
	srv := httptest.NewServer(fake.Server())

	t.Cleanup(srv.Close)

	cfg := Config{
		Bucket:         testBucket,
		Region:         testRegion,
		Endpoint:       srv.URL,
		ForcePathStyle: true,
	}

	store, err := newWithStaticCreds(
		t.Context(),
		cfg,
		"test-access-key",
		"test-secret-key",
	)
	require.NoError(t, err)

	return store
}

func TestNew_ValidConfig(t *testing.T) {
	store := newTestStore(t)
	assert.NotNil(t, store)
}

func TestStore_SaveAndGet(t *testing.T) {
	store := newTestStore(t)

	doc := core.Document{
		ID:          "owner/repo/getting-started.md",
		Repo:        "owner/repo",
		Path:        "getting-started.md",
		Title:       "Getting Started",
		Content:     "# Getting Started\n\nWelcome!",
		CommitSHA:   "abc123",
		UpdatedAt:   time.Now().UTC().Truncate(time.Nanosecond),
		ContentType: core.ContentTypeMarkdown,
	}

	err := store.Save(t.Context(), doc)
	require.NoError(t, err)

	got, err := store.Get(t.Context(), "owner/repo", "getting-started.md")
	require.NoError(t, err)

	assert.Equal(t, doc.ID, got.ID)
	assert.Equal(t, doc.Repo, got.Repo)
	assert.Equal(t, doc.Path, got.Path)
	assert.Equal(t, doc.Title, got.Title)
	assert.Equal(t, doc.Content, got.Content)
	assert.Equal(t, doc.CommitSHA, got.CommitSHA)
	assert.Equal(t, doc.ContentType, got.ContentType)
	assert.Equal(t, doc.UpdatedAt, got.UpdatedAt)
}

func TestStore_GetNotFound(t *testing.T) {
	store := newTestStore(t)

	_, err := store.Get(t.Context(), "owner/repo", "nonexistent.md")
	require.Error(t, err)
	assert.True(t, errors.Is(err, core.ErrNotFound))
}

func TestStore_Delete(t *testing.T) {
	store := newTestStore(t)

	doc := core.Document{
		ID:        "owner/repo/to-delete.md",
		Repo:      "owner/repo",
		Path:      "to-delete.md",
		Title:     "Delete Me",
		Content:   "# Delete Me",
		CommitSHA: "abc123",
		UpdatedAt: time.Now().UTC(),
	}

	err := store.Save(t.Context(), doc)
	require.NoError(t, err)

	err = store.Delete(t.Context(), "owner/repo", "to-delete.md")
	require.NoError(t, err)

	_, err = store.Get(t.Context(), "owner/repo", "to-delete.md")
	require.Error(t, err)
	assert.True(t, errors.Is(err, core.ErrNotFound))
}

func TestStore_DeleteNonexistent(t *testing.T) {
	store := newTestStore(t)

	// Deleting a non-existent object must be idempotent (no error).
	err := store.Delete(t.Context(), "owner/repo", "does-not-exist.md")
	assert.NoError(t, err)
}

func TestStore_List(t *testing.T) {
	store := newTestStore(t)

	docs := []core.Document{
		{
			ID:        "owner/repo/getting-started.md",
			Repo:      "owner/repo",
			Path:      "getting-started.md",
			Title:     "Getting Started",
			Content:   "# Getting Started",
			CommitSHA: "abc",
			UpdatedAt: time.Now().UTC().Truncate(time.Nanosecond),
		},
		{
			ID:        "owner/repo/api/overview.md",
			Repo:      "owner/repo",
			Path:      "api/overview.md",
			Title:     "API Overview",
			Content:   "# API Overview",
			CommitSHA: "def",
			UpdatedAt: time.Now().UTC().Truncate(time.Nanosecond),
		},
	}

	for _, doc := range docs {
		require.NoError(t, store.Save(t.Context(), doc))
	}

	list, err := store.List(t.Context(), "owner/repo")
	require.NoError(t, err)
	assert.Len(t, list, 2)

	// Results should be sorted by path.
	assert.Equal(t, "api/overview.md", list[0].Path)
	assert.Equal(t, "getting-started.md", list[1].Path)
}

func TestStore_ListEmpty(t *testing.T) {
	store := newTestStore(t)

	list, err := store.List(t.Context(), "owner/nonexistent-repo")
	require.NoError(t, err)
	assert.Empty(t, list)
}

func TestStore_ListRepos(t *testing.T) {
	store := newTestStore(t)

	doc := core.Document{
		ID:        "owner/repo/README.md",
		Repo:      "owner/repo",
		Path:      "README.md",
		Title:     "README",
		Content:   "# README",
		CommitSHA: "abc",
		UpdatedAt: time.Now().UTC(),
	}

	require.NoError(t, store.Save(t.Context(), doc))

	repos, err := store.ListRepos(t.Context())
	require.NoError(t, err)
	require.Len(t, repos, 1)
	assert.Equal(t, "owner/repo", repos[0].Name)
	assert.Equal(t, 1, repos[0].DocCount)
}

func TestStore_ListReposEmpty(t *testing.T) {
	store := newTestStore(t)

	repos, err := store.ListRepos(t.Context())
	require.NoError(t, err)
	assert.Empty(t, repos)
}

func TestStore_ListMultipleRepos(t *testing.T) {
	store := newTestStore(t)

	for _, repo := range []string{"owner/repo1", "owner/repo2"} {
		doc := core.Document{
			ID:        repo + "/readme.md",
			Repo:      repo,
			Path:      "readme.md",
			Title:     repo,
			Content:   "# " + repo,
			CommitSHA: "abc",
			UpdatedAt: time.Now().UTC(),
		}
		require.NoError(t, store.Save(t.Context(), doc))
	}

	repos, err := store.ListRepos(t.Context())
	require.NoError(t, err)
	assert.Len(t, repos, 2)
	// Sorted alphabetically.
	assert.Equal(t, "owner/repo1", repos[0].Name)
	assert.Equal(t, "owner/repo2", repos[1].Name)
}

func TestStore_SaveOverwritesExisting(t *testing.T) {
	store := newTestStore(t)

	doc := core.Document{
		ID:        "owner/repo/readme.md",
		Repo:      "owner/repo",
		Path:      "readme.md",
		Title:     "Original",
		Content:   "# Original",
		CommitSHA: "abc",
		UpdatedAt: time.Now().UTC().Truncate(time.Nanosecond),
	}

	require.NoError(t, store.Save(t.Context(), doc))

	doc.Title = "Updated"
	doc.Content = "# Updated"
	doc.CommitSHA = "def"

	require.NoError(t, store.Save(t.Context(), doc))

	got, err := store.Get(t.Context(), "owner/repo", "readme.md")
	require.NoError(t, err)
	assert.Equal(t, "Updated", got.Title)
	assert.Equal(t, "# Updated", got.Content)
	assert.Equal(t, "def", got.CommitSHA)
}

func TestStore_SaveAndGetAsset(t *testing.T) {
	store := newTestStore(t)

	data := []byte{0x89, 0x50, 0x4E, 0x47} // PNG magic bytes

	err := store.SaveAsset(t.Context(), "owner/repo", "images/arch.png", data)
	require.NoError(t, err)

	got, err := store.GetAsset(t.Context(), "owner/repo", "images/arch.png")
	require.NoError(t, err)
	assert.Equal(t, data, got)
}

func TestStore_GetAssetNotFound(t *testing.T) {
	store := newTestStore(t)

	_, err := store.GetAsset(t.Context(), "owner/repo", "nonexistent.png")
	require.Error(t, err)
	assert.True(t, errors.Is(err, core.ErrNotFound))
}

func TestStore_DeleteAsset(t *testing.T) {
	store := newTestStore(t)

	data := []byte("fake image data")

	err := store.SaveAsset(t.Context(), "owner/repo", "img.png", data)
	require.NoError(t, err)

	err = store.DeleteAsset(t.Context(), "owner/repo", "img.png")
	require.NoError(t, err)

	_, err = store.GetAsset(t.Context(), "owner/repo", "img.png")
	require.Error(t, err)
	assert.True(t, errors.Is(err, core.ErrNotFound))
}

func TestStore_DeleteAssetNonexistent(t *testing.T) {
	store := newTestStore(t)

	// Deleting a non-existent asset must be idempotent (no error).
	err := store.DeleteAsset(t.Context(), "owner/repo", "nope.png")
	assert.NoError(t, err)
}

func TestStore_ListAssets(t *testing.T) {
	store := newTestStore(t)

	// No assets yet — should return an empty result.
	paths, err := store.ListAssets(t.Context(), "owner/repo")
	require.NoError(t, err)
	assert.Empty(t, paths)

	// Save some assets.
	require.NoError(t, store.SaveAsset(t.Context(), "owner/repo", "images/b.png", []byte("b")))
	require.NoError(t, store.SaveAsset(t.Context(), "owner/repo", "images/a.png", []byte("a")))
	require.NoError(t, store.SaveAsset(t.Context(), "owner/repo", "diagrams/arch.svg", []byte("svg")))

	paths, err = store.ListAssets(t.Context(), "owner/repo")
	require.NoError(t, err)

	// Results should be sorted alphabetically.
	assert.Equal(t, []string{
		"diagrams/arch.svg",
		"images/a.png",
		"images/b.png",
	}, paths)
}

func TestStore_SaveAssetOverwrite(t *testing.T) {
	store := newTestStore(t)

	require.NoError(t, store.SaveAsset(t.Context(), "owner/repo", "img.png", []byte("version1")))
	require.NoError(t, store.SaveAsset(t.Context(), "owner/repo", "img.png", []byte("version2")))

	got, err := store.GetAsset(t.Context(), "owner/repo", "img.png")
	require.NoError(t, err)
	assert.Equal(t, []byte("version2"), got)
}

func TestStore_ContentTypeRoundTrip(t *testing.T) {
	store := newTestStore(t)

	doc := core.Document{
		ID:          "owner/repo/api.yaml",
		Repo:        "owner/repo",
		Path:        "api.yaml",
		Title:       "API Spec",
		Content:     "openapi: 3.0.0",
		CommitSHA:   "abc",
		UpdatedAt:   time.Now().UTC().Truncate(time.Nanosecond),
		ContentType: core.ContentTypeOpenAPI,
	}

	require.NoError(t, store.Save(t.Context(), doc))

	got, err := store.Get(t.Context(), "owner/repo", "api.yaml")
	require.NoError(t, err)
	assert.Equal(t, core.ContentTypeOpenAPI, got.ContentType)
}

func TestStore_GetDefaultsToMarkdownContentType(t *testing.T) {
	store := newTestStore(t)

	// Save a document with an empty content type; Get should default to markdown.
	doc := core.Document{
		ID:          "owner/repo/doc.md",
		Repo:        "owner/repo",
		Path:        "doc.md",
		Title:       "Doc",
		Content:     "# Doc",
		CommitSHA:   "abc",
		UpdatedAt:   time.Now().UTC().Truncate(time.Nanosecond),
		ContentType: "",
	}

	require.NoError(t, store.Save(t.Context(), doc))

	got, err := store.Get(t.Context(), "owner/repo", "doc.md")
	require.NoError(t, err)
	assert.Equal(t, core.ContentTypeMarkdown, got.ContentType)
}

func TestStore_InvalidPathRejectsTraversal(t *testing.T) {
	store := newTestStore(t)

	traversalPaths := []string{
		"../secret",
		"../../etc/passwd",
		"",
		"/absolute/path",
	}

	for _, p := range traversalPaths {
		doc := core.Document{
			Repo:    "owner/repo",
			Path:    p,
			Title:   "Bad",
			Content: "bad",
		}

		err := store.Save(t.Context(), doc)
		require.Error(t, err, "expected error for path %q", p)
		assert.True(t, errors.Is(err, core.ErrInvalidPath), "expected ErrInvalidPath for path %q, got %v", p, err)

		_, err = store.Get(t.Context(), "owner/repo", p)
		require.Error(t, err, "expected error for Get path %q", p)
		assert.True(t, errors.Is(err, core.ErrInvalidPath), "expected ErrInvalidPath for Get path %q, got %v", p, err)

		err = store.Delete(t.Context(), "owner/repo", p)
		require.Error(t, err, "expected error for Delete path %q", p)
		assert.True(t, errors.Is(err, core.ErrInvalidPath), "expected ErrInvalidPath for Delete path %q, got %v", p, err)
	}
}

func TestStore_InvalidAssetPathRejectsTraversal(t *testing.T) {
	store := newTestStore(t)

	traversalPaths := []string{
		"../secret.png",
		"",
		"/absolute.png",
	}

	for _, p := range traversalPaths {
		err := store.SaveAsset(t.Context(), "owner/repo", p, []byte("data"))
		require.Error(t, err, "expected error for asset path %q", p)
		assert.True(t, errors.Is(err, core.ErrInvalidPath), "expected ErrInvalidPath for asset path %q, got %v", p, err)

		_, err = store.GetAsset(t.Context(), "owner/repo", p)
		require.Error(t, err, "expected error for GetAsset path %q", p)
		assert.True(t, errors.Is(err, core.ErrInvalidPath), "expected ErrInvalidPath for GetAsset path %q, got %v", p, err)

		err = store.DeleteAsset(t.Context(), "owner/repo", p)
		require.Error(t, err, "expected error for DeleteAsset path %q", p)
		assert.True(t, errors.Is(err, core.ErrInvalidPath), "expected ErrInvalidPath for DeleteAsset path %q, got %v", p, err)
	}
}

func TestStore_InvalidRepoRejectsTraversal(t *testing.T) {
	store := newTestStore(t)

	invalidRepos := []string{
		"../other",
		"",
		"/absolute/repo",
	}

	for _, repo := range invalidRepos {
		doc := core.Document{
			Repo:    repo,
			Path:    "readme.md",
			Title:   "Bad",
			Content: "bad",
		}

		err := store.Save(t.Context(), doc)
		require.Error(t, err, "Save: expected error for repo %q", repo)
		assert.True(t, errors.Is(err, core.ErrInvalidPath), "Save: expected ErrInvalidPath for repo %q, got %v", repo, err)

		_, err = store.Get(t.Context(), repo, "readme.md")
		require.Error(t, err, "Get: expected error for repo %q", repo)
		assert.True(t, errors.Is(err, core.ErrInvalidPath), "Get: expected ErrInvalidPath for repo %q, got %v", repo, err)

		err = store.Delete(t.Context(), repo, "readme.md")
		require.Error(t, err, "Delete: expected error for repo %q", repo)
		assert.True(t, errors.Is(err, core.ErrInvalidPath), "Delete: expected ErrInvalidPath for repo %q, got %v", repo, err)

		_, err = store.List(t.Context(), repo)
		require.Error(t, err, "List: expected error for repo %q", repo)
		assert.True(t, errors.Is(err, core.ErrInvalidPath), "List: expected ErrInvalidPath for repo %q, got %v", repo, err)

		err = store.SaveAsset(t.Context(), repo, "img.png", []byte("data"))
		require.Error(t, err, "SaveAsset: expected error for repo %q", repo)
		assert.True(t, errors.Is(err, core.ErrInvalidPath), "SaveAsset: expected ErrInvalidPath for repo %q, got %v", repo, err)

		_, err = store.GetAsset(t.Context(), repo, "img.png")
		require.Error(t, err, "GetAsset: expected error for repo %q", repo)
		assert.True(t, errors.Is(err, core.ErrInvalidPath), "GetAsset: expected ErrInvalidPath for repo %q, got %v", repo, err)

		err = store.DeleteAsset(t.Context(), repo, "img.png")
		require.Error(t, err, "DeleteAsset: expected error for repo %q", repo)
		assert.True(t, errors.Is(err, core.ErrInvalidPath), "DeleteAsset: expected ErrInvalidPath for repo %q, got %v", repo, err)

		_, err = store.ListAssets(t.Context(), repo)
		require.Error(t, err, "ListAssets: expected error for repo %q", repo)
		assert.True(t, errors.Is(err, core.ErrInvalidPath), "ListAssets: expected ErrInvalidPath for repo %q, got %v", repo, err)
	}
}

// ---------------------------------------------------------------------------
// Pure-function unit tests
// ---------------------------------------------------------------------------

func TestNew_BasicConfig(t *testing.T) {
	// New() uses the standard AWS credential chain. LoadDefaultConfig succeeds
	// even without real credentials; no network call is made at construction time.
	cfg := Config{
		Bucket: "my-bucket",
		Region: "us-east-1",
	}

	store, err := New(t.Context(), cfg)
	require.NoError(t, err)
	assert.NotNil(t, store)
	assert.Equal(t, "my-bucket", store.bucket)
}

func TestParseUpdatedAt(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	nano := time.Now().UTC()

	fallback := now.Add(-time.Hour)

	tests := []struct {
		want     time.Time
		fallback *time.Time
		name     string
		value    string
	}{
		{
			name:     "RFC3339Nano parses correctly",
			value:    nano.Format(time.RFC3339Nano),
			fallback: nil,
			want:     nano.Truncate(time.Nanosecond),
		},
		{
			name:     "RFC3339 legacy format parses correctly",
			value:    now.Format(time.RFC3339),
			fallback: nil,
			want:     now,
		},
		{
			name:     "invalid value uses fallback",
			value:    "not-a-time",
			fallback: &fallback,
			want:     fallback,
		},
		{
			name:     "invalid value with nil fallback returns zero time",
			value:    "not-a-time",
			fallback: nil,
			want:     time.Time{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := parseUpdatedAt(tc.value, tc.fallback)
			// Truncate to second for RFC3339 comparison; for RFC3339Nano compare directly.
			if strings.Contains(tc.name, "RFC3339Nano") {
				assert.Equal(t, tc.want.UTC(), got.UTC())
			} else {
				assert.Equal(t, tc.want.UTC().Truncate(time.Second), got.UTC().Truncate(time.Second))
			}
		})
	}
}

func TestIsNotFound(t *testing.T) {
	tests := []struct {
		err  error
		name string
		want bool
	}{
		{
			name: "smithy APIError with NoSuchKey code",
			err:  &smithy.GenericAPIError{Code: "NoSuchKey", Message: "key not found"},
			want: true,
		},
		{
			name: "smithy APIError with NotFound code",
			err:  &smithy.GenericAPIError{Code: "NotFound", Message: "not found"},
			want: true,
		},
		{
			name: "smithy APIError with 404 code",
			err:  &smithy.GenericAPIError{Code: "404", Message: "not found"},
			want: true,
		},
		{
			name: "types.NoSuchKey",
			err:  &types.NoSuchKey{},
			want: true,
		},
		{
			name: "generic error",
			err:  errors.New("some other error"),
			want: false,
		},
		{
			name: "smithy APIError with other code",
			err:  &smithy.GenericAPIError{Code: "AccessDenied", Message: "access denied"},
			want: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, isNotFound(tc.err))
		})
	}
}

func TestValidateRelPath_DotPaths(t *testing.T) {
	// "." and ".." both resolve to directory root and must be rejected.
	err := validateRelPath(".")
	require.Error(t, err)
	assert.ErrorIs(t, err, core.ErrInvalidPath)

	err = validateRelPath("..")
	require.Error(t, err)
	assert.ErrorIs(t, err, core.ErrInvalidPath)
}

// ---------------------------------------------------------------------------
// failingS3Client — a minimal s3Client implementation for error-path testing.
// Each field controls which method returns an error.
// ---------------------------------------------------------------------------

type failingS3Client struct {
	putErr    error
	getErr    error
	deleteErr error
	listErr   error
	headErr   error
	// getBody, when non-nil, overrides the response body for GetObject.
	getBody io.ReadCloser
}

func (f *failingS3Client) PutObject(_ context.Context, _ *s3.PutObjectInput, _ ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	return nil, f.putErr
}

func (f *failingS3Client) GetObject(_ context.Context, _ *s3.GetObjectInput, _ ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}

	if f.getBody != nil {
		return &s3.GetObjectOutput{Body: f.getBody}, nil
	}

	return nil, errors.New("failingS3Client: no getBody configured")
}

func (f *failingS3Client) DeleteObject(_ context.Context, _ *s3.DeleteObjectInput, _ ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
	return nil, f.deleteErr
}

func (f *failingS3Client) ListObjectsV2(_ context.Context, _ *s3.ListObjectsV2Input, _ ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	// Return an empty, terminal page.
	return &s3.ListObjectsV2Output{}, nil
}

func (f *failingS3Client) HeadObject(_ context.Context, _ *s3.HeadObjectInput, _ ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
	return nil, f.headErr
}

// newStoreWithClient builds a Store directly from an s3Client, bypassing AWS SDK setup.
func newStoreWithClient(client s3Client) *Store {
	return &Store{client: client, bucket: testBucket}
}

// ---------------------------------------------------------------------------
// Error-path tests using failingS3Client
// ---------------------------------------------------------------------------

func TestStore_Save_PutObjectError(t *testing.T) {
	putErr := errors.New("put failed")
	store := newStoreWithClient(&failingS3Client{putErr: putErr})

	doc := core.Document{Repo: "owner/repo", Path: "doc.md", Title: "T", Content: "C"}

	err := store.Save(t.Context(), doc)
	require.Error(t, err)
	assert.ErrorContains(t, err, "failed to upload document")
}

func TestStore_Save_UpdateRepoMetaError(t *testing.T) {
	// First PutObject (document upload) succeeds; second (meta.json) fails.
	var callCount int

	metaErr := errors.New("meta put failed")

	client := &countingPutClient{metaErr: metaErr, callCount: &callCount}
	store := newStoreWithClient(client)

	doc := core.Document{Repo: "owner/repo", Path: "doc.md", Title: "T", Content: "C"}

	err := store.Save(t.Context(), doc)
	require.Error(t, err)
	assert.ErrorContains(t, err, "failed to update repo metadata")
}

// countingPutClient fails the second PutObject call (used for meta.json).
type countingPutClient struct {
	metaErr   error
	callCount *int
}

func (c *countingPutClient) PutObject(_ context.Context, _ *s3.PutObjectInput, _ ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	*c.callCount++
	if *c.callCount > 1 {
		return nil, c.metaErr
	}

	return &s3.PutObjectOutput{}, nil
}

func (c *countingPutClient) GetObject(_ context.Context, _ *s3.GetObjectInput, _ ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	return nil, errors.New("not implemented")
}

func (c *countingPutClient) DeleteObject(_ context.Context, _ *s3.DeleteObjectInput, _ ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
	return nil, errors.New("not implemented")
}

func (c *countingPutClient) ListObjectsV2(_ context.Context, _ *s3.ListObjectsV2Input, _ ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
	return nil, errors.New("not implemented")
}

func (c *countingPutClient) HeadObject(_ context.Context, _ *s3.HeadObjectInput, _ ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
	return nil, errors.New("not implemented")
}

func TestStore_Get_S3Error(t *testing.T) {
	// Non-404 S3 error should be wrapped and returned.
	getErr := errors.New("network error")
	store := newStoreWithClient(&failingS3Client{getErr: getErr})

	_, err := store.Get(t.Context(), "owner/repo", "doc.md")
	require.Error(t, err)
	assert.ErrorContains(t, err, "failed to get document")
}

func TestStore_Get_ReadBodyError(t *testing.T) {
	// Simulate a body that errors on read.
	store := newStoreWithClient(&failingS3Client{
		getBody: io.NopCloser(&errorReader{}),
	})

	_, err := store.Get(t.Context(), "owner/repo", "doc.md")
	require.Error(t, err)
	assert.ErrorContains(t, err, "failed to read document body")
}

// errorReader always returns an error from Read.
type errorReader struct{}

func (e *errorReader) Read(_ []byte) (int, error) {
	return 0, errors.New("read error")
}

func TestStore_Delete_S3Error(t *testing.T) {
	// A non-404 DeleteObject error must be propagated.
	deleteErr := errors.New("delete failed")
	store := newStoreWithClient(&failingS3Client{deleteErr: deleteErr})

	err := store.Delete(t.Context(), "owner/repo", "doc.md")
	require.Error(t, err)
	assert.ErrorContains(t, err, "failed to delete document")
}

func TestStore_List_PaginatorError(t *testing.T) {
	listErr := errors.New("list failed")
	store := newStoreWithClient(&failingS3Client{listErr: listErr})

	_, err := store.List(t.Context(), "owner/repo")
	require.Error(t, err)
	assert.ErrorContains(t, err, "failed to list documents")
}

func TestStore_List_HeadObjectSkipsOnError(t *testing.T) {
	// When HeadObject fails the entry should be skipped (slog.Warn) rather than
	// returning an error. We use gofakes3 for the listing but override HeadObject
	// via a wrapper that wraps the real client.
	base := newTestStore(t)

	doc := core.Document{
		Repo:    "owner/repo",
		Path:    "readme.md",
		Title:   "README",
		Content: "# README",
	}

	require.NoError(t, base.Save(t.Context(), doc))

	// Wrap the real client so HeadObject always fails.
	wrapped := &headFailClient{inner: base.client}
	store := &Store{client: wrapped, bucket: base.bucket}

	list, err := store.List(t.Context(), "owner/repo")
	require.NoError(t, err) // no error — bad entries are skipped
	assert.Empty(t, list)   // the one entry was skipped
}

// headFailClient delegates all methods to the inner client except HeadObject.
type headFailClient struct {
	inner s3Client
}

func (h *headFailClient) PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	return h.inner.PutObject(ctx, params, optFns...)
}

func (h *headFailClient) GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	return h.inner.GetObject(ctx, params, optFns...)
}

func (h *headFailClient) DeleteObject(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
	return h.inner.DeleteObject(ctx, params, optFns...)
}

func (h *headFailClient) ListObjectsV2(ctx context.Context, params *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
	return h.inner.ListObjectsV2(ctx, params, optFns...)
}

func (h *headFailClient) HeadObject(_ context.Context, _ *s3.HeadObjectInput, _ ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
	return nil, errors.New("head failed")
}

func TestStore_List_TitleFallback(t *testing.T) {
	// Upload a doc object with no title metadata so that List falls back to the
	// relative path as the title.
	store := newTestStore(t)
	relPath := "no-title.md"
	key := "owner/repo/" + docsPrefix + relPath

	_, err := store.client.PutObject(t.Context(), &s3.PutObjectInput{
		Bucket: aws.String(store.bucket),
		Key:    aws.String(key),
		Body:   strings.NewReader("# content"),
		// No Metadata — title will be missing.
	})
	require.NoError(t, err)

	list, err := store.List(t.Context(), "owner/repo")
	require.NoError(t, err)
	require.Len(t, list, 1)
	// Title should fall back to the relative path.
	assert.Equal(t, relPath, list[0].Title)
}

func TestStore_ListRepos_ReadRepoMetaSkipsOnError(t *testing.T) {
	// When readRepoMeta fails the repo is skipped (slog.Warn).
	base := newTestStore(t)

	doc := core.Document{
		Repo:    "owner/repo",
		Path:    "readme.md",
		Title:   "README",
		Content: "# README",
	}

	require.NoError(t, base.Save(t.Context(), doc))

	// Wrap the client so GetObject always fails (breaks readRepoMeta).
	wrapped := &getFailClient{inner: base.client}
	store := &Store{client: wrapped, bucket: base.bucket}

	repos, err := store.ListRepos(t.Context())
	require.NoError(t, err) // no error — bad repos are skipped
	assert.Empty(t, repos)
}

// getFailClient delegates all methods except GetObject which always errors.
type getFailClient struct {
	inner s3Client
}

func (g *getFailClient) PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	return g.inner.PutObject(ctx, params, optFns...)
}

func (g *getFailClient) GetObject(_ context.Context, _ *s3.GetObjectInput, _ ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	return nil, errors.New("get failed")
}

func (g *getFailClient) DeleteObject(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
	return g.inner.DeleteObject(ctx, params, optFns...)
}

func (g *getFailClient) ListObjectsV2(ctx context.Context, params *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
	return g.inner.ListObjectsV2(ctx, params, optFns...)
}

func (g *getFailClient) HeadObject(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
	return g.inner.HeadObject(ctx, params, optFns...)
}

func TestStore_ListRepos_CountDocsError(t *testing.T) {
	// When countDocs fails, the repo is still included but DocCount is 0.
	base := newTestStore(t)

	doc := core.Document{
		Repo:    "owner/repo",
		Path:    "readme.md",
		Title:   "README",
		Content: "# README",
	}

	require.NoError(t, base.Save(t.Context(), doc))

	// Wrap so ListObjectsV2 fails only for the docs/ prefix (countDocs), but
	// succeeds for owner-level and repo-level listing, and GetObject succeeds
	// for the meta.json fetch.
	wrapped := &countDocsFailClient{inner: base.client}
	store := &Store{client: wrapped, bucket: base.bucket}

	repos, err := store.ListRepos(t.Context())
	require.NoError(t, err)
	require.Len(t, repos, 1)
	assert.Equal(t, 0, repos[0].DocCount)
}

// countDocsFailClient fails ListObjectsV2 only when the prefix contains docsPrefix.
type countDocsFailClient struct {
	inner s3Client
}

func (c *countDocsFailClient) PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	return c.inner.PutObject(ctx, params, optFns...)
}

func (c *countDocsFailClient) GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	return c.inner.GetObject(ctx, params, optFns...)
}

func (c *countDocsFailClient) DeleteObject(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
	return c.inner.DeleteObject(ctx, params, optFns...)
}

func (c *countDocsFailClient) ListObjectsV2(_ context.Context, params *s3.ListObjectsV2Input, _ ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
	if params.Prefix != nil && strings.HasSuffix(aws.ToString(params.Prefix), docsPrefix) {
		return nil, errors.New("list docs failed")
	}
	// Delegate to inner for the owner/repo-level listing.
	return c.inner.ListObjectsV2(context.Background(), params)
}

func (c *countDocsFailClient) HeadObject(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
	return c.inner.HeadObject(ctx, params, optFns...)
}

func TestStore_SaveAsset_PutObjectError(t *testing.T) {
	putErr := errors.New("put failed")
	store := newStoreWithClient(&failingS3Client{putErr: putErr})

	err := store.SaveAsset(t.Context(), "owner/repo", "img.png", []byte("data"))
	require.Error(t, err)
	assert.ErrorContains(t, err, "failed to upload asset")
}

func TestStore_GetAsset_S3Error(t *testing.T) {
	getErr := errors.New("network error")
	store := newStoreWithClient(&failingS3Client{getErr: getErr})

	_, err := store.GetAsset(t.Context(), "owner/repo", "img.png")
	require.Error(t, err)
	assert.ErrorContains(t, err, "failed to get asset")
}

func TestStore_GetAsset_ReadBodyError(t *testing.T) {
	store := newStoreWithClient(&failingS3Client{
		getBody: io.NopCloser(&errorReader{}),
	})

	_, err := store.GetAsset(t.Context(), "owner/repo", "img.png")
	require.Error(t, err)
	assert.ErrorContains(t, err, "failed to read asset body")
}

func TestStore_DeleteAsset_S3Error(t *testing.T) {
	deleteErr := errors.New("delete failed")
	store := newStoreWithClient(&failingS3Client{deleteErr: deleteErr})

	err := store.DeleteAsset(t.Context(), "owner/repo", "img.png")
	require.Error(t, err)
	assert.ErrorContains(t, err, "failed to delete asset")
}

func TestStore_ListAssets_PaginatorError(t *testing.T) {
	listErr := errors.New("list failed")
	store := newStoreWithClient(&failingS3Client{listErr: listErr})

	_, err := store.ListAssets(t.Context(), "owner/repo")
	require.Error(t, err)
	assert.ErrorContains(t, err, "failed to list assets")
}

func TestStore_UpdateRepoMeta_PutObjectError(t *testing.T) {
	putErr := errors.New("put failed")
	store := newStoreWithClient(&failingS3Client{putErr: putErr})

	err := store.updateRepoMeta(t.Context(), "owner/repo", time.Now())
	require.Error(t, err)
	assert.ErrorContains(t, err, "failed to upload repo metadata")
}

func TestStore_ReadRepoMeta_S3Error(t *testing.T) {
	getErr := errors.New("get failed")
	store := newStoreWithClient(&failingS3Client{getErr: getErr})

	_, err := store.readRepoMeta(t.Context(), "owner/repo")
	require.Error(t, err)
	assert.ErrorContains(t, err, "failed to get repo metadata")
}

func TestStore_ReadRepoMeta_InvalidJSON(t *testing.T) {
	store := newStoreWithClient(&failingS3Client{
		getBody: io.NopCloser(bytes.NewReader([]byte("not-json"))),
	})

	_, err := store.readRepoMeta(t.Context(), "owner/repo")
	require.Error(t, err)
	assert.ErrorContains(t, err, "failed to decode repo metadata")
}

func TestStore_CountDocs_PaginatorError(t *testing.T) {
	listErr := errors.New("list failed")
	store := newStoreWithClient(&failingS3Client{listErr: listErr})

	count, err := store.countDocs(t.Context(), "owner/repo")
	require.Error(t, err)
	assert.ErrorContains(t, err, "failed to count documents")
	assert.Equal(t, 0, count)
}

func TestStore_ListRepos_InvalidRepoNesting(t *testing.T) {
	// Populate the store with an object that produces a prefix with an unexpected
	// number of slashes — exercises the strings.Count guard in ListRepos.
	base := newTestStore(t)

	// Put an object three levels deep so the repo prefix ends up as "a/b/c/"
	// which has 2 slashes and should be skipped.
	_, err := base.client.PutObject(t.Context(), &s3.PutObjectInput{
		Bucket: aws.String(base.bucket),
		Key:    aws.String("a/b/c/docs/readme.md"),
		Body:   strings.NewReader("content"),
	})
	require.NoError(t, err)

	repos, err := base.ListRepos(t.Context())
	require.NoError(t, err)
	assert.Empty(t, repos)
}

func TestStore_ListRepos_OwnerPaginatorError(t *testing.T) {
	// ListObjectsV2 fails on the very first (owner-level) call.
	listErr := errors.New("list owners failed")
	store := newStoreWithClient(&failingS3Client{listErr: listErr})

	_, err := store.ListRepos(t.Context())
	require.Error(t, err)
	assert.ErrorContains(t, err, "failed to list owners")
}

func TestStore_ListRepos_RepoPaginatorError(t *testing.T) {
	// The owner-level listing succeeds and returns one prefix; the second-level
	// (repo) listing then fails.
	base := newTestStore(t)

	// Save a doc so an owner prefix exists.
	doc := core.Document{
		Repo:    "owner/repo",
		Path:    "readme.md",
		Title:   "README",
		Content: "# README",
	}

	require.NoError(t, base.Save(t.Context(), doc))

	wrapped := &repoListFailClient{inner: base.client}
	store := &Store{client: wrapped, bucket: base.bucket}

	_, err := store.ListRepos(t.Context())
	require.Error(t, err)
	assert.ErrorContains(t, err, "failed to list repos for owner")
}

// repoListFailClient fails ListObjectsV2 only on the second call (repo-level).
type repoListFailClient struct {
	inner     s3Client
	callCount int
}

func (r *repoListFailClient) PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	return r.inner.PutObject(ctx, params, optFns...)
}

func (r *repoListFailClient) GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	return r.inner.GetObject(ctx, params, optFns...)
}

func (r *repoListFailClient) DeleteObject(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
	return r.inner.DeleteObject(ctx, params, optFns...)
}

func (r *repoListFailClient) ListObjectsV2(_ context.Context, params *s3.ListObjectsV2Input, _ ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
	r.callCount++
	if r.callCount > 1 {
		return nil, errors.New("list repos failed")
	}
	// First call (owner-level): delegate to inner.
	return r.inner.ListObjectsV2(context.Background(), params)
}

func (r *repoListFailClient) HeadObject(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
	return r.inner.HeadObject(ctx, params, optFns...)
}

func TestNew_WithEndpointAndPathStyle(t *testing.T) {
	// Exercise the Endpoint and ForcePathStyle branches inside New().
	// We use the gofakes3 test server so no real AWS credentials are needed.
	backend := s3mem.New()
	require.NoError(t, backend.CreateBucket(testBucket))

	fake := gofakes3.New(backend)
	srv := httptest.NewServer(fake.Server())

	t.Cleanup(srv.Close)

	cfg := Config{
		Bucket:         testBucket,
		Region:         testRegion,
		Endpoint:       srv.URL,
		ForcePathStyle: true,
	}

	// New() uses the default AWS credential chain. Set dummy env vars so the
	// chain resolves immediately without hitting real AWS endpoints.
	t.Setenv("AWS_ACCESS_KEY_ID", "test")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "test")

	store, err := New(t.Context(), cfg)
	require.NoError(t, err)
	assert.NotNil(t, store)
}

// ---------------------------------------------------------------------------
// json import sentinel — ensures the json package stays used.
// ---------------------------------------------------------------------------

var _ = json.Marshal

func TestStore_List_SkipsExactPrefixKey(t *testing.T) {
	// Place an object whose key is exactly the docs/ prefix (i.e. relPath == "").
	// List should skip it rather than appending an empty-path entry.
	store := newTestStore(t)

	prefixKey := "owner/repo/" + docsPrefix // ends with "docs/"

	_, err := store.client.PutObject(t.Context(), &s3.PutObjectInput{
		Bucket: aws.String(store.bucket),
		Key:    aws.String(prefixKey),
		Body:   strings.NewReader(""),
	})
	require.NoError(t, err)

	list, err := store.List(t.Context(), "owner/repo")
	require.NoError(t, err)
	assert.Empty(t, list)
}

func TestStore_ListAssets_SkipsExactPrefixKey(t *testing.T) {
	// Same as above but for the assets/ prefix.
	store := newTestStore(t)

	prefixKey := "owner/repo/" + assetsPrefix // ends with "assets/"

	_, err := store.client.PutObject(t.Context(), &s3.PutObjectInput{
		Bucket: aws.String(store.bucket),
		Key:    aws.String(prefixKey),
		Body:   strings.NewReader(""),
	})
	require.NoError(t, err)

	paths, err := store.ListAssets(t.Context(), "owner/repo")
	require.NoError(t, err)
	assert.Empty(t, paths)
}

func TestStore_ListRepos_SkipsEmptyOwnerPrefix(t *testing.T) {
	// When the owner-level listing returns a CommonPrefix with an empty string,
	// ListRepos should skip it (the continue branch at owner == "").
	store := newStoreWithClient(&emptyPrefixListClient{})

	repos, err := store.ListRepos(t.Context())
	require.NoError(t, err)
	assert.Empty(t, repos)
}

func TestStore_ListRepos_SkipsEmptyRepoPrefix(t *testing.T) {
	// When the repo-level listing returns a CommonPrefix with an empty string,
	// ListRepos should skip it (the continue branch at prefix == "").
	store := newStoreWithClient(&emptyRepoPrefixListClient{})

	repos, err := store.ListRepos(t.Context())
	require.NoError(t, err)
	assert.Empty(t, repos)
}

// emptyPrefixListClient returns one CommonPrefix with an empty Prefix string
// at the owner level so that the owner == "" guard is exercised.
type emptyPrefixListClient struct{}

func (e *emptyPrefixListClient) PutObject(_ context.Context, _ *s3.PutObjectInput, _ ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	return &s3.PutObjectOutput{}, nil
}

func (e *emptyPrefixListClient) GetObject(_ context.Context, _ *s3.GetObjectInput, _ ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	return nil, errors.New("not called")
}

func (e *emptyPrefixListClient) DeleteObject(_ context.Context, _ *s3.DeleteObjectInput, _ ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
	return &s3.DeleteObjectOutput{}, nil
}

func (e *emptyPrefixListClient) ListObjectsV2(_ context.Context, _ *s3.ListObjectsV2Input, _ ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
	return &s3.ListObjectsV2Output{
		CommonPrefixes: []types.CommonPrefix{
			{Prefix: aws.String("")}, // empty owner prefix → should be skipped
		},
	}, nil
}

func (e *emptyPrefixListClient) HeadObject(_ context.Context, _ *s3.HeadObjectInput, _ ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
	return nil, errors.New("not called")
}

// emptyRepoPrefixListClient returns a valid owner prefix on the first call and
// an empty repo prefix on the second call, exercising the prefix == "" guard.
type emptyRepoPrefixListClient struct {
	callCount int
}

func (e *emptyRepoPrefixListClient) PutObject(_ context.Context, _ *s3.PutObjectInput, _ ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	return &s3.PutObjectOutput{}, nil
}

func (e *emptyRepoPrefixListClient) GetObject(_ context.Context, _ *s3.GetObjectInput, _ ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	return nil, errors.New("not called")
}

func (e *emptyRepoPrefixListClient) DeleteObject(_ context.Context, _ *s3.DeleteObjectInput, _ ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
	return &s3.DeleteObjectOutput{}, nil
}

func (e *emptyRepoPrefixListClient) ListObjectsV2(_ context.Context, _ *s3.ListObjectsV2Input, _ ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
	e.callCount++
	switch e.callCount {
	case 1:
		// Owner-level: return one valid owner prefix.
		return &s3.ListObjectsV2Output{
			CommonPrefixes: []types.CommonPrefix{
				{Prefix: aws.String("owner/")},
			},
		}, nil
	default:
		// Repo-level: return one empty repo prefix → should be skipped.
		return &s3.ListObjectsV2Output{
			CommonPrefixes: []types.CommonPrefix{
				{Prefix: aws.String("")},
			},
		}, nil
	}
}

func (e *emptyRepoPrefixListClient) HeadObject(_ context.Context, _ *s3.HeadObjectInput, _ ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
	return nil, errors.New("not called")
}
