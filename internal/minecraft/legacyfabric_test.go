package minecraft

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchLegacyFabricLoadersAndProfile(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v2/versions/loader/1.8.9", func(w http.ResponseWriter, r *http.Request) {
		writeTestJSON(t, w, []any{
			map[string]any{"loader": map[string]any{"version": "0.16.9", "stable": true}},
			map[string]any{"version": "0.16.8"},
		})
	})
	mux.HandleFunc("/v2/versions/loader/1.8.9/0.16.9/profile/json", func(w http.ResponseWriter, r *http.Request) {
		writeTestJSON(t, w, map[string]any{
			"id":           "legacy-fabric-loader-0.16.9-1.8.9",
			"inheritsFrom": "1.8.9",
			"mainClass":    "net.fabricmc.loader.impl.launch.knot.KnotClient",
		})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	old := legacyFabricMetaURL
	defer func() { legacyFabricMetaURL = old }()
	legacyFabricMetaURL = server.URL
	hc := NewHTTPClient(Source{ID: "test"})
	hc.Client = server.Client()

	builds, err := FetchLegacyFabricLoaders(context.Background(), hc, "1.8.9")
	if err != nil {
		t.Fatal(err)
	}
	if len(builds) != 2 || builds[0].Version != "0.16.9" || !builds[0].Stable {
		t.Fatalf("builds = %#v", builds)
	}
	profile, err := FetchLegacyFabricProfile(context.Background(), hc, "1.8.9", "0.16.9")
	if err != nil {
		t.Fatal(err)
	}
	if len(profile) == 0 {
		t.Fatal("empty profile")
	}
}
