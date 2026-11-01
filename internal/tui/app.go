package tui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/MoonCC233/MCTL/internal/config"
	"github.com/MoonCC233/MCTL/internal/minecraft"
)

type pageID int

const (
	pageLaunch pageID = iota
	pageVersions
	pageDownload
	pageSettings
	pageTools
	pageAbout
	pageLogs
	pageCount
)

var pageNames = []string{"启动", "版本", "下载", "设置", "工具", "关于", "日志"}
var pageEnglish = []string{"LAUNCH", "VERSIONS", "DOWNLOAD", "SETTINGS", "TOOLS", "ABOUT", "LOGS"}
var pageNumbers = []string{"1", "2", "3", "4", "5", "6", "7"}

type toastState struct {
	Text  string
	Level int
	Until time.Time
}

// App is the root Bubble Tea model.
type App struct {
	width, height int
	store         *config.Store
	rt            *Runtime

	page     pageID
	modal    modalState
	frame    int
	toast    toastState
	quitting bool

	launch    launchPage
	versions  versionsPage
	download  downloadPage
	settings  settingsPage
	tools     toolsPage
	resources resourceBrowser
	logs      logsPage

	demo bool
}

// New builds the application.
func New(store *config.Store, rt *Runtime) *App {
	a := &App{store: store, rt: rt}
	a.settings.detected = rt.Java()
	a.download.installed = map[string]bool{}
	return a
}

// Init starts background loading and the UI clock.
func (a *App) Init() tea.Cmd {
	if a.demo {
		return nil
	}
	cfg := a.store.Snapshot()
	a.rt.ScanVersions()
	a.rt.LoadManifest(false)
	a.rt.DetectJava(cfg.JavaPath)
	return tea.Batch(a.waitEvent(), tickCmd())
}

func (a *App) waitEvent() tea.Cmd {
	return func() tea.Msg {
		return eventMsg{Msg: <-a.rt.Events}
	}
}

type tickMsg struct{}

func tickCmd() tea.Cmd {
	return tea.Tick(RefreshDelay, func(time.Time) tea.Msg { return tickMsg{} })
}

// Update implements tea.Model.
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m := msg.(type) {
	case tea.WindowSizeMsg:
		a.width, a.height = m.Width, m.Height
		return a, nil
	case tea.KeyMsg:
		return a.handleKey(m)
	case eventMsg:
		cmd := a.handleEvent(m.Msg)
		return a, tea.Batch(a.waitEvent(), cmd)
	case tickMsg:
		a.frame++
		if a.toast.Text != "" && time.Now().After(a.toast.Until) {
			a.toast = toastState{}
		}
		return a, tickCmd()
	}
	return a, nil
}

// View implements tea.Model.
func (a *App) View() string {
	if a.width == 0 || a.height == 0 {
		return "MCTL 正在初始化…"
	}
	if a.width < 76 || a.height < 22 {
		msg := styleAccent.Render("MCTL") + "\n\n" +
			styleMuted.Render("终端窗口过小") + "\n" +
			styleFaint.Render(fmt.Sprintf("当前 %d×%d，至少需要 76×22", a.width, a.height))
		return lipgloss.Place(a.width, a.height, lipgloss.Center, lipgloss.Center, msg)
	}
	base := a.renderBase()
	if a.modal.Kind != modalNone {
		return overlayCenter(base, a.renderModal(), a.width, a.height)
	}
	return base
}

func (a *App) renderBase() string {
	w, h := a.width, a.height
	bodyH := h - 4
	lines := []string{a.renderHeader()}
	lines = append(lines, styleBorder.Render(strings.Repeat("─", w)))
	lines = append(lines, a.renderBody(bodyH)...)
	lines = append(lines, styleBorder.Render(strings.Repeat("─", w)))
	lines = append(lines, a.renderFooter())
	for i := range lines {
		lines[i] = padRight(lines[i], w)
	}
	return strings.Join(lines, "\n")
}

func (a *App) renderHeader() string {
	cfg := a.store.Snapshot()
	left := " " + styleAccent.Render("◆") + " " + styleBold.Render("MCTL") + " " + styleFaint.Render("v"+Version) +
		styleFaint.Render("  ·  Minecraft 终端启动器")
	var right string
	if a.rt.Launching() {
		right = dot(colorGreen) + " " + styleGreen.Render("游戏运行中")
	} else if a.rt.Installing() {
		right = spinner(a.frame, "安装中")
	} else {
		java := "Java 未检测"
		if j := a.rt.Java(); j != nil {
			java = fmt.Sprintf("Java %d", j.Major)
		}
		src := minecraft.GetSource(cfg.DownloadSource).Name
		right = styleMuted.Render(fmt.Sprintf("%s  ·  %d MB  ·  %s", java, cfg.MaxMemory, src))
	}
	right += " "
	return padBetween(left, right, a.width)
}

func padBetween(left, right string, w int) string {
	lw, rw := lipgloss.Width(left), lipgloss.Width(right)
	if lw+rw+1 > w {
		left = ansiTruncate(left, max(w-rw-1, 0))
		lw = lipgloss.Width(left)
	}
	gap := max(w-lw-rw, 1)
	return left + strings.Repeat(" ", gap) + right
}

func ansiTruncate(s string, w int) string {
	return padRight(s, w)
}

func (a *App) renderFooter() string {
	cfg := a.store.Snapshot()
	var parts []string
	switch a.page {
	case pageLaunch:
		parts = []string{keyHint("↑↓", "选择"), keyHint("←→", "切换"), keyHint("enter", "启动"), keyHint("a", "账户"), keyHint("l", "日志")}
	case pageVersions:
		parts = []string{keyHint("↑↓", "选择"), keyHint("enter", "启动"), keyHint("i", "实例设置"), keyHint("d", "删除"), keyHint("o", "版本目录"), keyHint("/", "筛选"), keyHint("r", "刷新")}
	case pageDownload:
		parts = []string{keyHint("↑↓", "选择"), keyHint("enter", "安装"), keyHint("[ ]", "类型"), keyHint("i", "导入"), keyHint("o", "在线整合包"), keyHint("m", "筛选"), keyHint("/", "搜索"), keyHint("r", "刷新")}
	case pageSettings:
		parts = []string{keyHint("↑↓", "选择"), keyHint("enter", "编辑"), keyHint("←→", "调整"), keyHint("o", "打开游戏目录")}
	case pageTools:
		parts = []string{keyHint("↑↓", "选择"), keyHint("enter", "执行"), keyHint("x", "导出"), keyHint("u", "导入"), keyHint("d", "自检"), keyHint("t", "更新")}
	case pageAbout:
		parts = []string{keyHint("o", "打开游戏目录"), keyHint("g", "打开配置目录")}
	case pageLogs:
		parts = []string{keyHint("↑↓", "滚动"), keyHint("f", "跟随"), keyHint("c", "清空"), keyHint("pgup/pgdn", "翻页")}
	}
	parts = append(parts, keyHint("tab", "切换"), keyHint("?", "帮助"), keyHint("q", "退出"))
	left := " " + hints(parts...)
	var right string
	if a.modal.Kind == modalProgress {
		pct := int(a.modal.Percent * 100)
		right = spinner(a.frame, fmt.Sprintf("%s %d%%", a.modal.Stage, pct))
	} else if a.toast.Text != "" {
		right = dot(statusColor(a.toast.Level)) + " " + lipgloss.NewStyle().Foreground(statusColor(a.toast.Level)).Render(a.toast.Text)
	} else if a.rt.Launching() {
		right = dot(colorGreen) + " " + styleGreen.Render("游戏运行中")
	} else if a.rt.Installing() {
		right = spinner(a.frame, "安装中")
	} else {
		right = dot(colorGreen) + " " + styleMuted.Render("就绪")
	}
	right += " "
	if lipgloss.Width(left)+lipgloss.Width(right)+1 > a.width {
		left = ansiTruncate(left, max(a.width-lipgloss.Width(right)-1, 0))
	}
	_ = cfg
	return padBetween(left, right, a.width)
}

func (a *App) renderBody(h int) []string {
	w := a.width
	sidebarW := 24
	if w < 104 {
		sidebarW = 22
	}
	if w < 90 {
		sidebarW = 20
	}
	mainW := w - sidebarW - 1
	side := a.renderSidebar(h, sidebarW-1)
	page := a.renderPage(mainW, h)
	out := make([]string, h)
	for i := 0; i < h; i++ {
		s := ""
		if i < len(side) {
			s = side[i]
		}
		p := ""
		if i < len(page) {
			p = page[i]
		}
		out[i] = padRight(s, sidebarW-1) + styleBorder.Render("│") + padRight(p, mainW)
	}
	return out
}

func (a *App) renderSidebar(h, w int) []string {
	fullTop := a.sidebarTop(w, false)
	fullBottom := a.sidebarBottom(w, false)
	top, bottom := fullTop, fullBottom
	if h < len(fullTop)+len(fullBottom) {
		top = a.sidebarTop(w, true)
		bottom = a.sidebarBottom(w, true)
	}
	lines := append([]string{}, top...)
	space := h - len(lines) - len(bottom)
	if space < 0 {
		lines = lines[:max(h-len(bottom), 0)]
		space = h - len(lines) - len(bottom)
	}
	for i := 0; i < space; i++ {
		lines = append(lines, "")
	}
	lines = append(lines, bottom...)
	return fitLines(lines, h, w)
}

func (a *App) sidebarTop(w int, compact bool) []string {
	var top []string
	if !compact {
		top = append(top, "", " "+sectionLabel("工作区"), "")
	}
	for i := 0; i < int(pageCount); i++ {
		sel := pageID(i) == a.page
		label := pageNames[i]
		if compact {
			if sel {
				line := lipgloss.NewStyle().Background(colorPanel2).Foreground(colorAccent).Bold(true).Render(" ▸ " + label)
				line = padRight(line, w)
				line = lipgloss.NewStyle().Background(colorPanel2).Render(line)
				top = append(top, line)
			} else {
				top = append(top, " "+styleFaint.Render("·")+" "+styleText.Render(label))
			}
			continue
		}
		eng := pageEnglish[i]
		icon := styleFaint.Render("·")
		if sel {
			bg := lipgloss.NewStyle().Background(colorPanel2)
			line := bg.Foreground(colorAccent).Render(" ▸ ") + bg.Foreground(colorText).Bold(true).Render(label)
			line = padRight(line, w)
			line = lipgloss.NewStyle().Background(colorPanel2).Render(line)
			top = append(top, line)
			top = append(top, lipgloss.NewStyle().Background(colorPanel2).Foreground(colorMuted).Render(padRight("   "+eng, w)))
		} else {
			top = append(top, " "+icon+" "+styleText.Render(label))
			top = append(top, "   "+styleFaint.Render(eng))
		}
		if i < int(pageCount)-1 {
			top = append(top, "")
		}
	}
	return top
}

func (a *App) sidebarBottom(w int, compact bool) []string {
	cfg := a.store.Snapshot()
	acc := cfg.Account()
	ver := cfg.SelectedVersion
	loader := "未安装"
	for _, v := range a.launch.versions {
		if v.ID == ver {
			loader = v.Loader
			if v.MCVersion != "" {
				loader += " · " + v.MCVersion
			}
			break
		}
	}
	accountKind := "离线账户"
	shortKind := "离线"
	if acc.Type == "microsoft" {
		accountKind = "Microsoft 账户"
		shortKind = "Microsoft"
	}
	if compact {
		return []string{
			" " + styleBorder.Render(strings.Repeat("─", max(w-1, 0))),
			" " + sectionLabel("账户"),
			" " + dot(colorGreen) + " " + styleText.Render(acc.Name) + " " + styleFaint.Render("· "+shortKind),
			" " + sectionLabel("当前版本"),
			" " + styleText.Render(middleEllipsis(ver, w-2)),
			"   " + styleFaint.Render(middleEllipsis(loader, w-3)),
		}
	}
	bottom := []string{
		" " + styleBorder.Render(strings.Repeat("─", max(w-1, 0))),
		" " + sectionLabel("账户"),
		" " + dot(colorGreen) + " " + styleText.Render(acc.Name),
		"   " + styleFaint.Render(accountKind),
		"",
		" " + sectionLabel("当前版本"),
		" " + styleText.Render(middleEllipsis(ver, w-2)),
		"   " + styleFaint.Render(middleEllipsis(loader, w-3)),
	}
	if a.rt.Launching() {
		bottom = append(bottom, "", " "+dot(colorGreen)+" "+styleGreen.Render("游戏中"))
	}
	return bottom
}

func (a *App) renderPage(w, h int) []string {
	switch a.page {
	case pageLaunch:
		return a.viewLaunch(w, h)
	case pageVersions:
		return a.viewVersions(w, h)
	case pageDownload:
		return a.viewDownload(w, h)
	case pageSettings:
		return a.viewSettings(w, h)
	case pageTools:
		return a.viewTools(w, h)
	case pageAbout:
		return a.viewAbout(w, h)
	case pageLogs:
		return a.viewLogs(w, h)
	}
	return fitLines(nil, h, w)
}

func pageHeader(title, subtitle string, w int) []string {
	return []string{
		"  " + titleLine(title),
		"   " + styleMuted.Render(ansiTruncate(subtitle, max(w-4, 0))),
		"",
	}
}

func (a *App) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "ctrl+c" {
		return a.requestQuit()
	}
	if a.modal.Kind != modalNone {
		return a.handleModalKey(msg)
	}
	switch msg.String() {
	case "q":
		return a.requestQuit()
	case "?":
		a.openHelp()
		return a, nil
	case ":":
		a.openPalette()
		return a, nil
	case "tab":
		a.switchPage((a.page + 1) % pageCount)
		return a, nil
	case "shift+tab":
		a.switchPage((a.page - 1 + pageCount) % pageCount)
		return a, nil
	case "1", "2", "3", "4", "5", "6", "7":
		for i := range pageNumbers {
			if msg.String() == pageNumbers[i] {
				a.switchPage(pageID(i))
				return a, nil
			}
		}
	case "esc":
		if a.page != pageLaunch {
			a.switchPage(pageLaunch)
			return a, nil
		}
	}
	return a, a.handlePageKey(msg)
}

func (a *App) switchPage(p pageID) {
	if p < 0 || p >= pageCount {
		return
	}
	a.page = p
	switch p {
	case pageVersions:
		a.rt.ScanVersions()
	case pageDownload:
		if a.download.manifest == nil {
			a.rt.LoadManifest(false)
		}
		a.rt.ScanVersions()
	case pageTools:
		a.rt.ScanVersions()
	case pageLogs:
		a.logs.follow = true
	}
}

func (a *App) requestQuit() (tea.Model, tea.Cmd) {
	if a.rt.Launching() {
		a.openConfirm("退出 MCTL", "游戏正在运行。退出启动器将结束游戏进程，确定继续吗？", "quit", nil)
		return a, nil
	}
	if a.rt.Installing() {
		a.openConfirm("退出 MCTL", "安装仍在进行。退出将取消当前安装，确定继续吗？", "quit", nil)
		return a, nil
	}
	return a, tea.Quit
}

func (a *App) setToast(text string, level int) {
	a.toast = toastState{Text: text, Level: level, Until: time.Now().Add(5 * time.Second)}
}

func (a *App) saveCfg(fn func(*config.Config)) {
	if err := a.store.Update(fn); err != nil {
		a.setToast("保存配置失败: "+err.Error(), 2)
	}
}

func (a *App) handleEvent(msg tea.Msg) tea.Cmd {
	switch m := msg.(type) {
	case logEventMsg:
		return nil
	case versionsMsg:
		a.applyVersions(m.List, m.Err)
		return nil
	case manifestMsg:
		a.download.applyManifest(m.Manifest, m.Err)
		if m.Err != nil && a.page == pageDownload {
			a.setToast("获取版本列表失败，按 r 重试", 2)
		}
		return nil
	case javaMsg:
		a.launch.java = m.Info
		a.launch.javaErr = m.Err
		a.settings.detected = m.Info
		if m.Source == "config" && m.Err != nil {
			a.setToast("指定的 Java 无法使用，已回退到自动检测", 3)
		}
		return nil
	case progressMsg:
		if m.Scope == "install" {
			if a.modal.Kind == modalProgress {
				a.modal.Stage = m.P.Stage
				a.modal.Percent = ProgressPercent(m.P)
				a.modal.Detail = m.P.Current
				if m.P.Total > 0 {
					a.modal.Detail = fmt.Sprintf("%s  %d/%d", m.P.Current, m.P.Done, m.P.Total)
				} else if m.P.TotalBytes > 0 {
					a.modal.Detail = fmt.Sprintf("%s  %s/%s", m.P.Current, formatBytes(m.P.Bytes), formatBytes(m.P.TotalBytes))
				}
			}
		} else if m.Scope == "launch" {
			a.launch.stage = m.P.Stage
			a.launch.percent = ProgressPercent(m.P)
			a.launch.detail = m.P.Current
			if m.P.Total > 0 {
				a.launch.detail = fmt.Sprintf("%d/%d", m.P.Done, m.P.Total)
			}
		}
		return nil
	case installDoneMsg:
		if a.modal.Kind == modalProgress {
			a.modal = modalState{}
		}
		if m.Canceled {
			a.setToast("安装已取消", 3)
		} else if m.Err != nil {
			a.openError("安装失败", m.Err)
			a.setToast("安装失败", 2)
		} else {
			label := "安装完成"
			if m.Kind == "modpack" {
				label = "整合包导入完成"
			} else if m.Kind == "resource" {
				label = "资源安装完成"
			} else if m.Kind == "labymod" {
				label = "LabyMod 安装完成"
			}
			a.setToast(label+": "+m.ID, 1)
			a.launch.stage = ""
			a.launch.percent = 0
		}
		return nil
	case fabricLoadersMsg:
		a.download.loaderLoading = false
		if m.Err != nil {
			a.openError("获取 Fabric 版本失败", m.Err)
			return nil
		}
		if len(m.Loaders) == 0 {
			a.setToast("该版本没有可用的 Fabric 加载器", 3)
			return nil
		}
		items := make([]pickerItem, 0, len(m.Loaders))
		for _, l := range m.Loaders {
			desc := "测试版"
			if l.Stable {
				desc = "稳定版"
			}
			items = append(items, pickerItem{Label: l.Version, Desc: desc, Value: l.Version})
		}
		a.modal = modalState{
			Kind:     modalPicker,
			Title:    "选择 Fabric 加载器 · " + m.MC,
			Items:    items,
			Action:   "install-fabric-loader",
			Data:     m.MC,
			Selected: 0,
		}
		return nil
	case worldDoneMsg:
		if m.Err != nil {
			a.openError("存档操作失败", m.Err)
			return nil
		}
		if m.Action == "import" {
			a.setToast("存档已导入: "+m.Name, 1)
			a.openWorlds()
		} else {
			a.setToast("存档已备份: "+filepath.Base(m.Path), 1)
		}
		return nil
	case diagnoseDoneMsg:
		items := make([]pickerItem, 0, len(m.Checks))
		okCount := 0
		for _, c := range m.Checks {
			if c.Status == minecraft.DiagnoseOK {
				okCount++
			}
			items = append(items, pickerItem{Label: c.Name, Desc: c.Detail, Value: string(c.Status)})
		}
		a.modal = modalState{
			Kind:  modalDiagnostics,
			Title: fmt.Sprintf("环境自检 · %d/%d 通过", okCount, len(items)),
			Items: items,
		}
		return nil
	case updateCheckMsg:
		if m.Err != nil {
			a.openError("检查更新失败", m.Err)
			return nil
		}
		if m.Info == nil {
			a.setToast("没有找到发布信息", 3)
			return nil
		}
		if minecraft.IsNewerVersion(m.Info.TagName, m.Current) {
			a.openConfirm("发现新版本 "+m.Info.TagName,
				fmt.Sprintf("当前版本 v%s，最新版本 %s。按 enter 打开下载页面。", m.Current, m.Info.TagName),
				"open-release", m.Info.HTMLURL)
		} else {
			a.setToast("已是最新版本 v"+m.Current, 1)
		}
		return nil
	case instanceDoneMsg:
		if a.modal.Kind == modalProgress && a.modal.Action == "instance-op" {
			a.modal = modalState{}
		}
		if m.Err != nil {
			a.openError("实例操作失败", m.Err)
			return nil
		}
		if m.Action == "import" {
			msg := "实例已导入: " + m.Name
			if m.Warn != nil {
				msg = "实例已导入: " + m.Name + "（联网后可补全文件）"
			}
			a.setToast(msg, 1)
			a.switchPage(pageVersions)
		} else {
			a.setToast("实例已导出: "+filepath.Base(m.Path), 1)
		}
		return nil
	case resourcesMsg:
		a.resources.kind = m.Kind
		if m.Err != nil {
			a.openError("搜索在线资源失败", m.Err)
			return nil
		}
		if len(m.Projects) == 0 {
			a.setToast("没有找到匹配的资源", 3)
			return nil
		}
		items := make([]pickerItem, 0, len(m.Projects))
		for _, p := range m.Projects {
			desc := fmt.Sprintf("%d 次下载", p.Downloads)
			if p.Author != "" {
				desc = p.Author + " · " + desc
			}
			items = append(items, pickerItem{Label: p.Title, Desc: desc, Value: p.ID})
		}
		a.modal = modalState{Kind: modalPicker, Title: "在线资源 · " + resourceKindLabel(m.Kind), Items: items, Action: "install-resource"}
		return nil
	case modpacksMsg:
		if m.Err != nil {
			a.openError("搜索在线整合包失败", m.Err)
			return nil
		}
		if len(m.Projects) == 0 {
			a.setToast("没有找到匹配的整合包", 3)
			return nil
		}
		items := make([]pickerItem, 0, len(m.Projects))
		for _, p := range m.Projects {
			desc := fmt.Sprintf("%d 次下载", p.Downloads)
			if p.Author != "" {
				desc = p.Author + " · " + desc
			}
			items = append(items, pickerItem{Label: p.Title, Desc: desc, Value: p.ID})
		}
		a.modal = modalState{Kind: modalPicker, Title: "在线整合包 · Modrinth", Items: items, Action: "install-online-modpack"}
		return nil
	case microsoftCodeMsg:
		if a.modal.Kind == modalProgress && a.modal.Action == "cancel-auth" {
			a.modal.Stage = "等待 Microsoft 授权"
			a.modal.Detail = fmt.Sprintf("%s · 代码 %s", m.Code.VerificationURI, m.Code.UserCode)
		}
		return nil
	case microsoftDoneMsg:
		a.modal = modalState{}
		if m.Canceled {
			a.setToast("已取消 Microsoft 登录", 3)
			return nil
		}
		if m.Err != nil {
			a.openError("Microsoft 登录失败", m.Err)
			return nil
		}
		a.setToast("已登录 Microsoft 账户 "+m.Account.Name, 1)
		a.openAccounts()
		return nil
	case loaderVersionsMsg:
		a.download.loaderLoading = false
		name := "Forge"
		switch m.Kind {
		case "neoforge":
			name = "NeoForge"
		case "quilt":
			name = "Quilt"
		case "legacyfabric":
			name = "Legacy Fabric"
		case "optifine":
			name = "OptiFine"
		case "liteloader":
			name = "LiteLoader"
		case "cleanroom":
			name = "Cleanroom"
		case "labymod":
			name = "LabyMod"
		}
		if m.Err != nil {
			a.openError("获取 "+name+" 版本失败", m.Err)
			return nil
		}
		if len(m.Builds) == 0 {
			a.setToast("该版本没有可用的 "+name+" 构建", 3)
			return nil
		}
		items := make([]pickerItem, 0, len(m.Builds))
		for _, b := range m.Builds {
			desc := "测试版"
			if b.Stable {
				desc = "稳定版"
			}
			value := b.Version
			label := b.Version
			if m.Kind == "optifine" {
				value = b.Filename
			}
			if m.Kind == "labymod" {
				value = b.Channel
				label = b.Version + " · " + b.Channel
			}
			items = append(items, pickerItem{Label: label, Desc: desc, Value: value})
		}
		a.modal = modalState{
			Kind:     modalPicker,
			Title:    "选择 " + name + " 版本 · " + m.MC,
			Items:    items,
			Action:   "install-loader",
			Data:     loaderPickerData{Kind: m.Kind, MC: m.MC},
			Selected: 0,
		}
		return nil
	case launchStageMsg:
		a.launch.stage = m.Stage
		a.launch.percent = m.Percent
		a.launch.detail = m.Detail
		return nil
	case launchStartedMsg:
		a.launch.stage = "游戏运行中"
		a.launch.percent = 1
		a.setToast("游戏已启动", 1)
		return nil
	case launchExitMsg:
		a.launch.stage = ""
		a.launch.percent = 0
		a.launch.detail = ""
		if m.Stopped {
			a.setToast("游戏已停止", 3)
		} else if m.Err != nil {
			a.openError("游戏异常退出", m.Err)
		} else {
			a.setToast("游戏已退出", 0)
		}
		return nil
	}
	return nil
}

func (a *App) applyVersions(list []minecraft.InstalledVersion, err error) {
	if err != nil {
		a.setToast("读取版本失败: "+err.Error(), 2)
		return
	}
	a.launch.versions = list
	a.versions.list = list
	a.download.installed = map[string]bool{}
	for _, v := range list {
		a.download.installed[v.ID] = true
	}
	cfg := a.store.Snapshot()
	found := -1
	for i, v := range list {
		if v.ID == cfg.SelectedVersion {
			found = i
			break
		}
	}
	if found < 0 && len(list) > 0 {
		found = 0
		a.saveCfg(func(c *config.Config) { c.SelectedVersion = list[0].ID })
	}
	if found >= 0 {
		a.launch.versionIdx = found
	}
	if len(list) == 0 {
		a.launch.versionIdx = 0
	}
	if a.versions.idx >= len(a.filteredVersions()) {
		a.versions.idx = max(len(a.filteredVersions())-1, 0)
	}
}

func (a *App) startLaunch(versionID string) {
	if versionID == "" {
		a.setToast("请先选择一个已安装的版本", 3)
		return
	}
	if a.rt.Busy() {
		a.setToast("已有任务正在进行", 3)
		return
	}
	a.page = pageLaunch
	a.saveCfg(func(c *config.Config) { c.SelectedVersion = versionID })
	a.launch.stage = "准备启动"
	a.launch.percent = 0
	gameDir := a.store.Snapshot().GameDir
	for _, v := range a.launch.versions {
		if v.ID == versionID && v.GameDir != "" {
			gameDir = v.GameDir
			break
		}
	}
	a.rt.Launch(versionID, gameDir)
}

func (a *App) startResourceInstall(kind, projectID, name string) {
	if kind == "" || projectID == "" {
		a.setToast("无效的在线资源", 3)
		return
	}
	if kind == "mod" && a.store.Snapshot().SelectedVersion == "" {
		a.setToast("安装模组前请先在版本页选择版本", 3)
		return
	}
	if a.rt.Busy() {
		a.setToast("已有任务正在进行", 3)
		return
	}
	a.modal = modalState{
		Kind:    modalProgress,
		Title:   "在线资源 · " + name,
		Stage:   "获取版本",
		Percent: 0,
		Action:  "cancel-install",
	}
	a.rt.InstallResource(kind, projectID, name)
}

func (a *App) startOnlineModpack(projectID, name string) {
	if projectID == "" {
		a.setToast("无效的整合包", 3)
		return
	}
	if a.rt.Busy() {
		a.setToast("已有任务正在进行", 3)
		return
	}
	a.modal = modalState{
		Kind:    modalProgress,
		Title:   "在线整合包 · " + name,
		Stage:   "获取版本",
		Percent: 0,
		Action:  "cancel-install",
	}
	a.rt.InstallOnlineModpack(projectID, name)
}

func (a *App) startModpackImport(path string) {
	if path == "" {
		a.setToast("请填写整合包路径", 3)
		return
	}
	if a.rt.Busy() {
		a.setToast("已有任务正在进行", 3)
		return
	}
	a.modal = modalState{
		Kind:    modalProgress,
		Title:   "导入整合包",
		Stage:   "准备中",
		Percent: 0,
		Action:  "cancel-install",
	}
	a.rt.ImportModpack(path)
}

func (a *App) startInstall(kind, id, mc, loader string, entry *minecraft.VersionEntry) {
	if a.rt.Busy() {
		a.setToast("已有任务正在进行", 3)
		return
	}
	title := "安装 " + id
	switch kind {
	case "fabric":
		title = fmt.Sprintf("安装 Fabric %s · %s", loader, mc)
	case "quilt":
		title = fmt.Sprintf("安装 Quilt %s · %s", loader, mc)
	case "legacyfabric":
		title = fmt.Sprintf("安装 Legacy Fabric %s · %s", loader, mc)
	case "optifine":
		title = fmt.Sprintf("安装 OptiFine · %s", mc)
	case "liteloader":
		title = fmt.Sprintf("安装 LiteLoader %s · %s", loader, mc)
	case "cleanroom":
		title = fmt.Sprintf("安装 Cleanroom %s · %s", loader, mc)
	case "labymod":
		title = fmt.Sprintf("安装 LabyMod %s · %s", loader, mc)
	case "forge":
		title = fmt.Sprintf("安装 Forge %s · %s", loader, mc)
	case "neoforge":
		title = fmt.Sprintf("安装 NeoForge %s · %s", loader, mc)
	}
	a.modal = modalState{
		Kind:    modalProgress,
		Title:   title,
		Stage:   "准备中",
		Percent: 0,
		Action:  "cancel-install",
	}
	a.rt.Install(kind, id, mc, loader, entry)
}

func (a *App) openGameDir() {
	cfg := a.store.Snapshot()
	if err := os.MkdirAll(cfg.GameDir, 0o755); err != nil {
		a.setToast("无法创建游戏目录: "+err.Error(), 2)
		return
	}
	if err := openPath(cfg.GameDir); err != nil {
		a.setToast("打开目录失败: "+err.Error(), 2)
	}
}

func openPath(p string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("explorer", p)
	case "darwin":
		cmd = exec.Command("open", p)
	default:
		cmd = exec.Command("xdg-open", p)
	}
	return cmd.Start()
}
