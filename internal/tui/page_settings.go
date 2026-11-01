package tui

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/MoonCC233/MCTL/internal/config"
	"github.com/MoonCC233/MCTL/internal/minecraft"
)

const (
	settingText = iota
	settingNumber
	settingToggle
	settingChoice
	settingAction
)

type settingItem struct {
	key     string
	section string
	label   string
	desc    string
	kind    int
}

type settingsPage struct {
	idx      int
	detected *minecraft.JavaInfo
}

func (a *App) settingItems() []settingItem {
	return []settingItem{
		{key: "game_dir", section: "游戏", label: "游戏目录", desc: "Minecraft 数据目录，版本、模组和存档都保存在这里。", kind: settingText},
		{key: "download_source", section: "游戏", label: "下载源", desc: "BMCLAPI 镜像通常在国内更快，Mojang 官方源为原始地址。", kind: settingChoice},
		{key: "theme", section: "界面", label: "主题", desc: "切换终端配色：OpenCode 暗色、午夜蓝或浅色。", kind: settingChoice},
		{key: "language", section: "界面", label: "语言", desc: "中文 / English", kind: settingChoice},
		{key: "java_path", section: "Java", label: "Java 路径", desc: "留空则自动检测。也可以填写 java 可执行文件或 JAVA_HOME 目录。", kind: settingText},
		{key: "auto_java", section: "Java", label: "自动下载 Java", desc: "缺少兼容的 Java 时，从 Adoptium 自动下载所需 JRE。", kind: settingToggle},
		{key: "detect_java", section: "Java", label: "检测 Java", desc: "重新扫描系统与常见安装目录中的 Java 运行时。", kind: settingAction},
		{key: "max_memory", section: "启动", label: "最大内存", desc: "游戏可使用的最大堆内存（MB）。大型整合包建议 4096 以上。", kind: settingNumber},
		{key: "min_memory", section: "启动", label: "最小内存", desc: "JVM 初始堆内存（MB），通常保持 512 即可。", kind: settingNumber},
		{key: "width", section: "启动", label: "窗口宽度", desc: "游戏窗口宽度。全屏模式下此值会被忽略。", kind: settingNumber},
		{key: "height", section: "启动", label: "窗口高度", desc: "游戏窗口高度。全屏模式下此值会被忽略。", kind: settingNumber},
		{key: "fullscreen", section: "启动", label: "全屏启动", desc: "启动游戏时直接进入全屏模式。", kind: settingToggle},
		{key: "jvm_args", section: "启动", label: "JVM 参数", desc: "追加到 Java 命令行的参数，例如 -XX:+UseG1GC。", kind: settingText},
		{key: "game_args", section: "启动", label: "游戏参数", desc: "追加到 Minecraft 主类的参数，例如 --demo。", kind: settingText},
		{key: "ms_client_id", section: "账户", label: "微软客户端 ID", desc: "Azure 应用的客户端 ID，需要启用公共客户端设备代码流。", kind: settingText},
		{key: "ms_login", section: "账户", label: "Microsoft 登录", desc: "使用设备代码在浏览器中登录 Microsoft 账户。", kind: settingAction},
		{key: "cf_api_key", section: "整合包", label: "CurseForge Key", desc: "导入 CurseForge 整合包所需的 API Key，可在 CurseForge 开发者控制台申请。", kind: settingText},
		{key: "proxy", section: "网络", label: "HTTP 代理", desc: "例如 http://127.0.0.1:7890；留空表示直连并使用环境变量代理。", kind: settingText},
		{key: "check_update", section: "更新", label: "检查更新", desc: "检查最新发布版本并在有更新时打开下载页面。", kind: settingAction},
		{key: "feedback", section: "更新", label: "问题反馈", desc: "在浏览器中打开项目的 Issue 页面。", kind: settingAction},
		{key: "open_game", section: "目录", label: "打开游戏目录", desc: "在系统文件管理器中打开当前游戏目录。", kind: settingAction},
		{key: "open_config", section: "目录", label: "打开配置目录", desc: "打开 MCTL 配置与日志所在目录。", kind: settingAction},
	}
}

func (a *App) viewSettings(w, h int) []string {
	cfg := a.store.Snapshot()
	items := a.settingItems()
	if a.settings.idx >= len(items) {
		a.settings.idx = len(items) - 1
	}
	confDir, _ := config.Dir()
	lines := pageHeader("设置", tr("修改会立即保存")+" · "+middleEllipsis(confDir, max(w-24, 20)), w)
	contentH := h - len(lines)
	if contentH < 5 {
		return fitLines(lines, h, w)
	}
	descH := 6
	if contentH < 14 {
		descH = 0
	}
	listH := contentH - descH
	if descH > 0 {
		listH--
	}
	list := a.renderSettingsList(2, len(lines), w-4, listH, cfg, items)
	for _, l := range strings.Split(list, "\n") {
		lines = append(lines, "  "+l)
	}
	if descH > 0 {
		it := items[a.settings.idx]
		descW := max(w-12, 16)
		descText := wrapText(tr(it.desc), descW)
		if len(descText) > 2 {
			descText = descText[:2]
			descText[1] = ansiTruncate(descText[1], max(descW-1, 1)) + "…"
		}
		var desc []string
		for _, l := range descText {
			desc = append(desc, "  "+styleMuted.Render(l))
		}
		for len(desc) < 2 {
			desc = append(desc, "")
		}
		desc = append(desc, "", "  "+styleFaint.Render(tr("单击条目编辑或切换 · 滚轮滚动")))
		d := panel("提示", w-4, descH, false, desc)
		lines = append(lines, "")
		for _, l := range strings.Split(d, "\n") {
			lines = append(lines, "  "+l)
		}
	}
	return fitLines(lines, h, w)
}

func (a *App) renderSettingsList(x, y, w, h int, cfg config.Config, items []settingItem) string {
	inner := panelContentWidth(w)
	maxRows := panelContentHeight(h)
	a.addHit(x, y, x+w-1, y+h-1, "wheel", "settings")
	var all []string
	itemLine := make([]int, len(items))
	lastSection := ""
	for i, it := range items {
		if it.section != lastSection {
			lastSection = it.section
			all = append(all, " "+sectionLabel(it.section))
		}
		itemLine[i] = len(all)
		val := tr(a.settingValue(cfg, it))
		labelW := 16
		if inner < 34 {
			labelW = 12
		}
		prefix := "   "
		line := prefix + padRight(styleMuted.Render(tr(it.label)), labelW) + "  " + styleText.Render(ansiTruncate(val, max(inner-labelW-5, 8)))
		if i == a.settings.idx {
			prefix = styleAccent.Render(" ▸ ")
			line = prefix + padRight(styleText.Render(tr(it.label)), labelW) + "  " + styleText.Render(ansiTruncate(val, max(inner-labelW-5, 8)))
		}
		if i == a.settings.idx || a.hoverRow("settings-row", i) {
			all = append(all, lipglossRow(line, inner))
		} else {
			all = append(all, line)
		}
	}
	start := 0
	if len(all) > maxRows {
		start = itemLine[a.settings.idx] - maxRows/2
		if start < 0 {
			start = 0
		}
		if start > len(all)-maxRows {
			start = len(all) - maxRows
		}
	}
	end := min(start+maxRows, len(all))
	rows := append([]string{}, all[start:end]...)
	for i := range items {
		line := itemLine[i]
		if line < start || line >= end {
			continue
		}
		rowY := y + 1 + line - start
		a.addHit(x+1, rowY, x+w-2, rowY, "settings-row", i)
	}
	for len(rows) < maxRows {
		rows = append(rows, "")
	}
	return panelScroll(tr("偏好设置")+fmt.Sprintf(" · %d/%d", min(a.settings.idx+1, len(items)), len(items)), w, h, a.page == pageSettings, rows, len(all), start)
}

func (a *App) settingValue(cfg config.Config, it settingItem) string {
	switch it.key {
	case "game_dir":
		return middleEllipsis(cfg.GameDir, 40)
	case "download_source":
		return tr(minecraft.GetSource(cfg.DownloadSource).Name)
	case "theme":
		return ThemeName(cfg.Theme)
	case "language":
		return LanguageName(cfg.Language)
	case "java_path":
		if cfg.JavaPath != "" {
			return middleEllipsis(cfg.JavaPath, 40)
		}
		if a.settings.detected != nil {
			return trf("自动 · Java %d", a.settings.detected.Major)
		}
		return tr("自动 · 未检测到")
	case "auto_java":
		if cfg.AutoJava {
			return tr("开启")
		}
		return tr("关闭")
	case "max_memory":
		return fmt.Sprintf("%d MB", cfg.MaxMemory)
	case "min_memory":
		return fmt.Sprintf("%d MB", cfg.MinMemory)
	case "width":
		return strconv.Itoa(cfg.Width)
	case "height":
		return strconv.Itoa(cfg.Height)
	case "fullscreen":
		if cfg.Fullscreen {
			return tr("开启")
		}
		return tr("关闭")
	case "jvm_args":
		if strings.TrimSpace(cfg.JVMArgs) == "" {
			return tr("（无）")
		}
		return middleEllipsis(cfg.JVMArgs, 40)
	case "game_args":
		if strings.TrimSpace(cfg.GameArgs) == "" {
			return tr("（无）")
		}
		return middleEllipsis(cfg.GameArgs, 40)
	case "ms_client_id":
		if strings.TrimSpace(cfg.MicrosoftClientID) == "" {
			return tr("未配置")
		}
		return middleEllipsis(cfg.MicrosoftClientID, 40)
	case "ms_login":
		return "登录"
	case "cf_api_key":
		if strings.TrimSpace(cfg.CurseForgeAPIKey) == "" {
			return "未配置"
		}
		return middleEllipsis(cfg.CurseForgeAPIKey, 40)
	case "proxy":
		if strings.TrimSpace(cfg.Proxy) == "" {
			return "直连"
		}
		return middleEllipsis(cfg.Proxy, 40)
	case "detect_java", "open_game", "open_config":
		return "→"
	}
	return ""
}

func (a *App) settingsKey(msg tea.KeyMsg) tea.Cmd {
	items := a.settingItems()
	cfg := a.store.Snapshot()
	it := items[a.settings.idx]
	switch msg.String() {
	case "up", "k":
		a.settings.idx = (a.settings.idx - 1 + len(items)) % len(items)
	case "down", "j":
		a.settings.idx = (a.settings.idx + 1) % len(items)
	case "enter":
		a.editSetting(it, cfg)
	case "left", "h":
		a.adjustSetting(it, cfg, -1)
	case "right", "l":
		a.adjustSetting(it, cfg, 1)
	case "o":
		a.openGameDir()
	}
	return nil
}

func (a *App) editSetting(it settingItem, cfg config.Config) {
	switch it.kind {
	case settingToggle:
		a.toggleSetting(it, cfg)
	case settingChoice:
		a.adjustSetting(it, cfg, 1)
	case settingAction:
		a.runSettingAction(it)
	case settingText, settingNumber:
		a.modal = modalState{
			Kind:   modalInput,
			Title:  tr(it.label),
			Label:  tr(it.desc),
			Value:  []rune(a.rawSettingValue(cfg, it)),
			Action: "setting:" + it.key,
		}
	}
}

func (a *App) rawSettingValue(cfg config.Config, it settingItem) string {
	switch it.key {
	case "game_dir":
		return cfg.GameDir
	case "java_path":
		return cfg.JavaPath
	case "max_memory":
		return strconv.Itoa(cfg.MaxMemory)
	case "min_memory":
		return strconv.Itoa(cfg.MinMemory)
	case "width":
		return strconv.Itoa(cfg.Width)
	case "height":
		return strconv.Itoa(cfg.Height)
	case "jvm_args":
		return cfg.JVMArgs
	case "game_args":
		return cfg.GameArgs
	case "ms_client_id":
		return cfg.MicrosoftClientID
	case "cf_api_key":
		return cfg.CurseForgeAPIKey
	case "proxy":
		return cfg.Proxy
	}
	return ""
}

func (a *App) toggleSetting(it settingItem, cfg config.Config) {
	switch it.key {
	case "auto_java":
		a.saveCfg(func(c *config.Config) { c.AutoJava = !c.AutoJava })
		a.setToast("自动下载 Java: "+onOff(!cfg.AutoJava), 1)
	case "fullscreen":
		a.saveCfg(func(c *config.Config) { c.Fullscreen = !c.Fullscreen })
		a.setToast("全屏启动: "+onOff(!cfg.Fullscreen), 1)
	}
}

func onOff(b bool) string {
	if b {
		return "开启"
	}
	return "关闭"
}

func (a *App) adjustSetting(it settingItem, cfg config.Config, dir int) {
	switch it.key {
	case "download_source":
		next := "bmclapi"
		if cfg.DownloadSource == "bmclapi" {
			next = "official"
		}
		a.saveCfg(func(c *config.Config) { c.DownloadSource = next })
		a.rt.LoadManifest(true)
		a.setToast("下载源: "+minecraft.GetSource(next).Name, 1)
	case "theme":
		ids := ThemeIDs()
		idx := 0
		current := NormalizeTheme(cfg.Theme)
		for i, id := range ids {
			if id == current {
				idx = i
				break
			}
		}
		next := ids[(idx+dir+len(ids))%len(ids)]
		ApplyTheme(next)
		a.saveCfg(func(c *config.Config) { c.Theme = next })
		a.setToast("主题: "+ThemeName(next), 1)
	case "language":
		next := "en"
		if NormalizeLanguage(cfg.Language) == "en" {
			next = "zh"
		}
		SetLanguage(next)
		a.saveCfg(func(c *config.Config) { c.Language = next })
		a.setToast("Language: "+LanguageName(next), 1)
	case "auto_java":
		a.toggleSetting(it, cfg)
	case "fullscreen":
		a.toggleSetting(it, cfg)
	case "max_memory":
		v := clamp(cfg.MaxMemory+dir*256, 512, 65536)
		a.saveCfg(func(c *config.Config) { c.MaxMemory = v })
	case "min_memory":
		v := clamp(cfg.MinMemory+dir*256, 256, cfg.MaxMemory)
		a.saveCfg(func(c *config.Config) { c.MinMemory = v })
	case "width":
		v := clamp(cfg.Width+dir*16, 320, 7680)
		a.saveCfg(func(c *config.Config) { c.Width = v })
	case "height":
		v := clamp(cfg.Height+dir*16, 240, 4320)
		a.saveCfg(func(c *config.Config) { c.Height = v })
	}
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func (a *App) runSettingAction(it settingItem) {
	switch it.key {
	case "detect_java":
		cfg := a.store.Snapshot()
		a.rt.ScanJava(cfg.JavaPath)
		a.setToast("正在扫描 Java…", 0)
	case "ms_login":
		a.openMicrosoftLogin()
	case "open_game":
		a.openGameDir()
	case "open_config":
		dir, err := config.Dir()
		if err != nil {
			a.setToast("无法定位配置目录", 2)
			return
		}
		if err := openPath(dir); err != nil {
			a.setToast("打开配置目录失败: "+err.Error(), 2)
		}
	case "check_update":
		a.setToast("正在检查更新…", 0)
		a.rt.CheckUpdate()
	case "feedback":
		url := "https://github.com/" + RepoOwner + "/" + RepoName + "/issues"
		if err := openPath(url); err != nil {
			a.setToast("打开反馈页面失败: "+err.Error(), 2)
		}
	}
}

func (a *App) submitSetting(key, value string) {
	value = strings.TrimSpace(value)
	switch key {
	case "game_dir":
		if value == "" {
			a.setToast("游戏目录不能为空", 2)
			return
		}
		abs, err := filepath.Abs(value)
		if err != nil {
			a.setToast("路径无效: "+err.Error(), 2)
			return
		}
		a.saveCfg(func(c *config.Config) { c.GameDir = abs })
		a.rt.ScanVersions()
		a.setToast("游戏目录已更新", 1)
	case "java_path":
		a.saveCfg(func(c *config.Config) { c.JavaPath = value })
		a.rt.DetectJava(value)
		a.setToast("Java 路径已更新", 1)
	case "max_memory":
		v, err := strconv.Atoi(value)
		if err != nil || v < 512 {
			a.setToast("请输入不小于 512 的整数", 2)
			return
		}
		a.saveCfg(func(c *config.Config) { c.MaxMemory = clamp(v, 512, 65536) })
		a.setToast("最大内存已更新", 1)
	case "min_memory":
		v, err := strconv.Atoi(value)
		if err != nil || v < 256 {
			a.setToast("请输入不小于 256 的整数", 2)
			return
		}
		a.saveCfg(func(c *config.Config) { c.MinMemory = clamp(v, 256, c.MaxMemory) })
		a.setToast("最小内存已更新", 1)
	case "width":
		v, err := strconv.Atoi(value)
		if err != nil || v < 320 {
			a.setToast("宽度不能小于 320", 2)
			return
		}
		a.saveCfg(func(c *config.Config) { c.Width = clamp(v, 320, 7680) })
	case "height":
		v, err := strconv.Atoi(value)
		if err != nil || v < 240 {
			a.setToast("高度不能小于 240", 2)
			return
		}
		a.saveCfg(func(c *config.Config) { c.Height = clamp(v, 240, 4320) })
	case "jvm_args":
		a.saveCfg(func(c *config.Config) { c.JVMArgs = value })
		a.setToast("JVM 参数已保存", 1)
	case "game_args":
		a.saveCfg(func(c *config.Config) { c.GameArgs = value })
		a.setToast("游戏参数已保存", 1)
	case "ms_client_id":
		a.saveCfg(func(c *config.Config) { c.MicrosoftClientID = value })
		a.setToast("Microsoft 客户端 ID 已保存", 1)
	case "cf_api_key":
		a.saveCfg(func(c *config.Config) { c.CurseForgeAPIKey = value })
		a.setToast("CurseForge API Key 已保存", 1)
	case "proxy":
		if err := minecraft.SetProxy(value); err != nil {
			a.openError("代理地址无效", err)
			return
		}
		a.saveCfg(func(c *config.Config) { c.Proxy = value })
		if value == "" {
			a.setToast("已清除代理", 1)
		} else {
			a.setToast("代理已保存", 1)
		}
	}
}
