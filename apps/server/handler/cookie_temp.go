package handler

import (
	"fmt"
	"os"
	"path/filepath"
)

// writeTwikitCookieTempFile writes twikit cookie JSON to a temp file and returns the path
// and a cleanup function. If cookiesJSON is empty, returns empty path and no-op cleanup.
func writeTwikitCookieTempFile(cookiesJSON string) (string, func(), error) {
	if cookiesJSON == "" {
		return "", func() {}, nil
	}
	f, err := os.CreateTemp("", "goviral-twikit-cookies-*.json")
	if err != nil {
		return "", nil, fmt.Errorf("creating twikit cookie temp file: %w", err)
	}
	path := f.Name()
	if _, err := f.WriteString(cookiesJSON); err != nil {
		f.Close()
		os.Remove(path)
		return "", nil, fmt.Errorf("writing twikit cookie temp file: %w", err)
	}
	f.Close()
	return path, func() { os.Remove(path) }, nil
}

// writeLinkitinCookieTempDir writes linkitin cookie JSON to a temp directory as
// linkitin_cookies.json and returns the directory path and a cleanup function.
// The linkitin bridge script reads cookies from {configDir}/linkitin_cookies.json.
func writeLinkitinCookieTempDir(cookiesJSON string) (string, func(), error) {
	if cookiesJSON == "" {
		return "", func() {}, nil
	}
	dir, err := os.MkdirTemp("", "goviral-linkitin-config-*")
	if err != nil {
		return "", nil, fmt.Errorf("creating linkitin cookie temp dir: %w", err)
	}
	cookiePath := filepath.Join(dir, "linkitin_cookies.json")
	if err := os.WriteFile(cookiePath, []byte(cookiesJSON), 0600); err != nil {
		os.RemoveAll(dir)
		return "", nil, fmt.Errorf("writing linkitin cookie temp file: %w", err)
	}
	return dir, func() { os.RemoveAll(dir) }, nil
}
