package codeimg

func init() {
	RegisterTemplate(&TemplateSpec{
		Name:                "macos",
		Description:         "macOS window with traffic-light buttons",
		Selector:            ".capture-root",
		SupportsDescription: true,
		Template:            MustParseTemplate("macos", macosHTMLTemplate),
	})
}

const macosHTMLTemplate = `<!DOCTYPE html>
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
    padding: 40px;
    display: inline-block;
    min-width: 100%;
  }

  .capture-root {
    display: inline-block;
  }

  .macos-window {
    background: {{.Theme.CardBackground}};
    border-radius: 12px;
    overflow: hidden;
    box-shadow:
      0 0 0 1px rgba(0,0,0,0.12),
      0 2px 4px rgba(0,0,0,0.15),
      0 8px 24px rgba(0,0,0,0.25),
      0 20px 48px rgba(0,0,0,0.3);
  }

  /* ── Title bar ────────────────────────────────────────────────────── */
  .title-bar {
    display: flex;
    align-items: center;
    padding: 12px 16px;
    background: {{.Theme.HeaderBackground}};
    border-bottom: 1px solid {{.Theme.BorderColor}};
    position: relative;
  }

  .traffic-lights {
    display: flex;
    gap: 8px;
    z-index: 1;
  }

  .dot {
    width: 12px;
    height: 12px;
    border-radius: 50%;
    position: relative;
  }

  .dot::after {
    content: "";
    position: absolute;
    top: 1px;
    left: 2px;
    width: 8px;
    height: 5px;
    border-radius: 50%;
    background: linear-gradient(180deg, rgba(255,255,255,0.5) 0%, rgba(255,255,255,0) 100%);
  }

  .dot-close {
    background: linear-gradient(180deg, #ff6058 0%, #e0443e 100%);
    box-shadow: 0 0 1px rgba(0,0,0,0.3), inset 0 -1px 1px rgba(0,0,0,0.15);
  }

  .dot-minimize {
    background: linear-gradient(180deg, #ffc130 0%, #dea123 100%);
    box-shadow: 0 0 1px rgba(0,0,0,0.3), inset 0 -1px 1px rgba(0,0,0,0.15);
  }

  .dot-maximize {
    background: linear-gradient(180deg, #27ca40 0%, #1aab29 100%);
    box-shadow: 0 0 1px rgba(0,0,0,0.3), inset 0 -1px 1px rgba(0,0,0,0.15);
  }

  .title-text {
    position: absolute;
    left: 0;
    right: 0;
    text-align: center;
    font-size: 12px;
    font-weight: 500;
    color: {{.Theme.TextSecondary}};
    pointer-events: none;
  }

  /* ── Stat bar ─────────────────────────────────────────────────────── */
  .stat-bar {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 6px 16px;
    background: {{.Theme.HeaderBackground}};
    border-bottom: 1px solid {{.Theme.BorderColor}};
    font-size: 11px;
  }

  .stat-additions {
    color: {{.Theme.StatAdditionText}};
    background: {{.Theme.StatAdditionBg}};
    padding: 1px 7px;
    border-radius: 10px;
    font-weight: 600;
  }

  .stat-deletions {
    color: {{.Theme.StatDeletionText}};
    background: {{.Theme.StatDeletionBg}};
    padding: 1px 7px;
    border-radius: 10px;
    font-weight: 600;
  }

  .stat-lang {
    color: {{.Theme.TextSecondary}};
    margin-left: auto;
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

  .macos-description {
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
<div class="capture-root">
<div class="macos-window">
  <div class="title-bar">
    <div class="traffic-lights">
      <span class="dot dot-close"></span>
      <span class="dot dot-minimize"></span>
      <span class="dot dot-maximize"></span>
    </div>
    <span class="title-text">{{.Filename}}</span>
  </div>

  <div class="stat-bar">
    <span class="stat-additions">+{{.Additions}}</span>
    <span class="stat-deletions">-{{.Deletions}}</span>
    {{if .Language}}<span class="stat-lang">{{.Language}}</span>{{end}}
  </div>

  {{if .Description}}
  <div class="macos-description">{{.Description}}</div>
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
</div>
</body>
</html>
`
