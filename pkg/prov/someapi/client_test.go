package someapi

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	cli := New(Config{})

	assert.NotNil(t, cli, "New() should return a non-nil APIClient")
}

func TestAPIClient_CheckHealth_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	cfg := Config{
		BaseURL: ts.URL,
	}

	apiClient := New(cfg)

	err := apiClient.CheckHealth(t.Context())

	assert.NoError(t, err, "CheckHealth should not return an error for a healthy service")
}

func TestAPIClient_CheckHealth_Failure(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	newClient := New(Config{BaseURL: "invalid-url"})

	err := newClient.CheckHealth(t.Context())
	assert.Error(t, err, "CheckHealth should return an error for an unhealthy service")

	newClient = New(Config{BaseURL: ts.URL})
	err = newClient.CheckHealth(t.Context())
	assert.Error(t, err, "CheckHealth should return an error for an unhealthy service with status code 500")
}
