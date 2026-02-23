//go:build !compile

package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHealthCheck(t *testing.T) {
	api := &API{}

	req := httptest.NewRequest(http.MethodGet, "/livez", http.NoBody)
	rec := httptest.NewRecorder()

	api.healthCheck(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "text/plain", rec.Header().Get("Content-Type"))
	assert.Equal(t, "Ok", rec.Body.String())
}

func TestNewMux_RoutesRegistered(t *testing.T) {
	svc := NewMockService(t)
	views := NewMockViewRenderer(t)

	api := &API{
		svc:    svc,
		views:  views,
		config: Config{APIKeys: []string{"test-key"}},
	}

	mux := api.newMux()

	tests := []struct {
		name          string
		method        string
		path          string
		description   string
		wantStatusNot int
	}{
		{
			name:          "health check route exists",
			method:        http.MethodGet,
			path:          "/livez",
			wantStatusNot: http.StatusNotFound,
			description:   "health check should be registered",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, http.NoBody)
			rec := httptest.NewRecorder()

			mux.ServeHTTP(rec, req)

			assert.NotEqual(t, tt.wantStatusNot, rec.Code, tt.description)
		})
	}
}
