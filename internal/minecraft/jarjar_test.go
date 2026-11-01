package minecraft

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

func writeJar(t *testing.T, path string, entries map[string]string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	for name, content := range entries {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestInspectFabricMod(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sodium.jar")
	writeJar(t, path, map[string]string{
		"fabric.mod.json":         `{"id":"sodium","name":"Sodium","version":"0.6.9","description":"Fast renderer"}`,
		"META-INF/jars/inner.jar": "nested",
	})
	info := InspectModJar(path)
	if info.Err != "" {
		t.Fatal(info.Err)
	}
	if info.Loader != "Fabric" || info.ID != "sodium" || info.Name != "Sodium" || info.Version != "0.6.9" {
		t.Fatalf("info = %#v", info)
	}
	if info.Description != "Fast renderer" || len(info.Nested) != 1 || info.Nested[0] != "META-INF/jars/inner.jar" {
		t.Fatalf("info = %#v", info)
	}
	if info.Disabled {
		t.Fatal("jar should be enabled")
	}
}

func TestInspectQuiltMod(t *testing.T) {
	path := filepath.Join(t.TempDir(), "quilted.jar")
	writeJar(t, path, map[string]string{
		"quilt.mod.json": `{"quilt_loader":{"id":"quilted","version":"1.2.3","metadata":{"name":"Quilted","description":"Quilt mod"}}}`,
	})
	info := InspectModJar(path)
	if info.Err != "" {
		t.Fatal(info.Err)
	}
	if info.Loader != "Quilt" || info.ID != "quilted" || info.Name != "Quilted" || info.Version != "1.2.3" {
		t.Fatalf("info = %#v", info)
	}
}

func TestInspectForgeMod(t *testing.T) {
	path := filepath.Join(t.TempDir(), "forge.jar")
	writeJar(t, path, map[string]string{
		"META-INF/mods.toml": "modLoader=\"javafml\"\n[[mods]]\nmodId=\"example\"\ndisplayName=\"Example Mod\"\nversion=\"2.0.0\"\ndescription=\"Forge mod\"\n",
	})
	info := InspectModJar(path)
	if info.Err != "" {
		t.Fatal(info.Err)
	}
	if info.Loader != "Forge" || info.ID != "example" || info.Name != "Example Mod" || info.Version != "2.0.0" {
		t.Fatalf("info = %#v", info)
	}
}

func TestInspectNeoForgeMod(t *testing.T) {
	path := filepath.Join(t.TempDir(), "neo.jar")
	writeJar(t, path, map[string]string{
		"META-INF/neoforge.mods.toml": "[[mods]]\nmodId=\"neo\"\ndisplayName=\"Neo Mod\"\nversion=\"3.0.0\"\n",
	})
	info := InspectModJar(path)
	if info.Err != "" {
		t.Fatal(info.Err)
	}
	if info.Loader != "NeoForge" || info.ID != "neo" || info.Name != "Neo Mod" {
		t.Fatalf("info = %#v", info)
	}
}

func TestInspectLegacyModInfo(t *testing.T) {
	path := filepath.Join(t.TempDir(), "legacy.jar")
	writeJar(t, path, map[string]string{
		"mcmod.info": `[{"modid":"legacy","name":"Legacy Mod","version":"1.7.10"}]`,
	})
	info := InspectModJar(path)
	if info.Err != "" {
		t.Fatal(info.Err)
	}
	if info.Loader != "Forge" || info.ID != "legacy" || info.Name != "Legacy Mod" {
		t.Fatalf("info = %#v", info)
	}
}

func TestInspectModJarErrors(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "broken.jar")
	if err := os.WriteFile(path, []byte("not a zip"), 0o644); err != nil {
		t.Fatal(err)
	}
	info := InspectModJar(path)
	if info.Err == "" || info.Loader != "" {
		t.Fatalf("info = %#v", info)
	}
}

func TestInspectModsSortsAndFlagsDisabled(t *testing.T) {
	dir := t.TempDir()
	writeJar(t, filepath.Join(dir, "mods", "beta.jar"), map[string]string{
		"fabric.mod.json": `{"id":"beta","name":"Beta","version":"1"}`,
	})
	writeJar(t, filepath.Join(dir, "mods", "alpha.jar.disabled"), map[string]string{
		"fabric.mod.json": `{"id":"alpha","name":"Alpha","version":"1"}`,
	})
	if err := os.WriteFile(filepath.Join(dir, "mods", "notes.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	list, err := InspectMods(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Fatalf("list = %#v", list)
	}
	if list[0].File != "beta.jar" || list[0].Disabled {
		t.Fatalf("enabled jar should come first: %#v", list)
	}
	if list[1].File != "alpha.jar.disabled" || !list[1].Disabled {
		t.Fatalf("disabled jar = %#v", list[1])
	}
}

func TestInspectModsMissingDirectory(t *testing.T) {
	list, err := InspectMods(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 0 {
		t.Fatalf("list = %#v", list)
	}
}
