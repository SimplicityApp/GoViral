//go:build !embedweb

package main

import (
	"net/http"
)

// staticHandler returns a no-op handler when the web UI is not embedded.
// Build with -tags embedweb after copying apps/web/dist/ to apps/server/static/.
func staticHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "web UI not embedded; build with -tags embedweb", http.StatusNotFound)
	})
}
