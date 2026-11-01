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

func TestSelectModrinthVersionFilters(t *testing.T) {
	versions := []ModrinthVersion{
		{ID: "fabric-1", VersionType: "release", GameVersions: []string{"1.20.1"}, Loaders: []string{"fabric"}},
		{ID: "forge-1", VersionType: "release", GameVersions: []string{"1.20.1"}, Loaders: []string{"forge"}},
	}
	v, err := SelectModrinthVersion(versions, "1.20.1", "forge")
	if err != nil || v.ID != "forge-1" {
		t.Fatalf("v = %#v, err = %v", v, err)
	}
	if _, err := SelectModrinthVersion(versions, "1.19.4", "forge"); err == nil {
		t.Fatal("expected version mismatch error")
	}
}

func TestInstallResource(t *testing.T) {
	var baseURL string
	mux := http.NewServeMux()
	mux.HandleFunc("/v2/project/pack-id/version", func(w http.ResponseWriter, r *http.Request) {
		writeTestJSON(t, w, []any{
			map[string]any{
				"id": "v1", "project_id": "pack-id", "version_type": "release",
				"version_number": "1.0", "game_versions": []string{"1.20.1"},
				"files": []any{map[string]any{
					"url": baseURL + "/files/pack.zip", "filename": "pack.zip", "primary": true,
				}},
			},
		})
	})
	mux.HandleFunc("/files/pack.zip", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("pack"))
	})
	server := httptest.NewServer(mux)
	defer server.Close()
	baseURL = server.URL

	old := modrinthAPIBase
	defer func() { modrinthAPIBase = old }()
	modrinthAPIBase = server.URL + "/v2"
	hc := NewHTTPClient(Source{ID: "test"})
	hc.Client = server.Client()
	dir := t.TempDir()
	res, err := InstallResource(context.Background(), hc, "resourcepack", "pack-id", "", "", dir)
	if err != nil {
		t.Fatal(err)
	}
	if res.Path != filepath.Join(dir, "resourcepacks", "pack.zip") {
		t.Fatalf("path = %q", res.Path)
	}
	if data, err := os.ReadFile(res.Path); err != nil || string(data) != "pack" {
		t.Fatalf("file = %q, err = %v", data, err)
	}
}

func TestSearchModpacksFiltered(t *testing.T) {
	var facetsSeen []string
	mux := http.NewServeMux()
	mux.HandleFunc("/v2/search", func(w http.ResponseWriter, r *http.Request) {
		facetsSeen = append(facetsSeen, r.URL.Query().Get("facets"))
		writeTestJSON(t, w, map[string]any{"hits": []any{map[string]any{"project_id": "id", "title": "Pack"}}})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	old := modrinthAPIBase
	defer func() { modrinthAPIBase = old }()
	modrinthAPIBase = server.URL + "/v2"
	hc := NewHTTPClient(Source{ID: "test"})
	hc.Client = server.Client()

	projects, err := SearchModpacksFiltered(context.Background(), hc, "pack", ModpackFilter{GameVersion: "1.20.1", Loader: "fabric"}, 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(projects) != 1 || projects[0].ID != "id" {
		t.Fatalf("projects = %#v", projects)
	}
	plain, err := SearchModpacks(context.Background(), hc, "pack", 5)
	if err != nil || len(plain) != 1 {
		t.Fatalf("unfiltered search = %#v, err = %v", plain, err)
	}
	if len(facetsSeen) != 2 {
		t.Fatalf("facets calls = %#v", facetsSeen)
	}
	for _, want := range []string{"project_type:modpack", "versions:1.20.1", "categories:fabric"} {
		if !strings.Contains(facetsSeen[0], want) {
			t.Fatalf("filtered facets = %q, missing %q", facetsSeen[0], want)
		}
	}
	if strings.Contains(facetsSeen[1], "versions:") || strings.Contains(facetsSeen[1], "categories:") {
		t.Fatalf("unfiltered facets = %q", facetsSeen[1])
	}
}

func TestFetchRecentReleases(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/Owner/Repo/releases.atom", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/atom+xml")
		_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <entry><updated>2026-03-01T00:00:00Z</updated><link rel="alternate" href="https://example.invalid/releases/tag/v3"/><title>v3</title></entry>
  <entry><updated>2026-02-01T00:00:00Z</updated><link rel="alternate" href="https://example.invalid/releases/tag/v2"/><title>v2</title></entry>
  <entry><updated>2026-01-01T00:00:00Z</updated><link rel="alternate" href="https://example.invalid/releases/tag/v1"/><title>v1</title></entry>
</feed>`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	old := releasesFeedBase
	defer func() { releasesFeedBase = old }()
	releasesFeedBase = server.URL
	hc := NewHTTPClient(Source{ID: "test"})
	hc.Client = server.Client()

	releases, err := FetchRecentReleases(context.Background(), hc, "Owner", "Repo", 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(releases) != 2 || releases[0].TagName != "v3" || releases[1].TagName != "v2" {
		t.Fatalf("releases = %#v", releases)
	}
	all, err := FetchRecentReleases(context.Background(), hc, "Owner", "Repo", 0)
	if err != nil || len(all) != 3 {
		t.Fatalf("all = %#v, err = %v", all, err)
	}
}

func TestSearchModpacksAndLatestVersion(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v2/search", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("query") != "create" {
			t.Fatalf("query = %q", q.Get("query"))
		}
		if !strings.Contains(q.Get("facets"), "project_type:modpack") {
			t.Fatalf("facets = %q", q.Get("facets"))
		}
		writeTestJSON(t, w, map[string]any{
			"hits": []any{
				map[string]any{
					"project_id": "create-id", "slug": "create", "title": "Create",
					"description": "A tech modpack", "downloads": 1000,
				},
			},
		})
	})
	mux.HandleFunc("/v2/project/create-id/version", func(w http.ResponseWriter, r *http.Request) {
		writeTestJSON(t, w, []any{
			map[string]any{
				"id": "beta-id", "project_id": "create-id", "name": "beta",
				"version_number": "2.0-beta", "version_type": "beta",
				"files": []any{map[string]any{"url": "https://example.invalid/beta.mrpack", "filename": "beta.mrpack", "primary": true}},
			},
			map[string]any{
				"id": "release-id", "project_id": "create-id", "name": "release",
				"version_number": "1.0", "version_type": "release",
				"files": []any{
					map[string]any{"url": "https://example.invalid/extra.jar", "filename": "extra.jar", "primary": false},
					map[string]any{"url": "https://example.invalid/create.mrpack", "filename": "create.mrpack", "primary": true, "hashes": map[string]string{"sha1": "abc"}},
				},
			},
		})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	old := modrinthAPIBase
	defer func() { modrinthAPIBase = old }()
	modrinthAPIBase = server.URL + "/v2"
	hc := NewHTTPClient(Source{ID: "test"})
	hc.Client = server.Client()

	projects, err := SearchModpacks(context.Background(), hc, "create", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(projects) != 1 || projects[0].Title != "Create" {
		t.Fatalf("projects = %#v", projects)
	}
	v, err := FetchLatestModpackVersion(context.Background(), hc, "create-id")
	if err != nil {
		t.Fatal(err)
	}
	if v.ID != "release-id" {
		t.Fatalf("chosen version = %#v", v)
	}
	f, err := PrimaryModrinthFile(v)
	if err != nil {
		t.Fatal(err)
	}
	if f.Filename != "create.mrpack" || f.Hashes["sha1"] != "abc" {
		t.Fatalf("file = %#v", f)
	}
}
