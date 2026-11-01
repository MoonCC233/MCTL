package tui

import (
	"fmt"
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

func padRight(s string, w int) string {
	if w <= 0 {
		return ""
	}
	sw := lipgloss.Width(s)
	if sw > w {
		return ansi.Truncate(s, w, "…")
	}
	return s + strings.Repeat(" ", w-sw)
}

func padLeft(s string, w int) string {
	if w <= 0 {
		return ""
	}
	sw := lipgloss.Width(s)
	if sw > w {
		return ansi.Truncate(s, w, "…")
	}
	return strings.Repeat(" ", w-sw) + s
}

func centerText(s string, w int) string {
	if w <= 0 {
		return ""
	}
	sw := lipgloss.Width(s)
	if sw >= w {
		return ansi.Truncate(s, w, "…")
	}
	left := (w - sw) / 2
	return strings.Repeat(" ", left) + s + strings.Repeat(" ", w-sw-left)
}

func fitLines(lines []string, n, w int) []string {
	out := make([]string, n)
	for i := 0; i < n; i++ {
		if i < len(lines) {
			out[i] = padRight(lines[i], w)
		} else {
			out[i] = strings.Repeat(" ", max(w, 0))
		}
	}
	return out
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// panel draws a titled rounded box with exactly w columns and h rows.
func panel(title string, w, h int, focused bool, lines []string) string {
	return panelScroll(title, w, h, focused, lines, 0, 0)
}

// panelScroll renders a panel and, when total exceeds the visible height,
// draws a scrollbar thumb on the right border.
func panelScroll(title string, w, h int, focused bool, lines []string, total, offset int) string {
	if w < 6 || h < 2 {
		return ""
	}
	border := colorBorder
	titleColor := colorMuted
	if focused {
		border = colorAccent
		titleColor = colorAccent
	}
	bs := lipgloss.NewStyle().Foreground(border)
	ts := lipgloss.NewStyle().Foreground(titleColor).Bold(focused)
	t := " " + tr(title) + " "
	if lipgloss.Width(t) > w-7 {
		t = " " + ansi.Truncate(tr(title), max(w-8, 0), "…") + " "
	}
	dashes := w - 2 - lipgloss.Width(t) - 1
	if dashes < 1 {
		dashes = 1
	}
	top := bs.Render("╭─") + ts.Render(t) + bs.Render(strings.Repeat("─", dashes)+"╮")
	out := []string{padRight(top, w)}
	inner := w - 4
	visible := h - 2
	thumbStart, thumbEnd := -1, -1
	if total > visible && visible > 0 {
		thumb := max(visible*visible/total, 1)
		thumb = min(thumb, visible)
		span := total - visible
		pos := 0
		if span > 0 {
			pos = offset * (visible - thumb) / span
		}
		thumbStart = max(pos, 0)
		thumbEnd = min(thumbStart+thumb, visible)
	}
	thumbStyle := lipgloss.NewStyle().Foreground(colorAccent)
	for i := 0; i < visible; i++ {
		line := ""
		if i < len(lines) {
			line = lines[i]
		}
		right := bs.Render("│")
		if i >= thumbStart && i < thumbEnd {
			right = thumbStyle.Render("█")
		}
		out = append(out, bs.Render("│")+" "+padRight(line, inner)+" "+right)
	}
	out = append(out, bs.Render("╰"+strings.Repeat("─", max(w-2, 0))+"╯"))
	return strings.Join(out, "\n")
}

func panelContentWidth(w int) int  { return max(w-4, 0) }
func panelContentHeight(h int) int { return max(h-2, 0) }

func joinColumns(left, right string, gap int) []string {
	ll := strings.Split(left, "\n")
	rl := strings.Split(right, "\n")
	n := max(len(ll), len(rl))
	lw := 0
	for _, l := range ll {
		lw = max(lw, lipgloss.Width(l))
	}
	out := make([]string, n)
	for i := 0; i < n; i++ {
		l, r := "", ""
		if i < len(ll) {
			l = ll[i]
		}
		if i < len(rl) {
			r = rl[i]
		}
		out[i] = padRight(l, lw) + strings.Repeat(" ", gap) + r
	}
	return out
}

func keyHint(key, label string) string {
	// Mouse-first UI: hints render as clickable labels rather than key badges.
	return styleAccent.Render(tr(label))
}

func hints(parts ...string) string {
	return strings.Join(parts, "  ")
}

func badge(text string, c lipgloss.Color) string {
	return lipgloss.NewStyle().Foreground(c).Background(colorPanel2).Padding(0, 1).Render(text)
}

func dot(c lipgloss.Color) string {
	return lipgloss.NewStyle().Foreground(c).Render("●")
}

func progressBar(percent float64, width int) string {
	if width < 2 {
		return ""
	}
	if percent < 0 {
		percent = 0
	}
	if percent > 1 {
		percent = 1
	}
	filled := int(math.Round(percent * float64(width)))
	if filled > width {
		filled = width
	}
	return lipgloss.NewStyle().Foreground(colorAccent).Render(strings.Repeat("█", filled)) +
		lipgloss.NewStyle().Foreground(colorBorder).Render(strings.Repeat("░", width-filled))
}

func formatBytes(n int64) string {
	if n <= 0 {
		return "0 B"
	}
	units := []string{"B", "KB", "MB", "GB"}
	f := float64(n)
	i := 0
	for f >= 1024 && i < len(units)-1 {
		f /= 1024
		i++
	}
	if i == 0 {
		return fmt.Sprintf("%d B", n)
	}
	return fmt.Sprintf("%.1f %s", f, units[i])
}

func middleEllipsis(s string, w int) string {
	if w <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= w {
		return s
	}
	if w <= 1 {
		return "…"
	}
	r := []rune(s)
	keep := w - 1
	left := keep / 2
	right := keep - left
	if left+right > len(r) {
		return s
	}
	return string(r[:left]) + "…" + string(r[len(r)-right:])
}

func wrapText(s string, w int) []string {
	if w <= 0 {
		return nil
	}
	wrapped := ansi.Wrap(s, w, "")
	return strings.Split(wrapped, "\n")
}

func overlayCenter(base, box string, width, height int) string {
	baseLines := strings.Split(base, "\n")
	if len(baseLines) < height {
		baseLines = append(baseLines, make([]string, height-len(baseLines))...)
	}
	boxLines := strings.Split(box, "\n")
	boxW := 0
	for _, l := range boxLines {
		boxW = max(boxW, lipgloss.Width(l))
	}
	boxH := len(boxLines)
	if boxW <= 0 || boxH <= 0 {
		return base
	}
	x := max((width-boxW)/2, 0)
	y := max((height-boxH)/2, 0)
	for i := 0; i < boxH && y+i < len(baseLines); i++ {
		line := padRight(baseLines[y+i], width)
		left := ansi.Cut(line, 0, x)
		right := ansi.Cut(line, x+boxW, width)
		bl := padRight(boxLines[i], boxW)
		baseLines[y+i] = padRight(left+bl+right, width)
	}
	return strings.Join(baseLines, "\n")
}

func spinnerFrame(frame int) string {
	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	return frames[frame%len(frames)]
}

func spinner(frame int, label string) string {
	return lipgloss.NewStyle().Foreground(colorAccent).Render(spinnerFrame(frame)) + " " + styleMuted.Render(label)
}

func sectionLabel(s string) string {
	return styleSection.Render(strings.ToUpper(tr(s)))
}

func titleLine(title string) string {
	return styleAccent.Render("▌") + " " + styleTitle.Render(tr(title))
}

func statusColor(level int) lipgloss.Color {
	switch level {
	case 1:
		return colorGreen
	case 2:
		return colorRed
	case 3:
		return colorYellow
	default:
		return colorBlue
	}
}
