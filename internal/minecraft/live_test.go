//go:build live

package minecraft

import (
	"context"
	"testing"
	"time"
)

// TestLiveLoaderMetadata intentionally talks to the official loader endpoints.
// Run it with: go test -tags live -run TestLiveLoaderMetadata -v ./internal/minecraft
func TestLiveLoaderMetadata(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	hc := NewHTTPClient(GetSource("bmclapi"))
	hc.Client.Timeout = 30 * time.Second
	hc.Retries = 1

	cases := []struct {
		name string
		kind string
		mc   string
	}{
		{"Forge", "forge", "1.20.1"},
		{"NeoForge", "neoforge", "1.21.1"},
		{"Quilt", "quilt", "1.20.1"},
		{"LegacyFabric", "legacyfabric", "1.8.9"},
		{"LiteLoader", "liteloader", "1.12.2"},
		{"Cleanroom", "cleanroom", "1.12.2"},
		{"LabyMod", "labymod", "1.20.1"},
		{"OptiFine", "optifine", "1.20.1"},
	}
	t.Run("ModrinthSearch", func(t *testing.T) {
		projects, err := SearchModpacks(ctx, hc, "create", 3)
		if err != nil {
			t.Fatal(err)
		}
		if len(projects) == 0 {
			t.Fatal("Modrinth returned no modpacks")
		}
		t.Logf("Modrinth: %d projects, first=%q", len(projects), projects[0].Title)
		version, err := FetchLatestModpackVersion(ctx, hc, projects[0].ID)
		if err != nil {
			t.Fatal(err)
		}
		file, err := PrimaryModrinthFile(version)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("Modrinth latest: %s (%s)", version.VersionNumber, file.Filename)
	})

	t.Run("ReleaseFeed", func(t *testing.T) {
		info, err := FetchLatestRelease(ctx, hc, "MoonCC233", "MCTL")
		if err != nil {
			t.Fatal(err)
		}
		if info.TagName == "" || info.HTMLURL == "" {
			t.Fatalf("release = %#v", info)
		}
		t.Logf("Release feed: %s (%s)", info.TagName, info.HTMLURL)
	})

	t.Run("SpeedTest", func(t *testing.T) {
		sctx, cancel := context.WithTimeout(ctx, 75*time.Second)
		defer cancel()
		results := SpeedTest(sctx, hc, DefaultSpeedTargets()[:3], 64<<10)
		ok := 0
		for _, res := range results {
			if res.Err != nil {
				t.Logf("speed %s: %v", res.Target.Name, res.Err)
				continue
			}
			ok++
			t.Logf("speed %s: %d ms, %.2f MB/s", res.Target.Name, res.Latency.Milliseconds(), res.MBps)
		}
		if ok < 2 {
			t.Fatalf("only %d speed targets responded", ok)
		}
	})

	t.Run("NewsFeed", func(t *testing.T) {
		releases, err := FetchRecentReleases(ctx, hc, "MoonCC233", "MCTL", 5)
		if err != nil {
			t.Fatal(err)
		}
		if len(releases) == 0 || releases[0].TagName == "" {
			t.Fatalf("releases = %#v", releases)
		}
		t.Logf("News feed: %d releases, latest=%s", len(releases), releases[0].TagName)
	})

	t.Run("ModrinthFiltered", func(t *testing.T) {
		projects, err := SearchModpacksFiltered(ctx, hc, "create", ModpackFilter{GameVersion: "1.20.1", Loader: "fabric"}, 5)
		if err != nil {
			t.Fatal(err)
		}
		if len(projects) == 0 {
			t.Fatal("filtered Modrinth search returned no modpacks")
		}
		t.Logf("Modrinth filtered: %d projects, first=%q", len(projects), projects[0].Title)
	})

	t.Run("ModrinthResourcePack", func(t *testing.T) {
		projects, err := SearchProjects(ctx, hc, "faithful", "resourcepack", 3)
		if err != nil {
			t.Fatal(err)
		}
		if len(projects) == 0 {
			t.Fatal("Modrinth returned no resource packs")
		}
		versions, err := FetchProjectVersions(ctx, hc, projects[0].ID)
		if err != nil {
			t.Fatal(err)
		}
		version, err := SelectModrinthVersion(versions, "", "")
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("Modrinth resource pack: %q, version=%s", projects[0].Title, version.VersionNumber)
	})

	t.Run("LabyModProfile", func(t *testing.T) {
		builds, err := FetchLabyModBuilds(ctx, hc, "1.20.1")
		if err != nil {
			t.Fatal(err)
		}
		if len(builds) == 0 || builds[0].ManifestURL == "" {
			t.Fatal("no LabyMod profile")
		}
		if _, err := hc.Get(ctx, builds[0].ManifestURL); err != nil {
			t.Fatal(err)
		}
		libs, err := FetchLabyModLibraries(ctx, hc, builds[0].Channel)
		if err != nil {
			t.Fatal(err)
		}
		if len(libs) == 0 {
			t.Fatal("no LabyMod libraries")
		}
		t.Logf("LabyMod profile: %s, libraries=%d", builds[0].Version, len(libs))
	})

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var builds []LoaderBuild
			var err error
			switch tc.kind {
			case "quilt":
				builds, err = FetchQuiltLoaders(ctx, hc, tc.mc)
			case "legacyfabric":
				builds, err = FetchLegacyFabricLoaders(ctx, hc, tc.mc)
			case "optifine":
				builds, err = FetchOptiFineBuilds(ctx, hc, tc.mc)
			case "liteloader":
				builds, err = FetchLiteLoaderBuilds(ctx, hc, tc.mc)
			case "cleanroom":
				builds, err = FetchCleanroomBuilds(ctx, hc, tc.mc)
			case "labymod":
				builds, err = FetchLabyModBuilds(ctx, hc, tc.mc)
			default:
				builds, err = FetchLoaderBuilds(ctx, hc, tc.kind, tc.mc)
			}
			if err != nil {
				t.Fatalf("%s: %v", tc.name, err)
			}
			if len(builds) == 0 {
				t.Fatalf("%s returned no builds", tc.name)
			}
			t.Logf("%s: %d builds, first=%q stable=%v", tc.name, len(builds), builds[0].Version, builds[0].Stable)
		})
	}
}
