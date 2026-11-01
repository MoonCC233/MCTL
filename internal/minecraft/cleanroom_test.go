package minecraft

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchCleanroomBuilds(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/cleanroom.xml", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<metadata><versioning><versions>
			<version>0.7.13-alpha</version>
			<version>0.7.14-alpha</version>
		</versions></versioning></metadata>`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	old := cleanroomMetadataURL
	defer func() { cleanroomMetadataURL = old }()
	cleanroomMetadataURL = server.URL + "/cleanroom.xml"
	hc := NewHTTPClient(Source{ID: "test"})
	hc.Client = server.Client()

	builds, err := FetchCleanroomBuilds(context.Background(), hc, "1.12.2")
	if err != nil {
		t.Fatal(err)
	}
	if len(builds) != 2 || builds[0].Version != "0.7.14-alpha" || builds[0].Kind != "cleanroom" {
		t.Fatalf("builds = %#v", builds)
	}
	if got := builds[0].InstallerURL(); got != "https://repo.cleanroommc.com/releases/com/cleanroommc/cleanroom/0.7.14-alpha/cleanroom-0.7.14-alpha-installer.jar" {
		t.Fatalf("installer url = %q", got)
	}
	if other, err := FetchCleanroomBuilds(context.Background(), hc, "1.20.1"); err != nil || len(other) != 0 {
		t.Fatalf("non-1.12.2 builds = %#v, err = %v", other, err)
	}
}

func TestFindCleanroomVersion(t *testing.T) {
	dir := t.TempDir()
	writeJSON(t, VersionJSONPath(dir, "1.12.2-cleanroom-0.7.14-alpha"), map[string]any{
		"id":           "1.12.2-cleanroom-0.7.14-alpha",
		"inheritsFrom": "1.12.2",
	})
	writeJSON(t, VersionJSONPath(dir, "1.12.2"), map[string]any{
		"id":        "1.12.2",
		"mainClass": "net.minecraft.client.main.Main",
	})
	id, err := FindLoaderVersion(dir, "cleanroom", "1.12.2", "0.7.14-alpha")
	if err != nil {
		t.Fatal(err)
	}
	if id != "1.12.2-cleanroom-0.7.14-alpha" {
		t.Fatalf("id = %q", id)
	}
}
