package api

import (
	"log/slog"
	"net/http"
)

// healthCheck verifies the server is running and returns 200 OK.
func (a *API) healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)

	if _, err := w.Write([]byte("Ok")); err != nil {
		slog.ErrorContext(r.Context(), "Failed to write response", "error", err)

		return
	}
}
