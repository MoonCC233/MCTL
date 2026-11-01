package minecraft

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

var quiltMetaURL = "https://meta.quiltmc.org"

// FetchQuiltLoaders lists Quilt loader builds for a Minecraft version.
func FetchQuiltLoaders(ctx context.Context, hc *HTTPClient, mcVersion string) ([]LoaderBuild, error) {
	u := fmt.Sprintf("%s/v3/versions/loader/%s", quiltMetaURL, url.PathEscape(mcVersion))
	data, err := hc.Get(ctx, u)
	if err != nil {
		return nil, fmt.Errorf("获取 Quilt 加载器列表: %w", err)
	}
	var raw []json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("解析 Quilt 加载器列表: %w", err)
	}
	var out []LoaderBuild
	for _, item := range raw {
		var v struct {
			Version string `json:"version"`
			Loader  struct {
				Version string `json:"version"`
			} `json:"loader"`
		}
		if err := json.Unmarshal(item, &v); err != nil {
			continue
		}
		version := strings.TrimSpace(v.Loader.Version)
		if version == "" {
			version = strings.TrimSpace(v.Version)
		}
		if version == "" {
			continue
		}
		out = append(out, LoaderBuild{
			Version:   version,
			MCVersion: mcVersion,
			Kind:      "quilt",
			Stable:    !strings.Contains(strings.ToLower(version), "beta"),
		})
	}
	sortLoaderBuilds(out)
	return out, nil
}

// FetchQuiltProfile downloads an installable Quilt version profile.
func FetchQuiltProfile(ctx context.Context, hc *HTTPClient, mcVersion, loaderVersion string) ([]byte, error) {
	u := fmt.Sprintf("%s/v3/versions/loader/%s/%s/profile/json",
		quiltMetaURL, url.PathEscape(mcVersion), url.PathEscape(loaderVersion))
	data, err := hc.Get(ctx, u)
	if err != nil {
		return nil, fmt.Errorf("获取 Quilt 配置: %w", err)
	}
	var v VersionJSON
	if err := json.Unmarshal(data, &v); err != nil {
		return nil, fmt.Errorf("解析 Quilt 配置: %w", err)
	}
	if v.ID == "" {
		return nil, fmt.Errorf("Quilt 配置缺少版本 ID")
	}
	return data, nil
}
