//go:build !embedweb

package main

import (
	"io/fs"
	"os"
)

func extensionFS() fs.FS {
	// In dev mode, read extension files from the source tree.
	dir := "apps/extension"
	if info, err := os.Stat(dir); err == nil && info.IsDir() {
		return os.DirFS(dir)
	}
	return nil
}
