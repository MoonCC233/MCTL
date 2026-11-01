package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/MoonCC233/MCTL/internal/minecraft"
)

type downloadPage struct {
	mode     int // 0 vanilla, 1 fabric, 2 quilt, 3 forge, 4 neoforge
	manifest *minecraft.VersionManifest
	err      error
	loading  bool
	show     int // 0 release, 1 snapshot, 2 all
	idx      int
	filter   string

	loaderLoading bool
	installed     map[string]bool
}

var loaderModeNames = []string{"Minecraft 原版", "Fabric", "Quilt", "Legacy Fabric", "Forge", "NeoForge", "OptiFine"}

func (p *downloadPage) applyManifest(m *minecraft.VersionManifest, err error) {
	p.loading = false
	if err != nil {
		p.err = err
		return
	}
	p.err = nil
	p.manifest = m
	list := p.versions()
	if p.idx >= len(list) {
		p.idx = max(len(list)-1, 0)
	}
}

func (p *downloadPage) versions() []minecraft.VersionEntry {
	if p.manifest == nil {
		return nil
	}
	f := strings.ToLower(strings.TrimSpace(p.filter))
	var out []minecraft.VersionEntry
	for _, v := range p.manifest.Versions {
		if p.mode != 1 {
			switch p.show {
			case 0:
				if v.Type != "release" {
					continue
				}
			case 1:
				if v.Type == "release" {
					continue
				}
			}
		}
		if f != "" {
			hay := strings.ToLower(v.ID + " " + v.Type)
			if !strings.Contains(hay, f) {
				continue
			}
		}
		out = append(out, v)
	}
	return out
}

func (a *App) viewDownload(w, h int) []string {
	p := &a.download
	modeName := loaderModeNames[clamp(p.mode, 0, len(loaderModeNames)-1)]
	sub := "正在获取版本列表…"
	if p.manifest != nil {
		sub = fmt.Sprintf("%s · %d 个版本 · 下载源 %s", modeName, len(p.versions()), minecraft.GetSource(a.store.Snapshot().DownloadSource).Name)
	}
	if p.filter != "" {
		sub += " · 搜索：" + p.filter
	}
	lines := pageHeader("下载", sub, w)
	lines = append(lines, "  "+a.renderDownloadTabs())
	contentH := h - len(lines)
	if contentH < 5 {
		return fitLines(lines, h, w)
	}
	if p.loading {
		for i := 0; i < contentH; i++ {
			if i == contentH/2 {
				lines = append(lines, centerText(spinner(a.frame, "正在获取版本列表…"), w))
			} else {
				lines = append(lines, "")
			}
		}
		return fitLines(lines, h, w)
	}
	if p.err != nil && p.manifest == nil {
		lines = append(lines, "", "  "+styleRed.Render("获取版本列表失败"), "  "+styleMuted.Render(ansiTruncate(p.err.Error(), w-4)), "", "  "+keyHint("r", "重试"))
		return fitLines(lines, h, w)
	}
	list := p.versions()
	if p.idx >= len(list) {
		p.idx = max(len(list)-1, 0)
	}
	gap := 1
	leftW := max((w-4)*58/100, 34)
	rightW := w - 4 - gap - leftW
	if rightW < 22 {
		rightW = 22
		leftW = w - 4 - gap - rightW
	}
	left := a.renderDownloadList(leftW, contentH, list)
	right := a.renderDownloadDetail(rightW, contentH, list)
	cols := joinColumns(left, right, gap)
	for _, l := range cols {
		lines = append(lines, "  "+l)
	}
	return fitLines(lines, h, w)
}

func (a *App) renderDownloadTabs() string {
	p := &a.download
	tab := func(label string, selected bool) string {
		if selected {
			return lipgloss.NewStyle().Foreground(colorAccent).Background(colorPanel2).Bold(true).Padding(0, 1).Render(label)
		}
		return lipgloss.NewStyle().Foreground(colorMuted).Padding(0, 1).Render(label)
	}
	return tab("原版", p.mode == 0) + " " +
		tab("Fabric", p.mode == 1) + " " +
		tab("Quilt", p.mode == 2) + " " +
		tab("Legacy", p.mode == 3) + " " +
		tab("Forge", p.mode == 4) + " " +
		tab("NeoForge", p.mode == 5) + " " +
		tab("OptiFine", p.mode == 6)
}

func (a *App) renderDownloadList(w, h int, list []minecraft.VersionEntry) string {
	inner := panelContentWidth(w)
	maxRows := panelContentHeight(h)
	var rows []string
	if len(list) == 0 {
		rows = append(rows, "", "  "+styleFaint.Render("没有匹配的版本"))
	} else {
		start := 0
		if a.download.idx >= maxRows {
			start = a.download.idx - maxRows + 1
		}
		if start > len(list)-maxRows {
			start = max(len(list)-maxRows, 0)
		}
		end := min(start+maxRows, len(list))
		for i := start; i < end; i++ {
			v := list[i]
			sel := i == a.download.idx
			mark := " "
			if a.download.installed[v.ID] {
				mark = styleGreen.Render("✓")
			}
			nameW := max(inner-23, 10)
			name := ansiTruncate(v.ID, nameW)
			typ := typeLabel(v.Type)
			date := shortDate(v.ReleaseTime)
			if sel {
				line := " " + mark + " " + padRight(name, nameW) + "  " + padRight(typ, 6) + "  " + padRight(date, 10)
				rows = append(rows, lipglossRow(line, inner))
			} else {
				rows = append(rows, " "+mark+" "+padRight(name, nameW)+"  "+styleMuted.Render(padRight(typ, 6))+"  "+styleFaint.Render(padRight(date, 10)))
			}
		}
	}
	for len(rows) < panelContentHeight(h) {
		rows = append(rows, "")
	}
	if len(rows) > panelContentHeight(h) {
		rows = rows[:panelContentHeight(h)]
	}
	title := "版本列表"
	if a.download.mode != 0 {
		title = "选择游戏版本"
	}
	return panel(fmt.Sprintf("%s · %d", title, len(list)), w, h, a.page == pageDownload, rows)
}

func typeLabel(t string) string {
	switch t {
	case "release":
		return "正式"
	case "snapshot":
		return "快照"
	case "old_beta":
		return "Beta"
	case "old_alpha":
		return "Alpha"
	default:
		if t == "" {
			return "未知"
		}
		return ansiTruncate(t, 6)
	}
}

func (a *App) renderDownloadDetail(w, h int, list []minecraft.VersionEntry) string {
	inner := panelContentWidth(w)
	var rows []string
	if len(list) == 0 || a.download.idx >= len(list) {
		rows = append(rows, "", "  "+styleFaint.Render("选择左侧版本查看详情"))
	} else {
		v := list[a.download.idx]
		rows = append(rows,
			"",
			"  "+styleBold.Render(ansiTruncate(v.ID, inner-4)),
			"  "+styleMuted.Render(typeLabel(v.Type)+" · "+shortDate(v.ReleaseTime)),
			"",
			kv("版本号", v.ID, inner),
			kv("类型", v.Type, inner),
			kv("发布时间", shortDate(v.ReleaseTime), inner),
		)
		if a.download.installed[v.ID] {
			rows = append(rows, kv("状态", "已安装 ✓", inner))
		} else {
			rows = append(rows, kv("状态", "未安装", inner))
		}
		rows = append(rows, "", "  "+styleBorder.Render(strings.Repeat("─", max(inner-4, 0))))
		switch a.download.mode {
		case 0:
			if a.download.installed[v.ID] {
				rows = append(rows, "  "+styleAccent.Render("enter")+" "+styleMuted.Render("修复 / 重新安装"))
			} else {
				rows = append(rows, "  "+styleAccent.Render("enter")+" "+styleMuted.Render("安装此版本"))
			}
			rows = append(rows, "", "  "+styleFaint.Render("将下载客户端、依赖库与资源文件"))
		case 1:
			rows = append(rows, "  "+styleAccent.Render("enter")+" "+styleMuted.Render("选择 Fabric 加载器"))
			rows = append(rows, "", "  "+styleFaint.Render("将先安装原版 "+v.ID+"，"))
			rows = append(rows, "  "+styleFaint.Render("再写入 Fabric 配置并补全依赖"))
		case 2:
			rows = append(rows, "  "+styleAccent.Render("enter")+" "+styleMuted.Render("选择 Quilt 加载器"))
			rows = append(rows, "", "  "+styleFaint.Render("将先安装原版 "+v.ID+"，"))
			rows = append(rows, "  "+styleFaint.Render("再写入 Quilt 配置并补全依赖"))
		case 3:
			rows = append(rows, "  "+styleAccent.Render("enter")+" "+styleMuted.Render("选择 Legacy Fabric 加载器"))
			rows = append(rows, "", "  "+styleFaint.Render("将先安装原版 "+v.ID+"，"))
			rows = append(rows, "  "+styleFaint.Render("再写入 Legacy Fabric 配置并补全依赖"))
		case 4:
			rows = append(rows, "  "+styleAccent.Render("enter")+" "+styleMuted.Render("选择 Forge 版本"))
			rows = append(rows, "", "  "+styleFaint.Render("将先安装原版 "+v.ID+"，"))
			rows = append(rows, "  "+styleFaint.Render("再运行 Forge 安装器并补全依赖"))
		case 5:
			rows = append(rows, "  "+styleAccent.Render("enter")+" "+styleMuted.Render("选择 NeoForge 版本"))
			rows = append(rows, "", "  "+styleFaint.Render("将先安装原版 "+v.ID+"，"))
			rows = append(rows, "  "+styleFaint.Render("再运行 NeoForge 安装器并补全依赖"))
		case 6:
			rows = append(rows, "  "+styleAccent.Render("enter")+" "+styleMuted.Render("选择 OptiFine 版本"))
			rows = append(rows, "", "  "+styleFaint.Render("将先安装原版 "+v.ID+"，"))
			rows = append(rows, "  "+styleFaint.Render("再写入 OptiFine 配置并补全依赖"))
		}
	}
	for len(rows) < panelContentHeight(h) {
		rows = append(rows, "")
	}
	if len(rows) > panelContentHeight(h) {
		rows = rows[:panelContentHeight(h)]
	}
	return panel("详情", w, h, false, rows)
}

func (a *App) downloadKey(msg tea.KeyMsg) tea.Cmd {
	p := &a.download
	list := p.versions()
	switch msg.String() {
	case "up", "k":
		if len(list) > 0 {
			p.idx = (p.idx - 1 + len(list)) % len(list)
		}
	case "down", "j":
		if len(list) > 0 {
			p.idx = (p.idx + 1) % len(list)
		}
	case "pgup":
		p.idx = max(p.idx-10, 0)
	case "pgdown":
		if len(list) > 0 {
			p.idx = min(p.idx+10, len(list)-1)
		}
	case "home":
		p.idx = 0
	case "end":
		if len(list) > 0 {
			p.idx = len(list) - 1
		}
	case "left", "[":
		if p.mode > 0 {
			p.mode--
			p.idx = 0
			p.loaderLoading = false
		}
	case "right", "]":
		if p.mode < len(loaderModeNames)-1 {
			p.mode++
			p.idx = 0
			p.loaderLoading = false
			if p.manifest == nil && !p.loading {
				a.rt.LoadManifest(false)
			}
		}
	case "m":
		if p.mode != 1 {
			p.show = (p.show + 1) % 3
			p.idx = 0
			a.setToast("筛选: "+[]string{"正式版", "快照", "全部"}[p.show], 0)
		}
	case "/":
		a.modal = modalState{Kind: modalInput, Title: "搜索版本", Label: "版本号或类型", Value: []rune(p.filter), Placeholder: "例如 1.21 / snapshot", Action: "filter-download"}
	case "i":
		a.openInput("导入整合包", "Modrinth .mrpack 文件路径", "", `C:\Users\Alex\Downloads\pack.mrpack`, "import-modpack")
	case "r":
		a.rt.LoadManifest(true)
		a.setToast("正在刷新版本列表", 0)
	case "enter":
		if len(list) == 0 {
			return nil
		}
		v := list[p.idx]
		if p.mode != 0 {
			if p.loaderLoading {
				a.setToast("正在获取加载器版本，请稍候", 3)
				return nil
			}
			p.loaderLoading = true
			switch p.mode {
			case 1:
				a.setToast("正在获取 Fabric 加载器…", 0)
				a.rt.FetchFabricLoaders(v.ID)
			case 2:
				a.setToast("正在获取 Quilt 加载器…", 0)
				a.rt.FetchLoaderVersions("quilt", v.ID)
			case 3:
				a.setToast("正在获取 Legacy Fabric 加载器…", 0)
				a.rt.FetchLoaderVersions("legacyfabric", v.ID)
			case 4:
				a.setToast("正在获取 Forge 版本…", 0)
				a.rt.FetchLoaderVersions("forge", v.ID)
			case 5:
				a.setToast("正在获取 NeoForge 版本…", 0)
				a.rt.FetchLoaderVersions("neoforge", v.ID)
			case 6:
				a.setToast("正在获取 OptiFine 版本…", 0)
				a.rt.FetchLoaderVersions("optifine", v.ID)
			}
			return nil
		}
		if p.installed[v.ID] {
			a.openConfirm("重新安装", fmt.Sprintf("版本 %s 已安装。是否重新下载并修复文件？", v.ID), "install-version", v.ID)
		} else {
			entry := v
			a.startInstall("vanilla", v.ID, v.ID, "", &entry)
		}
	}
	return nil
}

func (a *App) installByID(id string) {
	m := a.download.manifest
	if m == nil {
		a.setToast("版本列表尚未加载", 3)
		return
	}
	entry, ok := m.Find(id)
	if !ok {
		a.setToast("找不到版本 "+id, 2)
		return
	}
	a.startInstall("vanilla", id, id, "", &entry)
}
