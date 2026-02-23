package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

type keyReqID struct{}

func NewReqID() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			reqID := uuid.New().String()

			ctx := context.WithValue(r.Context(), keyReqID{}, reqID)
			r = r.WithContext(ctx)

			// Call the next handler
			next.ServeHTTP(w, r)
		})
	}
}

// GetReqID extracts the request ID from the provided context.
// If no request ID is found, it returns an empty string.
func GetReqID(ctx context.Context) string {
	val := ctx.Value(keyReqID{})

	reqID, ok := val.(string)
	if ok {
		return reqID
	}

	return ""
}
