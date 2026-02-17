package auth

import (
	"context"
	"fmt"
	"net/http"
	"os/exec"
	"runtime"
)

// StartCallbackServer starts a temporary HTTP server on the given port,
// waits for an OAuth callback with an authorization code, then shuts down.
func StartCallbackServer(ctx context.Context, port int) (string, error) {
	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			errMsg := r.URL.Query().Get("error")
			if errMsg == "" {
				errMsg = "no authorization code received"
			}
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprintf(w, "<html><body><h2>Authorization failed: %s</h2><p>You can close this tab.</p></body></html>", errMsg)
			errCh <- fmt.Errorf("OAuth callback error: %s", errMsg)
			return
		}

		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, "<html><body><h2>Authorization complete!</h2><p>You can close this tab and return to the terminal.</p></body></html>")
		codeCh <- code
	})

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}

	// Start serving in a goroutine.
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- fmt.Errorf("starting callback server: %w", err)
		}
	}()

	defer server.Shutdown(context.Background())

	select {
	case code := <-codeCh:
		return code, nil
	case err := <-errCh:
		return "", err
	case <-ctx.Done():
		return "", fmt.Errorf("timed out waiting for authorization callback")
	}
}

// OpenBrowser opens the given URL in the user's default browser.
func OpenBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return fmt.Errorf("unsupported platform %s", runtime.GOOS)
	}
	return cmd.Start()
}
