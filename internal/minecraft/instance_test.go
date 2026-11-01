package minecraft

import (
	"archive/zip"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestExportImportVersionInstance(t *testing.T) {
	src := t.TempDir()
	writeJSON(t, VersionJSONPath(src, "1.20.1"), map[string]any{
		"id":        "1.20.1",
		"mainClass": "net.minecraft.client.main.Main",
	})
	writeJSON(t, VersionJSONPath(src, "fabric-loader-0.16.9-1.20.1"), map[string]any{
		"id":           "fabric-loader-0.16.9-1.20.1",
		"inheritsFrom": "1.20.1",
		"mainClass":    "net.fabricmc.loader.impl.launch.knot.KnotClient",
	})
	mustWrite(t, filepath.Join(src, "versions", "1.20.1", "1.20.1.jar"), "client")
	mustWrite(t, filepath.Join(src, "mods", "sodium.jar"), "mod")
	mustWrite(t, filepath.Join(src, "resourcepacks", "pack.zip"), "pack")
	mustWrite(t, filepath.Join(src, "config", "settings.txt"), "cfg")
	mustWrite(t, filepath.Join(src, "options.txt"), "options")
	mustWrite(t, filepath.Join(src, "servers.dat"), "servers")
	mustWrite(t, filepath.Join(src, "saves", "World", "level.dat"), "level")
	mustWrite(t, filepath.Join(src, "screenshots", "shot.png"), "png")
	mustWrite(t, filepath.Join(src, "logs", "latest.log"), "log")

	dest := filepath.Join(src, "exports", "fabric.zip")
	archive, err := ExportInstance(src, "fabric-loader-0.16.9-1.20.1", dest, true)
	if err != nil {
		t.Fatal(err)
	}
	if archive.FormatVersion != InstanceFormatVersion || archive.Source != "version" {
		t.Fatalf("archive = %#v", archive)
	}
	if archive.VersionID != "fabric-loader-0.16.9-1.20.1" || archive.MCVersion != "1.20.1" {
		t.Fatalf("archive = %#v", archive)
	}
	names := zipNames(t, dest)
	for _, want := range []string{
		"mctl-instance.json",
		"versions/1.20.1/1.20.1.json",
		"versions/1.20.1/1.20.1.jar",
		"versions/fabric-loader-0.16.9-1.20.1/fabric-loader-0.16.9-1.20.1.json",
		"mods/sodium.jar",
		"resourcepacks/pack.zip",
		"config/settings.txt",
		"options.txt",
		"servers.dat",
		"saves/World/level.dat",
		"screenshots/shot.png",
	} {
		if !names[want] {
			t.Fatalf("archive is missing %q; entries = %v", want, sortedKeys(names))
		}
	}
	if names["logs/latest.log"] {
		t.Fatal("logs should not be exported")
	}

	dst := t.TempDir()
	got, root, err := ImportInstance(dst, dest)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != archive.Name || got.VersionID != archive.VersionID {
		t.Fatalf("imported = %#v", got)
	}
	if !strings.HasPrefix(root, filepath.Join(dst, "instances")) {
		t.Fatalf("instance root = %q", root)
	}
	for _, rel := range []string{
		"mods/sodium.jar",
		"config/settings.txt",
		"saves/World/level.dat",
		"versions/1.20.1/1.20.1.jar",
		"versions/fabric-loader-0.16.9-1.20.1/fabric-loader-0.16.9-1.20.1.json",
		"versions/fabric-loader-0.16.9-1.20.1/mctl-pack.json",
	} {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(rel))); err != nil {
			t.Fatalf("imported file %q missing: %v", rel, err)
		}
	}
	if _, err := os.Stat(filepath.Join(root, "logs")); !os.IsNotExist(err) {
		t.Fatalf("logs should not be imported: %v", err)
	}

	rv, err := ResolveVersion(root, "fabric-loader-0.16.9-1.20.1")
	if err != nil {
		t.Fatal(err)
	}
	if rv.MainClass != "net.fabricmc.loader.impl.launch.knot.KnotClient" || rv.JarPath == "" {
		t.Fatalf("resolved = %#v", rv)
	}

	list, err := ScanVersions(dst)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].ID != archive.VersionID || list[0].GameDir != root {
		t.Fatalf("scan = %#v", list)
	}
	if list[0].MCVersion != "1.20.1" || list[0].Loader != "Fabric" {
		t.Fatalf("scan metadata = %#v", list[0])
	}

	if _, _, err := ImportInstance(dst, dest); err != nil {
		t.Fatal(err)
	}
	list, err = ScanVersions(dst)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Fatalf("second import should create a new instance: %#v", list)
	}
}

func TestExportInstanceWithoutSaves(t *testing.T) {
	src := t.TempDir()
	writeJSON(t, VersionJSONPath(src, "1.20.1"), map[string]any{"id": "1.20.1", "mainClass": "net.minecraft.client.main.Main"})
	mustWrite(t, filepath.Join(src, "versions", "1.20.1", "1.20.1.jar"), "client")
	mustWrite(t, filepath.Join(src, "saves", "World", "level.dat"), "level")
	mustWrite(t, filepath.Join(src, "mods", "a.jar"), "mod")

	dest := filepath.Join(src, "exports", "vanilla.zip")
	if _, err := ExportInstance(src, "1.20.1", dest, false); err != nil {
		t.Fatal(err)
	}
	names := zipNames(t, dest)
	if names["saves/World/level.dat"] {
		t.Fatal("saves should be excluded")
	}
	if !names["mods/a.jar"] {
		t.Fatal("mods should be included")
	}
}

func TestExportPackInstance(t *testing.T) {
	src := t.TempDir()
	instanceRoot := filepath.Join(src, "instances", "my-pack")
	writeJSON(t, VersionJSONPath(instanceRoot, "1.20.1"), map[string]any{"id": "1.20.1", "mainClass": "net.minecraft.client.main.Main"})
	packID := "pack-my-pack"
	writeJSON(t, VersionJSONPath(instanceRoot, packID), map[string]any{
		"id":           packID,
		"inheritsFrom": "1.20.1",
		"type":         "release",
	})
	writeJSON(t, filepath.Join(VersionPath(instanceRoot, packID), "mctl-pack.json"), map[string]any{
		"name": "My Pack", "versionId": "1.0", "mcVersion": "1.20.1", "loader": "1.20.1",
	})
	mustWrite(t, filepath.Join(instanceRoot, "versions", "1.20.1", "1.20.1.jar"), "client")
	mustWrite(t, filepath.Join(instanceRoot, "mods", "packmod.jar"), "mod")
	mustWrite(t, filepath.Join(instanceRoot, "saves", "World", "level.dat"), "level")

	dest := filepath.Join(src, "exports", "pack.zip")
	archive, err := ExportInstance(src, packID, dest, true)
	if err != nil {
		t.Fatal(err)
	}
	if archive.Source != "pack" || archive.Name != "My Pack" {
		t.Fatalf("archive = %#v", archive)
	}
	names := zipNames(t, dest)
	if !names["versions/1.20.1/1.20.1.json"] || !names["mods/packmod.jar"] || !names["versions/"+packID+"/mctl-pack.json"] {
		t.Fatalf("pack archive entries = %v", sortedKeys(names))
	}

	dst := t.TempDir()
	if _, root, err := ImportInstance(dst, dest); err != nil {
		t.Fatal(err)
	} else if _, err := ResolveVersion(root, packID); err != nil {
		t.Fatal(err)
	}
	list, err := ScanVersions(dst)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].ID != packID {
		t.Fatalf("scan = %#v", list)
	}
}

func TestImportInstanceRejectsUnsafeArchive(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "evil.zip")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	manifest, _ := json.Marshal(InstanceArchive{FormatVersion: InstanceFormatVersion, Name: "Evil", VersionID: "1.20.1", Source: "version"})
	w, _ := zw.Create("mctl-instance.json")
	_, _ = w.Write(manifest)
	w, _ = zw.Create("../escape.txt")
	_, _ = w.Write([]byte("bad"))
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	f.Close()
	if _, root, err := ImportInstance(dir, path); err == nil {
		t.Fatalf("expected unsafe path rejection, imported %q", root)
	}
	if _, err := os.Stat(filepath.Join(dir, "instances")); !os.IsNotExist(err) {
		t.Fatalf("failed import should not leave an instance: %v", err)
	}
}

func TestImportInstanceRejectsPlainZip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "plain.zip")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	w, _ := zw.Create("hello.txt")
	_, _ = w.Write([]byte("hello"))
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	f.Close()
	if _, _, err := ImportInstance(dir, path); err == nil || !strings.Contains(err.Error(), "mctl-instance.json") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func zipNames(t *testing.T, path string) map[string]bool {
	t.Helper()
	zr, err := zip.OpenReader(path)
	if err != nil {
		t.Fatal(err)
	}
	defer zr.Close()
	out := map[string]bool{}
	for _, f := range zr.File {
		if !f.FileInfo().IsDir() {
			out[f.Name] = true
		}
	}
	return out
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
