package minecraft

import (
	"archive/zip"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestFetchOptiFineBuilds(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/optifine/1.20.1", func(w http.ResponseWriter, r *http.Request) {
		writeTestJSON(t, w, []any{
			map[string]any{"mcversion": "1.20.1", "type": "HD_U", "patch": "I5", "filename": "OptiFine_1.20.1_HD_U_I5.jar"},
			map[string]any{"mcversion": "1.20.1", "type": "HD_U_I6", "patch": "pre2", "filename": "preview_OptiFine_1.20.1_HD_U_I6_pre2.jar"},
			map[string]any{"mcversion": "1.20.1", "type": "HD_U", "patch": "I6", "filename": "OptiFine_1.20.1_HD_U_I6.jar"},
		})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	old := optiFineAPI
	defer func() { optiFineAPI = old }()
	optiFineAPI = server.URL + "/optifine"
	hc := NewHTTPClient(Source{ID: "test"})
	hc.Client = server.Client()

	builds, err := FetchOptiFineBuilds(context.Background(), hc, "1.20.1")
	if err != nil {
		t.Fatal(err)
	}
	if len(builds) != 3 || !builds[0].Stable || builds[0].Filename != "OptiFine_1.20.1_HD_U_I6.jar" {
		t.Fatalf("builds = %#v", builds)
	}
	if builds[2].Stable {
		t.Fatalf("preview should be unstable: %#v", builds[2])
	}
}

func TestInstallOptiFine(t *testing.T) {
	dir := t.TempDir()
	writeJSON(t, VersionJSONPath(dir, "1.20.1"), map[string]any{
		"id":        "1.20.1",
		"mainClass": "net.minecraft.client.main.Main",
	})
	profile := map[string]any{
		"id":           "1.20.1-OptiFine_HD_U_I6",
		"inheritsFrom": "1.20.1",
		"mainClass":    "net.minecraft.launchwrapper.Launch",
		"libraries": []any{
			map[string]any{"name": "optifine:OptiFine:1.20.1_HD_U_I6"},
		},
	}
	var jar bytesBuffer
	zw := zip.NewWriter(&jar)
	w, _ := zw.Create("version.json")
	data, _ := json.Marshal(profile)
	_, _ = w.Write(data)
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/maven/com/optifine/1.20.1/OptiFine_1.20.1_HD_U_I6.jar", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(jar.Bytes())
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	oldBase := optiFineMavenBase
	defer func() { optiFineMavenBase = oldBase }()
	optiFineMavenBase = server.URL + "/maven/com/optifine"
	inst := NewInstaller(dir, Source{ID: "test"}, nil, nil)
	inst.HTTP.Client = server.Client()

	id, err := inst.InstallOptiFine(context.Background(), "1.20.1", "OptiFine_1.20.1_HD_U_I6.jar")
	if err != nil {
		t.Fatal(err)
	}
	if id != "1.20.1-OptiFine_HD_U_I6" {
		t.Fatalf("id = %q", id)
	}
	if _, err := os.Stat(VersionJSONPath(dir, id)); err != nil {
		t.Fatalf("version json missing: %v", err)
	}
	lib := filepath.Join(dir, "libraries", "optifine", "OptiFine", "1.20.1_HD_U_I6", "OptiFine-1.20.1_HD_U_I6.jar")
	if st, err := os.Stat(lib); err != nil || st.Size() == 0 {
		t.Fatalf("optifine library missing: %v", err)
	}
	rv, err := ResolveVersion(dir, id)
	if err != nil {
		t.Fatal(err)
	}
	if rv.InheritsFrom != "1.20.1" {
		t.Fatalf("inherits = %q", rv.InheritsFrom)
	}
}

type bytesBuffer struct {
	b []byte
}

func (b *bytesBuffer) Write(p []byte) (int, error) {
	b.b = append(b.b, p...)
	return len(p), nil
}

func (b *bytesBuffer) Bytes() []byte { return b.b }
