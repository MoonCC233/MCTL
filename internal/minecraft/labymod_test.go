package minecraft

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestFetchLabyModBuilds(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/manifest/production/latest.json", func(w http.ResponseWriter, r *http.Request) {
		writeTestJSON(t, w, map[string]any{
			"labyModVersion": "48.3",
			"minecraftVersions": []any{
				map[string]any{
					"tag": "1.20", "version": "1.20.1", "type": "release",
					"customManifestUrl": "https://example.invalid/production.json",
					"runtime":           map[string]any{"version": 21},
				},
			},
		})
	})
	mux.HandleFunc("/manifest/snapshot/latest.json", func(w http.ResponseWriter, r *http.Request) {
		writeTestJSON(t, w, map[string]any{
			"labyModVersion": "48.3",
			"minecraftVersions": []any{
				map[string]any{
					"tag": "1.20", "version": "1.20.1", "type": "release",
					"customManifestUrl": "https://example.invalid/snapshot.json",
					"runtime":           map[string]any{"version": 21},
				},
			},
		})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	oldManifest, oldInstaller := labyModManifestBase, labyModInstallerBase
	defer func() { labyModManifestBase, labyModInstallerBase = oldManifest, oldInstaller }()
	labyModManifestBase = server.URL + "/manifest"
	labyModInstallerBase = server.URL + "/installer"
	hc := NewHTTPClient(Source{ID: "test"})
	hc.Client = server.Client()

	builds, err := FetchLabyModBuilds(context.Background(), hc, "1.20.1")
	if err != nil {
		t.Fatal(err)
	}
	if len(builds) != 2 || builds[0].Channel != "production" || !builds[0].Stable || builds[1].Channel != "snapshot" {
		t.Fatalf("builds = %#v", builds)
	}
	if builds[0].RuntimeMajor != 21 || builds[0].ManifestURL == "" {
		t.Fatalf("metadata = %#v", builds[0])
	}
	if other, err := FetchLabyModBuilds(context.Background(), hc, "1.99"); err != nil || len(other) != 0 {
		t.Fatalf("unsupported builds = %#v, err = %v", other, err)
	}
}

func TestInstallLabyModProfile(t *testing.T) {
	var baseURL string
	mux := http.NewServeMux()
	mux.HandleFunc("/manifest/production/latest.json", func(w http.ResponseWriter, r *http.Request) {
		writeTestJSON(t, w, map[string]any{
			"labyModVersion":  "48.3",
			"commitReference": "e0629f72",
			"sha1":            hex.EncodeToString(func() []byte { s := sha1.Sum([]byte("laby")); return s[:] }()),
			"size":            4,
			"minecraftVersions": []any{map[string]any{
				"tag": "1.20", "version": "1.20.1", "type": "release",
				"customManifestUrl": baseURL + "/custom.json",
			}},
		})
	})
	mux.HandleFunc("/manifest/snapshot/latest.json", func(w http.ResponseWriter, r *http.Request) {
		writeTestJSON(t, w, map[string]any{
			"labyModVersion":    "48.3",
			"commitReference":   "e0629f72",
			"minecraftVersions": []any{},
		})
	})
	mux.HandleFunc("/custom.json", func(w http.ResponseWriter, r *http.Request) {
		writeTestJSON(t, w, map[string]any{
			"id":        "1.20.1",
			"mainClass": "net.minecraft.launchwrapper.Launch",
			"type":      "release",
			"arguments": map[string]any{"game": []string{"--dummyMinecraftClass", "net.minecraft.client.main.Main"}},
			"libraries": []any{},
		})
	})
	mux.HandleFunc("/libraries/production.json", func(w http.ResponseWriter, r *http.Request) {
		writeTestJSON(t, w, map[string]any{"libraries": []any{}})
	})
	mux.HandleFunc("/download/labymod4/production/e0629f72.jar", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("laby"))
	})
	server := httptest.NewServer(mux)
	defer server.Close()
	baseURL = server.URL

	oldManifest, oldLibs, oldDownload := labyModManifestBase, labyModLibrariesBase, labyModDownloadBase
	defer func() {
		labyModManifestBase, labyModLibrariesBase, labyModDownloadBase = oldManifest, oldLibs, oldDownload
	}()
	labyModManifestBase = server.URL + "/manifest"
	labyModLibrariesBase = server.URL + "/libraries"
	labyModDownloadBase = server.URL + "/download/labymod4"

	dir := t.TempDir()
	inst := NewInstaller(dir, Source{ID: "test"}, nil, nil)
	inst.HTTP.Client = server.Client()
	id, err := inst.InstallLabyModProfile(context.Background(), "1.20.1", "production")
	if err != nil {
		t.Fatal(err)
	}
	if id != "LabyMod-4-1.20-production" {
		t.Fatalf("id = %q", id)
	}
	rv, err := ResolveVersion(dir, id)
	if err != nil {
		t.Fatal(err)
	}
	if rv.MainClass != "net.minecraft.launchwrapper.Launch" {
		t.Fatalf("main class = %q", rv.MainClass)
	}
	found := false
	for _, l := range rv.Libraries {
		if l.Name == "net.labymod:LabyMod:4" {
			found = true
		}
	}
	if !found {
		t.Fatalf("LabyMod library missing: %#v", rv.Libraries)
	}
	jar := filepath.Join(dir, "libraries", "net", "labymod", "LabyMod", "4", "LabyMod-4.jar")
	if data, err := os.ReadFile(jar); err != nil || string(data) != "laby" {
		t.Fatalf("jar = %q, err = %v", data, err)
	}
}

func TestInstallLabyMod(t *testing.T) {
	dir := t.TempDir()
	mux := http.NewServeMux()
	mux.HandleFunc("/installer/production/java", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("laby-installer"))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	old := labyModInstallerBase
	defer func() { labyModInstallerBase = old }()
	labyModInstallerBase = server.URL + "/installer"
	inst := NewInstaller(dir, Source{ID: "test"}, nil, nil)
	inst.HTTP.Client = server.Client()
	path, err := inst.InstallLabyMod(context.Background(), "production")
	if err != nil {
		t.Fatal(err)
	}
	if path != filepath.Join(dir, "mctl-installers", "LabyMod4ProductionInstaller.jar") {
		t.Fatalf("path = %q", path)
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "laby-installer" {
		t.Fatalf("file = %q, err = %v", data, err)
	}
	if _, err := inst.InstallLabyMod(context.Background(), "unknown"); err == nil {
		t.Fatal("expected unknown channel error")
	}
}
