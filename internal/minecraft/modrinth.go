package minecraft

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
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

// ModpackFilter narrows an online modpack search.
type ModpackFilter struct {
	GameVersion string
	Loader      string
}

// SearchProjects searches Modrinth for a project type (modpack, resourcepack,
// shader or mod).
func SearchProjects(ctx context.Context, hc *HTTPClient, query, projectType string, limit int) ([]ModrinthProject, error) {
	return searchProjects(ctx, hc, query, projectType, nil, limit)
}

func searchProjects(ctx context.Context, hc *HTTPClient, query, projectType string, extra [][]string, limit int) ([]ModrinthProject, error) {
	if limit <= 0 {
		limit = 20
	}
	facets := [][]string{{"project_type:" + projectType}}
	facets = append(facets, extra...)
	facetJSON, err := json.Marshal(facets)
	if err != nil {
		return nil, err
	}
	q := url.Values{}
	q.Set("query", strings.TrimSpace(query))
	q.Set("limit", strconv.Itoa(limit))
	q.Set("index", "downloads")
	q.Set("facets", string(facetJSON))
	u := strings.TrimRight(modrinthAPIBase, "/") + "/search?" + q.Encode()
	data, err := hc.Get(ctx, u)
	if err != nil {
		return nil, fmt.Errorf("搜索 Modrinth %s: %w", projectType, err)
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

// SearchModpacks searches Modrinth for modpack projects.
func SearchModpacks(ctx context.Context, hc *HTTPClient, query string, limit int) ([]ModrinthProject, error) {
	return SearchModpacksFiltered(ctx, hc, query, ModpackFilter{}, limit)
}

// SearchModpacksFiltered searches modpacks with an optional game version and
// loader filter.
func SearchModpacksFiltered(ctx context.Context, hc *HTTPClient, query string, filter ModpackFilter, limit int) ([]ModrinthProject, error) {
	var extra [][]string
	if v := strings.TrimSpace(filter.GameVersion); v != "" {
		extra = append(extra, []string{"versions:" + v})
	}
	if l := strings.TrimSpace(filter.Loader); l != "" {
		extra = append(extra, []string{"categories:" + l})
	}
	return searchProjects(ctx, hc, query, "modpack", extra, limit)
}

// FetchProjectVersions returns all published versions of a project.
func FetchProjectVersions(ctx context.Context, hc *HTTPClient, projectID string) ([]ModrinthVersion, error) {
	u := strings.TrimRight(modrinthAPIBase, "/") + "/project/" + url.PathEscape(projectID) + "/version"
	data, err := hc.Get(ctx, u)
	if err != nil {
		return nil, fmt.Errorf("获取 Modrinth 版本: %w", err)
	}
	var versions []ModrinthVersion
	if err := json.Unmarshal(data, &versions); err != nil {
		return nil, fmt.Errorf("解析 Modrinth 版本: %w", err)
	}
	return versions, nil
}

// FetchLatestModpackVersion returns the preferred version for a project.
func FetchLatestModpackVersion(ctx context.Context, hc *HTTPClient, projectID string) (*ModrinthVersion, error) {
	versions, err := FetchProjectVersions(ctx, hc, projectID)
	if err != nil {
		return nil, err
	}
	return SelectModrinthVersion(versions, "", "")
}

// SelectModrinthVersion chooses a release compatible with the game version and
// loader. Empty filters pick the newest release.
func SelectModrinthVersion(versions []ModrinthVersion, gameVersion, loader string) (*ModrinthVersion, error) {
	var fallback *ModrinthVersion
	for i := range versions {
		v := &versions[i]
		if gameVersion != "" && len(v.GameVersions) > 0 && !containsString(v.GameVersions, gameVersion) {
			continue
		}
		if loader != "" && len(v.Loaders) > 0 && !containsString(v.Loaders, loader) {
			continue
		}
		if fallback == nil {
			fallback = v
		}
		if v.VersionType == "release" {
			return v, nil
		}
	}
	if fallback != nil {
		return fallback, nil
	}
	if len(versions) == 0 {
		return nil, fmt.Errorf("没有可用版本")
	}
	return nil, fmt.Errorf("没有适用于当前版本或加载器的文件")
}

func containsString(list []string, want string) bool {
	for _, s := range list {
		if strings.EqualFold(s, want) {
			return true
		}
	}
	return false
}

// ResourceInstall describes a downloaded online resource.
type ResourceInstall struct {
	Kind    string
	Name    string
	Path    string
	Version string
}

// ResourceSubdir maps a Modrinth project type to its game directory.
func ResourceSubdir(kind string) string {
	switch kind {
	case "resourcepack":
		return "resourcepacks"
	case "shader":
		return "shaderpacks"
	case "mod":
		return "mods"
	}
	return ""
}

// InstallResource downloads the best matching file for an online resource.
func InstallResource(ctx context.Context, hc *HTTPClient, kind, projectID, gameVersion, loader, gameDir string) (*ResourceInstall, error) {
	sub := ResourceSubdir(kind)
	if sub == "" {
		return nil, fmt.Errorf("未知资源类型: %s", kind)
	}
	versions, err := FetchProjectVersions(ctx, hc, projectID)
	if err != nil {
		return nil, err
	}
	if kind == "resourcepack" || kind == "shader" {
		gameVersion, loader = "", ""
	}
	version, err := SelectModrinthVersion(versions, gameVersion, loader)
	if err != nil {
		return nil, err
	}
	file, err := PrimaryModrinthFile(version)
	if err != nil {
		return nil, err
	}
	destDir := filepath.Join(gameDir, sub)
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return nil, err
	}
	name := filepath.Base(file.Filename)
	dest := filepath.Join(destDir, name)
	sha := ""
	if file.Hashes != nil {
		sha = file.Hashes["sha1"]
	}
	if err := hc.DownloadFile(ctx, file.URL, dest, sha, nil); err != nil {
		return nil, fmt.Errorf("下载 %s: %w", name, err)
	}
	return &ResourceInstall{Kind: kind, Name: name, Path: dest, Version: version.VersionNumber}, nil
}

// InstallDatapack downloads the best matching datapack into a world's own
// datapacks folder. Datapacks are world-scoped, so a world directory is
// required instead of the game directory.
func InstallDatapack(ctx context.Context, hc *HTTPClient, projectID, gameVersion, worldDir string) (*ResourceInstall, error) {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return nil, fmt.Errorf("缺少数据包项目 ID")
	}
	if st, err := os.Stat(filepath.Join(worldDir, "level.dat")); err != nil || st.IsDir() {
		return nil, fmt.Errorf("存档目录无效: %s", worldDir)
	}
	versions, err := FetchProjectVersions(ctx, hc, projectID)
	if err != nil {
		return nil, err
	}
	version, err := SelectModrinthVersion(versions, gameVersion, "")
	if err != nil {
		return nil, err
	}
	file, err := PrimaryModrinthFile(version)
	if err != nil {
		return nil, err
	}
	destDir := filepath.Join(worldDir, "datapacks")
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return nil, err
	}
	name := filepath.Base(file.Filename)
	dest := filepath.Join(destDir, name)
	sha := ""
	if file.Hashes != nil {
		sha = file.Hashes["sha1"]
	}
	if err := hc.DownloadFile(ctx, file.URL, dest, sha, nil); err != nil {
		return nil, fmt.Errorf("下载 %s: %w", name, err)
	}
	return &ResourceInstall{Kind: "datapack", Name: name, Path: dest, Version: version.VersionNumber}, nil
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
