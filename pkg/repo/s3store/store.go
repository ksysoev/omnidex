// Package s3store provides document storage backed by AWS S3 or any S3-compatible
// service (e.g. MinIO). It implements the same interface as pkg/repo/docstore so
// that the two backends are interchangeable at startup via configuration.
//
// Object layout inside the configured bucket:
//
//	{owner}/{repo}/
//	  meta.json                         – repo-level metadata (JSON)
//	  docs/{relative/path/to/doc}       – document content; per-document metadata
//	                                       stored as x-amz-meta-* object headers
//	  assets/{relative/path/to/file}    – binary asset body (no metadata headers)
//
// Document metadata fields stored as S3 custom object metadata headers:
//
//	x-amz-meta-title        – human-readable document title
//	x-amz-meta-updated-at   – RFC3339 timestamp of last update
//	x-amz-meta-commit-sha   – VCS commit SHA at ingest time
//	x-amz-meta-content-type – content type string (e.g. "markdown", "openapi")
//
// AWS credentials are never stored in configuration; they are sourced via the
// standard AWS SDK credential chain (environment variables →
// ~/.aws/credentials → IAM role).
package s3store

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	stdpath "path"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"

	"github.com/ksysoev/omnidex/pkg/core"
)

const (
	metaFileName = "meta.json"
	docsPrefix   = "docs/"
	assetsPrefix = "assets/"

	// S3 custom metadata header keys (lowercased; the SDK adds the x-amz-meta- prefix).
	metaKeyTitle       = "title"
	metaKeyUpdatedAt   = "updated-at"
	metaKeyCommitSHA   = "commit-sha"
	metaKeyContentType = "content-type"
)

// Config holds configuration for the S3-backed document store.
// AWS credentials are not stored here; they are sourced via the standard
// AWS credential chain (environment variables, ~/.aws/credentials, IAM role).
type Config struct {
	Endpoint       string `mapstructure:"endpoint"` // optional; for S3-compatible APIs such as MinIO
	Bucket         string `mapstructure:"bucket"`
	Region         string `mapstructure:"region"`
	ForcePathStyle bool   `mapstructure:"force_path_style"` // enable for MinIO and other path-style APIs
}

// s3Client defines the subset of the AWS S3 API used by Store.
// Using an interface makes the store testable without a real S3 connection.
type s3Client interface {
	PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
	GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
	DeleteObject(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error)
	ListObjectsV2(ctx context.Context, params *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error)
	HeadObject(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error)
}

// repoMeta holds the repo-level metadata persisted as meta.json.
type repoMeta struct {
	LastUpdated time.Time `json:"last_updated"`
	Name        string    `json:"name"`
}

// Store implements S3-backed document storage.
type Store struct {
	client s3Client
	bucket string
}

// New creates a new S3-backed Store using the provided configuration.
// Credentials are loaded via the standard AWS SDK credential chain.
func New(ctx context.Context, cfg Config) (*Store, error) {
	optFns := []func(*awsconfig.LoadOptions) error{
		awsconfig.WithRegion(cfg.Region),
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, optFns...)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	clientOptFns := []func(*s3.Options){}

	if cfg.Endpoint != "" {
		clientOptFns = append(clientOptFns, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
		})
	}

	if cfg.ForcePathStyle {
		clientOptFns = append(clientOptFns, func(o *s3.Options) {
			o.UsePathStyle = true
		})
	}

	client := s3.NewFromConfig(awsCfg, clientOptFns...)

	return &Store{client: client, bucket: cfg.Bucket}, nil
}

// newWithStaticCreds creates a Store using explicit static credentials.
// Intended for testing against MinIO or other S3-compatible services where
// the standard credential chain is not available.
func newWithStaticCreds(ctx context.Context, cfg Config, accessKey, secretKey string) (*Store, error) {
	optFns := []func(*awsconfig.LoadOptions) error{
		awsconfig.WithRegion(cfg.Region),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, optFns...)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	clientOptFns := []func(*s3.Options){}

	if cfg.Endpoint != "" {
		clientOptFns = append(clientOptFns, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
		})
	}

	if cfg.ForcePathStyle {
		clientOptFns = append(clientOptFns, func(o *s3.Options) {
			o.UsePathStyle = true
		})
	}

	client := s3.NewFromConfig(awsCfg, clientOptFns...)

	return &Store{client: client, bucket: cfg.Bucket}, nil
}

// validateRelPath rejects relative paths that are empty, absolute, or that
// escape the virtual prefix via directory traversal (e.g. "../docs/x").
// This mirrors the validation performed by the local docstore backend so that
// both backends present the same error semantics to callers.
func validateRelPath(relPath string) error {
	if relPath == "" {
		return fmt.Errorf("%w: path must not be empty", core.ErrInvalidPath)
	}

	if stdpath.IsAbs(relPath) {
		return fmt.Errorf("%w: path must not be absolute", core.ErrInvalidPath)
	}

	clean := stdpath.Clean(relPath)

	if clean == "." || clean == ".." {
		return fmt.Errorf("%w: path resolves to directory root", core.ErrInvalidPath)
	}

	if strings.HasPrefix(clean, "../") {
		return fmt.Errorf("%w: path attempts directory traversal", core.ErrInvalidPath)
	}

	return nil
}

// docKey returns the S3 object key for a document.
func docKey(repo, path string) string {
	return repo + "/" + docsPrefix + path
}

// assetKey returns the S3 object key for an asset.
func assetKey(repo, path string) string {
	return repo + "/" + assetsPrefix + path
}

// repoMetaKey returns the S3 object key for repo-level metadata.
func repoMetaKey(repo string) string {
	return repo + "/" + metaFileName
}

// parseUpdatedAt parses the updated-at metadata string. It accepts both
// RFC3339Nano (current format) and RFC3339 (legacy format written before the
// precision upgrade). When parsing fails or the value is absent, it falls back
// to fallback (typically the S3 LastModified timestamp).
func parseUpdatedAt(value string, fallback *time.Time) time.Time {
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339} {
		if t, err := time.Parse(layout, value); err == nil {
			return t
		}
	}

	if fallback != nil {
		return *fallback
	}

	return time.Time{}
}

// isNotFound returns true when the AWS SDK error represents a missing object (404).
func isNotFound(err error) bool {
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		code := apiErr.ErrorCode()

		return code == "NoSuchKey" || code == "NotFound" || code == "404"
	}

	var nsk *types.NoSuchKey

	return errors.As(err, &nsk)
}

// Save persists a document to S3. The content body is uploaded as the object
// body; per-document metadata is stored as x-amz-meta-* object headers.
// Repo-level metadata (meta.json) is updated after the document upload.
func (s *Store) Save(ctx context.Context, doc core.Document) error { //nolint:gocritic // Document is passed by value for immutability
	if err := validateRelPath(doc.Repo); err != nil {
		return err
	}

	if err := validateRelPath(doc.Path); err != nil {
		return err
	}

	metadata := map[string]string{
		metaKeyTitle:       doc.Title,
		metaKeyUpdatedAt:   doc.UpdatedAt.UTC().Format(time.RFC3339Nano),
		metaKeyCommitSHA:   doc.CommitSHA,
		metaKeyContentType: string(doc.ContentType),
	}

	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:   aws.String(s.bucket),
		Key:      aws.String(docKey(doc.Repo, doc.Path)),
		Body:     strings.NewReader(doc.Content),
		Metadata: metadata,
	})
	if err != nil {
		return fmt.Errorf("failed to upload document: %w", err)
	}

	if err := s.updateRepoMeta(ctx, doc.Repo, doc.UpdatedAt); err != nil {
		return fmt.Errorf("failed to update repo metadata: %w", err)
	}

	return nil
}

// Get retrieves a document from S3 by its repository and path.
// The document content is read from the object body; metadata is read from
// the x-amz-meta-* response headers.
func (s *Store) Get(ctx context.Context, repo, path string) (core.Document, error) {
	if err := validateRelPath(repo); err != nil {
		return core.Document{}, err
	}

	if err := validateRelPath(path); err != nil {
		return core.Document{}, err
	}

	resp, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(docKey(repo, path)),
	})
	if err != nil {
		if isNotFound(err) {
			return core.Document{}, fmt.Errorf("%w: %s/%s", core.ErrNotFound, repo, path)
		}

		return core.Document{}, fmt.Errorf("failed to get document: %w", err)
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return core.Document{}, fmt.Errorf("failed to read document body: %w", err)
	}

	meta := resp.Metadata

	updatedAt := parseUpdatedAt(meta[metaKeyUpdatedAt], resp.LastModified)

	ct := core.ContentType(meta[metaKeyContentType])
	if ct == "" {
		ct = core.ContentTypeMarkdown
	}

	return core.Document{
		ID:          repo + "/" + path,
		Repo:        repo,
		Path:        path,
		Title:       meta[metaKeyTitle],
		Content:     string(body),
		CommitSHA:   meta[metaKeyCommitSHA],
		UpdatedAt:   updatedAt,
		ContentType: ct,
	}, nil
}

// Delete removes a document from S3. Missing objects are silently ignored
// (idempotent behaviour matching the local docstore).
func (s *Store) Delete(ctx context.Context, repo, path string) error {
	if err := validateRelPath(repo); err != nil {
		return err
	}

	if err := validateRelPath(path); err != nil {
		return err
	}

	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(docKey(repo, path)),
	})
	if err != nil && !isNotFound(err) {
		return fmt.Errorf("failed to delete document: %w", err)
	}

	return nil
}

// List returns metadata for all documents stored under the given repository prefix.
// It uses ListObjectsV2 to enumerate the docs/ prefix then fetches metadata for
// each object via HeadObject.
func (s *Store) List(ctx context.Context, repo string) ([]core.DocumentMeta, error) {
	if err := validateRelPath(repo); err != nil {
		return nil, err
	}

	prefix := repo + "/" + docsPrefix

	var docs []core.DocumentMeta

	paginator := s3.NewListObjectsV2Paginator(s.client, &s3.ListObjectsV2Input{
		Bucket: aws.String(s.bucket),
		Prefix: aws.String(prefix),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list documents: %w", err)
		}

		for _, obj := range page.Contents {
			key := aws.ToString(obj.Key)
			relPath := strings.TrimPrefix(key, prefix)

			if relPath == "" {
				continue
			}

			head, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
				Bucket: aws.String(s.bucket),
				Key:    aws.String(key),
			})
			if err != nil {
				slog.WarnContext(ctx, "s3store: failed to head document object; skipping", "key", key, "err", err)
				continue
			}

			meta := head.Metadata

			updatedAt := parseUpdatedAt(meta[metaKeyUpdatedAt], head.LastModified)

			ct := core.ContentType(meta[metaKeyContentType])
			if ct == "" {
				ct = core.ContentTypeMarkdown
			}

			title := meta[metaKeyTitle]
			if title == "" {
				title = relPath
			}

			docs = append(docs, core.DocumentMeta{
				ID:          repo + "/" + relPath,
				Repo:        repo,
				Path:        relPath,
				Title:       title,
				UpdatedAt:   updatedAt,
				ContentType: ct,
			})
		}
	}

	sort.Slice(docs, func(i, j int) bool {
		return docs[i].Path < docs[j].Path
	})

	return docs, nil
}

// ListRepos returns metadata for all repositories discovered in the bucket.
// It uses two-level delimiter-based listing (owner/ then owner/repo/) to avoid
// scanning every object and scales proportionally to the number of repos rather
// than the total number of objects in the bucket.
func (s *Store) ListRepos(ctx context.Context) ([]core.RepoInfo, error) {
	var repos []core.RepoInfo

	// First level: enumerate {owner}/ common prefixes.
	ownerPaginator := s3.NewListObjectsV2Paginator(s.client, &s3.ListObjectsV2Input{
		Bucket:    aws.String(s.bucket),
		Delimiter: aws.String("/"),
	})

	for ownerPaginator.HasMorePages() {
		ownerPage, err := ownerPaginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list owners: %w", err)
		}

		for _, ownerPrefix := range ownerPage.CommonPrefixes {
			owner := aws.ToString(ownerPrefix.Prefix) // e.g. "owner/"
			if owner == "" {
				continue
			}

			// Second level: enumerate {owner}/{repo}/ common prefixes.
			repoPaginator := s3.NewListObjectsV2Paginator(s.client, &s3.ListObjectsV2Input{
				Bucket:    aws.String(s.bucket),
				Prefix:    aws.String(owner),
				Delimiter: aws.String("/"),
			})

			for repoPaginator.HasMorePages() {
				repoPage, err := repoPaginator.NextPage(ctx)
				if err != nil {
					return nil, fmt.Errorf("failed to list repos for owner %q: %w", owner, err)
				}

				for _, repoPrefix := range repoPage.CommonPrefixes {
					prefix := aws.ToString(repoPrefix.Prefix) // e.g. "owner/repo/"
					if prefix == "" {
						continue
					}

					// Strip trailing slash to get "owner/repo".
					repoName := strings.TrimSuffix(prefix, "/")

					// Validate exactly one slash — guards against unexpected nesting.
					if strings.Count(repoName, "/") != 1 {
						continue
					}

					meta, err := s.readRepoMeta(ctx, repoName)
					if err != nil {
						slog.WarnContext(ctx, "s3store: failed to read repo meta; skipping", "repo", repoName, "err", err)
						continue
					}

					docCount, err := s.countDocs(ctx, repoName)
					if err != nil {
						slog.WarnContext(ctx, "s3store: failed to count docs; using 0", "repo", repoName, "err", err)

						docCount = 0
					}

					repos = append(repos, core.RepoInfo{
						Name:        meta.Name,
						DocCount:    docCount,
						LastUpdated: meta.LastUpdated,
					})
				}
			}
		}
	}

	sort.Slice(repos, func(i, j int) bool {
		return repos[i].Name < repos[j].Name
	})

	return repos, nil
}

// SaveAsset writes a binary asset to S3.
func (s *Store) SaveAsset(ctx context.Context, repo, path string, data []byte) error {
	if err := validateRelPath(repo); err != nil {
		return err
	}

	if err := validateRelPath(path); err != nil {
		return err
	}

	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(assetKey(repo, path)),
		Body:   bytes.NewReader(data),
	})
	if err != nil {
		return fmt.Errorf("failed to upload asset: %w", err)
	}

	return nil
}

// GetAsset retrieves a binary asset from S3 by its repository and path.
func (s *Store) GetAsset(ctx context.Context, repo, path string) ([]byte, error) {
	if err := validateRelPath(repo); err != nil {
		return nil, err
	}

	if err := validateRelPath(path); err != nil {
		return nil, err
	}

	resp, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(assetKey(repo, path)),
	})
	if err != nil {
		if isNotFound(err) {
			return nil, fmt.Errorf("%w: asset %s/%s", core.ErrNotFound, repo, path)
		}

		return nil, fmt.Errorf("failed to get asset: %w", err)
	}

	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read asset body: %w", err)
	}

	return data, nil
}

// DeleteAsset removes a binary asset from S3. Missing objects are silently ignored.
func (s *Store) DeleteAsset(ctx context.Context, repo, path string) error {
	if err := validateRelPath(repo); err != nil {
		return err
	}

	if err := validateRelPath(path); err != nil {
		return err
	}

	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(assetKey(repo, path)),
	})
	if err != nil && !isNotFound(err) {
		return fmt.Errorf("failed to delete asset: %w", err)
	}

	return nil
}

// ListAssets returns all asset paths stored under the given repository prefix.
func (s *Store) ListAssets(ctx context.Context, repo string) ([]string, error) {
	if err := validateRelPath(repo); err != nil {
		return nil, err
	}

	prefix := repo + "/" + assetsPrefix

	var paths []string

	paginator := s3.NewListObjectsV2Paginator(s.client, &s3.ListObjectsV2Input{
		Bucket: aws.String(s.bucket),
		Prefix: aws.String(prefix),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list assets: %w", err)
		}

		for _, obj := range page.Contents {
			key := aws.ToString(obj.Key)
			relPath := strings.TrimPrefix(key, prefix)

			if relPath == "" {
				continue
			}

			paths = append(paths, relPath)
		}
	}

	sort.Strings(paths)

	return paths, nil
}

// updateRepoMeta writes the repo-level meta.json object to S3.
func (s *Store) updateRepoMeta(ctx context.Context, repo string, updatedAt time.Time) error {
	meta := repoMeta{
		Name:        repo,
		LastUpdated: updatedAt,
	}

	data, err := json.Marshal(meta)
	if err != nil {
		return fmt.Errorf("failed to marshal repo metadata: %w", err)
	}

	_, err = s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(repoMetaKey(repo)),
		Body:   bytes.NewReader(data),
	})
	if err != nil {
		return fmt.Errorf("failed to upload repo metadata: %w", err)
	}

	return nil
}

// readRepoMeta fetches and parses the repo-level meta.json object from S3.
func (s *Store) readRepoMeta(ctx context.Context, repo string) (*repoMeta, error) {
	resp, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(repoMetaKey(repo)),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get repo metadata: %w", err)
	}

	defer resp.Body.Close()

	var meta repoMeta
	if err := json.NewDecoder(resp.Body).Decode(&meta); err != nil {
		return nil, fmt.Errorf("failed to decode repo metadata: %w", err)
	}

	return &meta, nil
}

// countDocs returns the number of document objects stored under the repo docs/ prefix.
func (s *Store) countDocs(ctx context.Context, repo string) (int, error) {
	prefix := repo + "/" + docsPrefix
	count := 0

	paginator := s3.NewListObjectsV2Paginator(s.client, &s3.ListObjectsV2Input{
		Bucket: aws.String(s.bucket),
		Prefix: aws.String(prefix),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return count, fmt.Errorf("failed to count documents: %w", err)
		}

		count += len(page.Contents)
	}

	return count, nil
}
