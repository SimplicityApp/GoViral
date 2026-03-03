package middleware

import (
	"net/http"
	"strings"
)

// Auth returns middleware that validates the Authorization Bearer token
// against the configured API key. The /api/v1/health endpoint is excluded.
func Auth(apiKey string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip auth for health and OAuth endpoints (browser redirects can't carry headers)
			if r.URL.Path == "/api/v1/health" || strings.HasPrefix(r.URL.Path, "/api/v1/oauth/") {
				next.ServeHTTP(w, r)
				return
			}

			// If no API key is configured, skip auth
			if apiKey == "" {
				next.ServeHTTP(w, r)
				return
			}

			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				reqID := RequestIDFromContext(r.Context())
				WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing authorization header", reqID)
				return
			}

			token := strings.TrimPrefix(authHeader, "Bearer ")
			if token == authHeader || token != apiKey {
				reqID := RequestIDFromContext(r.Context())
				WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid API key", reqID)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
