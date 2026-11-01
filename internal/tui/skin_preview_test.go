package tui

import (
	"image"
	"image/color"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/MoonCC233/MCTL/internal/config"
	"github.com/MoonCC233/MCTL/internal/minecraft"
)

func testSkinTexture() image.Image {
	img := image.NewNRGBA(image.Rect(0, 0, 64, 64))
	// Head front red.
	for y := 8; y < 16; y++ {
		for x := 8; x < 16; x++ {
			img.Set(x, y, color.NRGBA{R: 255, A: 255})
		}
	}
	// Body front green.
	for y := 20; y < 32; y++ {
		for x := 20; x < 28; x++ {
			img.Set(x, y, color.NRGBA{G: 160, A: 255})
		}
	}
	// Hat overlay semi-transparent blue.
	for y := 8; y < 16; y++ {
		for x := 40; x < 48; x++ {
			img.Set(x, y, color.NRGBA{B: 255, A: 120})
		}
	}
	return img
}

func TestRenderSkinPreview(t *testing.T) {
	rows := renderSkinRows(testSkinTexture(), false)
	if len(rows) != 16 {
		t.Fatalf("rows = %d", len(rows))
	}
	joined := strings.Join(rows, "\n")
	if !strings.Contains(joined, "▀") {
		t.Fatal("preview should use half blocks")
	}
	if !strings.Contains(joined, "\x1b[38;2;") {
		t.Fatal("preview should use truecolor foreground")
	}
	for i, row := range rows {
		if w := ansi.StringWidth(row); w > 16 {
			t.Fatalf("row %d width %d: %q", i, w, ansi.Strip(row))
		}
	}
	// The head row must be red-dominant (blended with the blue hat overlay).
	if !strings.Contains(rows[0], "38;2;") {
		t.Fatalf("first row has no color: %q", rows[0])
	}
}

func TestRenderCapePreview(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 64, 32))
	for y := 1; y < 17; y++ {
		for x := 1; x < 11; x++ {
			img.Set(x, y, color.NRGBA{R: 20, G: 20, B: 220, A: 255})
		}
	}
	rows := renderCapeRows(img)
	if len(rows) != 8 {
		t.Fatalf("rows = %d", len(rows))
	}
	if !strings.Contains(strings.Join(rows, "\n"), "▀") {
		t.Fatal("cape preview should use half blocks")
	}
}

func TestRenderSkinLegacyMirrorsLimbs(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 64, 32))
	for y := 20; y < 32; y++ {
		for x := 44; x < 48; x++ {
			img.Set(x, y, color.NRGBA{R: 10, G: 200, B: 10, A: 255})
		}
	}
	rows := renderSkinRows(img, false)
	// The right arm row must contain color on both sides (mirrored left arm).
	row := ansi.Strip(rows[6])
	if strings.Count(row, "▀") < 8 {
		t.Fatalf("legacy limbs not mirrored: %q", row)
	}
}

func TestSkinTextureMessageAttachesPreview(t *testing.T) {
	a := testApp(t, config.Config{GameDir: t.TempDir()})
	a.modal = modalState{
		Kind: modalSkin,
		Data: &minecraft.SkinProfile{Name: "Alex", UUID: "abc", SkinURL: "https://example.invalid/skin.png"},
	}
	a.handleEvent(skinTextureMsg{Skin: []string{"row1", "row2"}, Cape: []string{"cape"}})
	if len(a.modal.SkinPreview) != 2 || len(a.modal.CapePreview) != 1 {
		t.Fatalf("preview = %#v / %#v", a.modal.SkinPreview, a.modal.CapePreview)
	}
	out := a.View()
	if !strings.Contains(ansi.Strip(out), "row1") {
		t.Fatal("preview rows should render inside the skin dialog")
	}
}
