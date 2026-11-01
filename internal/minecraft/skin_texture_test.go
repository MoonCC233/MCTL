package minecraft

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchTextureImage(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 64, 64))
	img.Set(8, 8, color.NRGBA{R: 255, A: 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(buf.Bytes())
	}))
	defer server.Close()

	hc := NewHTTPClient(Source{ID: "test"})
	hc.Client = server.Client()
	got, err := FetchTextureImage(context.Background(), hc, server.URL+"/skin.png")
	if err != nil {
		t.Fatal(err)
	}
	if got.Bounds().Dx() != 64 {
		t.Fatalf("bounds = %v", got.Bounds())
	}
	if _, err := FetchTextureImage(context.Background(), hc, ""); err == nil {
		t.Fatal("empty url should fail")
	}
}
