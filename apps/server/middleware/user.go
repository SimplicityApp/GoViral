package middleware

import (
	"context"
	"net/http"
	"regexp"
	"strings"
)

const userIDKey contextKey = "user_id"

var uuidV4Re = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

// UserIDFromContext extracts the user ID from the context.
func UserIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(userIDKey).(string)
	return id
}

// UserID returns middleware that extracts and validates the X-User-ID header,
// calls getOrCreate to upsert the user in the DB, and stores the ID in context.
func UserID(getOrCreate func(string) error) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip user ID validation for health and OAuth endpoints
			if r.URL.Path == "/api/v1/health" || strings.HasPrefix(r.URL.Path, "/api/v1/oauth/") {
				next.ServeHTTP(w, r)
				return
			}

			uid := r.Header.Get("X-User-ID")
			if uid == "" || !uuidV4Re.MatchString(uid) {
				reqID := RequestIDFromContext(r.Context())
				WriteError(w, http.StatusBadRequest, "INVALID_USER_ID", "missing or invalid X-User-ID header (must be UUID v4)", reqID)
				return
			}

			if err := getOrCreate(uid); err != nil {
				reqID := RequestIDFromContext(r.Context())
				WriteError(w, http.StatusInternalServerError, "USER_UPSERT_FAILED", "failed to register user", reqID)
				return
			}

			ctx := context.WithValue(r.Context(), userIDKey, uid)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
