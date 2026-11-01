package minecraft

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInstanceSettingsRoundTrip(t *testing.T) {
	dir := t.TempDir()
	writeJSON(t, VersionJSONPath(dir, "1.20.1"), map[string]any{"id": "1.20.1", "mainClass": "net.minecraft.client.main.Main"})

	settings, err := LoadInstanceSettings(dir, "1.20.1")
	if err != nil {
		t.Fatal(err)
	}
	if settings.MaxMemory != 0 || settings.Name != "" {
		t.Fatalf("missing file should be empty: %#v", settings)
	}

	in := &InstanceSettings{
		Name:      "  生存实例  ",
		Link:      "https://example.invalid/pack",
		MinMemory: 1024,
		MaxMemory: 8192,
		JVMArgs:   "-XX:+UseG1GC",
		GameArgs:  "--demo",
		Server:    "play.example.com:25566",
		JavaPath:  `C:\Java\bin\java.exe`,
	}
	if err := SaveInstanceSettings(dir, "1.20.1", in); err != nil {
		t.Fatal(err)
	}
	out, err := LoadInstanceSettings(dir, "1.20.1")
	if err != nil {
		t.Fatal(err)
	}
	if out.Name != "生存实例" || out.MaxMemory != 8192 || out.Server != "play.example.com:25566" {
		t.Fatalf("settings = %#v", out)
	}
	if _, err := os.Stat(InstanceSettingsPath(dir, "1.20.1")); err != nil {
		t.Fatalf("settings file missing: %v", err)
	}

	bad := &InstanceSettings{MinMemory: 4096, MaxMemory: 1024}
	if err := SaveInstanceSettings(dir, "1.20.1", bad); err == nil {
		t.Fatal("expected min > max error")
	}
	neg := &InstanceSettings{MaxMemory: -1}
	if err := SaveInstanceSettings(dir, "1.20.1", neg); err == nil {
		t.Fatal("expected negative memory error")
	}
	if err := SaveInstanceSettings(dir, "missing", &InstanceSettings{}); err == nil {
		t.Fatal("expected missing version directory error")
	}
}

func TestSplitServerAddress(t *testing.T) {
	cases := map[string][2]string{
		"play.example.com":       {"play.example.com", "25565"},
		"play.example.com:25566": {"play.example.com", "25566"},
		"127.0.0.1:25565":        {"127.0.0.1", "25565"},
		"[::1]:25565":            {"::1", "25565"},
		"host:notaport":          {"host:notaport", "25565"},
		"  spaced.example.com  ": {"spaced.example.com", "25565"},
	}
	for in, want := range cases {
		host, port := SplitServerAddress(in)
		if host != want[0] || port != want[1] {
			t.Fatalf("SplitServerAddress(%q) = %q, %q; want %q, %q", in, host, port, want[0], want[1])
		}
	}
	if host, port := SplitServerAddress(""); host != "" || port != "" {
		t.Fatalf("empty address = %q, %q", host, port)
	}
}

func TestInstanceSettingsPath(t *testing.T) {
	root := t.TempDir()
	instance := filepath.Join(root, "instances", "pack")
	if InstanceSettingsPath(instance, "pack-x") == InstanceSettingsPath(root, "pack-x") {
		t.Fatal("instances must keep separate settings")
	}
}

func TestInstanceSettingsIcon(t *testing.T) {
	dir := t.TempDir()
	writeJSON(t, VersionJSONPath(dir, "1.20.1"), map[string]any{"id": "1.20.1", "mainClass": "net.minecraft.client.main.Main"})
	if err := SaveInstanceSettings(dir, "1.20.1", &InstanceSettings{Name: "Pack", Icon: "🧱"}); err != nil {
		t.Fatal(err)
	}
	list, err := ScanVersions(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].Icon != "🧱" {
		t.Fatalf("scan = %#v", list)
	}
	if err := SaveInstanceSettings(dir, "1.20.1", &InstanceSettings{Icon: "🧱🧱🧱🧱🧱"}); err == nil {
		t.Fatal("expected icon length error")
	}
}

func TestInstanceSettingsAppearInScan(t *testing.T) {
	dir := t.TempDir()
	writeJSON(t, VersionJSONPath(dir, "1.20.1"), map[string]any{
		"id":        "1.20.1",
		"mainClass": "net.minecraft.client.main.Main",
	})
	if err := SaveInstanceSettings(dir, "1.20.1", &InstanceSettings{Name: "生存实例", Link: "https://example.invalid"}); err != nil {
		t.Fatal(err)
	}
	list, err := ScanVersions(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].Name != "生存实例" || list[0].Link != "https://example.invalid" {
		t.Fatalf("scan = %#v", list)
	}
}
