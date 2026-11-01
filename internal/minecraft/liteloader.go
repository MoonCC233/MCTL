package minecraft

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

var liteLoaderCatalogURL = "https://dl.liteloader.com/versions/versions.json"

type liteLoaderCatalog struct {
	Versions map[string]liteLoaderMC `json:"versions"`
}

type liteLoaderMC struct {
	Repo      liteLoaderRepo                        `json:"repo"`
	Artefacts map[string]map[string]liteLoaderBuild `json:"artefacts"`
	Snapshots map[string]json.RawMessage            `json:"snapshots"`
}

type liteLoaderRepo struct {
	Stream string `json:"stream"`
	Type   string `json:"type"`
	URL    string `json:"url"`
}

type liteLoaderBuild struct {
	TweakClass string    `json:"tweakClass"`
	Libraries  []Library `json:"libraries"`
	Stream     string    `json:"stream"`
	File       string    `json:"file"`
	Version    string    `json:"version"`
	MD5        string    `json:"md5"`
	Timestamp  string    `json:"timestamp"`
	Build      string    `json:"build"`
}

// FetchLiteLoaderBuilds returns the LiteLoader catalogue entries for one
// Minecraft version.
func FetchLiteLoaderBuilds(ctx context.Context, hc *HTTPClient, mcVersion string) ([]LoaderBuild, error) {
	data, err := hc.Get(ctx, liteLoaderCatalogURL)
	if err != nil {
		return nil, fmt.Errorf("获取 LiteLoader 版本列表: %w", err)
	}
	var catalog liteLoaderCatalog
	if err := json.Unmarshal(data, &catalog); err != nil {
		return nil, fmt.Errorf("解析 LiteLoader 版本列表: %w", err)
	}
	entry, ok := catalog.Versions[mcVersion]
	if !ok {
		return nil, nil
	}
	out := make([]LoaderBuild, 0, 4)
	seen := map[string]bool{}
	add := func(builds map[string]liteLoaderBuild, baseLibs []Library, stable bool) {
		for _, b := range builds {
			if b.Version == "" || b.File == "" {
				continue
			}
			if seen[b.Version+"\x00"+b.File] {
				continue
			}
			seen[b.Version+"\x00"+b.File] = true
			libs := append([]Library(nil), baseLibs...)
			libs = append(libs, b.Libraries...)
			out = append(out, LoaderBuild{
				Version:    b.Version,
				MCVersion:  mcVersion,
				Kind:       "liteloader",
				Stable:     stable && !strings.EqualFold(b.Stream, "SNAPSHOT"),
				Filename:   b.File,
				RepoURL:    entry.Repo.URL,
				TweakClass: b.TweakClass,
				Libraries:  libs,
			})
		}
	}
	for _, group := range entry.Artefacts {
		add(group, nil, true)
	}
	var snapLibs []Library
	for key, raw := range entry.Snapshots {
		if key == "libraries" {
			_ = json.Unmarshal(raw, &snapLibs)
		}
	}
	for key, raw := range entry.Snapshots {
		if key == "libraries" {
			continue
		}
		var group map[string]liteLoaderBuild
		if json.Unmarshal(raw, &group) == nil {
			add(group, snapLibs, false)
		}
	}
	sortLoaderBuilds(out)
	return out, nil
}

// InstallLiteLoader downloads a LiteLoader jar and writes a launchwrapper
// profile containing the LiteLoader tweak class and its libraries.
func (i *Installer) InstallLiteLoader(ctx context.Context, mcVersion, loaderVersion string) (string, error) {
	builds, err := FetchLiteLoaderBuilds(ctx, i.HTTP, mcVersion)
	if err != nil {
		return "", err
	}
	var build *LoaderBuild
	for idx := range builds {
		if builds[idx].Version == loaderVersion {
			build = &builds[idx]
			break
		}
	}
	if build == nil {
		return "", fmt.Errorf("找不到 LiteLoader %s · %s", mcVersion, loaderVersion)
	}
	if build.TweakClass == "" || build.RepoURL == "" {
		return "", fmt.Errorf("LiteLoader 版本缺少启动配置")
	}

	id := mcVersion + "-LiteLoader-" + loaderVersion
	coord := MavenCoord{Group: "com.mumfrey", Artifact: "liteloader", Version: loaderVersion, Ext: "jar"}
	jarPath := filepath.Join(i.GameDir, "libraries", filepath.FromSlash(coord.Path()))
	if err := os.MkdirAll(filepath.Dir(jarPath), 0o755); err != nil {
		return "", err
	}
	if st, err := os.Stat(jarPath); err != nil || st.Size() == 0 {
		base := strings.TrimRight(build.RepoURL, "/")
		u := fmt.Sprintf("%s/com/mumfrey/liteloader/%s/%s", base, url.PathEscape(loaderVersion), url.PathEscape(build.Filename))
		i.report(Progress{Stage: "下载 LiteLoader", Current: build.Filename}, true)
		i.logf("下载 LiteLoader %s", build.Filename)
		if err := i.HTTP.DownloadFile(ctx, u, jarPath, "", nil); err != nil {
			return "", fmt.Errorf("下载 LiteLoader: %w", err)
		}
	}

	libs := []any{
		map[string]any{"name": "com.mumfrey:liteloader:" + loaderVersion, "url": build.RepoURL},
	}
	for _, l := range build.Libraries {
		item := map[string]any{"name": l.Name}
		switch {
		case l.URL != "":
			item["url"] = l.URL
		case strings.HasPrefix(l.Name, "org.ow2.asm:"):
			item["url"] = "https://repo1.maven.org/maven2/"
		case strings.HasPrefix(l.Name, "net.minecraft:launchwrapper"):
			item["url"] = "https://libraries.minecraft.net/"
		case strings.HasPrefix(l.Name, "org.spongepowered:mixin"):
			item["url"] = "https://repo.spongepowered.org/maven/"
		}
		libs = append(libs, item)
	}
	profile, _ := json.Marshal(map[string]any{
		"id":           id,
		"inheritsFrom": mcVersion,
		"type":         "release",
		"mainClass":    "net.minecraft.launchwrapper.Launch",
		"libraries":    libs,
		"arguments": map[string]any{
			"game": []string{"--tweakClass", build.TweakClass},
		},
	})
	if err := i.InstallProfile(ctx, id, profile); err != nil {
		return "", err
	}
	return id, nil
}
