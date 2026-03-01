package codeimg

import (
	"bytes"
	"fmt"
	"html/template"
	"sort"
	"sync"
)

// TemplateSpec describes a visual template for rendering code diff images.
type TemplateSpec struct {
	// Name is the unique identifier (e.g. "github", "macos", "vscode").
	Name string
	// Description is a short human-readable summary.
	Description string
	// Selector is the CSS selector chromedp uses to screenshot the element.
	Selector string
	// SupportsDescription indicates the template has a description area.
	SupportsDescription bool
	// Template is the compiled HTML template.
	Template *template.Template
}

// sharedFuncMap is the template.FuncMap shared by all templates.
var sharedFuncMap = template.FuncMap{
	"lineNumStr": func(n int) string {
		if n == 0 {
			return ""
		}
		return fmt.Sprintf("%d", n)
	},
	"isAddition":   func(t DiffLineType) bool { return t == DiffLineAddition },
	"isDeletion":   func(t DiffLineType) bool { return t == DiffLineDeletion },
	"isHunkHeader": func(t DiffLineType) bool { return t == DiffLineHunkHeader },
	"isContext":    func(t DiffLineType) bool { return t == DiffLineContext },
}

var (
	templateRegistry = make(map[string]*TemplateSpec)
	templateMu       sync.RWMutex
)

// RegisterTemplate adds a template to the global registry. It is typically
// called from init() in each template_*.go file.
func RegisterTemplate(spec *TemplateSpec) {
	templateMu.Lock()
	defer templateMu.Unlock()
	templateRegistry[spec.Name] = spec
}

// LookupTemplate resolves a template by name. An empty or unknown name falls
// back to "github".
func LookupTemplate(name string) *TemplateSpec {
	templateMu.RLock()
	defer templateMu.RUnlock()
	if name == "" {
		name = "github"
	}
	if spec, ok := templateRegistry[name]; ok {
		return spec
	}
	// Fallback.
	return templateRegistry["github"]
}

// TemplateNames returns all registered template names in sorted order.
func TemplateNames() []string {
	templateMu.RLock()
	defer templateMu.RUnlock()
	names := make([]string, 0, len(templateRegistry))
	for n := range templateRegistry {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// RenderHTML renders the given diff data using the named template. If the
// template name is empty or unrecognised it falls back to "github".
func RenderHTML(data TemplateDiffData, templateName string) (string, error) {
	spec := LookupTemplate(templateName)
	if spec == nil {
		return "", fmt.Errorf("unknown template %q and fallback unavailable", templateName)
	}
	var buf bytes.Buffer
	if err := spec.Template.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("executing template %q: %w", spec.Name, err)
	}
	return buf.String(), nil
}

// MustParseTemplate is a convenience for template_*.go files: it creates a new
// template with the shared FuncMap and parses the given HTML string.
func MustParseTemplate(name, htmlStr string) *template.Template {
	return template.Must(template.New(name).Funcs(sharedFuncMap).Parse(htmlStr))
}
