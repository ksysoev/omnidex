package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewAuth_ValidKey(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	authMiddleware := NewAuth([]string{"test-key-123"})
	wrapped := authMiddleware(handler)

	req := httptest.NewRequest("POST", "/api/v1/docs", http.NoBody)
	req.Header.Set("Authorization", "Bearer test-key-123")

	w := httptest.NewRecorder()
	wrapped.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestNewAuth_InvalidKey(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	authMiddleware := NewAuth([]string{"test-key-123"})
	wrapped := authMiddleware(handler)

	req := httptest.NewRequest("POST", "/api/v1/docs", http.NoBody)
	req.Header.Set("Authorization", "Bearer wrong-key")

	w := httptest.NewRecorder()
	wrapped.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestNewAuth_MissingHeader(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	authMiddleware := NewAuth([]string{"test-key-123"})
	wrapped := authMiddleware(handler)

	req := httptest.NewRequest("POST", "/api/v1/docs", http.NoBody)

	w := httptest.NewRecorder()
	wrapped.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestNewAuth_InvalidFormat(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	authMiddleware := NewAuth([]string{"test-key-123"})
	wrapped := authMiddleware(handler)

	req := httptest.NewRequest("POST", "/api/v1/docs", http.NoBody)
	req.Header.Set("Authorization", "Basic test-key-123")

	w := httptest.NewRecorder()
	wrapped.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestNewAuth_MultipleKeys(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	authMiddleware := NewAuth([]string{"key-1", "key-2", "key-3"})
	wrapped := authMiddleware(handler)

	req := httptest.NewRequest("POST", "/api/v1/docs", http.NoBody)
	req.Header.Set("Authorization", "Bearer key-2")

	w := httptest.NewRecorder()
	wrapped.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestNewAuth_EmptyKeys(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	authMiddleware := NewAuth([]string{})
	wrapped := authMiddleware(handler)

	req := httptest.NewRequest("POST", "/api/v1/docs", http.NoBody)
	req.Header.Set("Authorization", "Bearer any-key")

	w := httptest.NewRecorder()
	wrapped.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
