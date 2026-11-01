package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MoonCC233/MCTL/internal/config"
	"github.com/MoonCC233/MCTL/internal/minecraft"
)

func newSettingsApp(t *testing.T) (*App, string) {
	t.Helper()
	dir := t.TempDir()
	versionDir := filepath.Join(dir, "versions", "1.20.1")
	if err := os.MkdirAll(versionDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(versionDir, "1.20.1.json"), []byte(`{"id":"1.20.1","mainClass":"net.minecraft.client.main.Main"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	a := testApp(t, config.Config{GameDir: dir, SelectedVersion: "1.20.1", MinMemory: 512, MaxMemory: 6144})
	return a, dir
}

func TestInstanceSettingsModal(t *testing.T) {
	a, _ := newSettingsApp(t)
	a.openInstanceSettings()
	if a.modal.Kind != modalInstanceSettings {
		t.Fatalf("modal = %#v", a.modal)
	}
	if len(a.modal.Items) != 9 {
		t.Fatalf("items = %#v", a.modal.Items)
	}
	out := a.View()
	for _, want := range []string{"实例设置", "实例名称", "实例图标", "最大内存", "自动进入服务器"} {
		if !strings.Contains(out, want) {
			t.Fatalf("modal output missing %q", want)
		}
	}
}

func TestSaveInstanceSetting(t *testing.T) {
	a, dir := newSettingsApp(t)
	a.runAction("save-instance-setting", "4096", pickerItem{}, instanceSettingEdit{GameDir: dir, VersionID: "1.20.1", Key: "max"})
	settings, err := minecraft.LoadInstanceSettings(dir, "1.20.1")
	if err != nil {
		t.Fatal(err)
	}
	if settings.MaxMemory != 4096 {
		t.Fatalf("max memory = %d", settings.MaxMemory)
	}
	if a.modal.Kind != modalInstanceSettings {
		t.Fatalf("should return to settings: %#v", a.modal)
	}

	edit := func(key, value string) {
		a.runAction("save-instance-setting", value, pickerItem{}, instanceSettingEdit{GameDir: dir, VersionID: "1.20.1", Key: key})
	}
	edit("name", "生存实例")
	edit("icon", "🧱")
	edit("server", "play.example.com:25566")
	edit("jvm", "-XX:+UseG1GC")
	settings, err = minecraft.LoadInstanceSettings(dir, "1.20.1")
	if err != nil {
		t.Fatal(err)
	}
	if settings.Name != "生存实例" || settings.Icon != "🧱" || settings.Server != "play.example.com:25566" || settings.JVMArgs != "-XX:+UseG1GC" {
		t.Fatalf("settings = %#v", settings)
	}
}

func TestSaveInstanceSettingRejectsInvalidValues(t *testing.T) {
	a, dir := newSettingsApp(t)
	edit := func(key, value string) {
		a.runAction("save-instance-setting", value, pickerItem{}, instanceSettingEdit{GameDir: dir, VersionID: "1.20.1", Key: key})
	}
	for _, tc := range []struct{ key, value string }{
		{"link", "ftp://example.com"},
		{"max", "abc"},
		{"max", "128"},
		{"server", "bad address"},
		{"min", "8192"},
		{"icon", "🧱🧱🧱🧱🧱"},
	} {
		edit(tc.key, tc.value)
		if a.modal.Kind != modalError {
			t.Fatalf("%s=%q should be rejected, modal = %#v", tc.key, tc.value, a.modal)
		}
	}
	settings, err := minecraft.LoadInstanceSettings(dir, "1.20.1")
	if err != nil {
		t.Fatal(err)
	}
	if settings.Name != "" || settings.MaxMemory != 0 {
		t.Fatalf("invalid values should not be saved: %#v", settings)
	}
}

func TestJoinArgs(t *testing.T) {
	cases := []struct{ base, extra, want string }{
		{"", "", ""},
		{"-XX:+UseG1GC", "", "-XX:+UseG1GC"},
		{"", "--demo", "--demo"},
		{" -Xms1G ", " --demo ", "-Xms1G --demo"},
	}
	for _, tc := range cases {
		if got := joinArgs(tc.base, tc.extra); got != tc.want {
			t.Fatalf("joinArgs(%q, %q) = %q, want %q", tc.base, tc.extra, got, tc.want)
		}
	}
}
