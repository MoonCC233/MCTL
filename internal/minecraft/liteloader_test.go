package minecraft

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestFetchAndInstallLiteLoader(t *testing.T) {
	var baseURL string
	mux := http.NewServeMux()
	mux.HandleFunc("/versions.json", func(w http.ResponseWriter, r *http.Request) {
		writeTestJSON(t, w, map[string]any{
			"versions": map[string]any{
				"1.12.2": map[string]any{
					"repo": map[string]any{"url": baseURL + "/repo/", "stream": "RELEASE"},
					"artefacts": map[string]any{
						"com.mumfrey:liteloader": map[string]any{
							"latest": map[string]any{
								"tweakClass": "com.mumfrey.liteloader.launch.LiteLoaderTweaker",
								"stream":     "RELEASE",
								"file":       "liteloader-1.12.2.jar",
								"version":    "1.12.2",
								"libraries":  []any{},
							},
						},
					},
				},
			},
		})
	})
	mux.HandleFunc("/repo/com/mumfrey/liteloader/1.12.2/liteloader-1.12.2.jar", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("lite"))
	})
	server := httptest.NewServer(mux)
	defer server.Close()
	baseURL = server.URL

	old := liteLoaderCatalogURL
	defer func() { liteLoaderCatalogURL = old }()
	liteLoaderCatalogURL = server.URL + "/versions.json"
	hc := NewHTTPClient(Source{ID: "test"})
	hc.Client = server.Client()

	builds, err := FetchLiteLoaderBuilds(context.Background(), hc, "1.12.2")
	if err != nil {
		t.Fatal(err)
	}
	if len(builds) != 1 || builds[0].Version != "1.12.2" || !builds[0].Stable {
		t.Fatalf("builds = %#v", builds)
	}

	dir := t.TempDir()
	writeJSON(t, VersionJSONPath(dir, "1.12.2"), map[string]any{
		"id":        "1.12.2",
		"mainClass": "net.minecraft.client.main.Main",
	})
	inst := NewInstaller(dir, Source{ID: "test"}, nil, nil)
	inst.HTTP.Client = server.Client()
	id, err := inst.InstallLiteLoader(context.Background(), "1.12.2", "1.12.2")
	if err != nil {
		t.Fatal(err)
	}
	if id != "1.12.2-LiteLoader-1.12.2" {
		t.Fatalf("id = %q", id)
	}
	lib := filepath.Join(dir, "libraries", "com", "mumfrey", "liteloader", "1.12.2", "liteloader-1.12.2.jar")
	if st, err := os.Stat(lib); err != nil || st.Size() == 0 {
		t.Fatalf("liteloader jar missing: %v", err)
	}
	rv, err := ResolveVersion(dir, id)
	if err != nil {
		t.Fatal(err)
	}
	if rv.MainClass != "net.minecraft.launchwrapper.Launch" {
		t.Fatalf("main class = %q", rv.MainClass)
	}
	args := rv.GameArguments(Features(0, 0))
	found := false
	for i := 0; i+1 < len(args); i++ {
		if args[i] == "--tweakClass" && args[i+1] == "com.mumfrey.liteloader.launch.LiteLoaderTweaker" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("tweak class missing from args: %#v", args)
	}
}
