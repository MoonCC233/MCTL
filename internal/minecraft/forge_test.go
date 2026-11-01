package minecraft

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNeoForgeMCVersion(t *testing.T) {
	cases := map[string]string{
		"21.1.80":  "1.21.1",
		"21.0.167": "1.21",
		"20.4.237": "1.20.4",
		"20.2.86":  "1.20.2",
		"19.4.0":   "",
		"bad":      "",
	}
	for in, want := range cases {
		if got := NeoForgeMCVersion(in); got != want {
			t.Fatalf("NeoForgeMCVersion(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestParseMavenMetadata(t *testing.T) {
	data := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<metadata>
  <groupId>net.minecraftforge</groupId>
  <versioning>
    <versions>
      <version>1.20.1-47.3.0</version>
      <version>1.20.1-47.2.0</version>
    </versions>
  </versioning>
</metadata>`)
	versions, err := parseMavenMetadata(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(versions) != 2 || versions[0] != "1.20.1-47.3.0" {
		t.Fatalf("versions = %#v", versions)
	}
}

func TestLoaderInstallerURL(t *testing.T) {
	forge := LoaderBuild{Kind: "forge", Version: "47.3.0", MCVersion: "1.20.1"}
	if got, want := forge.InstallerURL(), "https://maven.minecraftforge.net/net/minecraftforge/forge/1.20.1-47.3.0/forge-1.20.1-47.3.0-installer.jar"; got != want {
		t.Fatalf("forge url = %q, want %q", got, want)
	}
	neo := LoaderBuild{Kind: "neoforge", Version: "21.1.80", MCVersion: "1.21.1"}
	if got, want := neo.InstallerURL(), "https://maven.neoforged.net/releases/net/neoforged/neoforge/21.1.80/neoforge-21.1.80-installer.jar"; got != want {
		t.Fatalf("neoforge url = %q, want %q", got, want)
	}
}

func TestFindLoaderVersion(t *testing.T) {
	dir := t.TempDir()
	writeJSON(t, VersionJSONPath(dir, "1.20.1-forge-47.3.0"), map[string]any{
		"id":           "1.20.1-forge-47.3.0",
		"inheritsFrom": "1.20.1",
	})
	writeJSON(t, VersionJSONPath(dir, "1.20.1"), map[string]any{
		"id":        "1.20.1",
		"mainClass": "net.minecraft.client.main.Main",
	})
	id, err := FindLoaderVersion(dir, "forge", "1.20.1", "47.3.0")
	if err != nil {
		t.Fatal(err)
	}
	if id != "1.20.1-forge-47.3.0" {
		t.Fatalf("id = %q", id)
	}
	if _, err := FindLoaderVersion(dir, "neoforge", "1.21.1", "21.1.80"); err == nil {
		t.Fatal("expected missing NeoForge version error")
	}
}

func TestFetchLoaderBuilds(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/forge.xml", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<metadata><versioning><versions>
			<version>1.20.1-47.2.0</version>
			<version>1.20.1-47.4.0-beta</version>
			<version>1.20.1-47.3.0</version>
			<version>1.19.4-45.0.0</version>
		</versions></versioning></metadata>`))
	})
	mux.HandleFunc("/neoforge.xml", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<metadata><versioning><versions>
			<version>21.1.79</version>
			<version>21.1.80</version>
			<version>20.4.237</version>
		</versions></versioning></metadata>`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	oldForge, oldNeo := forgeMetadataURL, neoforgeMetadataURL
	defer func() { forgeMetadataURL, neoforgeMetadataURL = oldForge, oldNeo }()
	forgeMetadataURL = server.URL + "/forge.xml"
	neoforgeMetadataURL = server.URL + "/neoforge.xml"

	hc := NewHTTPClient(Source{ID: "test"})
	hc.Client = server.Client()
	forge, err := FetchLoaderBuilds(context.Background(), hc, "forge", "1.20.1")
	if err != nil {
		t.Fatal(err)
	}
	if len(forge) != 3 || forge[0].Version != "47.3.0" || forge[1].Version != "47.2.0" || forge[2].Stable {
		t.Fatalf("forge builds = %#v", forge)
	}
	neo, err := FetchLoaderBuilds(context.Background(), hc, "neoforge", "1.21.1")
	if err != nil {
		t.Fatal(err)
	}
	if len(neo) != 2 || neo[0].Version != "21.1.80" || neo[1].Version != "21.1.79" {
		t.Fatalf("neoforge builds = %#v", neo)
	}
}

func TestSortLoaderBuildsPrefersStable(t *testing.T) {
	list := []LoaderBuild{
		{Version: "47.4.0-beta", Stable: false},
		{Version: "47.3.0", Stable: true},
		{Version: "47.2.0", Stable: true},
	}
	sortLoaderBuilds(list)
	if list[0].Version != "47.3.0" || list[1].Version != "47.2.0" {
		t.Fatalf("unexpected order: %#v", list)
	}
}
