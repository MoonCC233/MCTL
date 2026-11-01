package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/MoonCC233/MCTL/internal/config"
	"github.com/MoonCC233/MCTL/internal/minecraft"
)

func mouseApp(t *testing.T) *App {
	t.Helper()
	a := testApp(t, config.Config{GameDir: t.TempDir(), SelectedVersion: "1.20.1"})
	a.width, a.height = 100, 30
	a.launch.versions = []minecraft.InstalledVersion{
		{ID: "1.20.1", MCVersion: "1.20.1", Loader: "Vanilla", Type: "release", Complete: true},
	}
	a.versions.list = a.launch.versions
	return a
}

func findHit(t *testing.T, a *App, action string, data any) hitRegion {
	t.Helper()
	a.View()
	for _, h := range a.hits {
		if h.action != action {
			continue
		}
		if data == nil || h.data == data {
			return h
		}
	}
	t.Fatalf("no hit for action %q data %#v", action, data)
	return hitRegion{}
}

func clickRegion(a *App, h hitRegion) {
	x := (h.x0 + h.x1) / 2
	y := (h.y0 + h.y1) / 2
	a.handleMouse(tea.MouseMsg{X: x, Y: y, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
}

func TestMouseSidebarSwitchesPage(t *testing.T) {
	a := mouseApp(t)
	clickRegion(a, findHit(t, a, "page", pageSettings))
	if a.page != pageSettings {
		t.Fatalf("page = %v", a.page)
	}
	clickRegion(a, findHit(t, a, "page", pageTools))
	if a.page != pageTools {
		t.Fatalf("page = %v", a.page)
	}
}

func TestMouseLaunchRows(t *testing.T) {
	a := mouseApp(t)
	a.page = pageLaunch
	clickRegion(a, findHit(t, a, "launch-version", nil))
	if a.modal.Kind != modalPicker || a.modal.Title != "选择游戏版本" {
		t.Fatalf("version picker = %#v", a.modal)
	}

	b := mouseApp(t)
	b.page = pageLaunch
	clickRegion(b, findHit(t, b, "launch-account", nil))
	if b.modal.Kind != modalAccounts {
		t.Fatalf("account modal = %#v", b.modal)
	}
}

func TestMouseSettingsRowOpensEditor(t *testing.T) {
	a := mouseApp(t)
	a.page = pageSettings
	clickRegion(a, findHit(t, a, "settings-row", 0))
	if a.modal.Kind != modalInput || a.modal.Action != "setting:game_dir" {
		t.Fatalf("modal = %#v", a.modal)
	}
}

func TestMouseToolRowRunsTool(t *testing.T) {
	a := mouseApp(t)
	a.page = pageTools
	// "收藏" is the fourth tool and opens the favorites picker.
	items := a.toolItems()
	idx := -1
	for i, it := range items {
		if it.action == "favorites" {
			idx = i
			break
		}
	}
	if idx < 0 {
		t.Fatal("favorites tool missing")
	}
	clickRegion(a, findHit(t, a, "tool-row", idx))
	if a.modal.Kind != modalPicker || a.modal.Action != "favorites-kind" {
		t.Fatalf("favorites modal = %#v", a.modal)
	}
}

func TestMouseModalItemAndConfirm(t *testing.T) {
	a := mouseApp(t)
	a.modal = modalState{
		Kind:   modalPicker,
		Items:  []pickerItem{{Label: "整合包", Desc: "Modpacks", Value: "modpacks"}},
		Action: "community-resource",
	}
	clickRegion(a, findHit(t, a, "modal-item", 0))
	if a.modal.Action != "community-pack-source" {
		t.Fatalf("modal = %#v", a.modal)
	}

	b := mouseApp(t)
	b.modal = modalState{Kind: modalConfirm, Title: "测试", Body: "确认操作", Action: "favorites"}
	clickRegion(b, findHit(t, b, "modal-confirm", nil))
	if b.modal.Action != "favorites-kind" {
		t.Fatalf("confirm did not run the action: %#v", b.modal)
	}

	c := mouseApp(t)
	c.modal = modalState{Kind: modalConfirm, Title: "测试", Body: "取消操作", Action: "favorites"}
	clickRegion(c, findHit(t, c, "modal-cancel", nil))
	if c.modal.Kind != modalNone {
		t.Fatalf("cancel should close the modal: %#v", c.modal)
	}
}

func TestMouseWheelScrollsVersions(t *testing.T) {
	a := mouseApp(t)
	a.page = pageVersions
	var list []minecraft.InstalledVersion
	for _, id := range []string{"1.20.1", "1.20.2", "1.20.3", "1.20.4"} {
		list = append(list, minecraft.InstalledVersion{ID: id, MCVersion: id, Loader: "Vanilla", Type: "release", Complete: true})
	}
	a.versions.list = list
	h := findHit(t, a, "wheel", "versions")
	a.handleMouse(tea.MouseMsg{
		X: (h.x0 + h.x1) / 2, Y: (h.y0 + h.y1) / 2,
		Action: tea.MouseActionPress, Button: tea.MouseButtonWheelDown,
	})
	if a.versions.idx != 1 {
		t.Fatalf("index = %d", a.versions.idx)
	}
	a.handleMouse(tea.MouseMsg{
		X: (h.x0 + h.x1) / 2, Y: (h.y0 + h.y1) / 2,
		Action: tea.MouseActionPress, Button: tea.MouseButtonWheelUp,
	})
	if a.versions.idx != 0 {
		t.Fatalf("index = %d", a.versions.idx)
	}
}

func TestMouseHelpAndOutsideClose(t *testing.T) {
	a := mouseApp(t)
	clickRegion(a, findHit(t, a, "help", nil))
	if a.modal.Kind != modalHelp {
		t.Fatalf("modal = %#v", a.modal)
	}
	a.View()
	a.handleMouse(tea.MouseMsg{X: 0, Y: 0, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
	if a.modal.Kind != modalNone {
		t.Fatalf("outside click should close the help modal: %#v", a.modal)
	}
}

func TestMouseQuitButton(t *testing.T) {
	a := mouseApp(t)
	h := findHit(t, a, "quit", nil)
	_, cmd := a.handleMouse(tea.MouseMsg{
		X: (h.x0 + h.x1) / 2, Y: (h.y0 + h.y1) / 2,
		Action: tea.MouseActionPress, Button: tea.MouseButtonLeft,
	})
	if cmd == nil {
		t.Fatal("quit button should return a command")
	}
}

func TestMouseRightClickVersionMenu(t *testing.T) {
	a := mouseApp(t)
	a.page = pageVersions
	h := findHit(t, a, "versions-row", 0)
	a.handleMouse(tea.MouseMsg{
		X: (h.x0 + h.x1) / 2, Y: (h.y0 + h.y1) / 2,
		Action: tea.MouseActionPress, Button: tea.MouseButtonRight,
	})
	if a.modal.Action != "context-action" {
		t.Fatalf("context menu = %#v", a.modal)
	}
	data, ok := a.modal.Data.(contextMenuData)
	if !ok || data.Kind != "versions" || data.Index != 0 {
		t.Fatalf("context data = %#v", a.modal.Data)
	}
	found := false
	for _, it := range a.modal.Items {
		if it.Value == "i" {
			found = true
		}
	}
	if !found {
		t.Fatalf("context items = %#v", a.modal.Items)
	}
	a.runAction("context-action", "", pickerItem{Label: "实例设置", Value: "i"}, data)
	if a.modal.Kind != modalInstanceSettings {
		t.Fatalf("context action should open instance settings: %#v", a.modal)
	}
}

func TestMouseRightClickDownloadMenu(t *testing.T) {
	a := mouseApp(t)
	a.page = pageDownload
	a.download.manifest = demoManifest(time.Now())
	a.download.idx = 0
	h := findHit(t, a, "download-row", 0)
	a.handleMouse(tea.MouseMsg{
		X: (h.x0 + h.x1) / 2, Y: (h.y0 + h.y1) / 2,
		Action: tea.MouseActionPress, Button: tea.MouseButtonRight,
	})
	if a.modal.Action != "context-action" {
		t.Fatalf("context menu = %#v", a.modal)
	}
	data, _ := a.modal.Data.(contextMenuData)
	if data.Kind != "download" {
		t.Fatalf("context data = %#v", a.modal.Data)
	}
	a.runAction("context-action", "", pickerItem{Label: "切换版本类型", Value: "mode"}, data)
	if a.modal.Action != "context-download-mode" {
		t.Fatalf("mode picker = %#v", a.modal)
	}
}

func TestMouseHoverUpdatesFooter(t *testing.T) {
	a := mouseApp(t)
	a.page = pageVersions
	h := findHit(t, a, "versions-row", 0)
	a.handleMouse(tea.MouseMsg{
		X: (h.x0 + h.x1) / 2, Y: (h.y0 + h.y1) / 2,
		Action: tea.MouseActionMotion,
	})
	out := a.View()
	if !strings.Contains(out, "右键更多") {
		t.Fatal("footer should show the hover hint")
	}
}

func TestMouseModalHoverHighlight(t *testing.T) {
	a := mouseApp(t)
	a.modal = modalState{
		Kind:   modalPicker,
		Items:  []pickerItem{{Label: "整合包", Value: "modpacks"}, {Label: "收藏", Value: "favorites"}},
		Action: "community-resource",
	}
	before := a.View()
	h := findHit(t, a, "modal-item", 1)
	a.handleMouse(tea.MouseMsg{
		X: (h.x0 + h.x1) / 2, Y: (h.y0 + h.y1) / 2,
		Action: tea.MouseActionMotion,
	})
	if a.hoverAction != "modal-item" || a.hoverData != 1 {
		t.Fatalf("hover = %q %#v", a.hoverAction, a.hoverData)
	}
	after := a.View()
	if before == after {
		t.Fatal("hovering a dialog row should change its highlight")
	}
}

func TestMouseOnScreenKeyboard(t *testing.T) {
	a := testApp(t, config.Config{GameDir: t.TempDir()})
	a.modal = modalState{Kind: modalInput, Title: "测试", Label: "名称", Action: "test-input"}
	clickRegion(a, findHit(t, a, "open-osk", nil))
	if a.modal.Kind != modalKeyboard {
		t.Fatalf("keyboard modal = %#v", a.modal)
	}
	for _, key := range []string{"a", "b", "space", "c"} {
		clickRegion(a, findHit(t, a, "osk-key", key))
	}
	clickRegion(a, findHit(t, a, "osk-done", nil))
	if a.modal.Kind != modalInput {
		t.Fatalf("should return to input: %#v", a.modal)
	}
	if got := string(a.modal.Value); got != "ab c" {
		t.Fatalf("value = %q", got)
	}

	// Backspace, shift and symbol page.
	clickRegion(a, findHit(t, a, "open-osk", nil))
	clickRegion(a, findHit(t, a, "osk-key", "back"))
	clickRegion(a, findHit(t, a, "osk-key", "shift"))
	clickRegion(a, findHit(t, a, "osk-key", "A"))
	clickRegion(a, findHit(t, a, "osk-key", "symbols"))
	clickRegion(a, findHit(t, a, "osk-key", ":"))
	clickRegion(a, findHit(t, a, "osk-done", nil))
	if got := string(a.modal.Value); got != "ab A:" {
		t.Fatalf("value = %q", got)
	}
}

func TestMouseOnScreenKeyboardCancel(t *testing.T) {
	a := testApp(t, config.Config{GameDir: t.TempDir()})
	a.modal = modalState{Kind: modalInput, Title: "测试", Label: "名称", Action: "test-input", Value: []rune("keep")}
	a.modal.Cursor = 4
	clickRegion(a, findHit(t, a, "open-osk", nil))
	clickRegion(a, findHit(t, a, "osk-key", "clear"))
	clickRegion(a, findHit(t, a, "osk-key", "x"))
	clickRegion(a, findHit(t, a, "osk-cancel", nil))
	if a.modal.Kind != modalInput || string(a.modal.Value) != "keep" {
		t.Fatalf("cancel should restore the input: %#v", a.modal)
	}
}

func TestMouseInputFieldOpensKeyboard(t *testing.T) {
	a := testApp(t, config.Config{GameDir: t.TempDir()})
	a.modal = modalState{Kind: modalInput, Title: "测试", Label: "名称", Action: "test-input"}
	hits := 0
	a.View()
	for _, h := range a.hits {
		if h.action == "open-osk" {
			hits++
		}
	}
	if hits < 2 {
		t.Fatalf("input field and button should both be clickable, hits=%d", hits)
	}
}

func TestKeyboardNavigationDisabled(t *testing.T) {
	a := mouseApp(t)
	for _, key := range []string{"3", "tab", "q", "?"} {
		a.handleKey(keyMsg(key))
	}
	if a.page != pageLaunch {
		t.Fatalf("keyboard changed page to %v", a.page)
	}
	if a.modal.Kind != modalNone {
		t.Fatalf("keyboard opened a modal: %#v", a.modal)
	}
}
