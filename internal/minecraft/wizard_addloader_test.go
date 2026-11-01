package minecraft

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestAddLoaderValidation(t *testing.T) {
	dir := t.TempDir()
	inst := NewInstaller(dir, Source{ID: "test"}, nil, nil)

	if _, err := inst.AddLoader(context.Background(), "fabric", "", "", "", false); err == nil {
		t.Fatal("expected missing Minecraft version error")
	}
	if _, err := inst.AddLoader(context.Background(), "", "1.20.1", "", "", false); err == nil {
		t.Fatal("expected missing loader error")
	}
	if _, err := inst.AddLoader(context.Background(), "optifine", "1.20.1", "", "", false); err == nil {
		t.Fatal("expected unsupported loader error")
	}
	if _, err := inst.AddLoader(context.Background(), "fabric", "1.20.1", "", "", false); err == nil {
		t.Fatal("expected missing base version error")
	}
}

func TestAddLoaderUpdatesPackMarker(t *testing.T) {
	var baseURL string
	mux := http.NewServeMux()
	mux.HandleFunc("/v2/versions/loader/1.20.1", func(w http.ResponseWriter, r *http.Request) {
		writeTestJSON(t, w, []any{
			map[string]any{"loader": map[string]any{"version": "0.16.9", "stable": true}},
		})
	})
	mux.HandleFunc("/v2/versions/loader/1.20.1/0.16.9/profile/json", func(w http.ResponseWriter, r *http.Request) {
		writeTestJSON(t, w, map[string]any{
			"id":           "fabric-loader-0.16.9-1.20.1",
			"inheritsFrom": "1.20.1",
			"mainClass":    "net.fabricmc.loader.impl.launch.knot.KnotClient",
			"libraries":    []any{},
		})
	})
	server := httptest.NewServer(mux)
	defer server.Close()
	baseURL = server.URL

	oldMeta := FabricMeta
	defer func() { FabricMeta = oldMeta }()
	FabricMeta = baseURL

	root := filepath.Join(t.TempDir(), "instances", "pack")
	writeJSON(t, VersionJSONPath(root, "1.20.1"), map[string]any{
		"id":        "1.20.1",
		"mainClass": "net.minecraft.client.main.Main",
	})
	mustWrite(t, filepath.Join(root, "versions", "1.20.1", "1.20.1.jar"), "client")
	writeJSON(t, filepath.Join(root, "versions", "1.20.1", "mctl-pack.json"), map[string]any{
		"name": "Pack", "versionId": "1.20.1", "mcVersion": "1.20.1", "loader": "1.20.1", "source": "wizard",
	})

	inst := NewInstaller(root, Source{ID: "test"}, nil, nil)
	inst.HTTP.Client = server.Client()
	id, err := inst.AddLoader(context.Background(), "fabric", "1.20.1", "0.16.9", "", false)
	if err != nil {
		t.Fatal(err)
	}
	if id != "fabric-loader-0.16.9-1.20.1" {
		t.Fatalf("id = %q", id)
	}
	if _, err := os.Stat(VersionJSONPath(root, id)); err != nil {
		t.Fatalf("loader profile missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "versions", "1.20.1", "mctl-pack.json")); !os.IsNotExist(err) {
		t.Fatalf("stale marker should be removed: %v", err)
	}
	marker, oldID, err := findPackMarker(root)
	if err != nil || marker == nil {
		t.Fatalf("marker = %#v, err = %v", marker, err)
	}
	if oldID != id || marker.VersionID != id || marker.MCVersion != "1.20.1" || marker.Name != "Pack" {
		t.Fatalf("marker = %#v (%s)", marker, oldID)
	}
	if _, err := ResolveVersion(root, id); err != nil {
		t.Fatal(err)
	}

	list, err := ScanVersions(filepath.Dir(filepath.Dir(root)))
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].ID != id || list[0].Name != "Pack" {
		t.Fatalf("scan = %#v", list)
	}
}
