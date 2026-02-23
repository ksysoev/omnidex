package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestUse_NoMiddleware(t *testing.T) {
	called := false
	handler := func(w http.ResponseWriter, r *http.Request) {
		called = true

		w.WriteHeader(http.StatusOK)
	}

	h := Use(handler)
	req := httptest.NewRequest("GET", "/", http.NoBody)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if !called {
		t.Error("handler was not called")
	}

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestUse_WithMiddleware(t *testing.T) {
	order := []string{}
	mw1 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "mw1")

			next.ServeHTTP(w, r)
		})
	}

	mw2 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "mw2")

			next.ServeHTTP(w, r)
		})
	}

	handler := func(w http.ResponseWriter, r *http.Request) {
		order = append(order, "handler")

		w.WriteHeader(http.StatusOK)
	}

	h := Use(handler, mw1, mw2)
	req := httptest.NewRequest("GET", "/", http.NoBody)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	expectedOrder := []string{"mw1", "mw2", "handler"}
	for i, v := range expectedOrder {
		if order[i] != v {
			t.Errorf("expected order[%d]=%s, got %s", i, v, order[i])
		}
	}
}
