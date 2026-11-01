package minecraft

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strings"
)

// FabricLoader is one loader build offered by Fabric Meta.
type FabricLoader struct {
	Version string
	Stable  bool
}

// FabricMeta is the public Fabric loader metadata endpoint.
var FabricMeta = "https://meta.fabricmc.net"

type fabricLoaderResponse struct {
	Loader struct {
		Version string `json:"version"`
		Stable  bool   `json:"stable"`
	} `json:"loader"`
}

// FetchFabricLoaders lists loader builds compatible with a game version.
func FetchFabricLoaders(ctx context.Context, hc *HTTPClient, mcVersion string) ([]FabricLoader, error) {
	u := fmt.Sprintf("%s/v2/versions/loader/%s", FabricMeta, url.PathEscape(mcVersion))
	data, err := hc.Get(ctx, u)
	if err != nil {
		return nil, fmt.Errorf("获取 Fabric 加载器列表: %w", err)
	}
	var raw []fabricLoaderResponse
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("解析 Fabric 加载器列表: %w", err)
	}
	out := make([]FabricLoader, 0, len(raw))
	for _, r := range raw {
		if r.Loader.Version == "" {
			continue
		}
		out = append(out, FabricLoader{Version: r.Loader.Version, Stable: r.Loader.Stable})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Stable != out[j].Stable {
			return out[i].Stable
		}
		return compareVersions(out[i].Version, out[j].Version) > 0
	})
	return out, nil
}

// FetchFabricProfile downloads an installable Fabric version profile.
func FetchFabricProfile(ctx context.Context, hc *HTTPClient, mcVersion, loaderVersion string) ([]byte, error) {
	u := fmt.Sprintf("%s/v2/versions/loader/%s/%s/profile/json", FabricMeta, url.PathEscape(mcVersion), url.PathEscape(loaderVersion))
	data, err := hc.Get(ctx, u)
	if err != nil {
		return nil, fmt.Errorf("获取 Fabric 配置: %w", err)
	}
	var v VersionJSON
	if err := json.Unmarshal(data, &v); err != nil {
		return nil, fmt.Errorf("解析 Fabric 配置: %w", err)
	}
	if v.ID == "" {
		return nil, fmt.Errorf("Fabric 配置缺少版本 ID")
	}
	return data, nil
}

func compareVersions(a, b string) int {
	pa := strings.Split(strings.TrimPrefix(a, "v"), ".")
	pb := strings.Split(strings.TrimPrefix(b, "v"), ".")
	n := len(pa)
	if len(pb) > n {
		n = len(pb)
	}
	for i := 0; i < n; i++ {
		var ai, bi int
		if i < len(pa) {
			ai = leadingNumber(pa[i])
		}
		if i < len(pb) {
			bi = leadingNumber(pb[i])
		}
		if ai != bi {
			if ai > bi {
				return 1
			}
			return -1
		}
	}
	return 0
}

func leadingNumber(s string) int {
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			break
		}
		n = n*10 + int(r-'0')
	}
	return n
}
