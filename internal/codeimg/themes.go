package codeimg

// ThemeColors holds all colour values needed by templates. Every template
// references these via CSS custom properties so any theme works with any
// template.
type ThemeColors struct {
	Name string

	// Body / container
	BodyBackground   string
	CardBackground   string
	HeaderBackground string
	BorderColor      string
	BoxShadow        string

	// Text
	TextPrimary   string
	TextSecondary string
	TextMuted     string

	// Diff — additions
	AdditionBg       string
	AdditionGutterBg string
	AdditionText     string
	AdditionGutter   string
	AdditionSign     string
	AdditionBorder   string

	// Diff — deletions
	DeletionBg       string
	DeletionGutterBg string
	DeletionText     string
	DeletionGutter   string
	DeletionSign     string
	DeletionBorder   string

	// Diff — hunk headers
	HunkBg     string
	HunkText   string
	HunkGutter string
	HunkBorder string

	// Diff — context lines
	ContextBg   string
	ContextText string

	// Stat badges
	StatAdditionText string
	StatAdditionBg   string
	StatDeletionText string
	StatDeletionBg   string

	// Gutter (line numbers)
	GutterText   string
	GutterBorder string

	// Template-specific accent / gradient
	AccentColor   string
	GradientStart string
	GradientEnd   string

	// Card template — description area
	DescriptionBg   string
	DescriptionText string
}

// ── Pre-defined themes ──────────────────────────────────────────────────────

var themeGitHubDark = ThemeColors{
	Name:             "github-dark",
	BodyBackground:   "#0d1117",
	CardBackground:   "#161b22",
	HeaderBackground: "#21262d",
	BorderColor:      "#30363d",
	BoxShadow:        "0 1px 3px rgba(0,0,0,0.4), 0 4px 16px rgba(0,0,0,0.3)",
	TextPrimary:      "#e6edf3",
	TextSecondary:    "#8b949e",
	TextMuted:        "#484f58",
	AdditionBg:       "#0f2d1a",
	AdditionGutterBg: "#112a1b",
	AdditionText:     "#aff5b4",
	AdditionGutter:   "#3fb950",
	AdditionSign:     "#3fb950",
	AdditionBorder:   "#1f3a27",
	DeletionBg:       "#2d0f0f",
	DeletionGutterBg: "#2a1111",
	DeletionText:     "#ffa198",
	DeletionGutter:   "#f85149",
	DeletionSign:     "#f85149",
	DeletionBorder:   "#3a1f1f",
	HunkBg:           "#1c2740",
	HunkText:         "#79c0ff",
	HunkGutter:       "#3d536b",
	HunkBorder:       "#243042",
	ContextBg:        "#161b22",
	ContextText:      "#e6edf3",
	StatAdditionText: "#3fb950",
	StatAdditionBg:   "rgba(63,185,80,0.12)",
	StatDeletionText: "#f85149",
	StatDeletionBg:   "rgba(248,81,73,0.12)",
	GutterText:       "#484f58",
	GutterBorder:     "#21262d",
	AccentColor:      "#58a6ff",
	GradientStart:    "#0d1117",
	GradientEnd:      "#161b22",
	DescriptionBg:    "#1c2128",
	DescriptionText:  "#c9d1d9",
}

var themeOneDarkPro = ThemeColors{
	Name:             "one-dark-pro",
	BodyBackground:   "#1e2127",
	CardBackground:   "#282c34",
	HeaderBackground: "#21252b",
	BorderColor:      "#3e4452",
	BoxShadow:        "0 2px 8px rgba(0,0,0,0.5), 0 8px 24px rgba(0,0,0,0.35)",
	TextPrimary:      "#abb2bf",
	TextSecondary:    "#5c6370",
	TextMuted:        "#4b5263",
	AdditionBg:       "#1a2e1a",
	AdditionGutterBg: "#1d331d",
	AdditionText:     "#98c379",
	AdditionGutter:   "#98c379",
	AdditionSign:     "#98c379",
	AdditionBorder:   "#2a3f2a",
	DeletionBg:       "#2e1a1a",
	DeletionGutterBg: "#331d1d",
	DeletionText:     "#e06c75",
	DeletionGutter:   "#e06c75",
	DeletionSign:     "#e06c75",
	DeletionBorder:   "#3f2a2a",
	HunkBg:           "#2c313a",
	HunkText:         "#61afef",
	HunkGutter:       "#4b5263",
	HunkBorder:       "#3e4452",
	ContextBg:        "#282c34",
	ContextText:      "#abb2bf",
	StatAdditionText: "#98c379",
	StatAdditionBg:   "rgba(152,195,121,0.12)",
	StatDeletionText: "#e06c75",
	StatDeletionBg:   "rgba(224,108,117,0.12)",
	GutterText:       "#4b5263",
	GutterBorder:     "#3e4452",
	AccentColor:      "#61afef",
	GradientStart:    "#1e2127",
	GradientEnd:      "#2c313a",
	DescriptionBg:    "#2c313a",
	DescriptionText:  "#abb2bf",
}

var themeDracula = ThemeColors{
	Name:             "dracula",
	BodyBackground:   "#1e1f29",
	CardBackground:   "#282a36",
	HeaderBackground: "#21222c",
	BorderColor:      "#44475a",
	BoxShadow:        "0 2px 8px rgba(0,0,0,0.6), 0 8px 24px rgba(0,0,0,0.4)",
	TextPrimary:      "#f8f8f2",
	TextSecondary:    "#6272a4",
	TextMuted:        "#44475a",
	AdditionBg:       "#1a2e1e",
	AdditionGutterBg: "#1d3321",
	AdditionText:     "#50fa7b",
	AdditionGutter:   "#50fa7b",
	AdditionSign:     "#50fa7b",
	AdditionBorder:   "#2a4030",
	DeletionBg:       "#3b1c2a",
	DeletionGutterBg: "#401e2e",
	DeletionText:     "#ff5555",
	DeletionGutter:   "#ff5555",
	DeletionSign:     "#ff5555",
	DeletionBorder:   "#4a2a38",
	HunkBg:           "#2d2f3e",
	HunkText:         "#bd93f9",
	HunkGutter:       "#44475a",
	HunkBorder:       "#44475a",
	ContextBg:        "#282a36",
	ContextText:      "#f8f8f2",
	StatAdditionText: "#50fa7b",
	StatAdditionBg:   "rgba(80,250,123,0.12)",
	StatDeletionText: "#ff5555",
	StatDeletionBg:   "rgba(255,85,85,0.12)",
	GutterText:       "#6272a4",
	GutterBorder:     "#44475a",
	AccentColor:      "#bd93f9",
	GradientStart:    "#282a36",
	GradientEnd:      "#44475a",
	DescriptionBg:    "#2d2f3e",
	DescriptionText:  "#f8f8f2",
}

var themeNord = ThemeColors{
	Name:             "nord",
	BodyBackground:   "#242933",
	CardBackground:   "#2e3440",
	HeaderBackground: "#3b4252",
	BorderColor:      "#434c5e",
	BoxShadow:        "0 2px 6px rgba(0,0,0,0.35), 0 6px 20px rgba(0,0,0,0.25)",
	TextPrimary:      "#eceff4",
	TextSecondary:    "#d8dee9",
	TextMuted:        "#4c566a",
	AdditionBg:       "#1e3328",
	AdditionGutterBg: "#21382c",
	AdditionText:     "#a3be8c",
	AdditionGutter:   "#a3be8c",
	AdditionSign:     "#a3be8c",
	AdditionBorder:   "#2e4a38",
	DeletionBg:       "#3b2328",
	DeletionGutterBg: "#40272c",
	DeletionText:     "#bf616a",
	DeletionGutter:   "#bf616a",
	DeletionSign:     "#bf616a",
	DeletionBorder:   "#4a3238",
	HunkBg:           "#353d4d",
	HunkText:         "#81a1c1",
	HunkGutter:       "#4c566a",
	HunkBorder:       "#434c5e",
	ContextBg:        "#2e3440",
	ContextText:      "#eceff4",
	StatAdditionText: "#a3be8c",
	StatAdditionBg:   "rgba(163,190,140,0.12)",
	StatDeletionText: "#bf616a",
	StatDeletionBg:   "rgba(191,97,106,0.12)",
	GutterText:       "#4c566a",
	GutterBorder:     "#3b4252",
	AccentColor:      "#88c0d0",
	GradientStart:    "#2e3440",
	GradientEnd:      "#3b4252",
	DescriptionBg:    "#353d4d",
	DescriptionText:  "#d8dee9",
}

var themeSolarizedLight = ThemeColors{
	Name:             "solarized-light",
	BodyBackground:   "#eee8d5",
	CardBackground:   "#fdf6e3",
	HeaderBackground: "#eee8d5",
	BorderColor:      "#d3cbb7",
	BoxShadow:        "0 1px 4px rgba(0,0,0,0.08), 0 4px 12px rgba(0,0,0,0.06)",
	TextPrimary:      "#657b83",
	TextSecondary:    "#93a1a1",
	TextMuted:        "#93a1a1",
	AdditionBg:       "#e6f2e6",
	AdditionGutterBg: "#d9eed9",
	AdditionText:     "#586e75",
	AdditionGutter:   "#859900",
	AdditionSign:     "#859900",
	AdditionBorder:   "#c4dcc4",
	DeletionBg:       "#f2e6e6",
	DeletionGutterBg: "#eed9d9",
	DeletionText:     "#586e75",
	DeletionGutter:   "#dc322f",
	DeletionSign:     "#dc322f",
	DeletionBorder:   "#dcc4c4",
	HunkBg:           "#e8e2cf",
	HunkText:         "#268bd2",
	HunkGutter:       "#93a1a1",
	HunkBorder:       "#d3cbb7",
	ContextBg:        "#fdf6e3",
	ContextText:      "#657b83",
	StatAdditionText: "#859900",
	StatAdditionBg:   "rgba(133,153,0,0.10)",
	StatDeletionText: "#dc322f",
	StatDeletionBg:   "rgba(220,50,47,0.10)",
	GutterText:       "#93a1a1",
	GutterBorder:     "#eee8d5",
	AccentColor:      "#268bd2",
	GradientStart:    "#fdf6e3",
	GradientEnd:      "#eee8d5",
	DescriptionBg:    "#eee8d5",
	DescriptionText:  "#657b83",
}

var themeMonokai = ThemeColors{
	Name:             "monokai",
	BodyBackground:   "#1e1e1e",
	CardBackground:   "#272822",
	HeaderBackground: "#1e1f1c",
	BorderColor:      "#3e3d32",
	BoxShadow:        "0 2px 8px rgba(0,0,0,0.5), 0 8px 24px rgba(0,0,0,0.35)",
	TextPrimary:      "#f8f8f2",
	TextSecondary:    "#75715e",
	TextMuted:        "#49483e",
	AdditionBg:       "#1e3a1e",
	AdditionGutterBg: "#224022",
	AdditionText:     "#a6e22e",
	AdditionGutter:   "#a6e22e",
	AdditionSign:     "#a6e22e",
	AdditionBorder:   "#2e4a2e",
	DeletionBg:       "#3a1e2a",
	DeletionGutterBg: "#40222e",
	DeletionText:     "#f92672",
	DeletionGutter:   "#f92672",
	DeletionSign:     "#f92672",
	DeletionBorder:   "#4a2e38",
	HunkBg:           "#2d2e27",
	HunkText:         "#66d9ef",
	HunkGutter:       "#49483e",
	HunkBorder:       "#3e3d32",
	ContextBg:        "#272822",
	ContextText:      "#f8f8f2",
	StatAdditionText: "#a6e22e",
	StatAdditionBg:   "rgba(166,226,46,0.12)",
	StatDeletionText: "#f92672",
	StatDeletionBg:   "rgba(249,38,114,0.12)",
	GutterText:       "#75715e",
	GutterBorder:     "#3e3d32",
	AccentColor:      "#66d9ef",
	GradientStart:    "#272822",
	GradientEnd:      "#3e3d32",
	DescriptionBg:    "#2d2e27",
	DescriptionText:  "#f8f8f2",
}

// themes is the canonical map of theme name → ThemeColors.
var themes = map[string]*ThemeColors{
	"github-dark":     &themeGitHubDark,
	"one-dark-pro":    &themeOneDarkPro,
	"dracula":         &themeDracula,
	"nord":            &themeNord,
	"solarized-light": &themeSolarizedLight,
	"monokai":         &themeMonokai,
}

// themeAliases maps shorthand names to canonical theme names for backward
// compatibility ("dark" was the only theme before this change).
var themeAliases = map[string]string{
	"dark":  "github-dark",
	"light": "solarized-light",
}

// LookupTheme resolves a theme by name (including aliases) and returns the
// ThemeColors. If name is empty or unrecognised it falls back to github-dark.
func LookupTheme(name string) ThemeColors {
	if canonical, ok := themeAliases[name]; ok {
		name = canonical
	}
	if t, ok := themes[name]; ok {
		return *t
	}
	return themeGitHubDark
}

// ThemeNames returns the canonical names of all available themes.
func ThemeNames() []string {
	names := make([]string, 0, len(themes))
	for n := range themes {
		names = append(names, n)
	}
	return names
}
