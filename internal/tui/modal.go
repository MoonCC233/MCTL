package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/MoonCC233/MCTL/internal/minecraft"
)

type modalKind int

const (
	modalNone modalKind = iota
	modalConfirm
	modalInput
	modalPicker
	modalProgress
	modalHelp
	modalAccounts
	modalPalette
	modalError
)

type pickerItem struct {
	Label string
	Desc  string
	Value string
}

type modalState struct {
	Kind        modalKind
	Title       string
	Body        string
	Label       string
	Placeholder string
	Action      string
	Stage       string
	Detail      string
	Value       []rune
	Cursor      int
	Items       []pickerItem
	Selected    int
	Data        any
	Percent     float64
	Err         error
	Query       []rune
}

func (a *App) openConfirm(title, body, action string, data any) {
	a.modal = modalState{Kind: modalConfirm, Title: title, Body: body, Action: action, Data: data}
}

func (a *App) openInput(title, label, value, placeholder, action string) {
	a.modal = modalState{Kind: modalInput, Title: title, Label: label, Value: []rune(value), Cursor: len([]rune(value)), Placeholder: placeholder, Action: action}
}

func (a *App) openError(title string, err error) {
	a.modal = modalState{Kind: modalError, Title: title, Err: err, Body: err.Error()}
}

func (a *App) openHelp() {
	a.modal = modalState{Kind: modalHelp, Title: "帮助"}
}

func (a *App) openAccounts() {
	cfg := a.store.Snapshot()
	items := make([]pickerItem, 0, len(cfg.Accounts))
	sel := 0
	for i, acc := range cfg.Accounts {
		desc := "离线账户 · " + minecraft.OfflineUUID(acc.Name)
		if acc.Type == "microsoft" {
			desc = "Microsoft 账户"
		}
		if acc.Name == cfg.SelectedAccount {
			if acc.Type == "microsoft" {
				desc = "当前账户 · Microsoft"
			} else {
				desc = "当前账户 · 离线"
			}
			sel = i
		}
		items = append(items, pickerItem{Label: acc.Name, Desc: desc, Value: acc.Name})
	}
	a.modal = modalState{Kind: modalAccounts, Title: "账户管理", Items: items, Selected: sel}
}

func (a *App) openPalette() {
	a.modal = modalState{Kind: modalPalette, Title: "命令面板", Items: a.paletteItems(), Query: []rune{}}
}

func (a *App) paletteItems() []pickerItem {
	return []pickerItem{
		{Label: "启动游戏", Desc: "启动当前选择的版本", Value: "cmd-launch"},
		{Label: "版本管理", Desc: "查看已安装的版本", Value: "cmd-page-versions"},
		{Label: "下载", Desc: "安装原版、Fabric、Forge 或 NeoForge", Value: "cmd-page-download"},
		{Label: "导入整合包", Desc: "导入本地 Modrinth / CurseForge 整合包", Value: "cmd-import-modpack"},
		{Label: "在线整合包", Desc: "搜索并安装 Modrinth 整合包", Value: "cmd-online-modpacks"},
		{Label: "在线资源", Desc: "搜索 Modrinth 资源包、光影或模组", Value: "cmd-online-resources"},
		{Label: "设置", Desc: "Java、内存与账户设置", Value: "cmd-page-settings"},
		{Label: "工具", Desc: "目录、Java、日志与整合包导入", Value: "cmd-page-tools"},
		{Label: "关于 MCTL", Desc: "版本、路径与项目信息", Value: "cmd-page-about"},
		{Label: "运行日志", Desc: "查看启动器与游戏输出", Value: "cmd-page-logs"},
		{Label: "添加离线账户", Desc: "创建新的离线账户", Value: "cmd-account"},
		{Label: "Microsoft 登录", Desc: "使用设备代码登录微软账户", Value: "cmd-ms-login"},
		{Label: "检测 Java", Desc: "重新扫描 Java 运行时", Value: "cmd-java"},
		{Label: "刷新版本列表", Desc: "重新扫描游戏目录", Value: "cmd-refresh"},
		{Label: "打开游戏目录", Desc: "在文件管理器中打开", Value: "cmd-folder"},
		{Label: "帮助", Desc: "查看快捷键", Value: "cmd-help"},
		{Label: "退出 MCTL", Desc: "关闭启动器", Value: "cmd-quit"},
	}
}

func (a *App) filteredPalette() []pickerItem {
	q := strings.ToLower(string(a.modal.Query))
	if q == "" {
		return a.modal.Items
	}
	var out []pickerItem
	for _, it := range a.modal.Items {
		if strings.Contains(strings.ToLower(it.Label+" "+it.Desc), q) {
			out = append(out, it)
		}
	}
	return out
}

func (a *App) handleModalKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m := &a.modal
	key := msg.String()
	if m.Kind == modalProgress {
		if key == "esc" {
			switch m.Action {
			case "cancel-install":
				a.rt.CancelInstall()
				m.Stage = "正在取消"
				a.setToast("正在取消安装…", 3)
			case "cancel-auth":
				a.rt.CancelMicrosoftLogin()
				a.modal = modalState{}
				a.setToast("已取消 Microsoft 登录", 3)
			}
			return a, nil
		}
		return a, nil
	}
	switch m.Kind {
	case modalHelp:
		if key == "esc" || key == "enter" || key == "q" || key == "?" {
			a.modal = modalState{}
		}
	case modalError:
		if key == "esc" || key == "enter" || key == "q" {
			a.modal = modalState{}
		}
	case modalConfirm:
		switch key {
		case "enter", "y":
			action, data := m.Action, m.Data
			a.modal = modalState{}
			return a, a.runAction(action, "", pickerItem{Value: ""}, data)
		case "esc", "n", "q":
			a.modal = modalState{}
		}
	case modalInput:
		switch key {
		case "esc":
			a.modal = modalState{}
		case "enter":
			value := string(m.Value)
			action := m.Action
			a.modal = modalState{}
			return a, a.runAction(action, value, pickerItem{}, nil)
		default:
			a.editInput(m, msg)
		}
	case modalPicker:
		switch key {
		case "esc":
			a.modal = modalState{}
		case "up", "k":
			if len(m.Items) > 0 {
				m.Selected = (m.Selected - 1 + len(m.Items)) % len(m.Items)
			}
		case "down", "j":
			if len(m.Items) > 0 {
				m.Selected = (m.Selected + 1) % len(m.Items)
			}
		case "pgup":
			m.Selected = max(m.Selected-8, 0)
		case "pgdown":
			m.Selected = min(m.Selected+8, len(m.Items)-1)
		case "home":
			m.Selected = 0
		case "end":
			if len(m.Items) > 0 {
				m.Selected = len(m.Items) - 1
			}
		case "enter":
			if len(m.Items) == 0 {
				return a, nil
			}
			it := m.Items[m.Selected]
			action, data := m.Action, m.Data
			a.modal = modalState{}
			return a, a.runAction(action, "", it, data)
		}
	case modalAccounts:
		switch key {
		case "esc":
			a.modal = modalState{}
		case "up", "k":
			if len(m.Items) > 0 {
				m.Selected = (m.Selected - 1 + len(m.Items)) % len(m.Items)
			}
		case "down", "j":
			if len(m.Items) > 0 {
				m.Selected = (m.Selected + 1) % len(m.Items)
			}
		case "enter":
			if len(m.Items) > 0 {
				it := m.Items[m.Selected]
				a.modal = modalState{}
				return a, a.runAction("select-account", "", it, nil)
			}
		case "a":
			a.openInput("添加离线账户", "玩家名（3-16 位字母、数字或下划线）", "", "Steve", "add-account")
		case "m":
			a.openMicrosoftLogin()
		case "d":
			if len(m.Items) > 0 {
				name := m.Items[m.Selected].Value
				a.openConfirm("删除账户", fmt.Sprintf("确定删除离线账户 %s 吗？", name), "remove-account", name)
			}
		}
	case modalPalette:
		switch key {
		case "esc":
			a.modal = modalState{}
		case "up":
			if len(a.filteredPalette()) > 0 {
				m.Selected = (m.Selected - 1 + len(a.filteredPalette())) % len(a.filteredPalette())
			}
		case "down":
			if len(a.filteredPalette()) > 0 {
				m.Selected = (m.Selected + 1) % len(a.filteredPalette())
			}
		case "enter":
			list := a.filteredPalette()
			if len(list) == 0 {
				return a, nil
			}
			it := list[m.Selected]
			a.modal = modalState{}
			return a, a.runAction("command", "", it, nil)
		default:
			a.editQuery(m, msg)
			if m.Selected >= len(a.filteredPalette()) {
				m.Selected = max(len(a.filteredPalette())-1, 0)
			}
		}
	}
	return a, nil
}

func (a *App) editInput(m *modalState, msg tea.KeyMsg) {
	switch msg.String() {
	case "backspace":
		if m.Cursor > 0 {
			m.Value = append(m.Value[:m.Cursor-1], m.Value[m.Cursor:]...)
			m.Cursor--
		}
	case "delete":
		if m.Cursor < len(m.Value) {
			m.Value = append(m.Value[:m.Cursor], m.Value[m.Cursor+1:]...)
		}
	case "left":
		if m.Cursor > 0 {
			m.Cursor--
		}
	case "right":
		if m.Cursor < len(m.Value) {
			m.Cursor++
		}
	case "home":
		m.Cursor = 0
	case "end":
		m.Cursor = len(m.Value)
	case "ctrl+u":
		m.Value = nil
		m.Cursor = 0
	case "ctrl+w":
		for m.Cursor > 0 && m.Value[m.Cursor-1] == ' ' {
			m.Value = append(m.Value[:m.Cursor-1], m.Value[m.Cursor:]...)
			m.Cursor--
		}
		for m.Cursor > 0 && m.Value[m.Cursor-1] != ' ' {
			m.Value = append(m.Value[:m.Cursor-1], m.Value[m.Cursor:]...)
			m.Cursor--
		}
	default:
		if len(msg.Runes) > 0 {
			runes := msg.Runes
			out := make([]rune, 0, len(m.Value)+len(runes))
			out = append(out, m.Value[:m.Cursor]...)
			out = append(out, runes...)
			out = append(out, m.Value[m.Cursor:]...)
			m.Value = out
			m.Cursor += len(runes)
		}
	}
}

func (a *App) editQuery(m *modalState, msg tea.KeyMsg) {
	switch msg.String() {
	case "backspace":
		if m.Cursor > 0 {
			m.Query = append(m.Query[:m.Cursor-1], m.Query[m.Cursor:]...)
			m.Cursor--
		}
	case "delete":
		if m.Cursor < len(m.Query) {
			m.Query = append(m.Query[:m.Cursor], m.Query[m.Cursor+1:]...)
		}
	case "left":
		if m.Cursor > 0 {
			m.Cursor--
		}
	case "right":
		if m.Cursor < len(m.Query) {
			m.Cursor++
		}
	case "home":
		m.Cursor = 0
	case "end":
		m.Cursor = len(m.Query)
	case "ctrl+u":
		m.Query = nil
		m.Cursor = 0
	default:
		if len(msg.Runes) > 0 {
			out := make([]rune, 0, len(m.Query)+len(msg.Runes))
			out = append(out, m.Query[:m.Cursor]...)
			out = append(out, msg.Runes...)
			out = append(out, m.Query[m.Cursor:]...)
			m.Query = out
			m.Cursor += len(msg.Runes)
		}
	}
}

func (a *App) renderModal() string {
	m := &a.modal
	maxW := max(a.width-8, 30)
	maxH := max(a.height-4, 8)
	boxW := 64
	switch m.Kind {
	case modalConfirm:
		boxW = 58
	case modalHelp:
		boxW = 76
	case modalError:
		boxW = 74
	case modalPalette:
		boxW = 66
	case modalAccounts:
		boxW = 68
	case modalPicker:
		boxW = 66
	case modalProgress:
		boxW = 66
	}
	boxW = min(boxW, maxW)
	inner := boxW - 4
	var content []string
	add := func(s string) { content = append(content, s) }

	switch m.Kind {
	case modalConfirm:
		add("")
		for _, l := range wrapText(m.Body, inner-2) {
			add("  " + styleMuted.Render(l))
		}
		add("")
		add("  " + keyHint("enter", "确认") + "  " + keyHint("esc", "取消"))
	case modalInput:
		add("")
		for _, l := range wrapText(m.Label, inner-2) {
			add("  " + styleMuted.Render(l))
		}
		add("")
		add("  " + a.inputView(m, inner-4))
		add("")
		add("  " + keyHint("enter", "确定") + "  " + keyHint("esc", "取消"))
	case modalPicker:
		add("")
		visible := min(10, max(maxH-12, 3))
		start := 0
		if m.Selected >= visible {
			start = m.Selected - visible + 1
		}
		if start > len(m.Items)-visible {
			start = max(len(m.Items)-visible, 0)
		}
		end := min(start+visible, len(m.Items))
		if len(m.Items) == 0 {
			add("  " + styleFaint.Render("没有可选项"))
		}
		for i := start; i < end; i++ {
			it := m.Items[i]
			line := "  " + padRight(ansiTruncate(it.Label, max(inner-22, 10)), max(inner-22, 10)) + "  " + styleFaint.Render(ansiTruncate(it.Desc, 18))
			if i == m.Selected {
				add(lipglossRow("  ▸ "+ansiTruncate(it.Label, max(inner-22, 10))+"  "+ansiTruncate(it.Desc, 18), inner))
			} else {
				add(line)
			}
		}
		add("")
		add("  " + keyHint("↑↓", "选择") + "  " + keyHint("enter", "确定") + "  " + keyHint("esc", "取消"))
	case modalAccounts:
		add("")
		for i, it := range m.Items {
			line := "  " + styleText.Render(padRight(it.Label, 16)) + "  " + styleFaint.Render(ansiTruncate(it.Desc, max(inner-24, 12)))
			if i == m.Selected {
				add(lipglossRow("  ▸ "+padRight(it.Label, 16)+"  "+ansiTruncate(it.Desc, max(inner-24, 12)), inner))
			} else {
				add(line)
			}
		}
		add("")
		add("  " + keyHint("enter", "选择") + "  " + keyHint("a", "离线") + "  " + keyHint("m", "Microsoft") + "  " + keyHint("d", "删除") + "  " + keyHint("esc", "关闭"))
	case modalProgress:
		add("")
		stage := m.Stage
		if stage == "" {
			stage = "准备中"
		}
		add("  " + styleText.Render(stage))
		add("")
		add("  " + progressBar(m.Percent, max(inner-4, 10)))
		add("  " + styleMuted.Render(fmt.Sprintf("%d%%", int(m.Percent*100))))
		if m.Detail != "" {
			add("  " + styleFaint.Render(ansiTruncate(m.Detail, inner-4)))
		}
		add("")
		add("  " + keyHint("esc", "取消安装"))
	case modalHelp:
		add("")
		for _, l := range helpLines() {
			add("  " + l)
		}
		add("")
		add("  " + keyHint("esc", "关闭"))
	case modalError:
		add("")
		for _, l := range wrapText(m.Body, inner-2) {
			add("  " + styleRed.Render(l))
		}
		add("")
		add("  " + keyHint("enter", "关闭"))
	case modalPalette:
		add("")
		add("  " + styleAccent.Render("> ") + a.paletteQueryView(inner-4))
		add("")
		list := a.filteredPalette()
		visible := min(9, max(maxH-12, 3))
		start := 0
		if m.Selected >= visible {
			start = m.Selected - visible + 1
		}
		if start > len(list)-visible {
			start = max(len(list)-visible, 0)
		}
		end := min(start+visible, len(list))
		if len(list) == 0 {
			add("  " + styleFaint.Render("没有匹配的命令"))
		}
		for i := start; i < end; i++ {
			it := list[i]
			if i == m.Selected {
				add(lipglossRow("  ▸ "+padRight(it.Label, 18)+"  "+ansiTruncate(it.Desc, max(inner-24, 10)), inner))
			} else {
				add("    " + styleText.Render(padRight(it.Label, 18)) + "  " + styleFaint.Render(ansiTruncate(it.Desc, max(inner-24, 10))))
			}
		}
		add("")
		add("  " + keyHint("↑↓", "选择") + "  " + keyHint("enter", "执行") + "  " + keyHint("esc", "关闭"))
	}
	if len(content)+2 > maxH {
		content = content[:max(maxH-2, 1)]
	}
	boxH := len(content) + 2
	return panel(m.Title, boxW, boxH, true, content)
}

func (a *App) inputView(m *modalState, width int) string {
	if width < 4 {
		width = 4
	}
	if len(m.Value) == 0 {
		cursor := styleAccent.Render("▏")
		if m.Placeholder != "" {
			return cursor + styleFaint.Render(ansiTruncate(m.Placeholder, width-1))
		}
		return cursor
	}
	start := 0
	avail := width - 1
	if m.Cursor > avail {
		start = m.Cursor - avail
	}
	end := min(start+avail, len(m.Value))
	before := string(m.Value[start:m.Cursor])
	after := string(m.Value[m.Cursor:end])
	left := before
	if lipgloss.Width(left) > width-1 {
		left = ansiTruncate(left, width-1)
	}
	return styleText.Render(left) + styleAccent.Render("▏") + styleText.Render(after)
}

func (a *App) paletteQueryView(width int) string {
	m := &a.modal
	if len(m.Query) == 0 {
		return styleAccent.Render("▏") + styleFaint.Render("输入命令…")
	}
	return styleText.Render(string(m.Query)) + styleAccent.Render("▏")
}

func helpLines() []string {
	accent := func(s string) string { return styleAccent.Render(s) }
	muted := func(s string) string { return styleMuted.Render(s) }
	return []string{
		styleBold.Render("MCTL") + " " + styleFaint.Render("v"+Version) + "  " + muted("Minecraft 终端启动器"),
		"",
		styleSection.Render("导航"),
		"  " + accent("tab / shift+tab") + muted("  切换页面"),
		"  " + accent("1 - 7") + muted("            快速跳转页面"),
		"  " + accent(":") + muted("                命令面板"),
		"  " + accent("?") + muted("                帮助"),
		"  " + accent("q") + muted("                退出"),
		"",
		styleSection.Render("启动页"),
		"  " + accent("↑↓ / ←→") + muted("         选择项目 / 切换值"),
		"  " + accent("enter") + muted("            启动游戏"),
		"  " + accent("a") + muted("                账户管理（含 Microsoft 登录）"),
		"  " + accent("s") + muted("                停止游戏"),
		"",
		styleSection.Render("版本 / 下载"),
		"  " + accent("enter") + muted("            启动 / 安装原版或加载器"),
		"  " + accent("d / o / / / r") + muted("    删除 / 目录 / 筛选 / 刷新"),
		"  " + accent("m / p / s / h / f") + muted("  模组 / 资源包 / 存档 / 截图 / 配置"),
		"  " + accent("i / o") + muted("            导入 / 在线整合包"),
		"",
		styleSection.Render("设置 / 工具 / 日志"),
		"  " + accent("enter / ←→") + muted("       编辑 / 调整"),
		"  " + accent("i / w / e / v / r") + muted("  导入 / 整合包 / 资源 / Java / 刷新"),
		"  " + accent("↑↓ / f / c") + muted("       滚动 / 跟随 / 清空"),
	}
}
