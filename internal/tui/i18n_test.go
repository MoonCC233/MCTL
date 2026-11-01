package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"

	"github.com/MoonCC233/MCTL/internal/config"
)

func TestLanguageSwitching(t *testing.T) {
	t.Cleanup(func() { SetLanguage("zh") })
	SetLanguage("en")
	if Language() != "en" {
		t.Fatalf("language = %q", Language())
	}
	pages := map[string][]string{
		"launch":   {"Launch game", "Account", "Minecraft terminal launcher"},
		"versions": {"Versions", "Details"},
		"download": {"Download", "Vanilla", "Fabric"},
		"settings": {"Settings", "Language", "Theme", "HTTP proxy", "Game directory"},
		"tools":    {"Tools", "New instance", "Speed test", "Self-test", "Skin / cape"},
		"about":    {"About", "SUPPORTED", "PROJECT"},
		"logs":     {"Runtime logs"},
	}
	for page, markers := range pages {
		out := RenderDemo(page, 160, 50)
		if lines := strings.Count(out, "\n") + 1; lines != 50 {
			t.Fatalf("%s: got %d lines", page, lines)
		}
		for _, want := range markers {
			if !strings.Contains(out, want) {
				t.Fatalf("%s page is missing English marker %q", page, want)
			}
		}
	}
	// English chrome must still fit a narrow terminal.
	for _, page := range []string{"launch", "versions", "download", "settings", "tools", "about", "logs"} {
		out := RenderDemo(page, 90, 24)
		for i, line := range strings.Split(out, "\n") {
			if got := lipgloss.Width(line); got > 90 {
				t.Fatalf("%s at 90x24: line %d width %d", page, i+1, got)
			}
		}
	}
}

func TestLanguageSettingCycle(t *testing.T) {
	t.Cleanup(func() { SetLanguage("zh") })
	SetLanguage("zh")
	a := testApp(t, config.Config{GameDir: t.TempDir(), Language: "zh"})

	a.adjustSetting(settingItem{key: "language"}, a.store.Snapshot(), 1)
	if Language() != "en" || a.store.Snapshot().Language != "en" {
		t.Fatalf("language = %q, config = %q", Language(), a.store.Snapshot().Language)
	}
	a.adjustSetting(settingItem{key: "language"}, a.store.Snapshot(), 1)
	if Language() != "zh" || a.store.Snapshot().Language != "zh" {
		t.Fatalf("language = %q, config = %q", Language(), a.store.Snapshot().Language)
	}
}

func TestLanguageNames(t *testing.T) {
	if NormalizeLanguage("en") != "en" || NormalizeLanguage("zh") != "zh" || NormalizeLanguage("fr") != "zh" {
		t.Fatal("language normalization is wrong")
	}
	if LanguageName("en") != "English" || LanguageName("zh") != "中文" {
		t.Fatal("language display names are wrong")
	}
}
