package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

type hitRegion struct {
	x0, y0, x1, y1 int
	action         string
	data           any
}

type clickStamp struct {
	key string
	at  time.Time
}

// addHit records a clickable rectangle in the current render pass. Coordinates
// are relative to the active hit origin (sidebar or page area).
func (a *App) addHit(x0, y0, x1, y1 int, action string, data any) {
	if x0 > x1 {
		x0, x1 = x1, x0
	}
	if y0 > y1 {
		y0, y1 = y1, y0
	}
	a.hits = append(a.hits, hitRegion{
		x0: a.hitOriginX + x0, y0: a.hitOriginY + y0,
		x1: a.hitOriginX + x1, y1: a.hitOriginY + y1,
		action: action, data: data,
	})
}

func (a *App) hitAt(x, y int) (hitRegion, bool) {
	for i := len(a.hits) - 1; i >= 0; i-- {
		h := a.hits[i]
		if x >= h.x0 && x <= h.x1 && y >= h.y0 && y <= h.y1 {
			return h, true
		}
	}
	return hitRegion{}, false
}

func stripANSI(s string) string { return ansi.Strip(s) }

func (a *App) inModal(x, y int) bool {
	return a.modalW > 0 && a.modalH > 0 &&
		x >= a.modalX && x < a.modalX+a.modalW &&
		y >= a.modalY && y < a.modalY+a.modalH
}

func (a *App) isDoubleClick(key string, now time.Time) bool {
	if a.lastClick.key == key && now.Sub(a.lastClick.at) < 450*time.Millisecond {
		a.lastClick = clickStamp{}
		return true
	}
	a.lastClick = clickStamp{key: key, at: now}
	return false
}

func keyMsg(key string) tea.KeyMsg {
	switch key {
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "up":
		return tea.KeyMsg{Type: tea.KeyUp}
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	case "left":
		return tea.KeyMsg{Type: tea.KeyLeft}
	case "right":
		return tea.KeyMsg{Type: tea.KeyRight}
	case "pgup":
		return tea.KeyMsg{Type: tea.KeyPgUp}
	case "pgdown":
		return tea.KeyMsg{Type: tea.KeyPgDown}
	}
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
}

type contextMenuData struct {
	Kind  string
	Index int
}

func (a *App) handleRightClick(x, y int) (tea.Model, tea.Cmd) {
	if a.modal.Kind != modalNone {
		return a, nil
	}
	h, ok := a.hitAt(x, y)
	if !ok {
		return a, nil
	}
	switch h.action {
	case "versions-row":
		idx, _ := h.data.(int)
		list := a.filteredVersions()
		if idx < 0 || idx >= len(list) {
			return a, nil
		}
		a.versions.idx = idx
		items := []pickerItem{
			{Label: "启动", Desc: "启动选中版本", Value: "enter"},
			{Label: "删除版本", Desc: "移除版本目录", Value: "d"},
			{Label: "打开版本目录", Desc: "在文件管理器中打开", Value: "o"},
			{Label: "实例设置", Desc: "内存 / JVM / 自动进服", Value: "i"},
			{Label: "实例安装 / 修复", Desc: "加载器 / 补全文件", Value: "l"},
			{Label: "导出实例", Desc: "打包为 zip", Value: "x"},
			{Label: "新建实例", Desc: "打开实例向导", Value: "+"},
		}
		a.modal = modalState{
			Kind:   modalPicker,
			Title:  tr("版本操作") + " · " + list[idx].ID,
			Items:  items,
			Data:   contextMenuData{Kind: "versions", Index: idx},
			Action: "context-action",
		}
	case "download-row":
		idx, _ := h.data.(int)
		list := a.download.versions()
		if idx < 0 || idx >= len(list) {
			return a, nil
		}
		a.download.idx = idx
		items := []pickerItem{
			{Label: "安装 / 修复", Desc: "安装选中版本", Value: "enter"},
			{Label: "切换版本类型", Desc: "原版 / Fabric / Forge / ...", Value: "mode"},
			{Label: "刷新版本清单", Desc: "重新获取列表", Value: "r"},
		}
		a.modal = modalState{
			Kind:   modalPicker,
			Title:  tr("下载操作") + " · " + list[idx].ID,
			Items:  items,
			Data:   contextMenuData{Kind: "download", Index: idx},
			Action: "context-action",
		}
	}
	return a, nil
}

func (a *App) openDownloadModePicker() {
	items := make([]pickerItem, 0, len(loaderModeNames))
	for i, name := range loaderModeNames {
		items = append(items, pickerItem{Label: tr(name), Desc: "", Value: fmt.Sprint(i)})
	}
	a.modal = modalState{Kind: modalPicker, Title: tr("版本类型"), Items: items, Action: "context-download-mode"}
}

func (a *App) hoverRow(action string, idx int) bool {
	return a.hoverAction == action && a.hoverData == idx
}

func (a *App) hoverHint() string {
	if a.modal.Kind != modalNone {
		return ""
	}
	switch a.hoverAction {
	case "page":
		return tr("单击切换页面")
	case "versions-row":
		return tr("单击选中 · 双击启动 · 右键更多")
	case "versions-action":
		return tr("单击执行操作")
	case "download-row":
		return tr("单击选中 · 双击安装 · 右键更多")
	case "download-tab":
		return tr("单击切换类型")
	case "download-action":
		return tr("单击安装 / 选择加载器")
	case "launch-version":
		return tr("单击选择版本")
	case "launch-account":
		return tr("单击管理账户")
	case "launch-start":
		return tr("单击启动 / 停止")
	case "launch-news":
		return tr("单击查看更新日志")
	case "settings-row":
		return tr("单击编辑或切换")
	case "tool-row":
		return tr("单击执行")
	case "modal-item":
		return tr("单击选择")
	case "help":
		return tr("单击打开帮助")
	case "quit":
		return tr("单击退出")
	}
	return ""
}

func (a *App) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	a.mouseX, a.mouseY = msg.X, msg.Y
	if msg.Action == tea.MouseActionMotion {
		if h, ok := a.hitAt(msg.X, msg.Y); ok {
			a.hoverAction, a.hoverData = h.action, h.data
		} else {
			a.hoverAction, a.hoverData = "", nil
		}
		return a, nil
	}
	if msg.Button == tea.MouseButtonRight {
		if msg.Action == tea.MouseActionPress {
			return a.handleRightClick(msg.X, msg.Y)
		}
		return a, nil
	}
	if msg.Button == tea.MouseButtonWheelUp {
		return a, a.mouseWheel(-1)
	}
	if msg.Button == tea.MouseButtonWheelDown {
		return a, a.mouseWheel(1)
	}
	if msg.Action != tea.MouseActionPress || msg.Button != tea.MouseButtonLeft {
		return a, nil
	}
	if a.modal.Kind != modalNone {
		if a.inModal(msg.X, msg.Y) {
			if h, ok := a.hitAt(msg.X, msg.Y); ok {
				return a.activateHit(h)
			}
			return a, nil
		}
		if a.modal.Kind != modalProgress {
			a.modal = modalState{}
		}
		return a, nil
	}
	if h, ok := a.hitAt(msg.X, msg.Y); ok {
		return a.activateHit(h)
	}
	return a, nil
}

func (a *App) activateHit(h hitRegion) (tea.Model, tea.Cmd) {
	switch h.action {
	case "quit":
		return a.requestQuit()
	case "help":
		a.openHelp()
	case "page":
		if p, ok := h.data.(pageID); ok {
			a.switchPage(p)
		}
	case "sidebar-account":
		a.openAccounts()
	case "sidebar-version":
		a.switchPage(pageVersions)
	case "launch-version":
		a.openVersionPicker()
	case "launch-account":
		a.openAccounts()
	case "launch-start":
		if a.rt.Launching() {
			a.openConfirm("停止游戏", "确定要结束当前游戏进程吗？未保存的游戏进度可能丢失。", "stop-game", nil)
		} else {
			a.startLaunch(a.launch.selectedVersion(a.store.Snapshot()))
		}
	case "launch-news":
		a.setToast("正在获取更新日志…", 0)
		a.rt.FetchNews()
	case "versions-row":
		idx, _ := h.data.(int)
		list := a.filteredVersions()
		if idx >= 0 && idx < len(list) {
			a.versions.idx = idx
		}
		if a.isDoubleClick(fmt.Sprintf("versions-row-%d", idx), time.Now()) {
			return a, a.versionsKey(keyMsg("enter"))
		}
	case "versions-action":
		if key, ok := h.data.(string); ok {
			return a, a.versionsKey(keyMsg(key))
		}
	case "download-tab":
		if mode, ok := h.data.(int); ok && mode >= 0 && mode < len(loaderModeNames) {
			if a.download.mode != mode {
				a.download.mode = mode
				a.download.idx = 0
				a.download.loaderLoading = false
				if a.download.manifest == nil && !a.download.loading {
					a.rt.LoadManifest(false)
				}
			}
		}
	case "download-next-mode":
		if a.download.mode < len(loaderModeNames)-1 {
			a.download.mode++
		} else {
			a.download.mode = 0
		}
		a.download.idx = 0
		a.download.loaderLoading = false
		if a.download.manifest == nil && !a.download.loading {
			a.rt.LoadManifest(false)
		}
	case "download-row":
		idx, _ := h.data.(int)
		list := a.download.versions()
		if idx >= 0 && idx < len(list) {
			a.download.idx = idx
		}
		if a.isDoubleClick(fmt.Sprintf("download-row-%d", idx), time.Now()) {
			return a, a.downloadKey(keyMsg("enter"))
		}
	case "download-action":
		if key, ok := h.data.(string); ok {
			return a, a.downloadKey(keyMsg(key))
		}
	case "settings-row":
		if idx, ok := h.data.(int); ok {
			items := a.settingItems()
			if idx >= 0 && idx < len(items) {
				a.settings.idx = idx
				return a, a.settingsKey(keyMsg("enter"))
			}
		}
	case "tool-row":
		if idx, ok := h.data.(int); ok {
			items := a.toolItems()
			if idx >= 0 && idx < len(items) {
				a.tools.idx = idx
				return a, a.runTool(items[idx].action)
			}
		}
	case "modal-item":
		if idx, ok := h.data.(int); ok && idx >= 0 && idx < len(a.modal.Items) {
			a.modal.Selected = idx
			return a.handleModalKey(keyMsg("enter"))
		}
	case "modal-confirm":
		return a.handleModalKey(keyMsg("enter"))
	case "modal-cancel":
		return a.handleModalKey(keyMsg("esc"))
	case "modal-favorite":
		a.togglePickerFavorite()
	case "modal-key":
		if key, ok := h.data.(string); ok {
			return a.handleModalKey(keyMsg(key))
		}
	case "modal-close":
		a.modal = modalState{}
	}
	return a, nil
}

func (a *App) mouseWheel(delta int) tea.Cmd {
	if a.modal.Kind != modalNone {
		key := "down"
		if delta < 0 {
			key = "up"
		}
		_, cmd := a.handleModalKey(keyMsg(key))
		return cmd
	}
	kind := ""
	if h, ok := a.hitAt(a.mouseX, a.mouseY); ok && h.action == "wheel" {
		kind, _ = h.data.(string)
	}
	switch kind {
	case "versions":
		list := a.filteredVersions()
		if len(list) > 0 {
			a.versions.idx = clamp(a.versions.idx+delta, 0, len(list)-1)
		}
	case "download":
		list := a.download.versions()
		if len(list) > 0 {
			a.download.idx = clamp(a.download.idx+delta, 0, len(list)-1)
		}
	case "settings":
		n := len(a.settingItems())
		if n > 0 {
			a.settings.idx = clamp(a.settings.idx+delta, 0, n-1)
		}
	case "tools":
		n := len(a.toolItems())
		if n > 0 {
			a.tools.idx = clamp(a.tools.idx+delta, 0, n-1)
		}
	case "about":
		a.about.offset = max(a.about.offset+delta*3, 0)
	case "logs":
		return a.logsKey(keyMsg("down"))
	default:
		switch a.page {
		case pageAbout:
			a.about.offset = max(a.about.offset+delta*3, 0)
		case pageLogs:
			return a.logsKey(keyMsg("down"))
		}
	}
	return nil
}

// registerModalHits maps rendered modal item rows to their indexes. Rows are
// matched by label prefix so every list-based dialog becomes clickable without
// duplicating each render path.
func (a *App) registerModalHits(content []string, m *modalState) {
	if len(m.Items) == 0 {
		return
	}
	for lineIdx, raw := range content {
		plain := strings.TrimSpace(ansi.Strip(raw))
		plain = strings.TrimPrefix(plain, "▸ ")
		if plain == "" {
			continue
		}
		for i, it := range m.Items {
			label := strings.TrimSpace(tr(it.Label))
			if label == "" {
				continue
			}
			probe := []rune(label)
			if len(probe) > 12 {
				probe = probe[:12]
			}
			if strings.HasPrefix(plain, string(probe)) {
				a.addHit(0, 1+lineIdx, 9999, 1+lineIdx, "modal-item", i)
				break
			}
		}
	}
}

// registerModalButton makes a text label clickable inside the dialog box.
func (a *App) registerModalButton(content []string, text, action string) {
	a.registerModalLabel(content, text, action, nil)
}

// registerModalKey wires a footer label to the same handler a keyboard action
// used to invoke.
func (a *App) registerModalKey(content []string, text, key string) {
	a.registerModalLabel(content, text, "modal-key", key)
}

func (a *App) registerModalLabel(content []string, text, action string, data any) {
	for lineIdx := len(content) - 1; lineIdx >= 0; lineIdx-- {
		plain := stripANSI(content[lineIdx])
		if !strings.Contains(plain, text) {
			continue
		}
		idx := strings.Index(plain, text)
		if idx < 0 {
			continue
		}
		x := lipgloss.Width(plain[:idx])
		w := lipgloss.Width(text)
		a.addHit(x, 1+lineIdx, x+w, 1+lineIdx, action, data)
		return
	}
}
