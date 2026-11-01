package minecraft

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestParseMaven(t *testing.T) {
	c, err := ParseMaven("net.example:demo:1.2.3:natives-windows@zip")
	if err != nil {
		t.Fatal(err)
	}
	if c.Group != "net.example" || c.Artifact != "demo" || c.Version != "1.2.3" {
		t.Fatalf("unexpected coordinate: %+v", c)
	}
	if c.Classifier != "natives-windows" || c.Ext != "zip" {
		t.Fatalf("unexpected classifier/ext: %+v", c)
	}
	if got, want := c.Path(), "net/example/demo/1.2.3/demo-1.2.3-natives-windows.zip"; got != want {
		t.Fatalf("path = %q, want %q", got, want)
	}
	if _, err := ParseMaven("broken"); err == nil {
		t.Fatal("expected invalid coordinate error")
	}
}

func TestRulesAndArguments(t *testing.T) {
	rules := []Rule{
		{Action: "allow"},
		{Action: "deny", OS: &OSRule{Name: "definitely-not-this-os"}},
	}
	if !RulesAllowed(rules, nil) {
		t.Fatal("platform-independent allow should be allowed")
	}
	deny := []Rule{{Action: "deny"}}
	if RulesAllowed(deny, nil) {
		t.Fatal("deny rule should win")
	}
	features := map[string]bool{"has_custom_resolution": true}
	arg := Argument{
		Values: []string{"--width", "1280"},
		Rules:  []Rule{{Action: "allow", Features: map[string]bool{"has_custom_resolution": true}}},
	}
	rv := &ResolvedVersion{GameArgs: []Argument{arg}}
	if got := rv.GameArguments(features); len(got) != 2 || got[0] != "--width" {
		t.Fatalf("unexpected arguments: %#v", got)
	}
	if got := rv.GameArguments(map[string]bool{"has_custom_resolution": false}); len(got) != 0 {
		t.Fatalf("arguments should be filtered: %#v", got)
	}
}

func TestResolveVersionInheritance(t *testing.T) {
	dir := t.TempDir()
	writeJSON(t, VersionJSONPath(dir, "1.20.1"), map[string]any{
		"id":        "1.20.1",
		"type":      "release",
		"mainClass": "net.minecraft.client.main.Main",
		"assets":    "5",
		"assetIndex": map[string]any{
			"id": "5", "url": "https://example.invalid/5.json",
		},
		"downloads": map[string]any{
			"client": map[string]any{"url": "https://example.invalid/client.jar", "size": 1},
		},
		"javaVersion": map[string]any{"majorVersion": 17},
		"libraries": []any{
			map[string]any{"name": "com.example:base:1.0"},
		},
		"arguments": map[string]any{
			"game": []any{"--username", "${auth_player_name}"},
			"jvm":  []any{"-Dbase=true"},
		},
	})
	writeJSON(t, VersionJSONPath(dir, "fabric-loader-0.16.9-1.20.1"), map[string]any{
		"id":           "fabric-loader-0.16.9-1.20.1",
		"inheritsFrom": "1.20.1",
		"mainClass":    "net.fabricmc.loader.impl.launch.knot.KnotClient",
		"libraries": []any{
			map[string]any{"name": "com.example:base:1.0"},
			map[string]any{"name": "net.fabricmc:fabric-loader:0.16.9"},
		},
		"arguments": map[string]any{
			"game": []any{"--fabric"},
		},
	})

	rv, err := ResolveVersion(dir, "fabric-loader-0.16.9-1.20.1")
	if err != nil {
		t.Fatal(err)
	}
	if rv.MainClass != "net.fabricmc.loader.impl.launch.knot.KnotClient" {
		t.Fatalf("main class = %q", rv.MainClass)
	}
	if rv.JarID != "1.20.1" {
		t.Fatalf("jar id = %q, want parent client jar", rv.JarID)
	}
	if len(rv.Chain) != 2 || rv.Chain[0] != "fabric-loader-0.16.9-1.20.1" {
		t.Fatalf("chain = %#v", rv.Chain)
	}
	if len(rv.Libraries) != 2 {
		t.Fatalf("libraries should be deduped, got %#v", rv.Libraries)
	}
	game := rv.GameArguments(Features(0, 0))
	if len(game) != 3 || game[2] != "--fabric" {
		t.Fatalf("merged game args = %#v", game)
	}
	if rv.JavaVersion == nil || rv.JavaVersion.MajorVersion != 17 {
		t.Fatalf("java requirement = %#v", rv.JavaVersion)
	}
}

func TestSplitArgs(t *testing.T) {
	got := SplitArgs(`-Xmx2G --name "Alex Smith" '--demo mode' --path "C:\\Games\\MC"`)
	want := []string{"-Xmx2G", "--name", "Alex Smith", "--demo mode", "--path", `C:\Games\MC`}
	if len(got) != len(want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("arg %d = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestOfflineUUID(t *testing.T) {
	got := OfflineUUID("Alex")
	if len(got) != 36 || got[8] != '-' || got[13] != '-' || got[18] != '-' || got[23] != '-' {
		t.Fatalf("malformed uuid: %q", got)
	}
	if got != OfflineUUID("Alex") {
		t.Fatal("offline uuid must be deterministic")
	}
	if got == OfflineUUID("Steve") {
		t.Fatal("different players must get different uuids")
	}
}

func writeJSON(t *testing.T, p string, v any) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, data, 0o644); err != nil {
		t.Fatal(err)
	}
}
