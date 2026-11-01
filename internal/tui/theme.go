package tui

import "github.com/charmbracelet/lipgloss"

// themePalette is one terminal color scheme.
type themePalette struct {
	ID     string
	Name   string
	Bg     string
	Panel  string
	Panel2 string
	Border string
	Text   string
	Muted  string
	Faint  string
	Accent string
	Blue   string
	Green  string
	Red    string
	Yellow string
	Purple string
	Cyan   string
}

var themes = []themePalette{
	{
		ID: "opencode", Name: "OpenCode 暗色",
		Bg: "#0d0d10", Panel: "#16161b", Panel2: "#1e1e25", Border: "#2c2c34",
		Text: "#e8e6e3", Muted: "#85858f", Faint: "#55555f", Accent: "#fab283",
		Blue: "#5c9cf5", Green: "#7fd88f", Red: "#e06c75", Yellow: "#e5c07b",
		Purple: "#9d7cd8", Cyan: "#56b6c2",
	},
	{
		ID: "midnight", Name: "午夜蓝",
		Bg: "#080d18", Panel: "#101a2c", Panel2: "#17243b", Border: "#24324a",
		Text: "#dbe7f6", Muted: "#7f92ad", Faint: "#4f5f77", Accent: "#6ed3ff",
		Blue: "#7aa2f7", Green: "#8fd694", Red: "#f07178", Yellow: "#e0c080",
		Purple: "#b39ddb", Cyan: "#7fdbca",
	},
	{
		ID: "light", Name: "浅色",
		Bg: "#f7f4ee", Panel: "#fffdf9", Panel2: "#ece7dc", Border: "#c9c2b4",
		Text: "#23211d", Muted: "#615c52", Faint: "#8d867a", Accent: "#b8541f",
		Blue: "#2563eb", Green: "#2f7d45", Red: "#b3362c", Yellow: "#8a6400",
		Purple: "#7048a8", Cyan: "#0e7490",
	},
}

var currentTheme = themes[0]

var (
	colorBg     = lipgloss.Color(currentTheme.Bg)
	colorPanel  = lipgloss.Color(currentTheme.Panel)
	colorPanel2 = lipgloss.Color(currentTheme.Panel2)
	colorBorder = lipgloss.Color(currentTheme.Border)
	colorText   = lipgloss.Color(currentTheme.Text)
	colorMuted  = lipgloss.Color(currentTheme.Muted)
	colorFaint  = lipgloss.Color(currentTheme.Faint)
	colorAccent = lipgloss.Color(currentTheme.Accent)
	colorBlue   = lipgloss.Color(currentTheme.Blue)
	colorGreen  = lipgloss.Color(currentTheme.Green)
	colorRed    = lipgloss.Color(currentTheme.Red)
	colorYellow = lipgloss.Color(currentTheme.Yellow)
	colorPurple = lipgloss.Color(currentTheme.Purple)
	colorCyan   = lipgloss.Color(currentTheme.Cyan)
)

var (
	styleText     lipgloss.Style
	styleMuted    lipgloss.Style
	styleFaint    lipgloss.Style
	styleAccent   lipgloss.Style
	styleBlue     lipgloss.Style
	styleGreen    lipgloss.Style
	styleRed      lipgloss.Style
	styleYellow   lipgloss.Style
	stylePurple   lipgloss.Style
	styleCyan     lipgloss.Style
	styleBorder   lipgloss.Style
	styleBold     lipgloss.Style
	styleTitle    lipgloss.Style
	styleSection  lipgloss.Style
	styleSelected lipgloss.Style
	styleKey      lipgloss.Style
)

func init() {
	setStyles()
}

func setStyles() {
	styleText = lipgloss.NewStyle().Foreground(colorText)
	styleMuted = lipgloss.NewStyle().Foreground(colorMuted)
	styleFaint = lipgloss.NewStyle().Foreground(colorFaint)
	styleAccent = lipgloss.NewStyle().Foreground(colorAccent)
	styleBlue = lipgloss.NewStyle().Foreground(colorBlue)
	styleGreen = lipgloss.NewStyle().Foreground(colorGreen)
	styleRed = lipgloss.NewStyle().Foreground(colorRed)
	styleYellow = lipgloss.NewStyle().Foreground(colorYellow)
	stylePurple = lipgloss.NewStyle().Foreground(colorPurple)
	styleCyan = lipgloss.NewStyle().Foreground(colorCyan)
	styleBorder = lipgloss.NewStyle().Foreground(colorBorder)
	styleBold = lipgloss.NewStyle().Foreground(colorText).Bold(true)
	styleTitle = lipgloss.NewStyle().Foreground(colorText).Bold(true)
	styleSection = lipgloss.NewStyle().Foreground(colorFaint).Bold(true)
	styleSelected = lipgloss.NewStyle().Foreground(colorText).Background(colorPanel2)
	styleKey = lipgloss.NewStyle().Foreground(colorBg).Background(colorAccent).Padding(0, 1).Bold(true)
}

// ThemeIDs lists the available theme identifiers.
func ThemeIDs() []string {
	out := make([]string, 0, len(themes))
	for _, t := range themes {
		out = append(out, t.ID)
	}
	return out
}

// ThemeName returns a display name, falling back to the default theme.
func ThemeName(id string) string {
	if language == "en" {
		switch NormalizeTheme(id) {
		case "midnight":
			return "Midnight"
		case "light":
			return "Light"
		default:
			return "OpenCode Dark"
		}
	}
	return paletteFor(id).Name
}

// NormalizeTheme maps unknown identifiers onto the default theme.
func NormalizeTheme(id string) string {
	for _, t := range themes {
		if t.ID == id {
			return id
		}
	}
	return themes[0].ID
}

func paletteFor(id string) themePalette {
	id = NormalizeTheme(id)
	for _, t := range themes {
		if t.ID == id {
			return t
		}
	}
	return themes[0]
}

// ApplyTheme switches the global terminal palette.
func ApplyTheme(id string) {
	p := paletteFor(id)
	currentTheme = p
	colorBg = lipgloss.Color(p.Bg)
	colorPanel = lipgloss.Color(p.Panel)
	colorPanel2 = lipgloss.Color(p.Panel2)
	colorBorder = lipgloss.Color(p.Border)
	colorText = lipgloss.Color(p.Text)
	colorMuted = lipgloss.Color(p.Muted)
	colorFaint = lipgloss.Color(p.Faint)
	colorAccent = lipgloss.Color(p.Accent)
	colorBlue = lipgloss.Color(p.Blue)
	colorGreen = lipgloss.Color(p.Green)
	colorRed = lipgloss.Color(p.Red)
	colorYellow = lipgloss.Color(p.Yellow)
	colorPurple = lipgloss.Color(p.Purple)
	colorCyan = lipgloss.Color(p.Cyan)
	setStyles()
}

// CurrentTheme returns the active theme identifier.
func CurrentTheme() string { return currentTheme.ID }
