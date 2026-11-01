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
	kind        string
	worldDir    string
	pendingID   string
	pendingName string
}

func resourceKindLabel(kind string) string {
	switch kind {
	case "resourcepack":
		return "资源包"
	case "shader":
		return "光影"
	case "mod":
		return "模组"
	case "datapack":
		return "数据包"
	case "world":
		return "世界"
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
		{label: "在线整合包", desc: "按游戏版本与加载器筛选，搜索并安装 Modrinth 整合包。", action: "online-modpacks"},
		{label: "CurseForge 整合包", desc: "使用 CurseForge API Key 搜索并安装整合包。", action: "curseforge-modpacks"},
		{label: "收藏", desc: "查看已收藏的整合包、资源包、光影和模组。", action: "favorites"},
		{label: "在线资源", desc: "搜索 Modrinth 资源包、光影、模组或数据包并安装到当前实例。", action: "online-resources"},
		{label: "存档管理", desc: "查看、打开、备份、导入或删除当前实例的存档。", action: "worlds"},
		{label: "本地资源", desc: "启用、停用或删除当前实例的模组、资源包和光影。", action: "local-resources"},
		{label: "投影管理", desc: "查看、打开或删除当前实例的投影文件。", action: "schematics"},
		{label: "模组信息", desc: "读取模组元数据、加载器、启用状态与内嵌 jar。", action: "mods-info"},
		{label: "截图管理", desc: "查看或删除当前实例的 Minecraft 截图。", action: "screenshots"},
		{label: "服务器列表", desc: "添加、编辑或删除当前实例保存的多人服务器。", action: "servers"},
		{label: "皮肤 / 披风", desc: "查询当前账户的公开皮肤与披风纹理。", action: "skin"},
		{label: "导出实例", desc: "把当前版本或整合包连同模组、存档与截图打包为 zip。", action: "export-instance"},
		{label: "导入实例", desc: "从 MCTL 实例 zip 创建独立实例并补全缺失文件。", action: "import-instance"},
		{label: "新建实例", desc: "按名称、目录、版本、加载器与 Java 创建独立实例。", action: "new-instance"},
		{label: "实例设置", desc: "为当前实例设置名称、主页、内存、JVM 参数与自动进入服务器。", action: "instance-settings"},
		{label: "实例安装", desc: "为当前实例安装加载器或补全游戏文件。", action: "instance-install"},
		{label: "环境自检", desc: "检查游戏目录、Java、版本、资源与下载源。", action: "diagnose"},
		{label: "检查更新", desc: "查询最新发布版本，有新版本时打开下载页面。", action: "update"},
		{label: "新闻 / 更新日志", desc: "查看最近的版本发布说明。", action: "news"},
		{label: "网络测速", desc: "测试下载源、Modrinth、GitHub 与 Adoptium 的延迟和吞吐。", action: "speedtest"},
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
		hint := []string{
			"  " + styleMuted.Render(ansiTruncate(tr(it.desc), max(w-10, 16))),
			"  " + keyHint("enter", "执行") + "  " + keyHint("+", "新建") + "  " + keyHint("x", "导出") + "  " + keyHint("u", "导入"),
			"  " + keyHint("y", "测速") + "  " + keyHint("d", "自检") + "  " + keyHint("t", "更新") + "  " + keyHint("h", "新闻"),
		}
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
		label := tr(it.label)
		desc := tr(it.desc)
		if i == a.tools.idx {
			rows = append(rows, lipglossRow("  ▸ "+padRight(label, 18)+"  "+ansiTruncate(desc, max(inner-24, 8)), inner))
		} else {
			rows = append(rows, "    "+styleText.Render(padRight(label, 18))+"  "+styleFaint.Render(ansiTruncate(desc, max(inner-24, 8))))
		}
	}
	for len(rows) < maxRows {
		rows = append(rows, "")
	}
	if len(rows) > maxRows {
		rows = rows[:maxRows]
	}
	return panel(tr("常用工具")+fmt.Sprintf(" · %d/%d", min(a.tools.idx+1, len(items)), len(items)), w, h, a.page == pageTools, rows)
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
	case "s":
		a.openWorlds()
	case "m":
		a.openLocalResourceTypePicker()
	case "z":
		a.openResourceFiles("schematic")
	case "b":
		a.openModsInfo()
	case "f":
		a.openFavorites()
	case "h":
		a.setToast("正在获取更新日志…", 0)
		a.rt.FetchNews()
	case "y":
		a.modal = modalState{Kind: modalSpeed, Title: "网络测速"}
		a.rt.RunSpeedTest()
	case "p":
		a.openScreenshots()
	case "n":
		a.openServers()
	case "x":
		a.openExportInstance()
	case "u":
		a.openImportInstance()
	case "+":
		a.openInstanceWizard()
	case "d":
		a.setToast("正在检测运行环境…", 0)
		a.rt.RunDiagnose()
	case "t":
		a.setToast("正在检查更新…", 0)
		a.rt.CheckUpdate()
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
	case "curseforge-modpacks":
		a.openCurseForgeSearch()
	case "favorites":
		a.openFavorites()
	case "online-resources":
		a.openResourceTypePicker()
	case "worlds":
		a.openWorlds()
	case "local-resources":
		a.openLocalResourceTypePicker()
	case "schematics":
		a.openResourceFiles("schematic")
	case "mods-info":
		a.openModsInfo()
	case "screenshots":
		a.openScreenshots()
	case "servers":
		a.openServers()
	case "skin":
		a.setToast("正在获取皮肤…", 0)
		a.rt.FetchSkin()
	case "export-instance":
		a.openExportInstance()
	case "import-instance":
		a.openImportInstance()
	case "instance-settings":
		a.openInstanceSettings()
	case "instance-install":
		a.openInstanceInstall()
	case "new-instance":
		a.openInstanceWizard()
	case "diagnose":
		a.setToast("正在检测运行环境…", 0)
		a.rt.RunDiagnose()
	case "update":
		a.setToast("正在检查更新…", 0)
		a.rt.CheckUpdate()
	case "news":
		a.setToast("正在获取更新日志…", 0)
		a.rt.FetchNews()
	case "speedtest":
		a.modal = modalState{Kind: modalSpeed, Title: "网络测速"}
		a.rt.RunSpeedTest()
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
