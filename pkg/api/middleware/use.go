package middleware

import (
	"net/http"
)

// Use applies a list of middleware functions to an http.Handler.
// Middlewares are applied in reverse order so that the first middleware
// wraps the handler last, preserving the expected chaining behavior.
func Use(handler func(http.ResponseWriter, *http.Request), middlewares ...func(http.Handler) http.Handler) http.Handler {
	var h http.Handler = http.HandlerFunc(handler)
	for i := len(middlewares); i > 0; i-- {
		h = middlewares[i-1](h)
	}

	return h
}
