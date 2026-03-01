package codeimg

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// samplePatch is a realistic unified diff for testing.
const samplePatch = `diff --git a/main.go b/main.go
index abc1234..def5678 100644
--- a/main.go
+++ b/main.go
@@ -10,7 +10,9 @@ func main() {
 	fmt.Println("hello")
-	fmt.Println("old line")
+	fmt.Println("new line")
+	fmt.Println("added line")
 	fmt.Println("world")
 }
`

func TestAllTemplatesCompile(t *testing.T) {
	data := FormatDiffForTemplate(samplePatch, 25)
	data.Filename = "main.go"
	data.Language = DetectLanguage("main.go")
	data.Description = "Refactored greeting logic for clarity"
	data.RepoName = "acme/myapp"

	for _, tmplName := range TemplateNames() {
		for _, themeName := range ThemeNames() {
			t.Run(tmplName+"/"+themeName, func(t *testing.T) {
				data.Theme = LookupTheme(themeName)
				html, err := RenderHTML(data, tmplName)
				if err != nil {
					t.Fatalf("RenderHTML(%s, %s): %v", tmplName, themeName, err)
				}
				if len(html) == 0 {
					t.Fatal("empty HTML output")
				}
				if !strings.Contains(html, "main.go") {
					t.Error("HTML missing filename")
				}
			})
		}
	}
}

func TestLookupThemeDefaults(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"", "github-dark"},
		{"dark", "github-dark"},
		{"light", "solarized-light"},
		{"github-dark", "github-dark"},
		{"dracula", "dracula"},
		{"nonexistent", "github-dark"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := LookupTheme(tt.input)
			if got.Name != tt.want {
				t.Errorf("LookupTheme(%q).Name = %q, want %q", tt.input, got.Name, tt.want)
			}
		})
	}
}

func TestLookupTemplateDefaults(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"", "github"},
		{"github", "github"},
		{"macos", "macos"},
		{"vscode", "vscode"},
		{"minimal", "minimal"},
		{"terminal", "terminal"},
		{"card", "card"},
		{"nonexistent", "github"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := LookupTemplate(tt.input)
			if got.Name != tt.want {
				t.Errorf("LookupTemplate(%q).Name = %q, want %q", tt.input, got.Name, tt.want)
			}
		})
	}
}

func TestRenderDiffHTMLBackwardCompat(t *testing.T) {
	data := FormatDiffForTemplate(samplePatch, 25)
	data.Filename = "main.go"
	data.Language = "Go"

	html, err := RenderDiffHTML(data)
	if err != nil {
		t.Fatalf("RenderDiffHTML: %v", err)
	}
	if !strings.Contains(html, "main.go") {
		t.Error("backward-compat RenderDiffHTML missing filename")
	}
	if !strings.Contains(html, "#0d1117") {
		t.Error("expected github-dark body background in default render")
	}
}

func TestAllTemplatesSupportsDescription(t *testing.T) {
	for _, name := range TemplateNames() {
		spec := LookupTemplate(name)
		if !spec.SupportsDescription {
			t.Errorf("template %q should have SupportsDescription=true", name)
		}
	}
}

// TestWriteSampleHTML writes one HTML file per template to a temp dir for
// manual visual inspection. Run with:
//
//	go test ./internal/codeimg/ -run TestWriteSampleHTML -v
//
// Then open the files in a browser.
func TestWriteSampleHTML(t *testing.T) {
	if os.Getenv("WRITE_SAMPLES") == "" {
		t.Skip("set WRITE_SAMPLES=1 to write sample HTML files")
	}

	dir := filepath.Join(os.TempDir(), "goviral-template-samples")
	os.MkdirAll(dir, 0o755)

	data := FormatDiffForTemplate(samplePatch, 25)
	data.Filename = "internal/server/handler.go"
	data.Language = DetectLanguage("handler.go")
	data.Description = "Refactored error handling to use structured errors with stack traces"
	data.RepoName = "acme/backend"

	for _, tmplName := range TemplateNames() {
		data.Theme = LookupTheme("dracula")
		html, err := RenderHTML(data, tmplName)
		if err != nil {
			t.Fatalf("RenderHTML(%s): %v", tmplName, err)
		}
		path := filepath.Join(dir, tmplName+".html")
		if err := os.WriteFile(path, []byte(html), 0o644); err != nil {
			t.Fatalf("writing %s: %v", path, err)
		}
		t.Logf("wrote %s", path)
	}
	t.Logf("\nOpen files in browser: open %s/*.html", dir)
}
