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
		{"OptiFine", "optifine", "1.20.1"},
	}
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
