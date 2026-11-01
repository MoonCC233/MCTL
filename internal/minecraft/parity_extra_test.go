package minecraft

import (
	"archive/zip"
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func worldZip(t *testing.T) []byte {
	t.Helper()
	var archive bytes.Buffer
	zw := zip.NewWriter(&archive)
	for name, content := range map[string]string{
		"World/level.dat":        "level",
		"World/region/r.0.0.mca": "region",
	} {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return archive.Bytes()
}

func TestDownloadWorldURL(t *testing.T) {
	archive := worldZip(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(archive)
	}))
	defer server.Close()

	hc := NewHTTPClient(Source{ID: "test"})
	hc.Client = server.Client()
	gameDir := t.TempDir()
	res, err := DownloadWorldURL(context.Background(), hc, server.URL+"/world.zip", gameDir)
	if err != nil {
		t.Fatal(err)
	}
	if res.Kind != "world" || res.Name != "World" {
		t.Fatalf("res = %#v", res)
	}
	want := filepath.Join(gameDir, "saves", "World")
	if res.Path != want {
		t.Fatalf("path = %q", res.Path)
	}
	if _, err := os.Stat(filepath.Join(want, "level.dat")); err != nil {
		t.Fatalf("level.dat missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(want, "region", "r.0.0.mca")); err != nil {
		t.Fatalf("region missing: %v", err)
	}

	worlds, err := ListWorlds(gameDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(worlds) != 1 || worlds[0].Name != "World" {
		t.Fatalf("worlds = %#v", worlds)
	}

	for _, bad := range []string{"", "ftp://example.com/world.zip", "not a url"} {
		if _, err := DownloadWorldURL(context.Background(), hc, bad, gameDir); err == nil {
			t.Fatalf("expected error for %q", bad)
		}
	}
}

func TestDownloadWorldURLRejectsInvalidArchive(t *testing.T) {
	var archive bytes.Buffer
	zw := zip.NewWriter(&archive)
	w, _ := zw.Create("readme.txt")
	_, _ = w.Write([]byte("no level.dat"))
	_ = zw.Close()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(archive.Bytes())
	}))
	defer server.Close()

	hc := NewHTTPClient(Source{ID: "test"})
	hc.Client = server.Client()
	gameDir := t.TempDir()
	if _, err := DownloadWorldURL(context.Background(), hc, server.URL+"/bad.zip", gameDir); err == nil {
		t.Fatal("expected invalid world archive error")
	}
	if entries, err := os.ReadDir(filepath.Join(gameDir, "saves")); err == nil && len(entries) != 0 {
		t.Fatalf("failed import left worlds behind: %v", entries)
	}
}

func TestSchematicResources(t *testing.T) {
	gameDir := t.TempDir()
	if ResourceSubdir("schematic") != "schematics" {
		t.Fatalf("subdir = %q", ResourceSubdir("schematic"))
	}
	dir := filepath.Join(gameDir, "schematics")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"house.litematic", "farm.schem", "notes.txt"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("data"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	list, err := ListResources(gameDir, "schematic")
	if err != nil {
		t.Fatal(err)
	}
	// ListResources lists every file; the UI shows the schematic folder as-is.
	if len(list) != 3 {
		t.Fatalf("list = %#v", list)
	}
	if err := DeleteResource(gameDir, "schematic", "house.litematic"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "house.litematic")); !os.IsNotExist(err) {
		t.Fatalf("file should be deleted: %v", err)
	}
	if err := DeleteResource(gameDir, "schematic", "../escape"); err == nil {
		t.Fatal("expected path traversal rejection")
	}
}
