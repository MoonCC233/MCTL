package tui

import (
	"strings"
	"testing"

	"github.com/MoonCC233/MCTL/internal/config"
	"github.com/MoonCC233/MCTL/internal/minecraft"
)

func TestApplyTheme(t *testing.T) {
	t.Cleanup(func() { ApplyTheme("opencode") })
	for _, id := range ThemeIDs() {
		ApplyTheme(id)
		if CurrentTheme() != id {
			t.Fatalf("current theme = %q, want %q", CurrentTheme(), id)
		}
		if ThemeName(id) == "" {
			t.Fatalf("theme %q has no name", id)
		}
	}
	ApplyTheme("light")
	if string(colorAccent) != "#b8541f" {
		t.Fatalf("light accent = %q", colorAccent)
	}
	ApplyTheme("does-not-exist")
	if CurrentTheme() != "opencode" {
		t.Fatalf("unknown theme should fall back, got %q", CurrentTheme())
	}
	if len(ThemeIDs()) != 3 {
		t.Fatalf("themes = %#v", ThemeIDs())
	}
}

func TestThemeSettingCycles(t *testing.T) {
	a := testApp(t, config.Config{GameDir: t.TempDir(), Theme: "opencode"})
	t.Cleanup(func() { ApplyTheme("opencode") })
	a.adjustSetting(settingItem{key: "theme"}, a.store.Snapshot(), 1)
	if got := a.store.Snapshot().Theme; got != "midnight" {
		t.Fatalf("theme = %q", got)
	}
	if CurrentTheme() != "midnight" {
		t.Fatalf("active theme = %q", CurrentTheme())
	}
	a.adjustSetting(settingItem{key: "theme"}, a.store.Snapshot(), -1)
	if got := a.store.Snapshot().Theme; got != "opencode" {
		t.Fatalf("theme = %q", got)
	}
}

func TestProxySetting(t *testing.T) {
	defer func() { _ = minecraft.SetProxy("") }()
	a := testApp(t, config.Config{GameDir: t.TempDir()})

	a.submitSetting("proxy", "127.0.0.1:7890")
	if a.modal.Kind != modalError {
		t.Fatalf("invalid proxy should show an error: %#v", a.modal)
	}
	if a.store.Snapshot().Proxy != "" {
		t.Fatalf("invalid proxy should not be saved: %q", a.store.Snapshot().Proxy)
	}

	a.submitSetting("proxy", "http://127.0.0.1:7890")
	if got := a.store.Snapshot().Proxy; got != "http://127.0.0.1:7890" {
		t.Fatalf("proxy = %q", got)
	}
	if !strings.Contains(a.toast.Text, "代理已保存") {
		t.Fatalf("toast = %q", a.toast.Text)
	}

	a.submitSetting("proxy", "")
	if a.store.Snapshot().Proxy != "" || !strings.Contains(a.toast.Text, "清除代理") {
		t.Fatalf("proxy = %q, toast = %q", a.store.Snapshot().Proxy, a.toast.Text)
	}
}

func TestSettingsIncludeThemeAndProxy(t *testing.T) {
	a := testApp(t, config.Config{GameDir: t.TempDir()})
	keys := map[string]bool{}
	for _, it := range a.settingItems() {
		keys[it.key] = true
	}
	for _, want := range []string{"theme", "proxy"} {
		if !keys[want] {
			t.Fatalf("settings are missing %q", want)
		}
	}
}
