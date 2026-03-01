package codeimg

func init() {
	RegisterTemplate(&TemplateSpec{
		Name:                "vscode",
		Description:         "VS Code editor with tabs, breadcrumbs, and status bar",
		Selector:            ".capture-root",
		SupportsDescription: true,
		Template:            MustParseTemplate("vscode", vscodeHTMLTemplate),
	})
}

const vscodeHTMLTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>{{.Filename}}</title>
<style>
  *, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }

  body {
    background: {{.Theme.BodyBackground}};
    font-family: ui-monospace, SFMono-Regular, "SF Mono", Menlo, Consolas,
                 "Liberation Mono", monospace;
    font-size: 13px;
    line-height: 1.55;
    color: {{.Theme.TextPrimary}};
    padding: 32px;
    display: inline-block;
    min-width: 100%;
  }

  .capture-root {
    display: inline-block;
  }

  .vscode-window {
    background: {{.Theme.CardBackground}};
    border-radius: 8px;
    overflow: hidden;
    box-shadow: {{.Theme.BoxShadow}};
    border: 1px solid {{.Theme.BorderColor}};
  }

  /* ── Title bar ────────────────────────────────────────────────────── */
  .vscode-titlebar {
    display: flex;
    align-items: center;
    padding: 0 12px;
    height: 32px;
    background: {{.Theme.HeaderBackground}};
    border-bottom: 1px solid {{.Theme.BorderColor}};
    font-size: 11px;
    color: {{.Theme.TextSecondary}};
  }

  .vscode-titlebar-dots {
    display: flex;
    gap: 6px;
    margin-right: 12px;
  }

  .vscode-titlebar-dots span {
    width: 10px;
    height: 10px;
    border-radius: 50%;
    background: {{.Theme.TextMuted}};
    opacity: 0.5;
  }

  .vscode-titlebar-text {
    flex: 1;
    text-align: center;
    color: {{.Theme.TextSecondary}};
    font-size: 11px;
  }

  /* ── Tab bar ──────────────────────────────────────────────────────── */
  .vscode-tabs {
    display: flex;
    background: {{.Theme.HeaderBackground}};
    border-bottom: 1px solid {{.Theme.BorderColor}};
    height: 36px;
    overflow: hidden;
  }

  .vscode-tab {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 0 16px;
    font-size: 12px;
    color: {{.Theme.TextSecondary}};
    border-right: 1px solid {{.Theme.BorderColor}};
    white-space: nowrap;
    position: relative;
  }

  .vscode-tab.active {
    background: {{.Theme.CardBackground}};
    color: {{.Theme.TextPrimary}};
  }

  .vscode-tab.active::after {
    content: "";
    position: absolute;
    bottom: 0;
    left: 0;
    right: 0;
    height: 2px;
    background: {{.Theme.AccentColor}};
  }

  .vscode-tab-icon {
    width: 14px;
    height: 14px;
    border-radius: 2px;
    background: {{.Theme.AccentColor}};
    opacity: 0.6;
    flex-shrink: 0;
  }

  /* ── Breadcrumb ──────────────────────────────────────────────────── */
  .vscode-breadcrumb {
    padding: 4px 16px;
    font-size: 11px;
    color: {{.Theme.TextSecondary}};
    background: {{.Theme.CardBackground}};
    border-bottom: 1px solid {{.Theme.BorderColor}};
  }

  .vscode-breadcrumb-sep {
    margin: 0 4px;
    opacity: 0.5;
  }

  /* ── Editor area ─────────────────────────────────────────────────── */
  .vscode-editor {
    display: flex;
    position: relative;
  }

  .vscode-editor-main {
    flex: 1;
    min-width: 0;
  }

  /* Mini-map hint */
  .vscode-minimap {
    width: 48px;
    background: {{.Theme.CardBackground}};
    border-left: 1px solid {{.Theme.BorderColor}};
    position: relative;
    overflow: hidden;
  }

  .vscode-minimap::before {
    content: "";
    position: absolute;
    top: 4px;
    left: 8px;
    right: 8px;
    bottom: 4px;
    background: linear-gradient(
      180deg,
      {{.Theme.TextMuted}} 0%,
      transparent 30%,
      {{.Theme.TextMuted}} 50%,
      transparent 70%,
      {{.Theme.TextMuted}} 100%
    );
    opacity: 0.08;
    border-radius: 2px;
  }

  /* ── Diff table ──────────────────────────────────────────────────── */
  .diff-table {
    width: 100%;
    border-collapse: collapse;
    font-size: 12px;
  }

  .diff-table td {
    padding: 0;
    vertical-align: top;
    white-space: pre;
  }

  /* Left margin colour indicator */
  .diff-marker {
    width: 3px;
    min-width: 3px;
    padding: 0;
  }

  tr.addition .diff-marker { background: {{.Theme.AdditionSign}}; }
  tr.deletion .diff-marker { background: {{.Theme.DeletionSign}}; }
  tr.hunk-header .diff-marker { background: {{.Theme.HunkText}}; }
  tr.context .diff-marker { background: transparent; }

  .ln {
    width: 1%;
    min-width: 36px;
    padding: 0 10px;
    text-align: right;
    color: {{.Theme.GutterText}};
    user-select: none;
    font-size: 11px;
  }

  .code {
    padding: 0 14px;
    width: 99%;
    overflow: hidden;
  }

  tr.addition td { background: {{.Theme.AdditionBg}}; }
  tr.addition td.ln { color: {{.Theme.AdditionGutter}}; }
  tr.addition td.code { color: {{.Theme.AdditionText}}; }

  tr.deletion td { background: {{.Theme.DeletionBg}}; }
  tr.deletion td.ln { color: {{.Theme.DeletionGutter}}; }
  tr.deletion td.code { color: {{.Theme.DeletionText}}; }

  tr.hunk-header td {
    background: {{.Theme.HunkBg}};
    color: {{.Theme.HunkText}};
    font-style: italic;
    padding-top: 2px;
    padding-bottom: 2px;
  }
  tr.hunk-header td.ln { color: {{.Theme.HunkGutter}}; }

  tr.context td {
    background: {{.Theme.ContextBg}};
    color: {{.Theme.ContextText}};
  }

  /* ── Status bar ──────────────────────────────────────────────────── */
  .vscode-statusbar {
    display: flex;
    align-items: center;
    gap: 16px;
    padding: 0 12px;
    height: 24px;
    background: {{.Theme.AccentColor}};
    font-size: 11px;
    color: #fff;
  }

  .vscode-statusbar-right {
    margin-left: auto;
    display: flex;
    gap: 16px;
  }

  .vscode-description {
    padding: 6px 16px;
    background: {{.Theme.DescriptionBg}};
    border-bottom: 1px solid {{.Theme.BorderColor}};
    font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Helvetica, Arial, sans-serif;
    font-size: 12px;
    line-height: 1.5;
    color: {{.Theme.DescriptionText}};
  }
</style>
</head>
<body>
<div class="capture-root">
<div class="vscode-window">
  <div class="vscode-titlebar">
    <div class="vscode-titlebar-dots">
      <span></span><span></span><span></span>
    </div>
    <div class="vscode-titlebar-text">{{.Filename}} — {{if .RepoName}}{{.RepoName}}{{else}}GoViral{{end}}</div>
  </div>

  <div class="vscode-tabs">
    <div class="vscode-tab active">
      <span class="vscode-tab-icon"></span>
      {{.Filename}}
    </div>
  </div>

  <div class="vscode-breadcrumb">
    {{if .RepoName}}{{.RepoName}}<span class="vscode-breadcrumb-sep">›</span>{{end}}{{.Filename}}
  </div>

  <div class="vscode-editor">
    <div class="vscode-editor-main">
      {{if .Description}}
      <div class="vscode-description">{{.Description}}</div>
      {{end}}
      <table class="diff-table">
        <tbody>
        {{range .Lines}}
        {{if isHunkHeader .Type}}
          <tr class="hunk-header">
            <td class="diff-marker"></td>
            <td class="ln" colspan="2"></td>
            <td class="code">{{.Content}}</td>
          </tr>
        {{else if isAddition .Type}}
          <tr class="addition">
            <td class="diff-marker"></td>
            <td class="ln">{{lineNumStr .OldLineNum}}</td>
            <td class="ln">{{lineNumStr .NewLineNum}}</td>
            <td class="code">{{.Content}}</td>
          </tr>
        {{else if isDeletion .Type}}
          <tr class="deletion">
            <td class="diff-marker"></td>
            <td class="ln">{{lineNumStr .OldLineNum}}</td>
            <td class="ln">{{lineNumStr .NewLineNum}}</td>
            <td class="code">{{.Content}}</td>
          </tr>
        {{else}}
          <tr class="context">
            <td class="diff-marker"></td>
            <td class="ln">{{lineNumStr .OldLineNum}}</td>
            <td class="ln">{{lineNumStr .NewLineNum}}</td>
            <td class="code">{{.Content}}</td>
          </tr>
        {{end}}
        {{end}}
        </tbody>
      </table>
    </div>
    <div class="vscode-minimap"></div>
  </div>

  <div class="vscode-statusbar">
    <span>{{if .Language}}{{.Language}}{{else}}Plain Text{{end}}</span>
    <span>+{{.Additions}} -{{.Deletions}}</span>
    <div class="vscode-statusbar-right">
      <span>UTF-8</span>
      <span>LF</span>
    </div>
  </div>
</div>
</div>
</body>
</html>
`
