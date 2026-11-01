package minecraft

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

var legacyFabricMetaURL = "https://meta.legacyfabric.net"

// FetchLegacyFabricLoaders lists Legacy Fabric loader builds.
func FetchLegacyFabricLoaders(ctx context.Context, hc *HTTPClient, mcVersion string) ([]LoaderBuild, error) {
	u := fmt.Sprintf("%s/v2/versions/loader/%s", legacyFabricMetaURL, url.PathEscape(mcVersion))
	data, err := hc.Get(ctx, u)
	if err != nil {
		return nil, fmt.Errorf("获取 Legacy Fabric 加载器列表: %w", err)
	}
	var raw []json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("解析 Legacy Fabric 加载器列表: %w", err)
	}
	var out []LoaderBuild
	for _, item := range raw {
		var v struct {
			Version string `json:"version"`
			Stable  bool   `json:"stable"`
			Loader  struct {
				Version string `json:"version"`
				Stable  bool   `json:"stable"`
			} `json:"loader"`
		}
		if err := json.Unmarshal(item, &v); err != nil {
			continue
		}
		version := strings.TrimSpace(v.Loader.Version)
		stable := v.Loader.Stable
		if version == "" {
			version = strings.TrimSpace(v.Version)
			stable = v.Stable
		}
		if version == "" {
			continue
		}
		out = append(out, LoaderBuild{
			Version:   version,
			MCVersion: mcVersion,
			Kind:      "legacyfabric",
			Stable:    stable,
		})
	}
	sortLoaderBuilds(out)
	return out, nil
}

// FetchLegacyFabricProfile downloads an installable Legacy Fabric profile.
func FetchLegacyFabricProfile(ctx context.Context, hc *HTTPClient, mcVersion, loaderVersion string) ([]byte, error) {
	u := fmt.Sprintf("%s/v2/versions/loader/%s/%s/profile/json",
		legacyFabricMetaURL, url.PathEscape(mcVersion), url.PathEscape(loaderVersion))
	data, err := hc.Get(ctx, u)
	if err != nil {
		return nil, fmt.Errorf("获取 Legacy Fabric 配置: %w", err)
	}
	var v VersionJSON
	if err := json.Unmarshal(data, &v); err != nil {
		return nil, fmt.Errorf("解析 Legacy Fabric 配置: %w", err)
	}
	if v.ID == "" {
		return nil, fmt.Errorf("Legacy Fabric 配置缺少版本 ID")
	}
	return data, nil
}
