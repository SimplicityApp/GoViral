package handler

import (
	"archive/zip"
	"io/fs"
	"net/http"
)

// ExtensionHandler serves the Chrome extension as a zip download.
type ExtensionHandler struct {
	extFS fs.FS
}

// NewExtensionHandler creates a new ExtensionHandler.
// extFS may be nil if extension files are not available.
func NewExtensionHandler(extFS fs.FS) *ExtensionHandler {
	return &ExtensionHandler{extFS: extFS}
}

// Download serves the extension directory as a zip file.
func (h *ExtensionHandler) Download(w http.ResponseWriter, r *http.Request) {
	if h.extFS == nil {
		http.Error(w, "extension files not available", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", `attachment; filename="goviral-extension.zip"`)

	zw := zip.NewWriter(w)
	defer zw.Close()

	err := fs.WalkDir(h.extFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, err := fs.ReadFile(h.extFS, path)
		if err != nil {
			return err
		}
		fw, err := zw.Create(path)
		if err != nil {
			return err
		}
		_, err = fw.Write(data)
		return err
	})
	if err != nil {
		http.Error(w, "failed to create zip", http.StatusInternalServerError)
	}
}
