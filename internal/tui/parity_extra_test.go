package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/MoonCC233/MCTL/internal/config"
	"github.com/MoonCC233/MCTL/internal/minecraft"
)

func TestWorldDownloadFlow(t *testing.T) {
	a := testApp(t, config.Config{GameDir: t.TempDir()})

	a.openCommunityResources()
	a.runAction("community-resource", "", pickerItem{Value: "world"}, nil)
	if a.modal.Action != "download-world-url" {
		t.Fatalf("world prompt = %#v", a.modal)
	}
	if a.modal.Kind != modalInput {
		t.Fatalf("world prompt kind = %v", a.modal.Kind)
	}

	// Avoid starting a network task in the test.
	a.rt.installing = true
	defer func() { a.rt.installing = false }()
	a.runAction("download-world-url", "https://example.invalid/world.zip", pickerItem{}, nil)
	if !strings.Contains(a.toast.Text, "已有任务") {
		t.Fatalf("toast = %q", a.toast.Text)
	}

	empty := testApp(t, config.Config{GameDir: t.TempDir()})
	empty.runAction("download-world-url", "  ", pickerItem{}, nil)
	if !strings.Contains(empty.toast.Text, "世界压缩包地址") {
		t.Fatalf("toast = %q", empty.toast.Text)
	}
}

func TestSchematicManager(t *testing.T) {
	dir := t.TempDir()
	schematicDir := filepath.Join(dir, "schematics")
	if err := os.MkdirAll(schematicDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(schematicDir, "house.litematic"), []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}
	a := testApp(t, config.Config{GameDir: dir})

	a.openResourceFiles("schematic")
	if a.modal.Kind != modalResources {
		t.Fatalf("modal = %#v", a.modal)
	}
	data, ok := a.modal.Data.(resourceModalData)
	if !ok || data.Kind != "schematic" || data.GameDir != dir {
		t.Fatalf("data = %#v", a.modal.Data)
	}
	if len(a.modal.Items) != 1 || a.modal.Items[0].Label != "house.litematic" {
		t.Fatalf("items = %#v", a.modal.Items)
	}
	out := a.View()
	for _, want := range []string{"投影", "house.litematic"} {
		if !strings.Contains(out, want) {
			t.Fatalf("schematic modal missing %q", want)
		}
	}
	if strings.Contains(out, "启用/停用") {
		t.Fatal("schematics should not offer enable/disable")
	}
}

func TestInstanceInstallPicker(t *testing.T) {
	dir := t.TempDir()
	a := testApp(t, config.Config{GameDir: dir, SelectedVersion: "1.20.1"})
	a.launch.versions = []minecraft.InstalledVersion{{ID: "1.20.1", MCVersion: "1.20.1", GameDir: dir}}
	a.page = pageVersions
	a.versions.list = a.launch.versions

	a.versionsKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	if a.modal.Action != "instance-install" {
		t.Fatalf("modal = %#v", a.modal)
	}
	if len(a.modal.Items) != 6 {
		t.Fatalf("items = %#v", a.modal.Items)
	}
	if a.modal.Items[0].Value != "repair" || a.modal.Items[1].Value != "fabric" {
		t.Fatalf("items = %#v", a.modal.Items)
	}
	data, ok := a.modal.Data.(instanceInstallData)
	if !ok || data.GameDir != dir || data.VersionID != "1.20.1" || data.MCVersion != "1.20.1" {
		t.Fatalf("data = %#v", a.modal.Data)
	}
	out := a.View()
	for _, want := range []string{"实例安装", "补全 / 修复文件", "安装 Fabric", "安装 NeoForge"} {
		if !strings.Contains(out, want) {
			t.Fatalf("picker missing %q", want)
		}
	}

	// Repair routing should open a cancellable progress modal without a network task.
	a.rt.installing = true
	defer func() { a.rt.installing = false }()
	a.runAction("instance-install", "", pickerItem{Value: "repair"}, data)
	if a.modal.Kind != modalProgress || !strings.Contains(a.modal.Title, "修复实例") {
		t.Fatalf("progress modal = %#v", a.modal)
	}
}

func TestInstanceInstallWithoutSelection(t *testing.T) {
	a := testApp(t, config.Config{GameDir: t.TempDir()})
	a.openInstanceInstall()
	if a.modal.Kind != modalNone {
		t.Fatalf("modal = %#v", a.modal)
	}
	if !strings.Contains(a.toast.Text, "选择一个版本") {
		t.Fatalf("toast = %q", a.toast.Text)
	}
}
