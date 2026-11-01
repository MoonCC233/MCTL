package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MoonCC233/MCTL/internal/config"
	"github.com/MoonCC233/MCTL/internal/minecraft"
)

func TestDatapackInstallFlow(t *testing.T) {
	dir := t.TempDir()
	worldDir := filepath.Join(dir, "saves", "World")
	if err := os.MkdirAll(worldDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(worldDir, "level.dat"), []byte("level"), 0o644); err != nil {
		t.Fatal(err)
	}
	a := testApp(t, config.Config{GameDir: dir, SelectedVersion: "1.20.1"})

	a.openCommunityResources()
	a.runAction("community-resource", "", pickerItem{Value: "datapack"}, nil)
	if a.modal.Action != "choose-datapack-world" {
		t.Fatalf("world picker = %#v", a.modal)
	}
	if len(a.modal.Items) != 1 || a.modal.Items[0].Value != worldDir {
		t.Fatalf("worlds = %#v", a.modal.Items)
	}

	a.runAction("choose-datapack-world", "", pickerItem{Value: worldDir}, nil)
	if a.modal.Action != "search-resources" || a.resources.kind != "datapack" {
		t.Fatalf("search prompt = %#v kind=%q", a.modal, a.resources.kind)
	}
	if a.resources.worldDir != worldDir {
		t.Fatalf("world dir = %q", a.resources.worldDir)
	}

	a.handleEvent(resourcesMsg{Kind: "datapack", Projects: []minecraft.ModrinthProject{
		{ID: "dp-id", Title: "Terralith", Downloads: 5000},
	}})
	if a.modal.Action != "install-datapack" {
		t.Fatalf("results modal = %#v", a.modal)
	}
	if world, ok := a.modal.Data.(string); !ok || world != worldDir {
		t.Fatalf("results data = %#v", a.modal.Data)
	}
	out := a.View()
	for _, want := range []string{"数据包", "Terralith"} {
		if !strings.Contains(out, want) {
			t.Fatalf("results modal missing %q", want)
		}
	}
}

func TestDatapackRequiresWorld(t *testing.T) {
	a := testApp(t, config.Config{GameDir: t.TempDir()})
	a.openDatapackWorldPicker()
	if a.modal.Kind != modalNone {
		t.Fatalf("modal = %#v", a.modal)
	}
	if !strings.Contains(a.toast.Text, "存档") {
		t.Fatalf("toast = %q", a.toast.Text)
	}
}

func TestDatapackFavoriteAsksForWorld(t *testing.T) {
	dir := t.TempDir()
	worldDir := filepath.Join(dir, "saves", "World")
	if err := os.MkdirAll(worldDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(worldDir, "level.dat"), []byte("level"), 0o644); err != nil {
		t.Fatal(err)
	}
	a := testApp(t, config.Config{GameDir: dir})
	if err := a.store.SetFavorite(config.Favorite{Kind: "datapack", ID: "dp-id", Name: "Terralith"}, true); err != nil {
		t.Fatal(err)
	}
	a.openFavorites()
	a.runAction("favorites-kind", "", pickerItem{Value: "datapack"}, nil)
	if a.modal.Action != "install-favorite" || len(a.modal.Items) != 1 {
		t.Fatalf("favorites = %#v", a.modal)
	}

	a.runAction("install-favorite", "", pickerItem{Value: "dp-id", Label: "Terralith"}, "datapack")
	if a.modal.Action != "choose-datapack-world" {
		t.Fatalf("favorite datapack should ask for a world: %#v", a.modal)
	}
	if a.resources.pendingID != "dp-id" || a.resources.pendingName != "Terralith" {
		t.Fatalf("pending = %q / %q", a.resources.pendingID, a.resources.pendingName)
	}
}
