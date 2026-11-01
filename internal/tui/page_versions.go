package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/MoonCC233/MCTL/internal/config"
	"github.com/MoonCC233/MCTL/internal/minecraft"
)

type versionsPage struct {
	list   []minecraft.InstalledVersion
	idx    int
	filter string
}

func (a *App) filteredVersions() []minecraft.InstalledVersion {
	f := strings.ToLower(strings.TrimSpace(a.versions.filter))
	if f == "" {
		return a.versions.list
	}
	var out []minecraft.InstalledVersion
	for _, v := range a.versions.list {
		hay := strings.ToLower(v.ID + " " + v.Name + " " + v.Loader + " " + v.MCVersion + " " + v.Type)
		if strings.Contains(hay, f) {
			out = append(out, v)
		}
	}
	return out
}

func (a *App) viewVersions(w, h int) []string {
	list := a.filteredVersions()
	sub := trf("已安装 %d 个版本", len(a.versions.list))
	if a.versions.filter != "" {
		sub += trf(" · 筛选：%s", a.versions.filter)
	}
	lines := pageHeader("版本管理", sub, w)
	contentH := h - len(lines)
	if contentH < 5 {
		return fitLines(lines, h, w)
	}
	if a.versions.idx >= len(list) {
		a.versions.idx = max(len(list)-1, 0)
	}
	gap := 1
	leftW := max((w-4)*58/100, 34)
	if leftW > w-4-26 {
		leftW = w - 4 - 26
	}
	rightW := w - 4 - gap - leftW
	if rightW < 20 {
		rightW = max(w-4-gap-leftW, 16)
		leftW = w - 4 - gap - rightW
	}
	left := a.renderVersionList(leftW, contentH, list)
	right := a.renderVersionDetail(rightW, contentH, list)
	cols := joinColumns(left, right, gap)
	for _, l := range cols {
		lines = append(lines, "  "+l)
	}
	return fitLines(lines, h, w)
}

func (a *App) renderVersionList(w, h int, list []minecraft.InstalledVersion) string {
	inner := panelContentWidth(w)
	maxRows := panelContentHeight(h)
	if maxRows < 1 {
		return panel("已安装", w, h, false, nil)
	}
	var rows []string
	if len(list) == 0 {
		rows = append(rows, "", "  "+styleFaint.Render("没有已安装的版本"), "  "+styleFaint.Render("按 n 前往下载页安装"))
	} else {
		start := 0
		if a.versions.idx >= maxRows {
			start = a.versions.idx - maxRows + 1
		}
		if start > len(list)-maxRows {
			start = max(len(list)-maxRows, 0)
		}
		end := min(start+maxRows, len(list))
		for i := start; i < end; i++ {
			v := list[i]
			sel := i == a.versions.idx
			nameW := max(inner-27, 10)
			if !v.Complete {
				nameW--
			}
			label := v.ID
			if v.Name != "" {
				label = v.Name + " · " + v.ID
			}
			if v.Icon != "" {
				label = v.Icon + " " + label
			}
			name := middleEllipsis(label, nameW)
			loader := ansiTruncate(v.Loader, 9)
			mc := ansiTruncate(v.MCVersion, 9)
			marker := " "
			if !v.Complete {
				marker = styleYellow.Render("!")
			}
			var line string
			if sel {
				line = lipglossRow("  ▸ "+padRight(name, nameW)+"  "+padRight(loader, 9)+"  "+padRight(mc, 9)+marker, inner)
			} else {
				line = "    " + padRight(name, nameW) + "  " + styleMuted.Render(padRight(loader, 9)) + "  " + styleFaint.Render(padRight(mc, 9)) + marker
			}
			rows = append(rows, line)
		}
	}
	for len(rows) < panelContentHeight(h) {
		rows = append(rows, "")
	}
	if len(rows) > panelContentHeight(h) {
		rows = rows[:panelContentHeight(h)]
	}
	return panel(trf("已安装 · %d", len(a.versions.list)), w, h, a.page == pageVersions, rows)
}

func lipglossRow(s string, w int) string {
	s = padRight(s, w)
	return styleSelected.Width(w).Render(s)
}

func (a *App) renderVersionDetail(w, h int, list []minecraft.InstalledVersion) string {
	inner := panelContentWidth(w)
	var rows []string
	if len(list) == 0 || a.versions.idx >= len(list) {
		rows = append(rows, "", "  "+styleFaint.Render("选择左侧版本查看详情"))
	} else {
		v := list[a.versions.idx]
		rows = append(rows,
			"",
			"  "+styleBold.Render(ansiTruncate(v.ID, inner-4)),
			"  "+styleMuted.Render(fmt.Sprintf("%s · Minecraft %s", v.Loader, v.MCVersion)),
			"",
			kv("类型", v.Type, inner),
			kv("Java", javaReq(v), inner),
			kv("大小", formatBytes(v.Size), inner),
			kv("模组", fmt.Sprintf("%d 个（共享）", v.Mods), inner),
			kv("发布", shortDate(v.ReleaseTime), inner),
			"",
		)
		if v.Name != "" {
			rows = append(rows, kv("名称", v.Name, inner))
		}
		if v.Icon != "" {
			rows = append(rows, kv("图标", v.Icon, inner))
		}
		if v.Link != "" {
			rows = append(rows, kv("主页", v.Link, inner))
		}
		rows = append(rows,
			"  "+styleMuted.Render("版本路径"),
			"  "+styleFaint.Render(middleEllipsis(v.Dir, inner-4)),
		)
		if v.GameDir != "" {
			rows = append(rows,
				"  "+styleMuted.Render("实例路径"),
				"  "+styleFaint.Render(middleEllipsis(v.GameDir, inner-4)),
			)
		}
		if !v.Complete {
			rows = append(rows, "", "  "+styleYellow.Render("⚠ 客户端文件不完整，启动时会自动补全"))
		}
		rows = append(rows, "", "  "+styleBorder.Render(strings.Repeat("─", max(inner-4, 0))))
		rows = append(rows,
			"  "+styleAccent.Render("enter")+" "+styleMuted.Render("启动游戏"),
			"  "+styleAccent.Render("d")+"     "+styleMuted.Render("删除版本"),
			"  "+styleAccent.Render("o")+"     "+styleMuted.Render("打开版本目录"),
			"  "+styleAccent.Render("m")+"     "+styleMuted.Render("打开模组目录"),
			"  "+styleAccent.Render("p")+"     "+styleMuted.Render("打开资源包目录"),
			"  "+styleAccent.Render("s")+"     "+styleMuted.Render("打开存档目录"),
			"  "+styleAccent.Render("h")+"     "+styleMuted.Render("打开截图目录"),
			"  "+styleAccent.Render("f")+"     "+styleMuted.Render("打开配置目录"),
			"  "+styleAccent.Render("i")+"     "+styleMuted.Render("实例设置"),
			"  "+styleAccent.Render("l")+"     "+styleMuted.Render("实例安装 / 修复"),
			"  "+styleAccent.Render("x")+"     "+styleMuted.Render("导出实例"),
			"  "+styleAccent.Render("+")+"     "+styleMuted.Render("新建实例"),
		)
	}
	for len(rows) < panelContentHeight(h) {
		rows = append(rows, "")
	}
	if len(rows) > panelContentHeight(h) {
		rows = rows[:panelContentHeight(h)]
	}
	return panel("详情", w, h, false, rows)
}

func versionSubdir(key string) string {
	switch key {
	case "m":
		return "mods"
	case "p":
		return "resourcepacks"
	case "s":
		return "saves"
	case "h":
		return "screenshots"
	case "f":
		return "config"
	}
	return ""
}

func (a *App) openVersionSubdir(v minecraft.InstalledVersion, sub string) {
	if sub == "" {
		return
	}
	root := v.GameDir
	if root == "" {
		root = a.store.Snapshot().GameDir
	}
	dir := filepath.Join(root, sub)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		a.setToast("无法创建目录: "+err.Error(), 2)
		return
	}
	if err := openPath(dir); err != nil {
		a.setToast("打开目录失败: "+err.Error(), 2)
	}
}

func kv(k, v string, w int) string {
	kw := 8
	return "  " + styleMuted.Render(padRight(tr(k), kw)) + "  " + styleText.Render(ansiTruncate(v, max(w-kw-6, 8)))
}

func javaReq(v minecraft.InstalledVersion) string {
	if v.JavaMajor > 0 {
		return fmt.Sprintf("Java %d+", v.JavaMajor)
	}
	return "自动"
}

func shortDate(v any) string {
	switch s := v.(type) {
	case time.Time:
		if s.IsZero() {
			return "未知"
		}
		return s.Format("2006-01-02")
	case string:
		if s == "" {
			return "未知"
		}
		t, err := time.Parse(time.RFC3339, s)
		if err != nil {
			return ansiTruncate(s, 10)
		}
		return t.Format("2006-01-02")
	}
	return "未知"
}

func (a *App) versionsKey(msg tea.KeyMsg) tea.Cmd {
	list := a.filteredVersions()
	switch msg.String() {
	case "up", "k":
		if len(list) > 0 {
			a.versions.idx = (a.versions.idx - 1 + len(list)) % len(list)
		}
	case "down", "j":
		if len(list) > 0 {
			a.versions.idx = (a.versions.idx + 1) % len(list)
		}
	case "pgup":
		a.versions.idx = max(a.versions.idx-8, 0)
	case "pgdown":
		if len(list) > 0 {
			a.versions.idx = min(a.versions.idx+8, len(list)-1)
		}
	case "home":
		a.versions.idx = 0
	case "end":
		if len(list) > 0 {
			a.versions.idx = len(list) - 1
		}
	case "enter":
		if len(list) > 0 {
			v := list[a.versions.idx]
			a.saveCfg(func(c *config.Config) { c.SelectedVersion = v.ID })
			a.startLaunch(v.ID)
		}
	case "d":
		if len(list) > 0 {
			v := list[a.versions.idx]
			a.openConfirm("删除版本", fmt.Sprintf("确定删除版本 %s 吗？此操作会移除版本目录，无法撤销。", v.ID), "delete-version", v.ID)
		}
	case "i":
		if len(list) > 0 {
			v := list[a.versions.idx]
			gameDir := v.GameDir
			if gameDir == "" {
				gameDir = a.store.Snapshot().GameDir
			}
			a.showInstanceSettings(gameDir, v.ID, "")
		}
	case "l":
		if len(list) > 0 {
			a.saveCfg(func(c *config.Config) { c.SelectedVersion = list[a.versions.idx].ID })
			a.openInstanceInstall()
		}
	case "x":
		if len(list) > 0 {
			v := list[a.versions.idx]
			a.saveCfg(func(c *config.Config) { c.SelectedVersion = v.ID })
			a.openExportInstance()
		}
	case "+":
		a.openInstanceWizard()
	case "o":
		if len(list) > 0 {
			if err := openPath(list[a.versions.idx].Dir); err != nil {
				a.setToast("打开目录失败: "+err.Error(), 2)
			}
		} else {
			a.openGameDir()
		}
	case "m", "p", "s", "h", "f":
		if len(list) > 0 {
			a.openVersionSubdir(list[a.versions.idx], versionSubdir(msg.String()))
		}
	case "n":
		a.switchPage(pageDownload)
	case "r":
		a.rt.ScanVersions()
		a.setToast("正在刷新版本列表", 0)
	case "/":
		a.modal = modalState{Kind: modalInput, Title: "筛选版本", Label: "名称 / 加载器 / 版本号", Value: []rune(a.versions.filter), Placeholder: "输入关键字，留空显示全部", Action: "filter-versions"}
	case "c":
		a.versions.filter = ""
		a.versions.idx = 0
	}
	return nil
}
