package codeimg

func init() {
	RegisterTemplate(&TemplateSpec{
		Name:                "terminal",
		Description:         "Retro terminal window with prompt and CRT aesthetic",
		Selector:            ".capture-root",
		SupportsDescription: true,
		Template:            MustParseTemplate("terminal", terminalHTMLTemplate),
	})
}

const terminalHTMLTemplate = `<!DOCTYPE html>
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
    padding: 36px;
    display: inline-block;
    min-width: 100%;
  }

  .capture-root {
    display: inline-block;
  }

  .terminal-window {
    background: {{.Theme.CardBackground}};
    border-radius: 8px;
    overflow: hidden;
    box-shadow:
      0 0 0 1px rgba(0,0,0,0.2),
      0 4px 12px rgba(0,0,0,0.3),
      0 16px 40px rgba(0,0,0,0.35);
    position: relative;
  }

  /* ── Scanline overlay ───────────────────────────────────────────── */
  .terminal-window::after {
    content: "";
    position: absolute;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    background: repeating-linear-gradient(
      0deg,
      transparent,
      transparent 2px,
      rgba(0,0,0,0.03) 2px,
      rgba(0,0,0,0.03) 4px
    );
    pointer-events: none;
    z-index: 10;
  }

  /* ── Title bar ──────────────────────────────────────────────────── */
  .terminal-titlebar {
    display: flex;
    align-items: center;
    padding: 10px 14px;
    background: {{.Theme.HeaderBackground}};
    border-bottom: 1px solid {{.Theme.BorderColor}};
  }

  .terminal-dots {
    display: flex;
    gap: 7px;
  }

  .terminal-dots span {
    width: 11px;
    height: 11px;
    border-radius: 50%;
    background: {{.Theme.TextMuted}};
  }

  .terminal-dots span:nth-child(1) { background: #ff5f57; }
  .terminal-dots span:nth-child(2) { background: #febc2e; }
  .terminal-dots span:nth-child(3) { background: #28c840; }

  .terminal-title {
    flex: 1;
    text-align: center;
    font-size: 12px;
    color: {{.Theme.TextSecondary}};
  }

  /* ── Terminal body ──────────────────────────────────────────────── */
  .terminal-body {
    padding: 12px 0;
  }

  .terminal-prompt {
    padding: 4px 16px 10px 16px;
    font-size: 12px;
    color: {{.Theme.AccentColor}};
    text-shadow: 0 0 6px {{.Theme.AccentColor}}44;
  }

  .terminal-prompt .prompt-char {
    color: {{.Theme.StatAdditionText}};
    margin-right: 6px;
    text-shadow: 0 0 8px {{.Theme.StatAdditionText}}44;
  }

  .terminal-prompt .prompt-cmd {
    color: {{.Theme.TextPrimary}};
    text-shadow: 0 0 4px {{.Theme.TextPrimary}}22;
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
    font-size: 11px;
  }

  .sign {
    width: 18px;
    min-width: 18px;
    padding: 0 2px 0 10px;
    text-align: center;
    user-select: none;
  }

  .code {
    padding: 0 14px;
    width: 99%;
    overflow: hidden;
  }

  tr.addition td { background: {{.Theme.AdditionBg}}; }
  tr.addition td.code {
    color: {{.Theme.AdditionText}};
    text-shadow: 0 0 4px {{.Theme.AdditionSign}}22;
  }
  tr.addition .sign {
    color: {{.Theme.AdditionSign}};
    text-shadow: 0 0 6px {{.Theme.AdditionSign}}33;
  }
  tr.addition td.ln { color: {{.Theme.AdditionGutter}}; }

  tr.deletion td { background: {{.Theme.DeletionBg}}; }
  tr.deletion td.code {
    color: {{.Theme.DeletionText}};
    text-shadow: 0 0 4px {{.Theme.DeletionSign}}22;
  }
  tr.deletion .sign {
    color: {{.Theme.DeletionSign}};
    text-shadow: 0 0 6px {{.Theme.DeletionSign}}33;
  }
  tr.deletion td.ln { color: {{.Theme.DeletionGutter}}; }

  tr.hunk-header td {
    background: {{.Theme.HunkBg}};
    color: {{.Theme.HunkText}};
    font-style: italic;
    padding-top: 2px;
    padding-bottom: 2px;
    text-shadow: 0 0 4px {{.Theme.HunkText}}22;
  }
  tr.hunk-header td.ln { color: {{.Theme.HunkGutter}}; }

  tr.context td {
    background: {{.Theme.ContextBg}};
    color: {{.Theme.ContextText}};
  }

  tr.addition .sign { color: {{.Theme.AdditionSign}}; }
  tr.deletion .sign { color: {{.Theme.DeletionSign}}; }
  tr.context  .sign { color: transparent; }
  tr.hunk-header .sign { color: {{.Theme.HunkText}}; }

  .terminal-description {
    padding: 0 0 6px 0;
    padding-left: 16px;
    padding-right: 16px;
    font-family: ui-monospace, SFMono-Regular, "SF Mono", Menlo, Consolas,
                 "Liberation Mono", monospace;
    font-size: 13px;
    line-height: 1.5;
    color: {{.Theme.DescriptionText}};
  }
</style>
</head>
<body>
<div class="capture-root">
<div class="terminal-window">
  <div class="terminal-titlebar">
    <div class="terminal-dots">
      <span></span><span></span><span></span>
    </div>
    <div class="terminal-title">bash — 80×24</div>
  </div>

  <div class="terminal-body">
    <div class="terminal-prompt">
      <span class="prompt-char">$</span>
      <span class="prompt-cmd">git diff {{.Filename}}</span>
    </div>

    {{if .Description}}
    <div class="terminal-description"># {{.Description}}</div>
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
</div>
</body>
</html>
`
