package codeimg

func init() {
	RegisterTemplate(&TemplateSpec{
		Name:                "card",
		Description:         "Social-media card with description area",
		Selector:            ".capture-root",
		SupportsDescription: true,
		Template:            MustParseTemplate("card", cardHTMLTemplate),
	})
}

const cardHTMLTemplate = `<!DOCTYPE html>
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

  .social-card {
    background: {{.Theme.CardBackground}};
    border-radius: 12px;
    overflow: hidden;
    box-shadow: {{.Theme.BoxShadow}};
    border: 1px solid {{.Theme.BorderColor}};
  }

  /* ── Card header ─────────────────────────────────────────────────── */
  .card-header {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 14px 18px;
    background: {{.Theme.HeaderBackground}};
    border-bottom: 1px solid {{.Theme.BorderColor}};
  }

  .card-avatar {
    width: 28px;
    height: 28px;
    border-radius: 50%;
    background: linear-gradient(135deg, {{.Theme.AccentColor}} 0%, {{.Theme.AccentColor}}88 100%);
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
  }

  .card-avatar-letter {
    color: #fff;
    font-size: 13px;
    font-weight: 700;
    font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Helvetica, Arial, sans-serif;
  }

  .card-repo-info {
    flex: 1;
    min-width: 0;
  }

  .card-repo-name {
    font-size: 13px;
    font-weight: 600;
    color: {{.Theme.TextPrimary}};
    font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Helvetica, Arial, sans-serif;
  }

  .card-filename {
    font-size: 11px;
    color: {{.Theme.TextSecondary}};
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .card-lang-badge {
    font-size: 10px;
    color: {{.Theme.AccentColor}};
    background: {{.Theme.AccentColor}}18;
    padding: 2px 8px;
    border-radius: 10px;
    font-weight: 600;
    white-space: nowrap;
  }

  .card-stats {
    display: flex;
    gap: 6px;
  }

  .stat-badge {
    font-size: 11px;
    font-weight: 600;
    padding: 2px 8px;
    border-radius: 10px;
  }

  .stat-badge-add {
    color: {{.Theme.StatAdditionText}};
    background: {{.Theme.StatAdditionBg}};
  }

  .stat-badge-del {
    color: {{.Theme.StatDeletionText}};
    background: {{.Theme.StatDeletionBg}};
  }

  /* ── Description area (conditional) ──────────────────────────────── */
  .card-description {
    padding: 14px 18px;
    background: {{.Theme.DescriptionBg}};
    border-bottom: 1px solid {{.Theme.BorderColor}};
    font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Helvetica, Arial, sans-serif;
    font-size: 13px;
    line-height: 1.6;
    color: {{.Theme.DescriptionText}};
  }

  /* ── Compact code ────────────────────────────────────────────────── */
  .diff-table {
    width: 100%;
    border-collapse: collapse;
    font-size: 11px;
    line-height: 1.45;
  }

  .diff-table td {
    padding: 0;
    vertical-align: top;
    white-space: pre;
  }

  .ln {
    width: 1%;
    min-width: 32px;
    padding: 0 8px;
    text-align: right;
    color: {{.Theme.GutterText}};
    user-select: none;
    font-size: 10px;
    border-right: 1px solid {{.Theme.GutterBorder}};
  }

  .sign {
    width: 16px;
    min-width: 16px;
    padding: 0 2px 0 8px;
    text-align: center;
    user-select: none;
    font-size: 10px;
  }

  .code {
    padding: 0 12px;
    width: 99%;
    overflow: hidden;
  }

  tr.addition td { background: {{.Theme.AdditionBg}}; }
  tr.addition td.ln {
    background: {{.Theme.AdditionGutterBg}};
    color: {{.Theme.AdditionGutter}};
    border-right-color: {{.Theme.AdditionBorder}};
  }
  tr.addition td.code { color: {{.Theme.AdditionText}}; }
  tr.addition .sign { color: {{.Theme.AdditionSign}}; }

  tr.deletion td { background: {{.Theme.DeletionBg}}; }
  tr.deletion td.ln {
    background: {{.Theme.DeletionGutterBg}};
    color: {{.Theme.DeletionGutter}};
    border-right-color: {{.Theme.DeletionBorder}};
  }
  tr.deletion td.code { color: {{.Theme.DeletionText}}; }
  tr.deletion .sign { color: {{.Theme.DeletionSign}}; }

  tr.hunk-header td {
    background: {{.Theme.HunkBg}};
    color: {{.Theme.HunkText}};
    font-style: italic;
    padding-top: 2px;
    padding-bottom: 2px;
  }
  tr.hunk-header td.ln {
    color: {{.Theme.HunkGutter}};
    border-right-color: {{.Theme.HunkBorder}};
  }
  tr.hunk-header .sign { color: {{.Theme.HunkText}}; }

  tr.context td {
    background: {{.Theme.ContextBg}};
    color: {{.Theme.ContextText}};
  }
  tr.context .sign { color: transparent; }

  /* ── Footer ──────────────────────────────────────────────────────── */
  .card-footer {
    padding: 8px 18px;
    border-top: 1px solid {{.Theme.BorderColor}};
    background: {{.Theme.HeaderBackground}};
    font-size: 10px;
    color: {{.Theme.TextMuted}};
    font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Helvetica, Arial, sans-serif;
    text-align: right;
  }
</style>
</head>
<body>
<div class="capture-root">
<div class="social-card">
  <div class="card-header">
    <div class="card-avatar">
      <span class="card-avatar-letter">{{if .RepoName}}{{slice .RepoName 0 1}}{{else}}G{{end}}</span>
    </div>
    <div class="card-repo-info">
      <div class="card-repo-name">{{if .RepoName}}{{.RepoName}}{{else}}Code Changes{{end}}</div>
      <div class="card-filename">{{.Filename}}</div>
    </div>
    {{if .Language}}<span class="card-lang-badge">{{.Language}}</span>{{end}}
    <div class="card-stats">
      <span class="stat-badge stat-badge-add">+{{.Additions}}</span>
      <span class="stat-badge stat-badge-del">-{{.Deletions}}</span>
    </div>
  </div>

  {{if .Description}}
  <div class="card-description">{{.Description}}</div>
  {{end}}

  <table class="diff-table">
    <tbody>
    {{range .Lines}}
    {{if isHunkHeader .Type}}
      <tr class="hunk-header">
        <td class="ln" colspan="2"></td>
        <td class="sign">@@</td>
        <td class="code">{{.Content}}</td>
      </tr>
    {{else if isAddition .Type}}
      <tr class="addition">
        <td class="ln">{{lineNumStr .OldLineNum}}</td>
        <td class="ln">{{lineNumStr .NewLineNum}}</td>
        <td class="sign">+</td>
        <td class="code">{{.Content}}</td>
      </tr>
    {{else if isDeletion .Type}}
      <tr class="deletion">
        <td class="ln">{{lineNumStr .OldLineNum}}</td>
        <td class="ln">{{lineNumStr .NewLineNum}}</td>
        <td class="sign">-</td>
        <td class="code">{{.Content}}</td>
      </tr>
    {{else}}
      <tr class="context">
        <td class="ln">{{lineNumStr .OldLineNum}}</td>
        <td class="ln">{{lineNumStr .NewLineNum}}</td>
        <td class="sign"> </td>
        <td class="code">{{.Content}}</td>
      </tr>
    {{end}}
    {{end}}
    </tbody>
  </table>

  <div class="card-footer">Built with GoViral</div>
</div>
</div>
</body>
</html>
`
