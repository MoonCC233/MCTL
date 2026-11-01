package minecraft

import (
	"bufio"
	"context"
	"encoding/xml"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

// LoaderBuild is one Forge or NeoForge build for a Minecraft version.
type LoaderBuild struct {
	Version   string
	MCVersion string
	Kind      string
	Stable    bool
}

var (
	forgeMetadataURL    = "https://maven.minecraftforge.net/net/minecraftforge/forge/maven-metadata.xml"
	neoforgeMetadataURL = "https://maven.neoforged.net/releases/net/neoforged/neoforge/maven-metadata.xml"
	forgeMavenBase      = "https://maven.minecraftforge.net/net/minecraftforge/forge"
	neoforgeMavenBase   = "https://maven.neoforged.net/releases/net/neoforged/neoforge"
)

type mavenMetadata struct {
	Versioning struct {
		Versions struct {
			Version []string `xml:"version"`
		} `xml:"versions"`
	} `xml:"versioning"`
}

// FetchLoaderBuilds lists Forge or NeoForge builds for a game version.
func FetchLoaderBuilds(ctx context.Context, hc *HTTPClient, kind, mcVersion string) ([]LoaderBuild, error) {
	switch kind {
	case "forge":
		return fetchForgeBuilds(ctx, hc, mcVersion)
	case "neoforge":
		return fetchNeoForgeBuilds(ctx, hc, mcVersion)
	default:
		return nil, fmt.Errorf("未知加载器: %s", kind)
	}
}

func fetchForgeBuilds(ctx context.Context, hc *HTTPClient, mcVersion string) ([]LoaderBuild, error) {
	data, err := hc.Get(ctx, forgeMetadataURL)
	if err != nil {
		return nil, fmt.Errorf("获取 Forge 版本列表: %w", err)
	}
	versions, err := parseMavenMetadata(data)
	if err != nil {
		return nil, fmt.Errorf("解析 Forge 版本列表: %w", err)
	}
	prefix := mcVersion + "-"
	var out []LoaderBuild
	for _, v := range versions {
		if !strings.HasPrefix(v, prefix) {
			continue
		}
		loader := strings.TrimPrefix(v, prefix)
		if loader == "" {
			continue
		}
		low := strings.ToLower(loader)
		out = append(out, LoaderBuild{
			Version:   loader,
			MCVersion: mcVersion,
			Kind:      "forge",
			Stable:    !strings.Contains(low, "beta") && !strings.Contains(low, "alpha"),
		})
	}
	sortLoaderBuilds(out)
	return out, nil
}

func fetchNeoForgeBuilds(ctx context.Context, hc *HTTPClient, mcVersion string) ([]LoaderBuild, error) {
	data, err := hc.Get(ctx, neoforgeMetadataURL)
	if err != nil {
		return nil, fmt.Errorf("获取 NeoForge 版本列表: %w", err)
	}
	versions, err := parseMavenMetadata(data)
	if err != nil {
		return nil, fmt.Errorf("解析 NeoForge 版本列表: %w", err)
	}
	var out []LoaderBuild
	for _, v := range versions {
		if neoForgeMCVersion(v) != mcVersion {
			continue
		}
		out = append(out, LoaderBuild{
			Version:   v,
			MCVersion: mcVersion,
			Kind:      "neoforge",
			Stable:    !strings.Contains(strings.ToLower(v), "beta"),
		})
	}
	sortLoaderBuilds(out)
	return out, nil
}

func parseMavenMetadata(data []byte) ([]string, error) {
	var m mavenMetadata
	if err := xml.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return m.Versioning.Versions.Version, nil
}

func sortLoaderBuilds(out []LoaderBuild) {
	sortSliceStable(out, func(i, j int) bool {
		if out[i].Stable != out[j].Stable {
			return out[i].Stable
		}
		return compareVersions(out[i].Version, out[j].Version) > 0
	})
}

func sortSliceStable[T any](s []T, less func(i, j int) bool) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && less(j, j-1); j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}

// NeoForgeMCVersion maps a NeoForge version such as 21.1.80 to 1.21.1.
func NeoForgeMCVersion(version string) string {
	return neoForgeMCVersion(version)
}

func neoForgeMCVersion(version string) string {
	parts := strings.Split(strings.TrimPrefix(strings.TrimSpace(version), "v"), ".")
	if len(parts) < 2 {
		return ""
	}
	major, err1 := strconv.Atoi(parts[0])
	minor, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil || major < 20 {
		return ""
	}
	if minor == 0 {
		return fmt.Sprintf("1.%d", major)
	}
	return fmt.Sprintf("1.%d.%d", major, minor)
}

// InstallerURL returns the official installer download for a loader build.
func (b LoaderBuild) InstallerURL() string {
	switch b.Kind {
	case "neoforge":
		return fmt.Sprintf("%s/%s/neoforge-%s-installer.jar",
			strings.TrimSuffix(neoforgeMavenBase, "/"),
			url.PathEscape(b.Version), url.PathEscape(b.Version))
	default:
		return fmt.Sprintf("%s/%s-%s/forge-%s-%s-installer.jar",
			strings.TrimSuffix(forgeMavenBase, "/"),
			url.PathEscape(b.MCVersion), url.PathEscape(b.Version),
			url.PathEscape(b.MCVersion), url.PathEscape(b.Version))
	}
}

// LoaderVersionCandidates lists likely version directory names created by an
// official loader installer.
func LoaderVersionCandidates(kind, mcVersion, loaderVersion string) []string {
	switch kind {
	case "neoforge":
		return []string{
			loaderVersion,
			"neoforge-" + loaderVersion,
			mcVersion + "-neoforge-" + loaderVersion,
		}
	default:
		return []string{
			mcVersion + "-forge-" + loaderVersion,
			mcVersion + "-forge" + loaderVersion,
			"forge-" + loaderVersion,
		}
	}
}

// FindLoaderVersion locates the version profile produced by a loader installer.
func FindLoaderVersion(gameDir, kind, mcVersion, loaderVersion string) (string, error) {
	for _, id := range LoaderVersionCandidates(kind, mcVersion, loaderVersion) {
		if _, err := ResolveVersion(gameDir, id); err == nil {
			return id, nil
		}
	}
	entries, err := os.ReadDir(filepath.Join(gameDir, "versions"))
	if err != nil {
		return "", fmt.Errorf("安装器未生成版本目录: %w", err)
	}
	needle := strings.ToLower(loaderVersion)
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		id := e.Name()
		low := strings.ToLower(id)
		if !strings.Contains(low, needle) {
			continue
		}
		if kind == "neoforge" {
			if !strings.Contains(low, "neoforge") {
				continue
			}
		} else if !strings.Contains(low, "forge") || strings.Contains(low, "neoforge") {
			continue
		}
		if _, err := ResolveVersion(gameDir, id); err == nil {
			return id, nil
		}
	}
	return "", fmt.Errorf("安装器运行结束，但没有找到 %s %s 的版本配置", kind, loaderVersion)
}

// InstallLoader downloads and runs the official Forge/NeoForge installer.
func (i *Installer) InstallLoader(ctx context.Context, kind, mcVersion, loaderVersion string, java *JavaInfo) (string, error) {
	if java == nil || java.Path == "" {
		return "", fmt.Errorf("运行 %s 安装器需要 Java", kind)
	}
	build := LoaderBuild{Version: loaderVersion, MCVersion: mcVersion, Kind: kind}
	installerDir := filepath.Join(i.GameDir, "mctl-installers")
	if err := os.MkdirAll(installerDir, 0o755); err != nil {
		return "", err
	}
	installer := filepath.Join(installerDir, fmt.Sprintf("%s-%s-%s-installer.jar", kind, mcVersion, loaderVersion))
	i.report(Progress{Stage: "下载安装器", Current: filepath.Base(installer)}, true)
	if st, err := os.Stat(installer); err == nil && st.Size() > 0 {
		i.logf("使用已缓存的 %s 安装器", kind)
	} else {
		i.logf("下载 %s 安装器 %s", kind, installer)
		if err := i.HTTP.DownloadFile(ctx, build.InstallerURL(), installer, "", nil); err != nil {
			return "", fmt.Errorf("下载 %s 安装器: %w", kind, err)
		}
	}

	i.report(Progress{Stage: "运行安装器", Current: filepath.Base(installer)}, true)
	if err := i.runInstaller(ctx, java.Path, installer); err != nil {
		return "", err
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	id, err := FindLoaderVersion(i.GameDir, kind, mcVersion, loaderVersion)
	if err != nil {
		return "", err
	}
	i.logf("安装器已生成版本 %s，开始补全游戏文件", id)
	if err := i.EnsureInstalled(ctx, id); err != nil {
		return "", err
	}
	return id, nil
}

func (i *Installer) runInstaller(ctx context.Context, javaPath, installer string) error {
	attempts := [][]string{
		{"-jar", installer, "--installClient", i.GameDir},
		{"-jar", installer, "--installClient"},
	}
	var lastErr error
	for n, args := range attempts {
		if n > 0 {
			i.logf("安装器返回错误，尝试兼容模式")
		}
		cmd := exec.CommandContext(ctx, javaPath, args...)
		cmd.Dir = i.GameDir
		hideWindow(cmd)
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			return err
		}
		stderr, err := cmd.StderrPipe()
		if err != nil {
			return err
		}
		if err := cmd.Start(); err != nil {
			lastErr = err
			continue
		}
		var wg sync.WaitGroup
		scan := func(r interface{ Read([]byte) (int, error) }, prefix string) {
			defer wg.Done()
			sc := bufio.NewScanner(r)
			sc.Buffer(make([]byte, 64*1024), 2*1024*1024)
			for sc.Scan() {
				line := strings.TrimRight(sc.Text(), "\r")
				if strings.TrimSpace(line) != "" {
					i.logf("%s%s", prefix, line)
				}
			}
		}
		wg.Add(2)
		go scan(stdout, "")
		go scan(stderr, "")
		wg.Wait()
		if err := cmd.Wait(); err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			lastErr = err
			continue
		}
		return nil
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("安装器没有正常退出")
	}
	return fmt.Errorf("运行 %s 安装器: %w", filepath.Base(installer), lastErr)
}
