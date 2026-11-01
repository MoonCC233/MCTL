package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/MoonCC233/MCTL/internal/config"
)

type toolsPage struct {
	idx int
}

type resourceBrowser struct {
	kind string
}

func resourceKindLabel(kind string) string {
	switch kind {
	case "resourcepack":
		return "资源包"
	case "shader":
		return "光影"
	case "mod":
		return "模组"
	}
	return kind
}

type toolItem struct {
	label  string
	desc   string
	action string
}

func (a *App) toolItems() []toolItem {
	return []toolItem{
		{label: "导入整合包", desc: "导入本地 Modrinth .mrpack 或 CurseForge 整合包到独立实例。", action: "import"},
		{label: "在线整合包", desc: "搜索 Modrinth 并下载安装最新的稳定整合包版本。", action: "online-modpacks"},
		{label: "在线资源", desc: "搜索 Modrinth 资源包、光影或模组并安装到当前实例。", action: "online-resources"},
		{label: "打开游戏目录", desc: "在系统文件管理器中打开当前游戏目录。", action: "open-game"},
		{label: "打开配置目录", desc: "打开 config.json 与日志文件所在目录。", action: "open-config"},
		{label: "检测 Java", desc: "重新扫描系统与常见安装目录中的 Java 运行时。", action: "java"},
		{label: "刷新版本与清单", desc: "重新扫描本地版本并刷新下载版本清单。", action: "refresh"},
		{label: "打开日志", desc: "查看启动器与游戏实时输出。", action: "logs"},
		{label: "清空日志", desc: "清空当前内存日志，不影响已写入的文件。", action: "clear-logs"},
		{label: "前往下载页", desc: "安装原版、Fabric、Quilt、Forge 或 NeoForge。", action: "download"},
		{label: "关于 MCTL", desc: "查看版本、配置路径与项目信息。", action: "about"},
	}
}

func (a *App) viewTools(w, h int) []string {
	items := a.toolItems()
	if a.tools.idx >= len(items) {
		a.tools.idx = len(items) - 1
	}
	lines := pageHeader("工具", "维护、目录与常用操作", w)
	contentH := h - len(lines)
	if contentH < 5 {
		return fitLines(lines, h, w)
	}
	hintH := 5
	if contentH < 12 {
		hintH = 0
	}
	listH := contentH - hintH
	if hintH > 0 {
		listH--
	}
	list := a.renderToolList(w-4, listH, items)
	for _, l := range strings.Split(list, "\n") {
		lines = append(lines, "  "+l)
	}
	if hintH > 0 {
		it := items[a.tools.idx]
		hint := []string{"  " + styleMuted.Render(ansiTruncate(it.desc, max(w-10, 16))), "", "  " + keyHint("enter", "执行") + "  " + keyHint("↑↓", "选择") + "  " + keyHint("i", "导入") + "  " + keyHint("w", "整合包") + "  " + keyHint("e", "资源")}
		p := panel("提示", w-4, hintH, false, hint)
		lines = append(lines, "")
		for _, l := range strings.Split(p, "\n") {
			lines = append(lines, "  "+l)
		}
	}
	return fitLines(lines, h, w)
}

func (a *App) renderToolList(w, h int, items []toolItem) string {
	inner := panelContentWidth(w)
	maxRows := panelContentHeight(h)
	start := 0
	if a.tools.idx >= maxRows {
		start = a.tools.idx - maxRows + 1
	}
	if start > len(items)-maxRows {
		start = max(len(items)-maxRows, 0)
	}
	end := min(start+maxRows, len(items))
	var rows []string
	if len(items) == 0 {
		rows = append(rows, "", "  "+styleFaint.Render("没有可用工具"))
	}
	for i := start; i < end; i++ {
		it := items[i]
		if i == a.tools.idx {
			rows = append(rows, lipglossRow("  ▸ "+padRight(it.label, 16)+"  "+ansiTruncate(it.desc, max(inner-22, 8)), inner))
		} else {
			rows = append(rows, "    "+styleText.Render(padRight(it.label, 16))+"  "+styleFaint.Render(ansiTruncate(it.desc, max(inner-22, 8))))
		}
	}
	for len(rows) < maxRows {
		rows = append(rows, "")
	}
	if len(rows) > maxRows {
		rows = rows[:maxRows]
	}
	return panel(fmt.Sprintf("常用工具 · %d", len(items)), w, h, a.page == pageTools, rows)
}

func (a *App) toolsKey(msg tea.KeyMsg) tea.Cmd {
	items := a.toolItems()
	switch msg.String() {
	case "up", "k":
		a.tools.idx = (a.tools.idx - 1 + len(items)) % len(items)
	case "down", "j":
		a.tools.idx = (a.tools.idx + 1) % len(items)
	case "enter":
		return a.runTool(items[a.tools.idx].action)
	case "i":
		a.openInput("导入整合包", "Modrinth .mrpack 或 CurseForge 压缩包路径", "", `C:\Users\Alex\Downloads\pack.mrpack`, "import-modpack")
	case "w":
		a.openOnlineModpacks()
	case "e":
		a.openResourceTypePicker()
	case "o":
		a.openGameDir()
	case "g":
		if dir, err := config.Dir(); err == nil {
			if err := openPath(dir); err != nil {
				a.setToast("打开配置目录失败: "+err.Error(), 2)
			}
		}
	case "v":
		a.rt.DetectJava(a.store.Snapshot().JavaPath)
		a.setToast("正在检测 Java…", 0)
	case "r":
		a.rt.ScanVersions()
		a.rt.LoadManifest(true)
		a.setToast("正在刷新版本与清单…", 0)
	case "l":
		a.switchPage(pageLogs)
	case "c":
		a.rt.Logs.Clear()
		a.setToast("日志已清空", 1)
	}
	return nil
}

func (a *App) runTool(action string) tea.Cmd {
	switch action {
	case "import":
		a.openInput("导入整合包", "Modrinth .mrpack 或 CurseForge 压缩包路径", "", `C:\Users\Alex\Downloads\pack.mrpack`, "import-modpack")
	case "online-modpacks":
		a.openOnlineModpacks()
	case "online-resources":
		a.openResourceTypePicker()
	case "open-game":
		a.openGameDir()
	case "open-config":
		dir, err := config.Dir()
		if err != nil {
			a.setToast("无法定位配置目录", 2)
			return nil
		}
		if err := openPath(dir); err != nil {
			a.setToast("打开配置目录失败: "+err.Error(), 2)
		}
	case "java":
		a.rt.DetectJava(a.store.Snapshot().JavaPath)
		a.setToast("正在检测 Java…", 0)
	case "refresh":
		a.rt.ScanVersions()
		a.rt.LoadManifest(true)
		a.setToast("正在刷新版本与清单…", 0)
	case "logs":
		a.switchPage(pageLogs)
	case "clear-logs":
		a.rt.Logs.Clear()
		a.setToast("日志已清空", 1)
	case "download":
		a.switchPage(pageDownload)
	case "about":
		a.switchPage(pageAbout)
	}
	return nil
}
