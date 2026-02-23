package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	mock "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestAPI_newMux_LivezRoute(t *testing.T) {
	mockSvc := NewMockService(t)
	mockSvc.EXPECT().CheckHealth(mock.Anything).Return(nil)

	a, err := New(Config{Listen: ":0"}, mockSvc)
	require.NoError(t, err)

	mux := a.newMux()

	req := httptest.NewRequest("GET", "/livez", http.NoBody)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	resp := w.Result()
	assert.Equal(t, http.StatusOK, resp.StatusCode, "expected status 200")

	body := w.Body.String()
	assert.Equal(t, "Ok", body)
}
