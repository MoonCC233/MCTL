package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/MoonCC233/MCTL/internal/config"
	"github.com/MoonCC233/MCTL/internal/logbuf"
)

func testApp(t *testing.T, cfg config.Config) *App {
	t.Helper()
	store := &config.Store{Cfg: cfg, Path: filepath.Join(t.TempDir(), "config.json")}
	rt := NewRuntime(store, logbuf.New(50))
	a := New(store, rt)
	a.width, a.height = 100, 30
	return a
}

func TestExportInstancePrompt(t *testing.T) {
	dir := t.TempDir()
	a := testApp(t, config.Config{GameDir: dir, SelectedVersion: "1.20.1"})

	a.openExportInstance()
	if a.modal.Kind != modalInput || a.modal.Action != "export-instance" {
		t.Fatalf("modal = %#v", a.modal)
	}
	if a.modal.Data != "1.20.1" {
		t.Fatalf("modal data = %#v", a.modal.Data)
	}
	value := string(a.modal.Value)
	if !strings.Contains(filepath.ToSlash(value), "/exports/") || !strings.HasSuffix(value, ".zip") {
		t.Fatalf("default export path = %q", value)
	}

	b := testApp(t, config.Config{GameDir: dir})
	b.openExportInstance()
	if b.modal.Kind != modalNone {
		t.Fatalf("missing selection should not open a prompt: %#v", b.modal)
	}
	if !strings.Contains(b.toast.Text, "选择一个版本") {
		t.Fatalf("toast = %q", b.toast.Text)
	}
}

func TestInstanceActionGuards(t *testing.T) {
	a := testApp(t, config.Config{GameDir: t.TempDir()})

	if cmd := a.runAction("import-instance", "   ", pickerItem{}, nil); cmd != nil {
		t.Fatal("empty import path should not start a task")
	}
	if a.modal.Kind != modalNone {
		t.Fatalf("empty import path opened a modal: %#v", a.modal)
	}
	if cmd := a.runAction("export-instance", "", pickerItem{}, "1.20.1"); cmd != nil {
		t.Fatal("empty export path should not start a task")
	}
}

func TestRuntimeExportInstance(t *testing.T) {
	dir := t.TempDir()
	versionDir := filepath.Join(dir, "versions", "1.20.1")
	if err := os.MkdirAll(versionDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(versionDir, "1.20.1.json"), []byte(`{"id":"1.20.1","mainClass":"net.minecraft.client.main.Main"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(versionDir, "1.20.1.jar"), []byte("client"), 0o644); err != nil {
		t.Fatal(err)
	}

	a := testApp(t, config.Config{GameDir: dir, SelectedVersion: "1.20.1"})
	dest := filepath.Join(dir, "exports", "export.zip")
	a.rt.ExportInstance("1.20.1", dest)

	var done instanceDoneMsg
	deadline := time.After(10 * time.Second)
	for done.Action == "" {
		select {
		case msg := <-a.rt.Events:
			if d, ok := msg.(instanceDoneMsg); ok {
				done = d
			}
		case <-deadline:
			t.Fatal("export timed out")
		}
	}
	if done.Err != nil {
		t.Fatal(done.Err)
	}
	if done.Action != "export" || done.Name != "1.20.1" {
		t.Fatalf("done = %#v", done)
	}
	if st, err := os.Stat(dest); err != nil || st.Size() == 0 {
		t.Fatalf("export missing: %v", err)
	}
}
