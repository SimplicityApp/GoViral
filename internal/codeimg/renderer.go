package codeimg

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/shuhao/goviral/pkg/models"
)

// renderTimeout is the maximum time allowed for a single RenderDiff call.
const renderTimeout = 15 * time.Second

// Renderer renders code diff images using a persistent headless Chrome
// instance.  Create one with NewRenderer and call Close when done.
//
// Renderer implements models.CodeImageRenderer.
type Renderer struct {
	allocCtx    context.Context
	allocCancel context.CancelFunc
}

// Compile-time interface compliance check.
var _ models.CodeImageRenderer = (*Renderer)(nil)

// NewRenderer starts a headless Chrome allocator and returns a ready Renderer.
// The browser process is shared across all RenderDiff calls; call Close when
// the Renderer is no longer needed so the process can be cleaned up.
func NewRenderer() (*Renderer, error) {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-setuid-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("hide-scrollbars", true),
		chromedp.Flag("mute-audio", true),
		// Avoid certificate errors when loading local file:// URLs.
		chromedp.Flag("allow-file-access-from-files", true),
		chromedp.Flag("disable-web-security", true),
	)

	allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(), opts...)

	// Verify the allocator works by creating (and immediately discarding) a
	// browser context.  This surfaces missing-Chrome errors early.
	probeCtx, probeCancel := chromedp.NewContext(allocCtx)
	defer probeCancel()
	if err := chromedp.Run(probeCtx); err != nil {
		allocCancel()
		return nil, fmt.Errorf("starting headless Chrome: %w", err)
	}

	return &Renderer{
		allocCtx:    allocCtx,
		allocCancel: allocCancel,
	}, nil
}

// RenderDiff renders a unified diff patch as a PNG screenshot.
//
// diff is the raw unified diff text (as returned by the GitHub API patch
// field).  filename is used both for display in the diff header and for
// language detection.  opts controls rendering behaviour; zero values are
// replaced with sensible defaults.
func (r *Renderer) RenderDiff(diff, filename string, opts models.RenderOptions) ([]byte, error) {
	// Apply defaults.
	if opts.Theme == "" {
		opts.Theme = "dark"
	}
	if opts.Template == "" {
		opts.Template = "github"
	}
	if opts.MaxLines <= 0 {
		opts.MaxLines = defaultMaxLines
	}
	if opts.Width <= 0 {
		opts.Width = 680
	}
	if opts.FontSize <= 0 {
		opts.FontSize = 13
	}

	// Build template data from the patch.
	data := FormatDiffForTemplate(diff, opts.MaxLines)
	data.Filename = filename
	data.Language = DetectLanguage(filename)
	data.Theme = LookupTheme(opts.Theme)
	data.Description = opts.Description
	data.RepoName = opts.RepoName

	return r.renderData(data, opts.Template, opts.Width)
}

// RenderDiffData renders a pre-built TemplateDiffData as a PNG image.
// This is used when the caller has already selected which lines to render
// (e.g. from an AI-selected code snippet).
func (r *Renderer) RenderDiffData(data TemplateDiffData, opts models.RenderOptions) ([]byte, error) {
	if opts.Theme == "" {
		opts.Theme = "dark"
	}
	if opts.Template == "" {
		opts.Template = "github"
	}
	if opts.Width <= 0 {
		opts.Width = 680
	}

	// Populate theme if not already set.
	if data.Theme.Name == "" {
		data.Theme = LookupTheme(opts.Theme)
	}
	if data.Description == "" {
		data.Description = opts.Description
	}
	if data.RepoName == "" {
		data.RepoName = opts.RepoName
	}

	return r.renderData(data, opts.Template, opts.Width)
}

// renderData is the shared screenshot pipeline used by both RenderDiff and
// RenderDiffData.
func (r *Renderer) renderData(data TemplateDiffData, templateName string, width int) ([]byte, error) {
	spec := LookupTemplate(templateName)
	if spec == nil {
		return nil, fmt.Errorf("unknown template %q", templateName)
	}

	html, err := RenderHTML(data, templateName)
	if err != nil {
		return nil, fmt.Errorf("rendering HTML for %s: %w", data.Filename, err)
	}

	// Write HTML to a temporary file so Chrome can load it as a proper page
	// (data: URLs have length limits and inconsistent behaviour across OSes).
	tmpFile, err := os.CreateTemp("", "goviral-diff-*.html")
	if err != nil {
		return nil, fmt.Errorf("creating temp HTML file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if _, err := tmpFile.WriteString(html); err != nil {
		tmpFile.Close()
		return nil, fmt.Errorf("writing temp HTML file: %w", err)
	}
	tmpFile.Close()

	fileURL := "file://" + filepath.ToSlash(tmpPath)

	// Create a browser tab context with a hard timeout.
	ctx, cancel := context.WithTimeout(r.allocCtx, renderTimeout)
	defer cancel()

	tabCtx, tabCancel := chromedp.NewContext(ctx)
	defer tabCancel()

	selector := spec.Selector

	var pngBytes []byte
	if err := chromedp.Run(tabCtx,
		chromedp.EmulateViewport(int64(width), 800),
		chromedp.Navigate(fileURL),
		chromedp.WaitVisible(selector, chromedp.ByQuery),
		chromedp.Screenshot(selector, &pngBytes, chromedp.ByQuery),
	); err != nil {
		return nil, fmt.Errorf("capturing screenshot for %s (template %s): %w", data.Filename, templateName, err)
	}

	if len(pngBytes) == 0 {
		return nil, fmt.Errorf("chromedp returned empty screenshot for %s", data.Filename)
	}

	return pngBytes, nil
}

// Close shuts down the underlying Chrome allocator and releases all associated
// resources.  It must be called when the Renderer is no longer needed.
func (r *Renderer) Close() {
	r.allocCancel()
}
