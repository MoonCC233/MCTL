package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MoonCC233/MCTL/internal/config"
	"github.com/MoonCC233/MCTL/internal/minecraft"
)

func wizardTestApp(t *testing.T) (*App, string) {
	t.Helper()
	dir := t.TempDir()
	versionDir := filepath.Join(dir, "versions", "fabric-loader-0.16.9-1.20.1")
	if err := os.MkdirAll(versionDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(versionDir, "fabric-loader-0.16.9-1.20.1.json"),
		[]byte(`{"id":"fabric-loader-0.16.9-1.20.1","inheritsFrom":"1.20.1","mainClass":"net.fabricmc.loader.impl.launch.knot.KnotClient"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	a := testApp(t, config.Config{GameDir: dir, SelectedVersion: "fabric-loader-0.16.9-1.20.1"})
	return a, dir
}

func TestInstanceWizardDefaults(t *testing.T) {
	a, _ := wizardTestApp(t)
	a.openInstanceWizard()
	if a.modal.Kind != modalInstanceWizard {
		t.Fatalf("modal = %#v", a.modal)
	}
	state := a.modal.Data.(instanceWizardState)
	if state.MCVersion != "1.20.1" || state.Loader != "vanilla" {
		t.Fatalf("state = %#v", state)
	}
	if len(a.modal.Items) != 7 || a.modal.Items[6].Value != "create" {
		t.Fatalf("items = %#v", a.modal.Items)
	}
	out := a.View()
	for _, want := range []string{"新建实例", "实例名称", "实例目录", "Minecraft 版本", "加载器", "创建实例"} {
		if !strings.Contains(out, want) {
			t.Fatalf("wizard missing %q", want)
		}
	}
}

func TestInstanceWizardEdits(t *testing.T) {
	a, _ := wizardTestApp(t)
	a.openInstanceWizard()
	state := a.modal.Data.(instanceWizardState)

	a.runAction("wizard-edit", "生存实例", pickerItem{}, instanceWizardEdit{State: state, Key: "name"})
	if a.modal.Kind != modalInstanceWizard {
		t.Fatalf("modal = %#v", a.modal)
	}
	state = a.modal.Data.(instanceWizardState)
	if state.Name != "生存实例" || a.modal.Selected != 0 {
		t.Fatalf("state = %#v selected = %d", state, a.modal.Selected)
	}

	a.runAction("wizard-edit", "custom/pack", pickerItem{}, instanceWizardEdit{State: state, Key: "dir"})
	state = a.modal.Data.(instanceWizardState)
	if state.Dir != "custom/pack" {
		t.Fatalf("state = %#v", state)
	}

	a.runAction("wizard-edit", "1.19.4", pickerItem{}, instanceWizardEdit{State: state, Key: "mc"})
	state = a.modal.Data.(instanceWizardState)
	if state.MCVersion != "1.19.4" {
		t.Fatalf("state = %#v", state)
	}

	a.runAction("wizard-loader", "", pickerItem{Value: "fabric"}, state)
	state = a.modal.Data.(instanceWizardState)
	if state.Loader != "fabric" {
		t.Fatalf("state = %#v", state)
	}
}

func TestInstanceWizardValidation(t *testing.T) {
	a, _ := wizardTestApp(t)
	a.openInstanceWizard()
	state := a.modal.Data.(instanceWizardState)
	state.MCVersion = ""

	a.wizardStart(state)
	if a.modal.Kind != modalInstanceWizard {
		t.Fatalf("missing version should keep the wizard open: %#v", a.modal)
	}
	if !strings.Contains(a.toast.Text, "Minecraft 版本") {
		t.Fatalf("toast = %q", a.toast.Text)
	}

	state.MCVersion = "1.20.1"
	state.Dir = "../outside"
	a.wizardStart(state)
	if a.modal.Kind != modalInstanceWizard {
		t.Fatalf("invalid dir should keep the wizard open: %#v", a.modal)
	}
	if !strings.Contains(a.toast.Text, "实例目录") {
		t.Fatalf("toast = %q", a.toast.Text)
	}
}

func TestWizardLoaderOptions(t *testing.T) {
	items := wizardLoaderOptions()
	if len(items) != len(minecraft.WizardLoaders) {
		t.Fatalf("items = %#v", items)
	}
	values := map[string]bool{}
	for _, it := range items {
		values[it.Value] = true
	}
	for _, kind := range []string{"vanilla", "fabric", "quilt", "legacyfabric", "forge", "neoforge"} {
		if !values[kind] {
			t.Fatalf("missing loader %q", kind)
		}
	}
}
