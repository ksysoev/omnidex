// Package docstore provides document storage backed by the filesystem.
package docstore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/ksysoev/omnidex/pkg/core"
)

const (
	metaFileName = "meta.json"
	docsDir      = "docs"
	assetsDir    = "assets"
)

// ErrNotFound is returned when a requested document or asset does not exist.
var ErrNotFound = errors.New("not found")

// ErrInvalidPath is returned when a document path attempts directory traversal.
var ErrInvalidPath = errors.New("invalid path: directory traversal not allowed")

// repoMeta holds metadata about an indexed repository.
type repoMeta struct {
	LastUpdated time.Time `json:"last_updated"`
	Name        string    `json:"name"`
}

// docMeta holds metadata about a single document stored on disk.
type docMeta struct {
	UpdatedAt   time.Time `json:"updated_at"`
	Title       string    `json:"title"`
	CommitSHA   string    `json:"commit_sha"`
	ContentType string    `json:"content_type,omitempty"` // defaults to "markdown" when empty
}

// Store implements filesystem-based document storage.
// Documents are stored in a directory tree: {basePath}/{owner}/{repo}/docs/{path}.
type Store struct {
	basePath string
	mu       sync.RWMutex
}

// New creates a new filesystem-based document store rooted at basePath.
func New(basePath string) (*Store, error) {
	absBase, err := filepath.Abs(basePath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve storage path: %w", err)
	}

	if err := os.MkdirAll(absBase, 0o750); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}

	return &Store{basePath: absBase}, nil
}

// validatePath ensures the given segments, when joined to the base path,
// do not escape the base directory via path traversal.
func (s *Store) validatePath(segments ...string) error {
	joined := filepath.Join(append([]string{s.basePath}, segments...)...)

	resolved, err := filepath.Abs(joined)
	if err != nil {
		return fmt.Errorf("%w: %s", ErrInvalidPath, err)
	}

	if !strings.HasPrefix(resolved, s.basePath+string(filepath.Separator)) && resolved != s.basePath {
		return fmt.Errorf("%w: resolved path escapes base directory", ErrInvalidPath)
	}

	return nil
}

// Save persists a document to the filesystem.
func (s *Store) Save(_ context.Context, doc core.Document) error { //nolint:gocritic // Document is passed by value for immutability
	if err := s.validatePath(doc.Repo, docsDir, doc.Path); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	repoDir := filepath.Join(s.basePath, doc.Repo)
	docDir := filepath.Join(repoDir, docsDir, filepath.Dir(doc.Path))

	if err := os.MkdirAll(docDir, 0o750); err != nil {
		return fmt.Errorf("failed to create document directory: %w", err)
	}

	// Write the markdown content.
	docPath := filepath.Join(repoDir, docsDir, doc.Path)

	if err := os.WriteFile(docPath, []byte(doc.Content), 0o600); err != nil {
		return fmt.Errorf("failed to write document: %w", err)
	}

	// Write document metadata alongside the content.
	meta := docMeta{
		Title:       doc.Title,
		CommitSHA:   doc.CommitSHA,
		UpdatedAt:   doc.UpdatedAt,
		ContentType: string(doc.ContentType),
	}

	metaPath := docPath + ".meta.json"

	metaData, err := json.Marshal(meta)
	if err != nil {
		return fmt.Errorf("failed to marshal document metadata: %w", err)
	}

	if err := os.WriteFile(metaPath, metaData, 0o600); err != nil {
		return fmt.Errorf("failed to write document metadata: %w", err)
	}

	// Update repo metadata.
	return s.updateRepoMeta(repoDir, doc.Repo, doc.UpdatedAt)
}

// Get retrieves a document by its repository and path.
func (s *Store) Get(_ context.Context, repo, path string) (core.Document, error) {
	if err := s.validatePath(repo, docsDir, path); err != nil {
		return core.Document{}, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	docPath := filepath.Join(s.basePath, repo, docsDir, path)

	content, err := os.ReadFile(docPath)
	if err != nil {
		if os.IsNotExist(err) {
			return core.Document{}, fmt.Errorf("%w: %s/%s", ErrNotFound, repo, path)
		}

		return core.Document{}, fmt.Errorf("failed to read document: %w", err)
	}

	meta, err := s.readDocMeta(docPath)
	if err != nil {
		return core.Document{}, err
	}

	ct := core.ContentType(meta.ContentType)
	if ct == "" {
		ct = core.ContentTypeMarkdown
	}

	return core.Document{
		ID:          repo + "/" + path,
		Repo:        repo,
		Path:        path,
		Title:       meta.Title,
		Content:     string(content),
		CommitSHA:   meta.CommitSHA,
		UpdatedAt:   meta.UpdatedAt,
		ContentType: ct,
	}, nil
}

// Delete removes a document from the filesystem.
func (s *Store) Delete(_ context.Context, repo, path string) error {
	if err := s.validatePath(repo, docsDir, path); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	docPath := filepath.Join(s.basePath, repo, docsDir, path)

	if err := os.Remove(docPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete document: %w", err)
	}

	// Also remove metadata file.
	metaPath := docPath + ".meta.json"
	_ = os.Remove(metaPath)

	// Clean up empty directories.
	s.cleanEmptyDirs(filepath.Dir(docPath), filepath.Join(s.basePath, repo, docsDir))

	return nil
}

// List returns metadata for all documents in a repository.
func (s *Store) List(_ context.Context, repo string) ([]core.DocumentMeta, error) {
	if err := s.validatePath(repo); err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	repoDocsDir := filepath.Join(s.basePath, repo, docsDir)

	var docs []core.DocumentMeta

	err := filepath.Walk(repoDocsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() || strings.HasSuffix(path, ".meta.json") {
			return nil
		}

		relPath, err := filepath.Rel(repoDocsDir, path)
		if err != nil {
			return fmt.Errorf("failed to compute relative path: %w", err)
		}

		meta, err := s.readDocMeta(path)
		if err != nil {
			// If no metadata file, use file info.
			meta = &docMeta{
				Title:     relPath,
				UpdatedAt: info.ModTime(),
			}
		}

		ct := core.ContentType(meta.ContentType)
		if ct == "" {
			ct = core.ContentTypeMarkdown
		}

		docs = append(docs, core.DocumentMeta{
			ID:          repo + "/" + relPath,
			Repo:        repo,
			Path:        relPath,
			Title:       meta.Title,
			UpdatedAt:   meta.UpdatedAt,
			ContentType: ct,
		})

		return nil
	})
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}

		return nil, fmt.Errorf("failed to list documents: %w", err)
	}

	sort.Slice(docs, func(i, j int) bool {
		return docs[i].Path < docs[j].Path
	})

	return docs, nil
}

// ListRepos returns metadata for all indexed repositories.
func (s *Store) ListRepos(_ context.Context) ([]core.RepoInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var repos []core.RepoInfo

	owners, err := os.ReadDir(s.basePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read storage directory: %w", err)
	}

	for _, owner := range owners {
		if !owner.IsDir() {
			continue
		}

		repoEntries, err := os.ReadDir(filepath.Join(s.basePath, owner.Name()))
		if err != nil {
			continue
		}

		for _, repoEntry := range repoEntries {
			if !repoEntry.IsDir() {
				continue
			}

			repoName := owner.Name() + "/" + repoEntry.Name()
			repoDir := filepath.Join(s.basePath, repoName)

			meta, err := s.readRepoMeta(repoDir)
			if err != nil {
				continue
			}

			docCount := s.countDocs(filepath.Join(repoDir, docsDir))

			repos = append(repos, core.RepoInfo{
				Name:        meta.Name,
				DocCount:    docCount,
				LastUpdated: meta.LastUpdated,
			})
		}
	}

	sort.Slice(repos, func(i, j int) bool {
		return repos[i].Name < repos[j].Name
	})

	return repos, nil
}

func (s *Store) updateRepoMeta(repoDir, repoName string, updatedAt time.Time) error {
	meta := repoMeta{
		Name:        repoName,
		LastUpdated: updatedAt,
	}

	data, err := json.Marshal(meta)
	if err != nil {
		return fmt.Errorf("failed to marshal repo metadata: %w", err)
	}

	metaPath := filepath.Join(repoDir, metaFileName)

	if err := os.WriteFile(metaPath, data, 0o600); err != nil {
		return fmt.Errorf("failed to write repo metadata: %w", err)
	}

	return nil
}

func (s *Store) readRepoMeta(repoDir string) (*repoMeta, error) {
	data, err := os.ReadFile(filepath.Join(repoDir, metaFileName))
	if err != nil {
		return nil, fmt.Errorf("failed to read repo metadata: %w", err)
	}

	var meta repoMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, fmt.Errorf("failed to unmarshal repo metadata: %w", err)
	}

	return &meta, nil
}

func (s *Store) readDocMeta(docPath string) (*docMeta, error) {
	data, err := os.ReadFile(docPath + ".meta.json")
	if err != nil {
		return nil, fmt.Errorf("failed to read document metadata: %w", err)
	}

	var meta docMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, fmt.Errorf("failed to unmarshal document metadata: %w", err)
	}

	return &meta, nil
}

func (s *Store) countDocs(dir string) int {
	count := 0

	_ = filepath.Walk(dir, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && !strings.HasSuffix(info.Name(), ".meta.json") {
			count++
		}

		return nil
	})

	return count
}

func (s *Store) cleanEmptyDirs(dir, stopAt string) {
	for dir != stopAt {
		entries, err := os.ReadDir(dir)
		if err != nil || len(entries) > 0 {
			return
		}

		_ = os.Remove(dir)
		dir = filepath.Dir(dir)
	}
}

// validateAssetRelPath rejects asset path values that could escape the assets
// subdirectory. validatePath (which checks containment within basePath) is
// insufficient on its own because a path like "../docs/readme.md" still resolves
// to a location under basePath, allowing the asset API to access doc files.
//
// Rules enforced here (before the absolute-path check in validatePath):
//   - path must not be empty or "."
//   - path must not be absolute
//   - cleaned path must not equal ".." or start with "../" (OS-separator aware)
func validateAssetRelPath(assetPath string) error {
	if assetPath == "" {
		return fmt.Errorf("%w: asset path must not be empty", ErrInvalidPath)
	}

	if filepath.IsAbs(assetPath) {
		return fmt.Errorf("%w: asset path must not be absolute", ErrInvalidPath)
	}

	clean := filepath.Clean(assetPath)

	if clean == "." || clean == ".." {
		return fmt.Errorf("%w: asset path resolves to directory root", ErrInvalidPath)
	}

	if strings.HasPrefix(clean, ".."+string(os.PathSeparator)) {
		return fmt.Errorf("%w: asset path attempts directory traversal", ErrInvalidPath)
	}

	return nil
}

// SaveAsset writes a binary asset to {basePath}/{repo}/assets/{path}.
// No metadata sidecar is created; MIME type is detected from the file extension at serve time.
func (s *Store) SaveAsset(_ context.Context, repo, path string, data []byte) error {
	if err := validateAssetRelPath(path); err != nil {
		return err
	}

	if err := s.validatePath(repo, assetsDir, path); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	assetDir := filepath.Join(s.basePath, repo, assetsDir, filepath.Dir(path))

	if err := os.MkdirAll(assetDir, 0o750); err != nil {
		return fmt.Errorf("failed to create asset directory: %w", err)
	}

	assetPath := filepath.Join(s.basePath, repo, assetsDir, path)

	if err := os.WriteFile(assetPath, data, 0o600); err != nil {
		return fmt.Errorf("failed to write asset: %w", err)
	}

	return nil
}

// GetAsset reads a binary asset from the store by its repository and path.
func (s *Store) GetAsset(_ context.Context, repo, path string) ([]byte, error) {
	if err := validateAssetRelPath(path); err != nil {
		return nil, err
	}

	if err := s.validatePath(repo, assetsDir, path); err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	assetPath := filepath.Join(s.basePath, repo, assetsDir, path)

	data, err := os.ReadFile(assetPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: asset %s/%s", ErrNotFound, repo, path)
		}

		return nil, fmt.Errorf("failed to read asset: %w", err)
	}

	return data, nil
}

// DeleteAsset removes a binary asset from the store and cleans up empty parent directories.
func (s *Store) DeleteAsset(_ context.Context, repo, path string) error {
	if err := validateAssetRelPath(path); err != nil {
		return err
	}

	if err := s.validatePath(repo, assetsDir, path); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	assetPath := filepath.Join(s.basePath, repo, assetsDir, path)

	if err := os.Remove(assetPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete asset: %w", err)
	}

	s.cleanEmptyDirs(filepath.Dir(assetPath), filepath.Join(s.basePath, repo, assetsDir))

	return nil
}

// ListAssets returns all asset paths for a repository.
func (s *Store) ListAssets(_ context.Context, repo string) ([]string, error) {
	if err := s.validatePath(repo); err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	repoAssetsDir := filepath.Join(s.basePath, repo, assetsDir)

	var paths []string

	err := filepath.Walk(repoAssetsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(repoAssetsDir, path)
		if err != nil {
			return fmt.Errorf("failed to compute relative path: %w", err)
		}

		paths = append(paths, filepath.ToSlash(relPath))

		return nil
	})
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}

		return nil, fmt.Errorf("failed to list assets: %w", err)
	}

	sort.Strings(paths)

	return paths, nil
}
