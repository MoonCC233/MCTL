package minecraft

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

var modrinthAPIBase = "https://api.modrinth.com/v2"

// ModrinthProject is a search result for an online modpack.
type ModrinthProject struct {
	ID          string `json:"project_id"`
	Slug        string `json:"slug"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Author      string `json:"author"`
	Downloads   int64  `json:"downloads"`
	IconURL     string `json:"icon_url"`
}

type modrinthSearchResponse struct {
	Hits []ModrinthProject `json:"hits"`
}

// ModrinthVersion is one published version of a Modrinth project.
type ModrinthVersion struct {
	ID            string         `json:"id"`
	ProjectID     string         `json:"project_id"`
	Name          string         `json:"name"`
	VersionNumber string         `json:"version_number"`
	VersionType   string         `json:"version_type"`
	GameVersions  []string       `json:"game_versions"`
	Loaders       []string       `json:"loaders"`
	Files         []ModrinthFile `json:"files"`
}

// ModrinthFile is a downloadable file attached to a version.
type ModrinthFile struct {
	URL      string            `json:"url"`
	Filename string            `json:"filename"`
	Size     int64             `json:"size"`
	Primary  bool              `json:"primary"`
	Hashes   map[string]string `json:"hashes"`
}

// SearchModpacks searches Modrinth for modpack projects.
func SearchModpacks(ctx context.Context, hc *HTTPClient, query string, limit int) ([]ModrinthProject, error) {
	if limit <= 0 {
		limit = 20
	}
	q := url.Values{}
	q.Set("query", strings.TrimSpace(query))
	q.Set("limit", strconv.Itoa(limit))
	q.Set("index", "downloads")
	q.Set("facets", `[["project_type:modpack"]]`)
	u := strings.TrimRight(modrinthAPIBase, "/") + "/search?" + q.Encode()
	data, err := hc.Get(ctx, u)
	if err != nil {
		return nil, fmt.Errorf("搜索 Modrinth 整合包: %w", err)
	}
	var resp modrinthSearchResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("解析 Modrinth 搜索结果: %w", err)
	}
	out := make([]ModrinthProject, 0, len(resp.Hits))
	for _, p := range resp.Hits {
		if p.ID == "" {
			continue
		}
		out = append(out, p)
	}
	return out, nil
}

// FetchLatestModpackVersion returns the preferred version for a project.
func FetchLatestModpackVersion(ctx context.Context, hc *HTTPClient, projectID string) (*ModrinthVersion, error) {
	u := strings.TrimRight(modrinthAPIBase, "/") + "/project/" + url.PathEscape(projectID) + "/version"
	data, err := hc.Get(ctx, u)
	if err != nil {
		return nil, fmt.Errorf("获取 Modrinth 版本: %w", err)
	}
	var versions []ModrinthVersion
	if err := json.Unmarshal(data, &versions); err != nil {
		return nil, fmt.Errorf("解析 Modrinth 版本: %w", err)
	}
	if len(versions) == 0 {
		return nil, fmt.Errorf("该整合包没有可用版本")
	}
	chosen := &versions[0]
	for i := range versions {
		if versions[i].VersionType == "release" {
			chosen = &versions[i]
			break
		}
	}
	if len(chosen.Files) == 0 {
		return nil, fmt.Errorf("该整合包版本没有文件")
	}
	return chosen, nil
}

// PrimaryModrinthFile returns the primary file of a version, or the first.
func PrimaryModrinthFile(v *ModrinthVersion) (ModrinthFile, error) {
	if v == nil || len(v.Files) == 0 {
		return ModrinthFile{}, fmt.Errorf("整合包版本没有文件")
	}
	for _, f := range v.Files {
		if f.Primary && f.URL != "" {
			return f, nil
		}
	}
	for _, f := range v.Files {
		if f.URL != "" {
			return f, nil
		}
	}
	return ModrinthFile{}, fmt.Errorf("整合包文件没有下载地址")
}
