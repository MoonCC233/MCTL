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

func TestSearchCurseForgeModpacks(t *testing.T) {
	var gotKey, gotClass, gotVersion, gotQuery string
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/mods/search", func(w http.ResponseWriter, r *http.Request) {
		gotKey = r.Header.Get("x-api-key")
		gotClass = r.URL.Query().Get("classId")
		gotVersion = r.URL.Query().Get("gameVersion")
		gotQuery = r.URL.Query().Get("searchFilter")
		writeTestJSON(t, w, map[string]any{
			"data": []any{map[string]any{
				"id": 42, "name": "Better MC", "slug": "better-mc",
				"summary": "A pack", "downloadCount": 1000, "classId": 4471,
				"authors": []any{map[string]any{"name": "Author"}},
				"latestFiles": []any{
					map[string]any{"id": 2, "displayName": "1.0", "fileName": "pack.zip", "releaseType": 1, "downloadUrl": "https://example.invalid/pack.zip", "fileLength": 10},
					map[string]any{"id": 3, "displayName": "beta", "fileName": "beta.zip", "releaseType": 2, "downloadUrl": "https://example.invalid/beta.zip"},
				},
			}},
		})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	oldBase := curseForgeAPIBase
	defer func() { curseForgeAPIBase = oldBase }()
	curseForgeAPIBase = server.URL + "/v1"
	hc := NewHTTPClient(Source{ID: "test"})
	hc.Client = server.Client()

	mods, err := SearchCurseForgeModpacks(context.Background(), hc, "test-key", "better", "1.20.1", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(mods) != 1 || mods[0].ID != 42 || mods[0].Name != "Better MC" {
		t.Fatalf("mods = %#v", mods)
	}
	if gotKey != "test-key" || gotClass != "4471" || gotVersion != "1.20.1" || gotQuery != "better" {
		t.Fatalf("request key=%q class=%q version=%q query=%q", gotKey, gotClass, gotVersion, gotQuery)
	}
	file, err := LatestCurseForgeFile(mods[0])
	if err != nil {
		t.Fatal(err)
	}
	if file.ID != 2 || file.ReleaseType != 1 {
		t.Fatalf("file = %#v", file)
	}

	if _, err := SearchCurseForgeModpacks(context.Background(), hc, "", "x", "", 5); err == nil {
		t.Fatal("missing API key should fail")
	}
}

func TestSearchCurseForgeRejectsBadKey(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/mods/search", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":"forbidden"}`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	oldBase := curseForgeAPIBase
	defer func() { curseForgeAPIBase = oldBase }()
	curseForgeAPIBase = server.URL + "/v1"
	hc := NewHTTPClient(Source{ID: "test"})
	hc.Client = server.Client()

	_, err := SearchCurseForgeModpacks(context.Background(), hc, "bad", "x", "", 5)
	if err == nil || !strings.Contains(err.Error(), "API Key") {
		t.Fatalf("err = %v", err)
	}
}

func TestLatestCurseForgeFileFallbacks(t *testing.T) {
	mod := CurseForgeMod{
		Name: "Pack",
		LatestFiles: []CurseForgeFile{
			{ID: 1, FileName: "no-url.zip"},
			{ID: 2, FileName: "beta.zip", ReleaseType: 2, DownloadURL: "https://example.invalid/beta.zip"},
			{ID: 3, FileName: "alpha.zip", ReleaseType: 3, DownloadURL: "https://example.invalid/alpha.zip"},
		},
	}
	file, err := LatestCurseForgeFile(mod)
	if err != nil {
		t.Fatal(err)
	}
	if file.ID != 2 {
		t.Fatalf("file = %#v", file)
	}
	if _, err := LatestCurseForgeFile(CurseForgeMod{Name: "Empty"}); err == nil {
		t.Fatal("expected no-file error")
	}
}

func TestDownloadCurseForgeModpack(t *testing.T) {
	var baseURL string
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/mods/42/files/2/download-url", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-api-key") != "test-key" {
			t.Fatalf("api key = %q", r.Header.Get("x-api-key"))
		}
		writeTestJSON(t, w, map[string]any{"data": baseURL + "/pack.zip"})
	})
	mux.HandleFunc("/pack.zip", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("zipdata"))
	})
	server := httptest.NewServer(mux)
	defer server.Close()
	baseURL = server.URL

	oldBase, oldClient := curseForgeAPIBase, curseForgeHTTPClient
	defer func() { curseForgeAPIBase, curseForgeHTTPClient = oldBase, oldClient }()
	curseForgeAPIBase = server.URL + "/v1"
	curseForgeHTTPClient = server.Client()

	hc := NewHTTPClient(Source{ID: "test"})
	hc.Client = server.Client()
	destDir := t.TempDir()
	path, file, err := DownloadCurseForgeModpack(context.Background(), hc, "test-key", CurseForgeMod{
		ID:   42,
		Name: "Pack",
		LatestFiles: []CurseForgeFile{
			{ID: 2, FileName: "pack.zip", ReleaseType: 1, DownloadURL: "https://example.invalid/pack.zip"},
		},
	}, destDir, nil)
	if err != nil {
		t.Fatal(err)
	}
	if file.ID != 2 || path != filepath.Join(destDir, "pack.zip") {
		t.Fatalf("path = %q, file = %#v", path, file)
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "zipdata" {
		t.Fatalf("data = %q, err = %v", data, err)
	}
}

func TestDownloadCurseForgeModpackNoFile(t *testing.T) {
	hc := NewHTTPClient(Source{ID: "test"})
	if _, _, err := DownloadCurseForgeModpack(context.Background(), hc, "key", CurseForgeMod{Name: "Empty"}, t.TempDir(), nil); err == nil {
		t.Fatal("expected missing file error")
	}
}
