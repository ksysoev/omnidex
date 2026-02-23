package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
)

func TestNewReqID_MiddlewareSetsReqID(t *testing.T) {
	var gotReqID string

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotReqID = GetReqID(r.Context())

		w.WriteHeader(http.StatusOK)
	})

	middleware := NewReqID()
	h := middleware(handler)

	req := httptest.NewRequest("GET", "/", http.NoBody)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if gotReqID == "" {
		t.Error("expected non-empty request ID in context")
	}

	if _, err := uuid.Parse(gotReqID); err != nil {
		t.Errorf("expected valid UUID, got %q: %v", gotReqID, err)
	}
}

func TestGetReqID_ReturnsEmptyStringIfNotSet(t *testing.T) {
	reqID := GetReqID(t.Context())
	if reqID != "" {
		t.Errorf("expected empty string, got %q", reqID)
	}
}

func TestGetReqID_ReturnsReqIDIfSet(t *testing.T) {
	ctx := context.WithValue(t.Context(), keyReqID{}, "test-id")

	reqID := GetReqID(ctx)
	if reqID != "test-id" {
		t.Errorf("expected 'test-id', got %q", reqID)
	}
}
