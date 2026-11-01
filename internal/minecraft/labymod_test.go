package minecraft

import (
	"context"
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
