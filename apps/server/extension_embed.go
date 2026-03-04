//go:build embedweb

package main

import (
	"embed"
	"io/fs"
)

//go:embed extension
var embeddedExtension embed.FS

func extensionFS() fs.FS {
	sub, err := fs.Sub(embeddedExtension, "extension")
	if err != nil {
		return nil
	}
	return sub
}
