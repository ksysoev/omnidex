package api

import (
	"errors"
	"log/slog"
	"mime"
	"net/http"
	"path/filepath"

	"github.com/ksysoev/omnidex/pkg/repo/docstore"
)

// assetPage handles GET /assets/{owner}/{repo}/{path...} - serves a binary asset.
func (a *API) assetPage(w http.ResponseWriter, r *http.Request) {
	owner := r.PathValue("owner")
	repo := r.PathValue("repo")
	path := r.PathValue("path")

	if owner == "" || repo == "" || path == "" {
		http.NotFound(w, r)
		return
	}

	fullRepo := owner + "/" + repo

	data, err := a.svc.GetAsset(r.Context(), fullRepo, path)
	if err != nil {
		if errors.Is(err, docstore.ErrNotFound) {
			http.NotFound(w, r)
			return
		}

		if errors.Is(err, docstore.ErrInvalidPath) {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		slog.ErrorContext(r.Context(), "Failed to get asset", "error", err, "repo", fullRepo, "path", path)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)

		return
	}

	contentType := mime.TypeByExtension(filepath.Ext(path))
	if contentType == "" {
		contentType = http.DetectContentType(data)
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "public, max-age=86400")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Security-Policy", "sandbox")

	if _, err := w.Write(data); err != nil { //nolint:gosec // Binary asset data with explicit Content-Type; not user-controlled HTML
		slog.ErrorContext(r.Context(), "Failed to write asset response", "error", err)
	}
}
