package minecraft

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

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
