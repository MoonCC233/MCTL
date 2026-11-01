package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/MoonCC233/MCTL/internal/config"
	"github.com/MoonCC233/MCTL/internal/minecraft"
)

func TestFavoriteToggleInPicker(t *testing.T) {
	a := testApp(t, config.Config{GameDir: t.TempDir()})
	a.modal = modalState{
		Kind:   modalPicker,
		Items:  []pickerItem{{Label: "Create", Desc: "1.0", Value: "create-id"}},
		Action: "install-online-modpack",
	}
	a.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	if !a.store.Snapshot().IsFavorite("modpack", "create-id") {
		t.Fatal("favorite should be stored")
	}
	if !strings.HasPrefix(a.modal.Items[0].Label, "★ ") {
		t.Fatalf("label = %q", a.modal.Items[0].Label)
	}

	a.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	if a.store.Snapshot().IsFavorite("modpack", "create-id") {
		t.Fatal("favorite should be removed")
	}
	if strings.HasPrefix(a.modal.Items[0].Label, "★ ") {
		t.Fatalf("label = %q", a.modal.Items[0].Label)
	}
}

func TestFavoritesList(t *testing.T) {
	a := testApp(t, config.Config{GameDir: t.TempDir()})
	if err := a.store.SetFavorite(config.Favorite{Kind: "modpack", ID: "pack-id", Name: "My Pack"}, true); err != nil {
		t.Fatal(err)
	}
	a.openFavorites()
	if a.modal.Kind != modalPicker || a.modal.Action != "favorites-kind" {
		t.Fatalf("modal = %#v", a.modal)
	}
	a.runAction("favorites-kind", "", pickerItem{Value: "modpack"}, nil)
	if a.modal.Action != "install-favorite" {
		t.Fatalf("modal = %#v", a.modal)
	}
	if len(a.modal.Items) != 1 || a.modal.Items[0].Label != "My Pack" || a.modal.Items[0].Value != "pack-id" {
		t.Fatalf("items = %#v", a.modal.Items)
	}

	a.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	if a.store.Snapshot().IsFavorite("modpack", "pack-id") {
		t.Fatal("d should remove the favorite")
	}
}

func TestOnlineModpackFilterFlow(t *testing.T) {
	a := testApp(t, config.Config{GameDir: t.TempDir()})
	a.openOnlineModpacks()
	if a.modal.Action != "online-modpack-version" {
		t.Fatalf("modal = %#v", a.modal)
	}
	a.runAction("online-modpack-version", "1.20.1", pickerItem{}, nil)
	if a.modal.Action != "online-modpack-loader" {
		t.Fatalf("modal = %#v", a.modal)
	}
	if a.modal.Data != "1.20.1" {
		t.Fatalf("data = %#v", a.modal.Data)
	}

	a.runAction("online-modpack-loader", "", pickerItem{Value: "fabric"}, "1.20.1")
	if a.modal.Action != "search-modpacks" {
		t.Fatalf("modal = %#v", a.modal)
	}
	filter, ok := a.modal.Data.(minecraft.ModpackFilter)
	if !ok || filter.GameVersion != "1.20.1" || filter.Loader != "fabric" {
		t.Fatalf("filter = %#v", a.modal.Data)
	}
}
