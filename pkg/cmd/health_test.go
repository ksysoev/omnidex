package cmd

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRunHealthCheck_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/livez", r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)

		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	err := runHealthCheck(t.Context(), srv.URL)
	assert.NoError(t, err)
}

func TestRunHealthCheck_ServerDown(t *testing.T) {
	err := runHealthCheck(t.Context(), "http://localhost:1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "health check failed")
}

func TestRunHealthCheck_NonOKStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	err := runHealthCheck(t.Context(), srv.URL)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "health check returned status 503")
}

func TestRunHealthCheck_InvalidURL(t *testing.T) {
	err := runHealthCheck(t.Context(), "://invalid")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create request")
}

func TestRunHealthCheck_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := runHealthCheck(ctx, "http://localhost:8080")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "health check failed")
}

func TestNewHealthCmd(t *testing.T) {
	cmd := newHealthCmd()

	assert.Equal(t, "health", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Long)

	urlFlag := cmd.Flags().Lookup("url")
	assert.NotNil(t, urlFlag)
	assert.Equal(t, "http://localhost:8080", urlFlag.DefValue)
}
