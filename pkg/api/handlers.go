package api

import (
	"log/slog"
	"net/http"
)

// CheckHealth verifies the health of the service and returns an error
// if the service is unhealthy.
func (a *API) healthCheck(w http.ResponseWriter, r *http.Request) {
	if err := a.svc.CheckHealth(r.Context()); err != nil {
		slog.ErrorContext(r.Context(), "Health check failed", "error", err)

		http.Error(w, "Internal Server Error", http.StatusInternalServerError)

		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)

	if _, err := w.Write([]byte("Ok")); err != nil {
		slog.ErrorContext(r.Context(), "Failed to write response", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)

		return
	}
}
