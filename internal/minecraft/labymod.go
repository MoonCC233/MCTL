package minecraft

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var (
	labyModManifestBase  = "https://releases.r2.labymod.net/api/v1/manifest"
	labyModInstallerBase = "https://releases.labymod.net/api/v1/installer"
)

type labyModManifest struct {
	LabyModVersion    string             `json:"labyModVersion"`
	MinecraftVersions []labyModMCVersion `json:"minecraftVersions"`
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
