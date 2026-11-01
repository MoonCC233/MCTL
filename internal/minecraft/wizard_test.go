package minecraft

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveInstanceDir(t *testing.T) {
	gameDir := filepath.Join(t.TempDir(), "game")
	root := filepath.Join(gameDir, "instances")

	got, err := ResolveInstanceDir(gameDir, "My Pack", "1.20.1", "")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(got, root) || !strings.Contains(filepath.Base(got), "my-pack") {
		t.Fatalf("default dir = %q", got)
	}

	rel, err := ResolveInstanceDir(gameDir, "Name", "1.20.1", "custom-pack")
	if err != nil {
		t.Fatal(err)
	}
	if rel != filepath.Join(root, "custom-pack") {
		t.Fatalf("relative dir = %q", rel)
	}

	abs := filepath.Join(root, "absolute")
	if got, err := ResolveInstanceDir(gameDir, "", "", abs); err != nil || got != abs {
		t.Fatalf("absolute dir = %q, err = %v", got, err)
	}

	for _, bad := range []string{"../outside", "../../outside", "custom/pack", `custom\pack`, ".", "..", filepath.Join(t.TempDir(), "outside"), filepath.Join(root, "nested", "dir")} {
		if _, err := ResolveInstanceDir(gameDir, "Name", "1.20.1", bad); err == nil {
			t.Fatalf("expected rejection for %q", bad)
		}
	}
}

func TestWizardLoaderAllowed(t *testing.T) {
	for _, kind := range WizardLoaders {
		if !wizardLoaderAllowed(kind) {
			t.Fatalf("%q should be allowed", kind)
		}
	}
	for _, kind := range []string{"optifine", "cleanroom", "liteloader", "unknown"} {
		if wizardLoaderAllowed(kind) {
			t.Fatalf("%q should not be allowed", kind)
		}
	}
}

func TestCreateInstanceVanilla(t *testing.T) {
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
			"id": "1.20.1", "type": "release", "mainClass": "net.minecraft.client.main.Main",
			"javaVersion": map[string]any{"majorVersion": 17},
			"downloads": map[string]any{
				"client": map[string]any{"url": baseURL + "/client.jar", "size": 6},
			},
			"libraries": []any{},
		})
	})
	mux.HandleFunc("/client.jar", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("client"))
	})
	server := httptest.NewServer(mux)
	defer server.Close()
	baseURL = server.URL

	dir := t.TempDir()
	inst := NewInstaller(dir, Source{ID: "test", ManifestURL: baseURL + "/manifest.json"}, nil, nil)
	inst.HTTP.Client = server.Client()

	id, root, err := inst.CreateInstance(context.Background(), InstanceWizardOptions{
		Name:      "生存实例",
		MCVersion: "1.20.1",
		Loader:    "vanilla",
	})
	if err != nil {
		t.Fatal(err)
	}
	if id != "1.20.1" {
		t.Fatalf("id = %q", id)
	}
	if !strings.HasPrefix(root, filepath.Join(dir, "instances")) {
		t.Fatalf("root = %q", root)
	}
	for _, rel := range []string{
		"versions/1.20.1/1.20.1.json",
		"versions/1.20.1/1.20.1.jar",
		"versions/1.20.1/mctl-pack.json",
		"versions/1.20.1/mctl-settings.json",
	} {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(rel))); err != nil {
			t.Fatalf("missing %q: %v", rel, err)
		}
	}
	list, err := ScanVersions(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].ID != id || list[0].Name != "生存实例" || list[0].GameDir != root {
		t.Fatalf("scan = %#v", list)
	}

	if _, _, err := inst.CreateInstance(context.Background(), InstanceWizardOptions{
		Name: "生存实例", MCVersion: "1.20.1", Loader: "vanilla",
	}); err == nil {
		t.Fatal("expected duplicate directory error")
	}
}

func TestCreateInstanceRejectsBadOptions(t *testing.T) {
	inst := NewInstaller(t.TempDir(), Source{ID: "test"}, nil, nil)
	if _, _, err := inst.CreateInstance(context.Background(), InstanceWizardOptions{}); err == nil {
		t.Fatal("expected missing version error")
	}
	if _, _, err := inst.CreateInstance(context.Background(), InstanceWizardOptions{
		MCVersion: "1.20.1", Loader: "optifine",
	}); err == nil {
		t.Fatal("expected unsupported loader error")
	}
	if _, _, err := inst.CreateInstance(context.Background(), InstanceWizardOptions{
		MCVersion: "1.20.1", Dir: "../escape",
	}); err == nil {
		t.Fatal("expected directory escape error")
	}
}
