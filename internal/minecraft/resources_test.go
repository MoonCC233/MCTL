package minecraft

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLocalResourceToggleAndDelete(t *testing.T) {
	dir := t.TempDir()
	mods := filepath.Join(dir, "mods")
	mustWrite(t, filepath.Join(mods, "alpha.jar"), "a")
	mustWrite(t, filepath.Join(mods, "beta.jar.disabled"), "b")

	list, err := ListResources(dir, "mod")
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 || list[0].Name != "alpha.jar" || list[1].Name != "beta.jar.disabled" {
		t.Fatalf("list = %#v", list)
	}
	if _, err := ToggleResource(dir, "mod", "alpha.jar"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(mods, "alpha.jar.disabled")); err != nil {
		t.Fatalf("alpha should be disabled: %v", err)
	}
	if _, err := ToggleResource(dir, "mod", "beta.jar.disabled"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(mods, "beta.jar")); err != nil {
		t.Fatalf("beta should be enabled: %v", err)
	}
	if err := DeleteResource(dir, "mod", "alpha.jar.disabled"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(mods, "alpha.jar.disabled")); !os.IsNotExist(err) {
		t.Fatalf("alpha should be deleted: %v", err)
	}
	if err := DeleteResource(dir, "mod", `..\outside.jar`); err == nil {
		t.Fatal("expected traversal rejection")
	}
}

func TestListResourcesMissingDir(t *testing.T) {
	list, err := ListResources(t.TempDir(), "shader")
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 0 {
		t.Fatalf("list = %#v", list)
	}
}
