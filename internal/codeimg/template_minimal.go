package codeimg

func init() {
	RegisterTemplate(&TemplateSpec{
		Name:                "minimal",
		Description:         "Floating card on gradient background (ray.so style)",
		Selector:            ".capture-root",
		SupportsDescription: true,
		Template:            MustParseTemplate("minimal", minimalHTMLTemplate),
	})
}

const minimalHTMLTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>{{.Filename}}</title>
<style>
  *, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }

  body {
    margin: 0;
    padding: 0;
    font-family: ui-monospace, SFMono-Regular, "SF Mono", Menlo, Consolas,
                 "Liberation Mono", monospace;
    font-size: 14px;
    line-height: 1.6;
    color: {{.Theme.TextPrimary}};
    display: inline-block;
    min-width: 100%;
  }

  .capture-root {
    background: linear-gradient(135deg, {{.Theme.GradientStart}} 0%, {{.Theme.GradientEnd}} 50%, {{.Theme.AccentColor}}22 100%);
    padding: 48px;
    display: inline-block;
  }

  .minimal-card {
    background: {{.Theme.CardBackground}};
    border-radius: 16px;
    overflow: hidden;
    box-shadow:
      0 0 0 1px rgba(255,255,255,0.04),
      0 4px 8px rgba(0,0,0,0.15),
      0 12px 32px rgba(0,0,0,0.25),
      0 24px 64px rgba(0,0,0,0.3);
  }

  /* ── Optional filename pill ──────────────────────────────────────── */
  .filename-pill {
    padding: 16px 24px 0 24px;
  }

  .filename-pill span {
    display: inline-block;
    font-size: 11px;
    color: {{.Theme.TextSecondary}};
    background: {{.Theme.HeaderBackground}};
    padding: 3px 10px;
    border-radius: 6px;
    letter-spacing: 0.02em;
  }

  /* ── Code area ───────────────────────────────────────────────────── */
  .minimal-code {
    padding: 16px 0 24px 0;
  }

  .diff-table {
    width: 100%;
    border-collapse: collapse;
    font-size: 13px;
    letter-spacing: 0.01em;
  }

  .diff-table td {
    padding: 0;
    vertical-align: top;
    white-space: pre;
  }

  .ln {
    width: 1%;
    min-width: 40px;
    padding: 1px 12px;
    text-align: right;
    color: {{.Theme.GutterText}};
    user-select: none;
    font-size: 11px;
    opacity: 0.5;
  }

  .code {
    padding: 1px 24px 1px 16px;
    width: 99%;
    overflow: hidden;
  }

  /* Subtle row backgrounds — no sign column, no borders */
  tr.addition td {
    background: {{.Theme.AdditionBg}};
  }
  tr.addition td.code {
    color: {{.Theme.AdditionText}};
  }
  tr.addition td.ln {
    color: {{.Theme.AdditionGutter}};
    opacity: 0.7;
  }

  tr.deletion td {
    background: {{.Theme.DeletionBg}};
  }
  tr.deletion td.code {
    color: {{.Theme.DeletionText}};
  }
  tr.deletion td.ln {
    color: {{.Theme.DeletionGutter}};
    opacity: 0.7;
  }

  tr.hunk-header td {
    background: transparent;
    color: {{.Theme.HunkText}};
    font-style: italic;
    padding-top: 6px;
    padding-bottom: 4px;
    opacity: 0.6;
  }

  tr.context td {
    background: transparent;
    color: {{.Theme.ContextText}};
  }

  .minimal-description {
    padding: 0 0 10px 0;
    font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Helvetica, Arial, sans-serif;
    font-size: 12px;
    line-height: 1.5;
    color: {{.Theme.DescriptionText}};
    padding-left: 24px;
    padding-right: 24px;
  }
</style>
</head>
<body>
<div class="capture-root">
<div class="minimal-card">
  {{if .Filename}}<div class="filename-pill"><span>{{.Filename}}</span></div>{{end}}
  {{if .Description}}
  <div class="minimal-description">{{.Description}}</div>
  {{end}}
  <div class="minimal-code">
    <table class="diff-table">
      <tbody>
      {{range .Lines}}
      {{if isHunkHeader .Type}}
        <tr class="hunk-header">
          <td class="ln"></td>
          <td class="code">{{.Content}}</td>
        </tr>
      {{else if isAddition .Type}}
        <tr class="addition">
          <td class="ln">{{lineNumStr .NewLineNum}}</td>
          <td class="code">{{.Content}}</td>
        </tr>
      {{else if isDeletion .Type}}
        <tr class="deletion">
          <td class="ln">{{lineNumStr .OldLineNum}}</td>
          <td class="code">{{.Content}}</td>
        </tr>
      {{else}}
        <tr class="context">
          <td class="ln">{{lineNumStr .NewLineNum}}</td>
          <td class="code">{{.Content}}</td>
        </tr>
      {{end}}
      {{end}}
      </tbody>
    </table>
  </div>
</div>
</div>
</body>
</html>
`
