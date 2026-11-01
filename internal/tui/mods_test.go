package tui

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MoonCC233/MCTL/internal/config"
)

func writeModJar(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	w, err := zw.Create("fabric.mod.json")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte(`{"id":"sodium","name":"Sodium","version":"0.6.9","description":"Fast renderer"}`)); err != nil {
		t.Fatal(err)
	}
	w, err = zw.Create("META-INF/jars/inner.jar")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte("nested")); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestModsInfoModalAndDetail(t *testing.T) {
	dir := t.TempDir()
	writeModJar(t, filepath.Join(dir, "mods", "sodium.jar"))
	writeModJar(t, filepath.Join(dir, "mods", "beta.jar.disabled"))

	a := testApp(t, config.Config{GameDir: dir})
	a.openModsInfo()
	if a.modal.Kind != modalMods {
		t.Fatalf("modal = %#v", a.modal)
	}
	if len(a.modal.Items) != 2 {
		t.Fatalf("items = %#v", a.modal.Items)
	}
	out := a.View()
	for _, want := range []string{"模组信息", "sodium.jar", "Fabric", "已禁用"} {
		if !strings.Contains(out, want) {
			t.Fatalf("modal output missing %q", want)
		}
	}

	data := a.modal.Data.(modsInspectorData)
	a.openModDetail(data.Mods[0])
	if a.modal.Kind != modalModDetail {
		t.Fatalf("detail modal = %#v", a.modal)
	}
	detail := a.View()
	for _, want := range []string{"模组 · sodium.jar", "Sodium", "0.6.9", "Fast renderer", "内嵌 jar · 1"} {
		if !strings.Contains(detail, want) {
			t.Fatalf("detail missing %q", want)
		}
	}
}
