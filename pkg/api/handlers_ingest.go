package api

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/ksysoev/omnidex/pkg/core"
)

// ingestDocs handles POST /api/v1/docs - batch document ingest from GitHub Actions.
func (a *API) ingestDocs(w http.ResponseWriter, r *http.Request) {
	var req core.IngestRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.ErrorContext(r.Context(), "Failed to decode ingest request", "error", err)
		http.Error(w, "invalid request body", http.StatusBadRequest)

		return
	}

	if req.Repo == "" {
		http.Error(w, "repo field is required", http.StatusBadRequest)
		return
	}

	if len(req.Documents) == 0 {
		http.Error(w, "documents field is required and must not be empty", http.StatusBadRequest)
		return
	}

	resp, err := a.svc.IngestDocuments(r.Context(), req)
	if err != nil {
		slog.ErrorContext(r.Context(), "Failed to ingest documents", "error", err)
		http.Error(w, "failed to process documents", http.StatusInternalServerError)

		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		slog.ErrorContext(r.Context(), "Failed to encode response", "error", err)
	}
}

// listRepos handles GET /api/v1/repos - list all indexed repositories.
func (a *API) listRepos(w http.ResponseWriter, r *http.Request) {
	repos, err := a.svc.ListRepos(r.Context())
	if err != nil {
		slog.ErrorContext(r.Context(), "Failed to list repos", "error", err)
		http.Error(w, "failed to list repositories", http.StatusInternalServerError)

		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(map[string]any{"repos": repos}); err != nil {
		slog.ErrorContext(r.Context(), "Failed to encode response", "error", err)
	}
}
