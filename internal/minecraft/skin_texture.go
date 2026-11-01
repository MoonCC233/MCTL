package minecraft

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/png"
	"strings"
)

// FetchTextureImage downloads and decodes a PNG skin or cape texture.
func FetchTextureImage(ctx context.Context, hc *HTTPClient, rawURL string) (image.Image, error) {
	if strings.TrimSpace(rawURL) == "" {
		return nil, fmt.Errorf("缺少纹理地址")
	}
	if hc == nil {
		return nil, fmt.Errorf("缺少 HTTP 客户端")
	}
	data, err := hc.Get(ctx, rawURL)
	if err != nil {
		return nil, fmt.Errorf("下载纹理: %w", err)
	}
	if len(data) > 4<<20 {
		return nil, fmt.Errorf("纹理文件过大")
	}
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("解析纹理: %w", err)
	}
	if img.Bounds().Dx() < 64 || img.Bounds().Dy() < 32 {
		return nil, fmt.Errorf("纹理尺寸异常: %dx%d", img.Bounds().Dx(), img.Bounds().Dy())
	}
	return img, nil
}
