package middleware

import (
	"net/http"
	"strings"
)

// CORS returns middleware that handles Cross-Origin Resource Sharing.
func CORS(allowedOrigins []string) func(http.Handler) http.Handler {
	originSet := make(map[string]bool, len(allowedOrigins))
	for _, o := range allowedOrigins {
		originSet[o] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			if origin != "" && isAllowed(origin, originSet, allowedOrigins) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, Accept, X-User-ID")
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Max-Age", "86400")
			}

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func isAllowed(origin string, originSet map[string]bool, allowedOrigins []string) bool {
	if originSet["*"] {
		return true
	}
	if originSet[origin] {
		return true
	}
	// Support wildcard subdomains like "https://*.vercel.app"
	for _, allowed := range allowedOrigins {
		if idx := strings.Index(allowed, "*."); idx >= 0 {
			prefix := allowed[:idx] // "https://"
			suffix := allowed[idx+1:] // ".vercel.app"
			if strings.HasPrefix(origin, prefix) && strings.HasSuffix(origin, suffix) {
				// Ensure there's something between prefix and suffix (not empty subdomain)
				middle := origin[len(prefix) : len(origin)-len(suffix)]
				if len(middle) > 0 && !strings.Contains(middle, "/") {
					return true
				}
			}
		}
	}
	return false
}
