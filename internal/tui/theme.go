package tui

import "github.com/charmbracelet/lipgloss"

// The palette is inspired by modern terminal UIs: near-black canvas, warm
// peach primary, cool blue secondary and restrained status colors.
var (
	colorBg     = lipgloss.Color("#0d0d10")
	colorPanel  = lipgloss.Color("#16161b")
	colorPanel2 = lipgloss.Color("#1e1e25")
	colorBorder = lipgloss.Color("#2c2c34")
	colorText   = lipgloss.Color("#e8e6e3")
	colorMuted  = lipgloss.Color("#85858f")
	colorFaint  = lipgloss.Color("#55555f")
	colorAccent = lipgloss.Color("#fab283")
	colorBlue   = lipgloss.Color("#5c9cf5")
	colorGreen  = lipgloss.Color("#7fd88f")
	colorRed    = lipgloss.Color("#e06c75")
	colorYellow = lipgloss.Color("#e5c07b")
	colorPurple = lipgloss.Color("#9d7cd8")
	colorCyan   = lipgloss.Color("#56b6c2")
)

var (
	styleText     = lipgloss.NewStyle().Foreground(colorText)
	styleMuted    = lipgloss.NewStyle().Foreground(colorMuted)
	styleFaint    = lipgloss.NewStyle().Foreground(colorFaint)
	styleAccent   = lipgloss.NewStyle().Foreground(colorAccent)
	styleBlue     = lipgloss.NewStyle().Foreground(colorBlue)
	styleGreen    = lipgloss.NewStyle().Foreground(colorGreen)
	styleRed      = lipgloss.NewStyle().Foreground(colorRed)
	styleYellow   = lipgloss.NewStyle().Foreground(colorYellow)
	stylePurple   = lipgloss.NewStyle().Foreground(colorPurple)
	styleCyan     = lipgloss.NewStyle().Foreground(colorCyan)
	styleBorder   = lipgloss.NewStyle().Foreground(colorBorder)
	styleBold     = lipgloss.NewStyle().Foreground(colorText).Bold(true)
	styleTitle    = lipgloss.NewStyle().Foreground(colorText).Bold(true)
	styleSection  = lipgloss.NewStyle().Foreground(colorFaint).Bold(true)
	styleSelected = lipgloss.NewStyle().Foreground(colorText).Background(colorPanel2)
	styleKey      = lipgloss.NewStyle().Foreground(colorBg).Background(colorAccent).Padding(0, 1).Bold(true)
)
