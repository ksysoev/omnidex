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
