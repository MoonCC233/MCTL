package minecraft

import (
	"archive/zip"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestSafeJoin(t *testing.T) {
	root := t.TempDir()
	if _, err := safeJoin(root, "mods/example.jar"); err != nil {
		t.Fatal(err)
	}
	badPaths := []string{"../outside.jar", "../../outside.jar", "/absolute.jar", ""}
	if runtime.GOOS == "windows" {
		badPaths = append(badPaths, `..\outside.jar`)
	}
	for _, bad := range badPaths {
		if _, err := safeJoin(root, bad); err == nil {
			t.Fatalf("expected path rejection for %q", bad)
		}
	}
}

func TestPackSlug(t *testing.T) {
	a := packSlug("My Great Pack!", "1.2.3")
	b := packSlug("My Great Pack!", "1.2.3")
	if a != b {
		t.Fatalf("slug should be deterministic: %q != %q", a, b)
	}
	if !strings.HasPrefix(a, "my-great-pack-") || strings.ContainsAny(a, " !") {
		t.Fatalf("unexpected slug: %q", a)
	}
	if len(a) > 41 {
		t.Fatalf("slug too long: %q", a)
	}
}

func TestModpackTasks(t *testing.T) {
	root := t.TempDir()
	idx := &modpackIndex{
		Files: []modpackFile{
			{Path: "mods/a.jar", Downloads: []string{"https://example.invalid/a.jar"}, Hashes: map[string]string{"sha1": "abc"}},
			{Path: "mods/server.jar", Downloads: []string{"https://example.invalid/s.jar"}, Env: map[string]string{"client": "unsupported"}},
			{Path: "../escape.jar", Downloads: []string{"https://example.invalid/e.jar"}},
		},
	}
	if _, err := modpackTasks(idx, root); err == nil {
		t.Fatal("expected unsafe path error")
	}
	idx.Files = idx.Files[:2]
	tasks, err := modpackTasks(idx, root)
	if err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 1 || tasks[0].SHA1 != "abc" {
		t.Fatalf("tasks = %#v", tasks)
	}
}

func TestImportModpackCreatesInstance(t *testing.T) {
	var baseURL string
	mux := http.NewServeMux()
	mux.HandleFunc("/manifest.json", func(w http.ResponseWriter, r *http.Request) {
		writeTestJSON(t, w, map[string]any{
			"latest": map[string]string{"release": "1.20.1", "snapshot": "1.20.1"},
			"versions": []any{map[string]any{
				"id":          "1.20.1",
				"type":        "release",
				"url":         baseURL + "/1.20.1.json",
				"releaseTime": "2024-01-01T00:00:00Z",
			}},
		})
	})
	mux.HandleFunc("/1.20.1.json", func(w http.ResponseWriter, r *http.Request) {
		writeTestJSON(t, w, map[string]any{
			"id":          "1.20.1",
			"type":        "release",
			"mainClass":   "net.minecraft.client.main.Main",
			"javaVersion": map[string]any{"majorVersion": 17},
			"downloads": map[string]any{
				"client": map[string]any{"url": baseURL + "/client.jar", "size": 6},
			},
			"libraries": []any{map[string]any{
				"name": "com.example:lib:1.0",
				"downloads": map[string]any{
					"artifact": map[string]any{"url": baseURL + "/lib.jar", "size": 3},
				},
			}},
		})
	})
	mux.HandleFunc("/client.jar", func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("client")) })
	mux.HandleFunc("/lib.jar", func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("lib")) })
	mux.HandleFunc("/mod.jar", func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("mod")) })
	server := httptest.NewServer(mux)
	defer server.Close()
	baseURL = server.URL

	dir := t.TempDir()
	modSum := sha1.Sum([]byte("mod"))
	mrpack := filepath.Join(dir, "test.mrpack")
	f, err := os.Create(mrpack)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	index := map[string]any{
		"formatVersion": 1,
		"game":          "minecraft",
		"versionId":     "1.0.0",
		"name":          "Test Pack",
		"files": []any{map[string]any{
			"path":      "mods/test.jar",
			"hashes":    map[string]string{"sha1": hex.EncodeToString(modSum[:])},
			"downloads": []string{baseURL + "/mod.jar"},
			"fileSize":  3,
		}},
		"dependencies": map[string]string{"minecraft": "1.20.1"},
	}
	data, _ := json.Marshal(index)
	w, _ := zw.Create("modrinth.index.json")
	_, _ = w.Write(data)
	w, _ = zw.Create("overrides/config/test.txt")
	_, _ = w.Write([]byte("hello override"))
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	f.Close()

	inst := NewInstaller(dir, Source{ID: "test", ManifestURL: baseURL + "/manifest.json"}, nil, nil)
	id, err := inst.ImportModpack(context.Background(), mrpack, "", false, "")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(id, "pack-test-pack-") {
		t.Fatalf("pack id = %q", id)
	}
	slug := strings.TrimPrefix(id, "pack-")
	instanceRoot := filepath.Join(dir, "instances", slug)
	for _, p := range []string{
		filepath.Join(instanceRoot, "mods", "test.jar"),
		filepath.Join(instanceRoot, "config", "test.txt"),
		filepath.Join(instanceRoot, "versions", id, id+".json"),
		filepath.Join(instanceRoot, "versions", id, "mctl-pack.json"),
	} {
		if _, err := os.Stat(p); err != nil {
			t.Fatalf("missing %s: %v", p, err)
		}
	}
	rv, err := ResolveVersion(instanceRoot, id)
	if err != nil {
		t.Fatal(err)
	}
	if rv.InheritsFrom != "1.20.1" || rv.MainClass != "net.minecraft.client.main.Main" {
		t.Fatalf("resolved = %#v", rv)
	}
	list, err := ScanVersions(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].ID != id || list[0].GameDir != instanceRoot || list[0].MCVersion != "1.20.1" {
		t.Fatalf("scan = %#v", list)
	}
	if !list[0].Complete {
		t.Fatal("pack should be complete after import")
	}
}

func TestParseCurseLoader(t *testing.T) {
	for _, tc := range []struct {
		id, kind, version string
	}{
		{"forge-47.3.0", "forge", "47.3.0"},
		{"neoforge-21.1.80", "neoforge", "21.1.80"},
		{"fabric-0.15.11", "fabric", "0.15.11"},
		{"quilt-0.20.0", "quilt", "0.20.0"},
	} {
		kind, version := parseCurseLoader([]curseLoader{{ID: tc.id}})
		if kind != tc.kind || version != tc.version {
			t.Fatalf("%s -> %q %q", tc.id, kind, version)
		}
	}
}

func TestFileNameFromURL(t *testing.T) {
	if got := fileNameFromURL("https://cdn.example/mods/example.jar?x=1", 1, 2); got != "example.jar" {
		t.Fatalf("name = %q", got)
	}
	if got := fileNameFromURL("not a url", 1, 2); got != "1-2.jar" {
		t.Fatalf("fallback = %q", got)
	}
}

func TestImportCurseForgePack(t *testing.T) {
	var baseURL string
	mux := http.NewServeMux()
	mux.HandleFunc("/manifest.json", func(w http.ResponseWriter, r *http.Request) {
		writeTestJSON(t, w, map[string]any{
			"latest": map[string]string{"release": "1.20.1", "snapshot": "1.20.1"},
			"versions": []any{map[string]any{
				"id": "1.20.1", "type": "release", "url": baseURL + "/1.20.1.json",
			}},
		})
	})
	mux.HandleFunc("/1.20.1.json", func(w http.ResponseWriter, r *http.Request) {
		writeTestJSON(t, w, map[string]any{
			"id":          "1.20.1",
			"type":        "release",
			"mainClass":   "net.minecraft.client.main.Main",
			"javaVersion": map[string]any{"majorVersion": 17},
			"downloads": map[string]any{
				"client": map[string]any{"url": baseURL + "/client.jar", "size": 6},
			},
			"libraries": []any{map[string]any{
				"name": "com.example:lib:1.0",
				"downloads": map[string]any{
					"artifact": map[string]any{"url": baseURL + "/lib.jar", "size": 3},
				},
			}},
		})
	})
	mux.HandleFunc("/client.jar", func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("client")) })
	mux.HandleFunc("/lib.jar", func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("lib")) })
	mux.HandleFunc("/files/mod.jar", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-api-key") != "" {
			t.Fatalf("download should not send the API key")
		}
		_, _ = w.Write([]byte("mod"))
	})
	mux.HandleFunc("/v1/mods/123/files/456/download-url", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-api-key") != "test-key" {
			t.Fatalf("api key = %q", r.Header.Get("x-api-key"))
		}
		writeTestJSON(t, w, map[string]any{"data": baseURL + "/files/mod.jar"})
	})
	server := httptest.NewServer(mux)
	defer server.Close()
	baseURL = server.URL

	oldBase, oldClient := curseForgeAPIBase, curseForgeHTTPClient
	defer func() { curseForgeAPIBase, curseForgeHTTPClient = oldBase, oldClient }()
	curseForgeAPIBase = server.URL + "/v1"
	curseForgeHTTPClient = server.Client()

	dir := t.TempDir()
	zipPath := filepath.Join(dir, "curse.zip")
	f, err := os.Create(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	manifest := map[string]any{
		"minecraft": map[string]any{
			"version":    "1.20.1",
			"modLoaders": []any{},
		},
		"name":    "Curse Pack",
		"version": "1.0.0",
		"files": []any{map[string]any{
			"projectID": 123, "fileID": 456, "required": true,
		}},
	}
	data, _ := json.Marshal(manifest)
	w, _ := zw.Create("manifest.json")
	_, _ = w.Write(data)
	w, _ = zw.Create("overrides/config/test.txt")
	_, _ = w.Write([]byte("curse override"))
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	f.Close()

	inst := NewInstaller(dir, Source{ID: "test", ManifestURL: baseURL + "/manifest.json"}, nil, nil)
	if _, err := inst.ImportModpack(context.Background(), zipPath, "", false, ""); err == nil {
		t.Fatal("expected missing CurseForge API key error")
	}
	id, err := inst.ImportModpack(context.Background(), zipPath, "", false, "test-key")
	if err != nil {
		t.Fatal(err)
	}
	slug := strings.TrimPrefix(id, "pack-")
	instanceRoot := filepath.Join(dir, "instances", slug)
	for _, p := range []string{
		filepath.Join(instanceRoot, "mods", "mod.jar"),
		filepath.Join(instanceRoot, "config", "test.txt"),
		filepath.Join(instanceRoot, "versions", id, "mctl-pack.json"),
	} {
		if _, err := os.Stat(p); err != nil {
			t.Fatalf("missing %s: %v", p, err)
		}
	}
	rv, err := ResolveVersion(instanceRoot, id)
	if err != nil {
		t.Fatal(err)
	}
	if rv.InheritsFrom != "1.20.1" {
		t.Fatalf("inherits = %q", rv.InheritsFrom)
	}
	list, err := ScanVersions(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].ID != id || list[0].GameDir != instanceRoot {
		t.Fatalf("scan = %#v", list)
	}
}

func TestImportModpackRejectsPlainZip(t *testing.T) {
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

	inst := NewInstaller(dir, Sources[0], nil, nil)
	if _, err := inst.ImportModpack(context.Background(), path, "", false, ""); err == nil {
		t.Fatal("expected invalid modpack error")
	} else if !strings.Contains(err.Error(), "modrinth.index.json") {
		t.Fatalf("unexpected error: %v", err)
	}
}
