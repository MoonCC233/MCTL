package minecraft

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

func TestDiagnoseReportsEnvironment(t *testing.T) {
	dir := t.TempDir()
	writeJSON(t, VersionJSONPath(dir, "1.20.1"), map[string]any{
		"id":        "1.20.1",
		"mainClass": "net.minecraft.client.main.Main",
	})
	mustWrite(t, filepath.Join(dir, "versions", "1.20.1", "1.20.1.jar"), "client")

	checks := Diagnose(context.Background(), nil, DiagnoseOptions{GameDir: dir, VersionID: "1.20.1"})
	byName := make(map[string]DiagnoseCheck, len(checks))
	for _, c := range checks {
		byName[c.Name] = c
	}
	if len(checks) < 4 {
		t.Fatalf("checks = %#v", checks)
	}
	if c := byName["游戏目录"]; c.Status != DiagnoseOK {
		t.Fatalf("game dir check = %#v", c)
	}
	if c := byName["游戏版本"]; c.Status != DiagnoseOK || c.Detail == "" {
		t.Fatalf("version check = %#v", c)
	}
	if c := byName["资源文件"]; c.Status != DiagnoseWarn {
		t.Fatalf("assets check = %#v", c)
	}
	if _, ok := byName["下载源"]; ok {
		t.Fatal("nil client should skip the download source check")
	}
}

func TestDiagnoseMissingVersion(t *testing.T) {
	dir := t.TempDir()
	checks := Diagnose(context.Background(), nil, DiagnoseOptions{GameDir: dir})
	byName := make(map[string]DiagnoseCheck, len(checks))
	for _, c := range checks {
		byName[c.Name] = c
	}
	if c := byName["游戏版本"]; c.Status != DiagnoseWarn {
		t.Fatalf("version check = %#v", c)
	}
}

func TestDiagnoseDownloadSource(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/manifest.json", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"versions":[]}`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	hc := NewHTTPClient(Source{ID: "test", Name: "测试源", ManifestURL: server.URL + "/manifest.json"})
	hc.Client = server.Client()
	checks := Diagnose(context.Background(), hc, DiagnoseOptions{GameDir: t.TempDir()})
	var found *DiagnoseCheck
	for i := range checks {
		if checks[i].Name == "下载源" {
			found = &checks[i]
		}
	}
	if found == nil || found.Status != DiagnoseOK || found.Detail == "" {
		t.Fatalf("download source check = %#v", found)
	}
}

func TestFetchLatestRelease(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/MoonCC233/MCTL/releases.atom", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") == "" {
			t.Fatal("missing user agent")
		}
		w.Header().Set("Content-Type", "application/atom+xml")
		_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <entry>
    <updated>2026-01-01T00:00:00Z</updated>
    <link rel="alternate" type="text/html" href="https://example.invalid/releases/tag/v1.0.17"/>
    <title>MCTL v1.0.17</title>
    <content type="html">notes</content>
  </entry>
  <entry>
    <updated>2025-12-01T00:00:00Z</updated>
    <link rel="alternate" type="text/html" href="https://example.invalid/releases/tag/v1.0.16"/>
    <title>MCTL v1.0.16</title>
  </entry>
</feed>`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	old := releasesFeedBase
	defer func() { releasesFeedBase = old }()
	releasesFeedBase = server.URL
	hc := NewHTTPClient(Source{ID: "test"})
	hc.Client = server.Client()

	info, err := FetchLatestRelease(context.Background(), hc, "MoonCC233", "MCTL")
	if err != nil {
		t.Fatal(err)
	}
	if info.TagName != "v1.0.17" || info.HTMLURL != "https://example.invalid/releases/tag/v1.0.17" {
		t.Fatalf("info = %#v", info)
	}
	if !IsNewerVersion(info.TagName, "1.0.16") {
		t.Fatal("v1.0.17 should be newer than 1.0.16")
	}
	if IsNewerVersion("v1.0.16", "1.0.16") {
		t.Fatal("equal versions are not newer")
	}
	if IsNewerVersion("v1.0.2", "1.0.16") {
		t.Fatal("v1.0.2 is older than 1.0.16")
	}
}
