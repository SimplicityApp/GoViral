package codeimg

import "fmt"

func init() {
	RegisterTemplate(&TemplateSpec{
		Name:                "github",
		Description:         "GitHub-style diff view",
		Selector:            ".diff-box",
		SupportsDescription: true,
		Template:            MustParseTemplate("github", githubHTMLTemplate),
	})
}

// githubHTMLTemplate is the GitHub-style diff template. Colours are injected
// from {{.Theme}} so it works with any ThemeColors palette.
const githubHTMLTemplate = `<!DOCTYPE html>
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
    padding: 24px;
    display: inline-block;
    min-width: 100%;
  }

  .diff-box {
    background: {{.Theme.CardBackground}};
    border: 1px solid {{.Theme.BorderColor}};
    border-radius: 6px;
    overflow: hidden;
    box-shadow: {{.Theme.BoxShadow}};
    width: 100%;
  }

  /* ── Header ─────────────────────────────────────────────────────────── */
  .diff-header {
    display: flex;
    align-items: center;
    gap: 10px;
    background: {{.Theme.HeaderBackground}};
    border-bottom: 1px solid {{.Theme.BorderColor}};
    padding: 8px 14px;
    flex-wrap: wrap;
  }

  .diff-filename {
    flex: 1;
    font-size: 13px;
    font-weight: 600;
    color: {{.Theme.TextPrimary}};
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .diff-lang {
    font-size: 11px;
    color: {{.Theme.TextSecondary}};
    font-weight: 400;
  }

  .diff-stat {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    font-size: 12px;
    font-weight: 600;
    white-space: nowrap;
  }

  .stat-additions {
    color: {{.Theme.StatAdditionText}};
    background: {{.Theme.StatAdditionBg}};
    padding: 1px 7px;
    border-radius: 10px;
  }

  .stat-deletions {
    color: {{.Theme.StatDeletionText}};
    background: {{.Theme.StatDeletionBg}};
    padding: 1px 7px;
    border-radius: 10px;
  }

  /* ── Diff table ──────────────────────────────────────────────────────── */
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

  .ln {
    width: 1%;
    min-width: 36px;
    padding: 0 10px;
    text-align: right;
    color: {{.Theme.GutterText}};
    user-select: none;
    border-right: 1px solid {{.Theme.GutterBorder}};
    font-size: 11px;
  }

  .code {
    padding: 0 14px;
    width: 99%;
    overflow: hidden;
  }

  /* ── Row colours ─────────────────────────────────────────────────────── */
  tr.addition td { background: {{.Theme.AdditionBg}}; }
  tr.addition td.ln {
    background: {{.Theme.AdditionGutterBg}};
    color: {{.Theme.AdditionGutter}};
    border-right-color: {{.Theme.AdditionBorder}};
  }
  tr.addition td.code { color: {{.Theme.AdditionText}}; }

  tr.deletion td { background: {{.Theme.DeletionBg}}; }
  tr.deletion td.ln {
    background: {{.Theme.DeletionGutterBg}};
    color: {{.Theme.DeletionGutter}};
    border-right-color: {{.Theme.DeletionBorder}};
  }
  tr.deletion td.code { color: {{.Theme.DeletionText}}; }

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

  tr.context td {
    background: {{.Theme.ContextBg}};
    color: {{.Theme.ContextText}};
  }
  tr.context:nth-child(even) td {
    background: {{.Theme.ContextBg}};
  }

  .sign {
    width: 18px;
    min-width: 18px;
    padding: 0 2px 0 10px;
    text-align: center;
    user-select: none;
  }
  tr.addition .sign { color: {{.Theme.AdditionSign}}; }
  tr.deletion .sign { color: {{.Theme.DeletionSign}}; }
  tr.context  .sign { color: transparent; }
  tr.hunk-header .sign { color: {{.Theme.HunkText}}; }

  .github-description {
    padding: 10px 16px;
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
<div class="diff-box">
  <div class="diff-header">
    <span class="diff-filename">
      {{.Filename}}{{if .Language}}<span class="diff-lang"> — {{.Language}}</span>{{end}}
    </span>
    <div class="diff-stat">
      <span class="stat-additions">+{{.Additions}}</span>
      <span class="stat-deletions">-{{.Deletions}}</span>
    </div>
  </div>

  {{if .Description}}
  <div class="github-description">{{.Description}}</div>
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
</div>
</body>
</html>
`

// RenderDiffHTML executes the github diff template against data and returns
// the full HTML string. Retained for backward compatibility — new callers
// should use RenderHTML(data, templateName) instead.
func RenderDiffHTML(data TemplateDiffData) (string, error) {
	// Ensure a theme is set so CSS variables are populated.
	if data.Theme.Name == "" {
		data.Theme = LookupTheme("github-dark")
	}
	html, err := RenderHTML(data, "github")
	if err != nil {
		return "", fmt.Errorf("executing diff template: %w", err)
	}
	return html, nil
}
