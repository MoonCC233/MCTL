package tui

import (
	"errors"
	"fmt"
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

func TestPanelScrollbar(t *testing.T) {
	lines := []string{"a", "b", "c", "d", "e"}
	long := panelScroll("列表", 40, 7, false, lines, 100, 50)
	if !strings.Contains(long, "█") {
		t.Fatal("long list should render a scrollbar thumb")
	}
	if got := strings.Count(long, "\n") + 1; got != 7 {
		t.Fatalf("panel height = %d", got)
	}
	short := panelScroll("列表", 40, 7, false, lines, len(lines), 0)
	if strings.Contains(short, "█") {
		t.Fatal("short list should not render a scrollbar")
	}
}

func TestVersionsPageShowsScrollbar(t *testing.T) {
	a := mouseApp(t)
	a.page = pageVersions
	var list []minecraft.InstalledVersion
	for i := 0; i < 40; i++ {
		list = append(list, minecraft.InstalledVersion{ID: fmt.Sprintf("1.20.%d", i), MCVersion: "1.20", Loader: "Vanilla", Type: "release", Complete: true})
	}
	a.versions.list = list
	a.versions.idx = 10
	out := a.View()
	if !strings.Contains(out, "█") {
		t.Fatal("long version list should show a scrollbar")
	}
}

func TestMouseOSKCursorAndInsert(t *testing.T) {
	a := testApp(t, config.Config{GameDir: t.TempDir()})
	a.modal = modalState{Kind: modalInput, Title: "测试", Label: "名称", Action: "test-input", Value: []rune("ab c"), Cursor: 4}
	clickRegion(a, findHit(t, a, "open-osk", nil))
	for _, key := range []string{"left", "left", "x", "home", "symbols", ">", "end", "!"} {
		clickRegion(a, findHit(t, a, "osk-key", key))
	}
	clickRegion(a, findHit(t, a, "osk-done", nil))
	if got := string(a.modal.Value); got != ">abx c!" {
		t.Fatalf("value = %q", got)
	}
	if a.modal.Cursor != len([]rune(">abx c!")) {
		t.Fatalf("cursor = %d", a.modal.Cursor)
	}
}

func TestMouseModalRightClickMenu(t *testing.T) {
	dir := t.TempDir()
	a := testApp(t, config.Config{GameDir: dir})
	a.modal = modalState{
		Kind:  modalScreenshots,
		Title: "截图管理",
		Items: []pickerItem{{Label: "shot.png", Value: "shot.png"}},
		Data:  dir,
	}
	h := findHit(t, a, "modal-item", 0)
	a.handleMouse(tea.MouseMsg{
		X: (h.x0 + h.x1) / 2, Y: (h.y0 + h.y1) / 2,
		Action: tea.MouseActionPress, Button: tea.MouseButtonRight,
	})
	if a.modal.Action != "modal-context" {
		t.Fatalf("context menu = %#v", a.modal)
	}
	if a.contextReturn == nil {
		t.Fatal("context menu should remember the original dialog")
	}
	// First item is Delete.
	clickRegion(a, findHit(t, a, "modal-item", 0))
	if a.modal.Kind != modalConfirm || a.modal.Action != "delete-screenshot" {
		t.Fatalf("delete confirm = %#v", a.modal)
	}
	if a.contextReturn != nil {
		t.Fatal("context return should be cleared after an action")
	}
}

func TestMouseModalContextCancelRestoresList(t *testing.T) {
	a := testApp(t, config.Config{GameDir: t.TempDir()})
	a.modal = modalState{
		Kind:  modalScreenshots,
		Title: "截图管理",
		Items: []pickerItem{{Label: "shot.png", Value: "shot.png"}},
		Data:  t.TempDir(),
	}
	h := findHit(t, a, "modal-item", 0)
	a.handleMouse(tea.MouseMsg{
		X: (h.x0 + h.x1) / 2, Y: (h.y0 + h.y1) / 2,
		Action: tea.MouseActionPress, Button: tea.MouseButtonRight,
	})
	clickRegion(a, findHit(t, a, "modal-cancel", nil))
	if a.modal.Kind != modalScreenshots {
		t.Fatalf("cancel should restore the list: %#v", a.modal)
	}
	if a.contextReturn != nil {
		t.Fatal("context return should be cleared")
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

func TestJavaListMessageOpensPicker(t *testing.T) {
	a := mouseApp(t)
	list := []minecraft.JavaInfo{
		{Path: `D:\Program Files\Java\jdk-25.0.3.9\bin\java.exe`, Version: "25.0.3", Major: 25, Vendor: "Eclipse Adoptium"},
		{Path: `D:\Program Files\Java\jdk-21\bin\java.exe`, Version: "21.0.11", Major: 21, Vendor: "Eclipse Adoptium"},
	}
	a.handleEvent(javaListMsg{List: list})
	if a.modal.Kind != modalPicker || a.modal.Action != "select-java" {
		t.Fatalf("java picker = %#v", a.modal)
	}
	if len(a.modal.Items) != 3 {
		t.Fatalf("items = %#v", a.modal.Items)
	}
	if a.modal.Items[1].Value != list[0].Path {
		t.Fatalf("first runtime = %#v", a.modal.Items[1])
	}
	clickRegion(a, findHit(t, a, "modal-item", 2))
	if got := a.store.Snapshot().JavaPath; got != list[1].Path {
		t.Fatalf("config java path = %q", got)
	}
}

func TestJavaPickerAutomaticClearsOverride(t *testing.T) {
	a := mouseApp(t)
	a.saveCfg(func(c *config.Config) { c.JavaPath = `D:\Java\jdk-21\bin\java.exe` })
	a.openJavaPicker([]minecraft.JavaInfo{{Path: `D:\Java\jdk-21\bin\java.exe`, Major: 21}})
	if !strings.HasPrefix(a.modal.Items[1].Label, "★ ") {
		t.Fatalf("current runtime should be marked: %#v", a.modal.Items[1])
	}
	clickRegion(a, findHit(t, a, "modal-item", 0))
	if got := a.store.Snapshot().JavaPath; got != "" {
		t.Fatalf("config java path should be cleared, got %q", got)
	}
}

func TestJavaPickerEmptyListKeepsState(t *testing.T) {
	a := mouseApp(t)
	a.openJavaPicker(nil)
	if a.modal.Kind != modalNone {
		t.Fatalf("modal = %#v", a.modal)
	}
}

func TestCurseForgeProjectFilesPicker(t *testing.T) {
	a := mouseApp(t)
	mod := minecraft.CurseForgeMod{ID: 285109, Name: "RLCraft"}
	files := []minecraft.CurseForgeFile{
		{ID: 11, DisplayName: "RLCraft 2.9.3", FileName: "rlcraft.zip", ReleaseType: 1, FileLength: 2048, DownloadURL: "https://example.invalid/a.zip"},
		{ID: 12, DisplayName: "RLCraft 2.9.2", FileName: "old.zip", ReleaseType: 2},
	}
	a.handleEvent(curseForgeProjectMsg{Mod: mod, Files: files})
	if a.modal.Kind != modalPicker || a.modal.Action != "install-curseforge-file" {
		t.Fatalf("file picker = %#v", a.modal)
	}
	if len(a.modal.Items) != 2 {
		t.Fatalf("items = %#v", a.modal.Items)
	}
	data, ok := a.modal.Data.(curseForgeFileData)
	if !ok || data.Mod.ID != 285109 || len(data.Files) != 2 {
		t.Fatalf("data = %#v", a.modal.Data)
	}
	if !strings.Contains(a.modal.Items[0].Desc, "正式版") {
		t.Fatalf("first item = %#v", a.modal.Items[0])
	}
}

func TestCurseForgeProjectErrorShowsDialog(t *testing.T) {
	a := mouseApp(t)
	a.handleEvent(curseForgeProjectMsg{Err: errors.New("403 forbidden")})
	if a.modal.Kind != modalError {
		t.Fatalf("modal = %#v", a.modal)
	}
}

func TestCurseForgeByIDToolOpensInput(t *testing.T) {
	a := testApp(t, config.Config{GameDir: t.TempDir(), CurseForgeAPIKey: "key"})
	a.width, a.height = 100, 30
	a.page = pageTools
	idx := -1
	for i, it := range a.toolItems() {
		if it.action == "curseforge-by-id" {
			idx = i
			break
		}
	}
	if idx < 0 {
		t.Fatal("curseforge-by-id tool missing")
	}
	clickRegion(a, findHit(t, a, "tool-row", idx))
	if a.modal.Kind != modalInput || a.modal.Action != "load-curseforge-project" {
		t.Fatalf("modal = %#v", a.modal)
	}
}
