package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/MoonCC233/MCTL/internal/config"
	"github.com/MoonCC233/MCTL/internal/minecraft"
)

func TestVersionsPageExportAndNewInstance(t *testing.T) {
	a := testApp(t, config.Config{GameDir: t.TempDir(), SelectedVersion: "1.20.1"})
	a.launch.versions = []minecraft.InstalledVersion{{ID: "1.20.1", MCVersion: "1.20.1"}}
	a.versions.list = a.launch.versions
	a.page = pageVersions

	a.versionsKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	if a.modal.Action != "export-instance" {
		t.Fatalf("x should open the export prompt: %#v", a.modal)
	}
	if a.store.Snapshot().SelectedVersion != "1.20.1" {
		t.Fatalf("selected version = %q", a.store.Snapshot().SelectedVersion)
	}

	a.versionsKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'+'}})
	if a.modal.Kind != modalInstanceWizard {
		t.Fatalf("+ should open the instance wizard: %#v", a.modal)
	}
}

func TestLaunchPageInstanceSettingsShortcut(t *testing.T) {
	a := testApp(t, config.Config{GameDir: t.TempDir(), SelectedVersion: "1.20.1", MaxMemory: 4096})
	a.launch.versions = []minecraft.InstalledVersion{{ID: "1.20.1", MCVersion: "1.20.1"}}
	a.page = pageLaunch
	a.launchKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	if a.modal.Kind != modalInstanceSettings {
		t.Fatalf("i should open instance settings: %#v", a.modal)
	}
}

func TestDownloadCommunityResources(t *testing.T) {
	a := testApp(t, config.Config{GameDir: t.TempDir()})
	a.page = pageDownload
	a.downloadKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	if a.modal.Action != "community-resource" {
		t.Fatalf("c should open community resources: %#v", a.modal)
	}

	// Modpacks ask for a source, resources go straight to the search prompt.
	a.runAction("community-resource", "", pickerItem{Value: "modpacks"}, nil)
	if a.modal.Action != "community-pack-source" {
		t.Fatalf("modpacks source picker = %#v", a.modal)
	}
	a.runAction("community-pack-source", "", pickerItem{Value: "modrinth"}, nil)
	if a.modal.Action != "online-modpack-version" {
		t.Fatalf("modrinth should start the modpack flow: %#v", a.modal)
	}

	b := testApp(t, config.Config{GameDir: t.TempDir()})
	b.openCommunityResources()
	b.runAction("community-resource", "", pickerItem{Value: "mod"}, nil)
	if b.modal.Action != "search-resources" || b.resources.kind != "mod" {
		t.Fatalf("mod resource flow = %#v kind=%q", b.modal, b.resources.kind)
	}
}
