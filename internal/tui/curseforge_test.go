package tui

import (
	"strings"
	"testing"

	"github.com/MoonCC233/MCTL/internal/config"
	"github.com/MoonCC233/MCTL/internal/minecraft"
)

func TestCurseForgeRequiresAPIKey(t *testing.T) {
	a := testApp(t, config.Config{GameDir: t.TempDir()})
	a.openCurseForgeSearch()
	if a.modal.Kind != modalError {
		t.Fatalf("modal = %#v", a.modal)
	}
	if !strings.Contains(a.modal.Body, "API Key") {
		t.Fatalf("body = %q", a.modal.Body)
	}
}

func TestCurseForgeSearchPrompt(t *testing.T) {
	a := testApp(t, config.Config{GameDir: t.TempDir(), CurseForgeAPIKey: "key"})
	a.openCurseForgeSearch()
	if a.modal.Kind != modalInput || a.modal.Action != "search-curseforge" {
		t.Fatalf("modal = %#v", a.modal)
	}
}

func TestCurseForgeResultsModal(t *testing.T) {
	a := testApp(t, config.Config{GameDir: t.TempDir(), CurseForgeAPIKey: "key"})
	a.handleEvent(curseForgeModsMsg{Mods: []minecraft.CurseForgeMod{
		{ID: 42, Name: "Better MC", DownloadCount: 1000},
	}})
	if a.modal.Kind != modalPicker || a.modal.Action != "install-curseforge" {
		t.Fatalf("modal = %#v", a.modal)
	}
	if len(a.modal.Items) != 1 || a.modal.Items[0].Value != "42" {
		t.Fatalf("items = %#v", a.modal.Items)
	}
	out := a.View()
	if !strings.Contains(out, "Better MC") || !strings.Contains(out, "1000") {
		t.Fatalf("modal output missing project details")
	}

	mods, ok := a.modal.Data.([]minecraft.CurseForgeMod)
	if !ok || len(mods) != 1 || mods[0].ID != 42 {
		t.Fatalf("data = %#v", a.modal.Data)
	}

	a.runAction("install-curseforge", "", pickerItem{Value: "999"}, mods)
	if !strings.Contains(a.toast.Text, "无效") {
		t.Fatalf("toast = %q", a.toast.Text)
	}
}
