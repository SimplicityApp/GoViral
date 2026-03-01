package models

import "strings"

// lockfileNames is the set of exact filenames that are lock/generated files.
var lockfileNames = map[string]bool{
	"go.sum":              true,
	"go.work.sum":         true,
	"package-lock.json":   true,
	"yarn.lock":           true,
	"pnpm-lock.yaml":      true,
	"Cargo.lock":          true,
	"Pipfile.lock":        true,
	"poetry.lock":         true,
	"composer.lock":       true,
	"Gemfile.lock":        true,
	"packages.lock.json":  true,
	"project.assets.json": true,
	"flake.lock":          true,
}

// generatedSuffixes are filename suffixes that indicate machine-generated code.
var generatedSuffixes = []string{
	".pb.go",
	".pb.gw.go",
	"_gen.go",
	".gen.go",
	"_generated.go",
	".generated.go",
	"_mock.go",
	".min.js",
	".min.css",
	".d.ts",
}

// sourceExtensions lists extensions considered "real" source code.
var sourceExtensions = map[string]bool{
	".go":    true,
	".ts":    true,
	".tsx":   true,
	".js":    true,
	".jsx":   true,
	".py":    true,
	".rs":    true,
	".java":  true,
	".kt":    true,
	".swift": true,
	".c":     true,
	".cpp":   true,
	".cc":    true,
	".h":     true,
	".hpp":   true,
	".cs":    true,
	".rb":    true,
	".php":   true,
	".scala": true,
	".ex":    true,
	".exs":   true,
	".zig":   true,
	".lua":   true,
	".sh":    true,
	".bash":  true,
}

// configExtensions lists extensions that are typically config/data files.
var configExtensions = map[string]bool{
	".json": true,
	".yaml": true,
	".yml":  true,
	".toml": true,
	".xml":  true,
	".ini":  true,
	".env":  true,
}

// IsLockfile returns true when filename is a lock file or a known generated file.
func IsLockfile(filename string) bool {
	base := fileBaseName(filename)
	if lockfileNames[base] {
		return true
	}
	for _, suffix := range generatedSuffixes {
		if strings.HasSuffix(filename, suffix) {
			return true
		}
	}
	return false
}

// IsSourceFile returns true when filename is a recognised source-code file.
func IsSourceFile(filename string) bool {
	return sourceExtensions[fileExtLower(filename)]
}

// IsConfigFile returns true when filename looks like a configuration / data file.
func IsConfigFile(filename string) bool {
	return configExtensions[fileExtLower(filename)]
}

// fileBaseName returns the last path component of a forward-slash-delimited path.
func fileBaseName(path string) string {
	if idx := strings.LastIndexByte(path, '/'); idx >= 0 {
		return path[idx+1:]
	}
	return path
}

// fileExtLower returns the lowercase file extension including the leading dot.
func fileExtLower(filename string) string {
	base := fileBaseName(filename)
	if idx := strings.LastIndexByte(base, '.'); idx > 0 {
		return strings.ToLower(base[idx:])
	}
	return ""
}
