//go:build live

package tui

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"

	"github.com/MoonCC233/MCTL/internal/minecraft"
)

// TestLiveSkinPreview renders a real texture so the preview pipeline is
// exercised end to end against Mojang's texture CDN.
func TestLiveSkinPreview(t *testing.T) {
	if os.Getenv("MCTL_SKIP_LIVE") != "" {
		t.Skip("MCTL_SKIP_LIVE set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	hc := minecraft.NewHTTPClient(minecraft.GetSource("bmclapi"))
	profile, err := minecraft.FetchSkinProfile(ctx, hc, "069a79f444e94726a5befca90e38aaf5")
	if err != nil {
		t.Fatal(err)
	}
	img, err := minecraft.FetchTextureImage(ctx, hc, profile.SkinURL)
	if err != nil {
		t.Fatal(err)
	}
	rows := renderSkinRows(img, profile.Slim)
	if len(rows) != 16 {
		t.Fatalf("rows = %d", len(rows))
	}
	t.Logf("skin: %s texture=%dx%d", profile.Name, img.Bounds().Dx(), img.Bounds().Dy())
	for _, row := range rows {
		t.Logf("%s", ansi.Strip(row))
	}
}
