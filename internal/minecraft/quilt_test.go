package minecraft

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchQuiltLoadersAndProfile(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v3/versions/loader/1.20.1", func(w http.ResponseWriter, r *http.Request) {
		writeTestJSON(t, w, []any{
			map[string]any{"loader": map[string]string{"version": "0.20.0-beta.9"}},
			map[string]any{"version": "0.19.0"},
		})
	})
	mux.HandleFunc("/v3/versions/loader/1.20.1/0.19.0/profile/json", func(w http.ResponseWriter, r *http.Request) {
		writeTestJSON(t, w, map[string]any{
			"id":           "quilt-loader-0.19.0-1.20.1",
			"inheritsFrom": "1.20.1",
			"mainClass":    "org.quiltmc.loader.impl.launch.knot.KnotClient",
		})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	old := quiltMetaURL
	defer func() { quiltMetaURL = old }()
	quiltMetaURL = server.URL
	hc := NewHTTPClient(Source{ID: "test"})
	hc.Client = server.Client()

	builds, err := FetchQuiltLoaders(context.Background(), hc, "1.20.1")
	if err != nil {
		t.Fatal(err)
	}
	if len(builds) != 2 || builds[0].Version != "0.19.0" || !builds[0].Stable || builds[1].Stable {
		t.Fatalf("quilt builds = %#v", builds)
	}
	profile, err := FetchQuiltProfile(context.Background(), hc, "1.20.1", "0.19.0")
	if err != nil {
		t.Fatal(err)
	}
	if len(profile) == 0 {
		t.Fatal("empty profile")
	}
}
