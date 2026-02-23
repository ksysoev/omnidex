package middleware

import (
	"crypto/subtle"
	"net/http"
	"strings"
)

// NewAuth creates a middleware that validates API key authentication.
// It checks the Authorization header for a valid Bearer token against the provided list of valid keys.
// If no valid keys are configured, all requests are rejected.
func NewAuth(validKeys []string) func(http.Handler) http.Handler {
	keySet := make(map[string]struct{}, len(validKeys))

	for _, k := range validKeys {
		if k != "" {
			keySet[k] = struct{}{}
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "missing authorization header", http.StatusUnauthorized)
				return
			}

			token, found := strings.CutPrefix(authHeader, "Bearer ")
			if !found {
				http.Error(w, "invalid authorization format", http.StatusUnauthorized)
				return
			}

			if !isValidKey(token, keySet) {
				http.Error(w, "invalid API key", http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func isValidKey(token string, validKeys map[string]struct{}) bool {
	for key := range validKeys {
		if subtle.ConstantTimeCompare([]byte(token), []byte(key)) == 1 {
			return true
		}
	}

	return false
}
