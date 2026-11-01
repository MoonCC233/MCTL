package tui

import (
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

type logsPage struct {
	offset int
	follow bool
}

func (a *App) viewLogs(w, h int) []string {
	lines := a.rt.Logs.Lines()
	state := tr("跟随中")
	if !a.logs.follow {
		state = tr("已暂停")
	}
	sub := trf("%d 行 · %s · 日志同时写入配置目录", len(lines), state)
	head := pageHeader("运行日志", sub, w)
	a.addHit(2, len(head), w-2, h-1, "wheel", "logs")
	contentH := h - len(head)
	if contentH < 4 {
		return fitLines(head, h, w)
	}
	maxRows := panelContentHeight(contentH)
	if maxRows < 1 {
		return fitLines(head, h, w)
	}
	maxOffset := max(len(lines)-maxRows, 0)
	if a.logs.follow {
		a.logs.offset = maxOffset
	} else {
		a.logs.offset = clamp(a.logs.offset, 0, maxOffset)
	}
	start := a.logs.offset
	end := min(start+maxRows, len(lines))
	var rows []string
	if len(lines) == 0 {
		rows = append(rows, "", "  "+styleFaint.Render("暂无日志 · 启动游戏或执行操作后会显示在这里"))
	} else {
		for i := start; i < end; i++ {
			rows = append(rows, renderLogLine(lines[i], i+1, panelContentWidth(w-4)))
		}
	}
	for len(rows) < maxRows {
		rows = append(rows, "")
	}
	if len(rows) > maxRows {
		rows = rows[:maxRows]
	}
	title := tr("日志")
	if a.logs.follow {
		title += tr(" · 跟随")
	}
	p := panel(title, w-4, contentH, a.page == pageLogs, rows)
	for _, l := range strings.Split(p, "\n") {
		head = append(head, "  "+l)
	}
	return fitLines(head, h, w)
}

func renderLogLine(line string, num, w int) string {
	line = ansi.Strip(line)
	gutter := styleFaint.Render(padLeft(strconv.Itoa(num), 4)) + "  "
	bodyW := max(w-8, 1)
	body := ansiTruncate(line, bodyW)
	style := styleText
	low := strings.ToLower(line)
	switch {
	case strings.Contains(low, "[error]") || strings.Contains(low, "exception") || strings.Contains(low, "caused by"):
		style = styleRed
	case strings.Contains(low, "[warn]"):
		style = styleYellow
	case strings.Contains(line, "[MCTL]"):
		style = styleAccent
	case strings.Contains(low, "downloading") || strings.Contains(low, "下载"):
		style = styleMuted
	}
	return "  " + gutter + style.Render(body)
}

func (a *App) logsKey(msg tea.KeyMsg) tea.Cmd {
	total := a.rt.Logs.Len()
	rows := max(a.height-5, 1)
	maxOffset := max(total-rows, 0)
	switch msg.String() {
	case "up", "k":
		if a.logs.follow {
			a.logs.follow = false
			a.logs.offset = max(maxOffset-1, 0)
		} else {
			a.logs.offset = max(a.logs.offset-1, 0)
		}
	case "down", "j":
		if !a.logs.follow {
			a.logs.offset++
			if a.logs.offset >= maxOffset {
				a.logs.follow = true
			}
		}
	case "pgup":
		a.logs.follow = false
		a.logs.offset = max(a.logs.offset-rows, 0)
	case "pgdown":
		a.logs.offset = min(a.logs.offset+rows, maxOffset)
		if a.logs.offset >= maxOffset {
			a.logs.follow = true
		}
	case "home":
		a.logs.follow = false
		a.logs.offset = 0
	case "end":
		a.logs.follow = true
	case "f":
		a.logs.follow = !a.logs.follow
		if a.logs.follow {
			a.logs.offset = maxOffset
		}
	case "c":
		a.rt.Logs.Clear()
		a.logs.offset = 0
		a.logs.follow = true
		a.setToast("日志已清空", 1)
	}
	return nil
}
