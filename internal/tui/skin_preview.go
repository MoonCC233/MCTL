package tui

import (
	"fmt"
	"image"
	"image/color"
	"strings"
)

type texPart struct {
	dstX, dstY int
	w, h       int
	srcX, srcY int
}

// skinParts describes the front view of a Minecraft skin: arms, body, head and
// legs on a 16x32 canvas.
func skinParts(slim bool) (base, overlay []texPart) {
	armW := 4
	armRightX, armLeftX := 0, 12
	if slim {
		armW = 3
		armRightX, armLeftX = 1, 12
	}
	base = []texPart{
		{dstX: 4, dstY: 0, w: 8, h: 8, srcX: 8, srcY: 8},    // head
		{dstX: 4, dstY: 8, w: 8, h: 12, srcX: 20, srcY: 20}, // body
		{dstX: armRightX, dstY: 8, w: armW, h: 12, srcX: 44, srcY: 20},
		{dstX: armLeftX, dstY: 8, w: armW, h: 12, srcX: 36, srcY: 52},
		{dstX: 4, dstY: 20, w: 4, h: 12, srcX: 4, srcY: 20},  // right leg
		{dstX: 8, dstY: 20, w: 4, h: 12, srcX: 20, srcY: 52}, // left leg
	}
	overlay = []texPart{
		{dstX: 4, dstY: 0, w: 8, h: 8, srcX: 40, srcY: 8},   // hat
		{dstX: 4, dstY: 8, w: 8, h: 12, srcX: 20, srcY: 36}, // jacket
		{dstX: armRightX, dstY: 8, w: armW, h: 12, srcX: 44, srcY: 36},
		{dstX: armLeftX, dstY: 8, w: armW, h: 12, srcX: 52, srcY: 52},
		{dstX: 4, dstY: 20, w: 4, h: 12, srcX: 4, srcY: 36},
		{dstX: 8, dstY: 20, w: 4, h: 12, srcX: 4, srcY: 52},
	}
	return base, overlay
}

// renderSkinRows composes a front-view skin preview as ANSI half-block rows.
func renderSkinRows(img image.Image, slim bool) []string {
	const width, height = 16, 32
	canvas := make([]color.NRGBA, width*height)
	base, overlay := skinParts(slim)
	if img.Bounds().Dy() < 64 {
		// Legacy 64x32 skins have no left-side limbs or body overlays; mirror
		// the right-side regions so the preview stays complete.
		base[3].srcX, base[3].srcY = base[2].srcX, base[2].srcY
		base[5].srcX, base[5].srcY = base[4].srcX, base[4].srcY
		overlay = overlay[:1]
	}
	for _, p := range base {
		blitRegion(img, canvas, width, height, p)
	}
	for _, p := range overlay {
		blitRegion(img, canvas, width, height, p)
	}
	return halfBlockRows(canvas, width, height)
}

// renderCapeRows renders the cape face stored at (1,1) in a 64x32 cape texture.
func renderCapeRows(img image.Image) []string {
	const width, height = 10, 16
	canvas := make([]color.NRGBA, width*height)
	blitRegion(img, canvas, width, height, texPart{w: 10, h: 16, srcX: 1, srcY: 1})
	return halfBlockRows(canvas, width, height)
}

func blitRegion(src image.Image, canvas []color.NRGBA, width, height int, p texPart) {
	for y := 0; y < p.h; y++ {
		dy := p.dstY + y
		if dy < 0 || dy >= height {
			continue
		}
		for x := 0; x < p.w; x++ {
			dx := p.dstX + x
			if dx < 0 || dx >= width {
				continue
			}
			r, g, b, a := src.At(p.srcX+x, p.srcY+y).RGBA()
			if a == 0 {
				continue
			}
			sa := float64(a) / 65535.0
			i := dy*width + dx
			dst := canvas[i]
			if dst.A == 0 {
				canvas[i] = color.NRGBA{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8), A: uint8(a >> 8)}
				continue
			}
			da := float64(dst.A) / 255.0
			outA := sa + da*(1-sa)
			if outA <= 0 {
				continue
			}
			canvas[i] = color.NRGBA{
				R: uint8((float64(r>>8)*sa + float64(dst.R)*da*(1-sa)) / outA),
				G: uint8((float64(g>>8)*sa + float64(dst.G)*da*(1-sa)) / outA),
				B: uint8((float64(b>>8)*sa + float64(dst.B)*da*(1-sa)) / outA),
				A: uint8(outA * 255),
			}
		}
	}
}

// halfBlockRows converts a pixel canvas into terminal rows. Each character
// shows two vertical pixels using the upper half block.
func halfBlockRows(canvas []color.NRGBA, width, height int) []string {
	rows := make([]string, 0, height/2)
	for y := 0; y+1 < height; y += 2 {
		var b strings.Builder
		for x := 0; x < width; x++ {
			top := canvas[y*width+x]
			bottom := canvas[(y+1)*width+x]
			switch {
			case top.A == 0 && bottom.A == 0:
				b.WriteByte(' ')
			case top.A == 0:
				fmt.Fprintf(&b, "\x1b[38;2;%d;%d;%dm▄\x1b[0m", bottom.R, bottom.G, bottom.B)
			case bottom.A == 0:
				fmt.Fprintf(&b, "\x1b[38;2;%d;%d;%dm▀\x1b[0m", top.R, top.G, top.B)
			default:
				fmt.Fprintf(&b, "\x1b[38;2;%d;%d;%d;48;2;%d;%d;%dm▀\x1b[0m",
					top.R, top.G, top.B, bottom.R, bottom.G, bottom.B)
			}
		}
		rows = append(rows, strings.TrimRight(b.String(), " "))
	}
	return rows
}
