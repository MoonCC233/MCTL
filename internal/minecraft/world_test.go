package minecraft

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

func TestWorldBackupImportDelete(t *testing.T) {
	dir := t.TempDir()
	worldDir := filepath.Join(dir, "saves", "My World")
	mustWrite(t, filepath.Join(worldDir, "level.dat"), "level")
	mustWrite(t, filepath.Join(worldDir, "region", "r.0.0.mca"), "region")

	list, err := ListWorlds(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].Name != "My World" || list[0].Size == 0 {
		t.Fatalf("worlds = %#v", list)
	}

	backup := WorldBackupPath(dir, "My World")
	if err := BackupWorld(dir, "My World", backup); err != nil {
		t.Fatal(err)
	}
	if st, err := os.Stat(backup); err != nil || st.Size() == 0 {
		t.Fatalf("backup missing: %v", err)
	}

	other := t.TempDir()
	name, err := ImportWorld(other, backup)
	if err != nil {
		t.Fatal(err)
	}
	if name != "My World" {
		t.Fatalf("imported name = %q", name)
	}
	if data, err := os.ReadFile(filepath.Join(other, "saves", "My World", "level.dat")); err != nil || string(data) != "level" {
		t.Fatalf("level.dat = %q, err = %v", data, err)
	}
	if err := DeleteWorld(dir, "My World"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(worldDir); !os.IsNotExist(err) {
		t.Fatalf("world should be deleted: %v", err)
	}
	if err := DeleteWorld(dir, `..\outside`); err == nil {
		t.Fatal("expected traversal rejection")
	}
}

func TestImportWorldRejectsInvalidZip(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "bad.zip")
	f, err := os.Create(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	w, _ := zw.Create("notes.txt")
	_, _ = w.Write([]byte("not a world"))
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	f.Close()
	if _, err := ImportWorld(dir, zipPath); err == nil {
		t.Fatal("expected invalid world error")
	}
}
