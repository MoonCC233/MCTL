package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/MoonCC233/MCTL/internal/config"
	"github.com/MoonCC233/MCTL/internal/minecraft"
)

type homeAction struct {
	label  string
	desc   string
	action string
	value  string
}

func homeActions() []homeAction {
	return []homeAction{
		{label: "启动游戏", desc: "使用当前实例与账户启动", action: "launch"},
		{label: "下载版本", desc: "原版与 Fabric / Forge / NeoForge 等加载器", action: "page", value: "download"},
		{label: "新建实例", desc: "独立目录、版本、加载器与 Java", action: "new-instance"},
		{label: "导入整合包", desc: "Modrinth .mrpack 或 CurseForge 压缩包", action: "import"},
		{label: "打开游戏目录", desc: "在系统文件管理器中打开", action: "open-game"},
		{label: "检测 Java", desc: "列出所有检测到的运行时", action: "java"},
	}
}

// viewHome renders the landing page: release news, shortcuts and status.
func (a *App) viewHome(w, h int) []string {
	cfg := a.store.Snapshot()
	sub := "最新动态、快捷入口与运行状态"
	if len(a.launch.versions) > 0 {
		sub = trf("已安装 %d 个版本 · 最新发布与常用操作", len(a.launch.versions))
	}
	lines := pageHeader("主页", sub, w)
	contentH := h - len(lines)
	if contentH < 6 {
		return fitLines(lines, h, w)
	}
	gap := 1
	leftW := max((w-4-gap)*58/100, 34)
	if leftW > w-4-gap-28 {
		leftW = w - 4 - gap - 28
	}
	rightW := w - 4 - gap - leftW
	if rightW < 26 {
		rightW = 26
		leftW = w - 4 - gap - rightW
	}
	if leftW < 20 {
		leftW = max((w-4-gap)/2, 18)
		rightW = w - 4 - gap - leftW
	}

	newsH := contentH
	left := a.renderHomeNews(2, len(lines), leftW, newsH)
	actionH := min(len(homeActions())+2, max(contentH-7, 5))
	statusH := min(7, contentH-actionH-1)
	if statusH < 5 {
		statusH = 0
		actionH = contentH
	}
	rightTop := a.renderHomeActions(2+leftW+gap, len(lines), rightW, actionH)
	var rightLines []string
	rightLines = append(rightLines, strings.Split(rightTop, "\n")...)
	if statusH > 0 {
		rightLines = append(rightLines, "")
		rightLines = append(rightLines, strings.Split(a.renderHomeStatus(2+leftW+gap, len(lines)+actionH+1, rightW, statusH, cfg), "\n")...)
	}
	cols := joinColumns(left, strings.Join(rightLines, "\n"), gap)
	for _, l := range cols {
		lines = append(lines, "  "+l)
	}
	return fitLines(lines, h, w)
}

func (a *App) renderHomeNews(x, y, w, h int) string {
	p := &a.launch
	inner := panelContentWidth(w)
	var rows []string
	switch {
	case len(p.news) == 0 && p.newsErr != nil:
		rows = append(rows, "", "  "+styleRed.Render(tr("新闻获取失败")), "  "+styleFaint.Render(tr("单击此处重试")))
	case len(p.news) == 0:
		rows = append(rows, "", "  "+styleFaint.Render(tr("正在获取新闻…")))
	default:
		for _, rel := range p.news {
			date := ""
			if t, err := time.Parse(time.RFC3339, rel.PublishedAt); err == nil {
				date = t.Format("01-02")
			}
			title := strings.TrimSpace(rel.Name)
			if title == "" {
				title = rel.TagName
			}
			if title == "" {
				continue
			}
			label := ansiTruncate(title, max(inner-12, 8))
			line := "  " + styleFaint.Render(date) + " " + styleAccent.Render("▸") + " " + styleText.Render(label)
			if date == "" {
				line = "  " + styleAccent.Render("▸") + " " + styleText.Render(ansiTruncate(title, max(inner-6, 8)))
			}
			rows = append(rows, line)
		}
		rows = append(rows, "", "  "+styleFaint.Render(tr("单击查看完整更新日志")))
	}
	for len(rows) < panelContentHeight(h) {
		rows = append(rows, "")
	}
	if len(rows) > panelContentHeight(h) {
		rows = rows[:panelContentHeight(h)]
	}
	a.addHit(x, y, x+w-1, y+h-1, "home-news", nil)
	return panel("新闻 / 公告", w, h, a.page == pageHome, rows)
}

func (a *App) renderHomeActions(x, y, w, h int) string {
	items := homeActions()
	var rows []string
	for i, it := range items {
		row := "  " + styleAccent.Render("▸") + " " + styleText.Render(padRight(tr(it.label), 14)) + " " + styleFaint.Render(ansiTruncate(tr(it.desc), max(panelContentWidth(w)-20, 8)))
		if a.hoverRow("home-action", i) {
			rows = append(rows, lipglossRow(row, panelContentWidth(w)))
		} else {
			rows = append(rows, row)
		}
	}
	for len(rows) < panelContentHeight(h) {
		rows = append(rows, "")
	}
	if len(rows) > panelContentHeight(h) {
		rows = rows[:panelContentHeight(h)]
	}
	for i := range items {
		rowY := y + 1 + i
		if rowY <= y+h-2 {
			a.addHit(x+1, rowY, x+w-2, rowY, "home-action", items[i])
		}
	}
	return panel("快捷操作", w, h, false, rows)
}

func (a *App) renderHomeStatus(x, y, w, h int, cfg config.Config) string {
	inner := panelContentWidth(w)
	java := tr("未检测到")
	if j := a.rt.Java(); j != nil {
		java = fmt.Sprintf("Java %d · %s", j.Major, j.Vendor)
	}
	account := tr("未选择")
	for _, acc := range cfg.Accounts {
		if acc.Name == cfg.SelectedAccount {
			account = acc.Name
			break
		}
	}
	version := cfg.SelectedVersion
	if version == "" {
		version = tr("未选择")
	}
	rows := []string{
		"  " + styleMuted.Render(padRight(tr("Java"), 8)) + styleText.Render(ansiTruncate(java, max(inner-12, 8))),
		"  " + styleMuted.Render(padRight(tr("账户"), 8)) + styleText.Render(ansiTruncate(account, max(inner-12, 8))),
		"  " + styleMuted.Render(padRight(tr("实例"), 8)) + styleText.Render(ansiTruncate(version, max(inner-12, 8))),
		"  " + styleMuted.Render(padRight(tr("下载源"), 8)) + styleText.Render(tr(minecraft.GetSource(cfg.DownloadSource).Name)),
	}
	if h > 6 {
		rows = append(rows, "  "+styleFaint.Render(ansiTruncate(middleEllipsis(cfg.GameDir, max(inner-2, 8)), max(inner-2, 8))))
	}
	for len(rows) < panelContentHeight(h) {
		rows = append(rows, "")
	}
	if len(rows) > panelContentHeight(h) {
		rows = rows[:panelContentHeight(h)]
	}
	return panel("运行状态", w, h, false, rows)
}
