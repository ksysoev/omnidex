package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestAPI_healthCheck_OK(t *testing.T) {
	mockSvc := NewMockService(t)
	mockSvc.On("CheckHealth", mock.Anything).Return(nil)
	api := &API{svc: mockSvc}
	req := httptest.NewRequest("GET", "/health", http.NoBody)
	w := httptest.NewRecorder()

	api.healthCheck(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	if ct := resp.Header.Get("Content-Type"); ct != "text/plain" {
		t.Errorf("expected Content-Type text/plain, got %q", ct)
	}

	body := w.Body.String()
	if body != "Ok" {
		t.Errorf("expected body 'Ok', got %q", body)
	}
}

func TestAPI_healthCheck_Error(t *testing.T) {
	mockSvc := NewMockService(t)
	mockSvc.On("CheckHealth", mock.Anything).Return(assert.AnError)
	api := &API{svc: mockSvc}
	req := httptest.NewRequest("GET", "/health", http.NoBody)
	w := httptest.NewRecorder()

	api.healthCheck(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", resp.StatusCode)
	}
}
