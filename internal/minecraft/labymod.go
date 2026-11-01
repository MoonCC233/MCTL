package minecraft

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var (
	labyModManifestBase  = "https://releases.r2.labymod.net/api/v1/manifest"
	labyModLibrariesBase = "https://releases.r2.labymod.net/api/v1/libraries"
	labyModDownloadBase  = "https://releases.r2.labymod.net/api/v1/download/labymod4"
	labyModInstallerBase = "https://releases.labymod.net/api/v1/installer"
)

type labyModManifest struct {
	LabyModVersion    string             `json:"labyModVersion"`
	CommitReference   string             `json:"commitReference"`
	SHA1              string             `json:"sha1"`
	Size              int64              `json:"size"`
	MinecraftVersions []labyModMCVersion `json:"minecraftVersions"`
}

type labyModLibrary struct {
	Name             string `json:"name"`
	URL              string `json:"url"`
	MinecraftVersion string `json:"minecraftVersion"`
	SHA1             string `json:"sha1"`
	Size             int64  `json:"size"`
}

type labyModLibrariesResponse struct {
	Libraries []labyModLibrary `json:"libraries"`
}

type labyModMCVersion struct {
	Tag               string `json:"tag"`
	Version           string `json:"version"`
	Type              string `json:"type"`
	CustomManifestURL string `json:"customManifestUrl"`
	Runtime           struct {
		Version int `json:"version"`
	} `json:"runtime"`
}

// FetchLabyModBuilds returns production and snapshot installer entries that
// support the selected Minecraft version.
func FetchLabyModBuilds(ctx context.Context, hc *HTTPClient, mcVersion string) ([]LoaderBuild, error) {
	var out []LoaderBuild
	for _, channel := range []string{"production", "snapshot"} {
		manifest, err := fetchLabyModManifest(ctx, hc, channel)
		if err != nil {
			return nil, err
		}
		var match *labyModMCVersion
		for i := range manifest.MinecraftVersions {
			v := &manifest.MinecraftVersions[i]
			if v.Version == mcVersion || v.Tag == mcVersion {
				match = v
				break
			}
		}
		if match == nil {
			continue
		}
		filename := "LabyMod4ProductionInstaller.jar"
		if channel == "snapshot" {
			filename = "LabyMod4SnapshotInstaller.jar"
		}
		out = append(out, LoaderBuild{
			Version:      manifest.LabyModVersion,
			MCVersion:    mcVersion,
			Kind:         "labymod",
			Stable:       channel == "production",
			Filename:     filename,
			Channel:      channel,
			ManifestURL:  match.CustomManifestURL,
			RuntimeMajor: match.Runtime.Version,
		})
	}
	sortLoaderBuilds(out)
	return out, nil
}

func fetchLabyModManifest(ctx context.Context, hc *HTTPClient, channel string) (*labyModManifest, error) {
	u := fmt.Sprintf("%s/%s/latest.json", strings.TrimRight(labyModManifestBase, "/"), channel)
	data, err := hc.Get(ctx, u)
	if err != nil {
		return nil, fmt.Errorf("获取 LabyMod %s 清单: %w", channel, err)
	}
	var m labyModManifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("解析 LabyMod %s 清单: %w", channel, err)
	}
	if m.LabyModVersion == "" {
		return nil, fmt.Errorf("LabyMod %s 清单缺少版本号", channel)
	}
	return &m, nil
}

// FetchLabyModLibraries returns the extra libraries required by a channel.
func FetchLabyModLibraries(ctx context.Context, hc *HTTPClient, channel string) ([]labyModLibrary, error) {
	u := fmt.Sprintf("%s/%s.json", strings.TrimRight(labyModLibrariesBase, "/"), channel)
	data, err := hc.Get(ctx, u)
	if err != nil {
		return nil, fmt.Errorf("获取 LabyMod 依赖库: %w", err)
	}
	var resp labyModLibrariesResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("解析 LabyMod 依赖库: %w", err)
	}
	return resp.Libraries, nil
}

// InstallLabyModProfile creates a launchable LabyMod version profile from the
// official custom manifest and channel library catalogue.
func (i *Installer) InstallLabyModProfile(ctx context.Context, mcVersion, channel string) (string, error) {
	if channel != "production" && channel != "snapshot" {
		return "", fmt.Errorf("未知 LabyMod 通道: %s", channel)
	}
	manifest, err := fetchLabyModManifest(ctx, i.HTTP, channel)
	if err != nil {
		return "", err
	}
	var match *labyModMCVersion
	for idx := range manifest.MinecraftVersions {
		v := &manifest.MinecraftVersions[idx]
		if v.Version == mcVersion || v.Tag == mcVersion {
			match = v
			break
		}
	}
	if match == nil {
		return "", fmt.Errorf("LabyMod 不支持 Minecraft %s", mcVersion)
	}
	customData, err := i.HTTP.Get(ctx, match.CustomManifestURL)
	if err != nil {
		return "", fmt.Errorf("获取 LabyMod 自定义清单: %w", err)
	}
	var profile map[string]any
	if err := json.Unmarshal(customData, &profile); err != nil {
		return "", fmt.Errorf("解析 LabyMod 自定义清单: %w", err)
	}
	channelLibs, err := FetchLabyModLibraries(ctx, i.HTTP, channel)
	if err != nil {
		return "", err
	}
	var libs []any
	if existing, ok := profile["libraries"].([]any); ok {
		libs = append(libs, existing...)
	}
	for _, l := range channelLibs {
		if l.MinecraftVersion != "" && l.MinecraftVersion != "all" && l.MinecraftVersion != mcVersion {
			continue
		}
		item, err := labyModLibraryJSON(l)
		if err != nil {
			return "", err
		}
		libs = append(libs, item)
	}
	libs = append(libs, map[string]any{
		"name": "net.labymod:LabyMod:4",
		"downloads": map[string]any{
			"artifact": map[string]any{
				"path": "net/labymod/LabyMod/4/LabyMod-4.jar",
				"sha1": manifest.SHA1,
				"size": manifest.Size,
				"url": fmt.Sprintf("%s/%s/%s.jar",
					strings.TrimRight(labyModDownloadBase, "/"), channel, manifest.CommitReference),
			},
		},
	})
	tag := match.Tag
	if tag == "" {
		tag = match.Version
	}
	id := fmt.Sprintf("LabyMod-4-%s-%s", tag, channel)
	now := time.Now().UTC().Format(time.RFC3339)
	profile["id"] = id
	profile["time"] = now
	profile["releaseTime"] = now
	profile["libraries"] = libs
	profile["labymod_data"] = map[string]any{
		"channelType":     channel,
		"version":         manifest.LabyModVersion,
		"versionType":     match.Type,
		"commitReference": manifest.CommitReference,
	}
	if mcVersion == "1.8.9" || mcVersion == "1.12.2" {
		profile = map[string]any{
			"id":           id,
			"inheritsFrom": mcVersion,
			"type":         "release",
			"time":         now,
			"releaseTime":  now,
			"mainClass":    "net.minecraft.launchwrapper.Launch",
			"minecraftArguments": "--username ${auth_player_name} --version ${version_name} " +
				"--gameDir ${game_directory} --assetsDir ${assets_root} --assetIndex ${assets_index_name} " +
				"--uuid ${auth_uuid} --accessToken ${auth_access_token} --userProperties ${user_properties} " +
				"--userType ${user_type} --dummyMainClass net.minecraft.client.main.Main " +
				"--tweakClass net.labymod.core.loader.vanilla.launchwrapper.LabyModLaunchWrapperTweaker",
			"libraries":   libs,
			"javaVersion": map[string]any{"majorVersion": 21},
			"labymod_data": map[string]any{
				"channelType":     channel,
				"version":         manifest.LabyModVersion,
				"versionType":     match.Type,
				"commitReference": manifest.CommitReference,
			},
		}
	}
	data, err := json.Marshal(profile)
	if err != nil {
		return "", err
	}
	if err := i.InstallProfile(ctx, id, data); err != nil {
		return "", err
	}
	return id, nil
}

func labyModLibraryJSON(l labyModLibrary) (map[string]any, error) {
	coord, err := ParseMaven(l.Name)
	if err != nil {
		return nil, fmt.Errorf("解析 LabyMod 依赖 %s: %w", l.Name, err)
	}
	return map[string]any{
		"name": l.Name,
		"downloads": map[string]any{
			"artifact": map[string]any{
				"path": coord.Path(),
				"sha1": l.SHA1,
				"size": l.Size,
				"url":  l.URL,
			},
		},
	}, nil
}

// LabyModInstallerURL returns the official installer endpoint for a channel.
func LabyModInstallerURL(channel string) string {
	return fmt.Sprintf("%s/%s/java", strings.TrimRight(labyModInstallerBase, "/"), channel)
}

// InstallLabyMod downloads the official LabyMod installer into the instance
// cache. LabyMod is a client installer rather than a launch profile, so the
// installer itself is kept for the user to run.
func (i *Installer) InstallLabyMod(ctx context.Context, channel string) (string, error) {
	switch channel {
	case "production", "snapshot":
	default:
		return "", fmt.Errorf("未知 LabyMod 通道: %s", channel)
	}
	filename := "LabyMod4ProductionInstaller.jar"
	if channel == "snapshot" {
		filename = "LabyMod4SnapshotInstaller.jar"
	}
	dir := filepath.Join(i.GameDir, "mctl-installers")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	dest := filepath.Join(dir, filename)
	if st, err := os.Stat(dest); err == nil && st.Size() > 0 {
		i.logf("使用已下载的 LabyMod 安装器")
		return dest, nil
	}
	i.report(Progress{Stage: "下载 LabyMod 安装器", Current: filename}, true)
	i.logf("下载 LabyMod %s 安装器", channel)
	if err := i.HTTP.DownloadFile(ctx, LabyModInstallerURL(channel), dest, "", nil); err != nil {
		return "", fmt.Errorf("下载 LabyMod 安装器: %w", err)
	}
	i.report(Progress{Stage: "LabyMod 安装器已下载", Done: 1, Total: 1}, true)
	return dest, nil
}
