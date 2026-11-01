package minecraft

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// InstanceWizardOptions configures creation of a new isolated instance.
type InstanceWizardOptions struct {
	Name          string
	Dir           string
	MCVersion     string
	Loader        string
	LoaderVersion string
	JavaPath      string
	AutoJava      bool
}

// WizardLoaders lists the loader choices offered by the instance wizard.
var WizardLoaders = []string{"vanilla", "fabric", "quilt", "legacyfabric", "forge", "neoforge"}

// ResolveInstanceDir validates and resolves the instance directory. The
// instance must be a single directory directly under <gameDir>/instances.
func ResolveInstanceDir(gameDir, name, mcVersion, dir string) (string, error) {
	root := filepath.Join(gameDir, "instances")
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return filepath.Join(root, packSlug(name, mcVersion)), nil
	}
	var target string
	if filepath.IsAbs(dir) {
		target = filepath.Clean(dir)
		if filepath.Dir(target) != filepath.Clean(root) {
			return "", fmt.Errorf("实例目录必须直接位于 %s 下", root)
		}
	} else {
		if strings.ContainsAny(dir, `/\`) || dir == "." || dir == ".." {
			return "", fmt.Errorf("实例目录只能填写单层目录名")
		}
		target = filepath.Join(root, dir)
	}
	rel, err := filepath.Rel(filepath.Clean(root), target)
	if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || strings.Contains(rel, string(filepath.Separator)) {
		return "", fmt.Errorf("实例目录必须直接位于 %s 下", root)
	}
	return target, nil
}

// CreateInstance installs a Minecraft version and optional loader into a new
// isolated instance and returns its version ID and directory.
func (i *Installer) CreateInstance(ctx context.Context, opts InstanceWizardOptions) (string, string, error) {
	mcVersion := strings.TrimSpace(opts.MCVersion)
	if mcVersion == "" {
		return "", "", fmt.Errorf("请填写 Minecraft 版本")
	}
	name := strings.TrimSpace(opts.Name)
	if name == "" {
		name = mcVersion
	}
	kind := strings.ToLower(strings.TrimSpace(opts.Loader))
	if kind == "" {
		kind = "vanilla"
	}
	if !wizardLoaderAllowed(kind) {
		return "", "", fmt.Errorf("不支持的加载器: %s", kind)
	}
	root, err := ResolveInstanceDir(i.GameDir, name, mcVersion, opts.Dir)
	if err != nil {
		return "", "", err
	}
	if entries, err := os.ReadDir(root); err == nil && len(entries) > 0 {
		return "", "", fmt.Errorf("实例目录已存在且不为空: %s", root)
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return "", "", err
	}
	target := i.forGameDir(root)
	cleanup := func() { _ = os.RemoveAll(root) }

	target.report(Progress{Stage: "安装基础版本", Current: mcVersion}, true)
	if err := target.InstallVersion(ctx, mcVersion, nil); err != nil {
		cleanup()
		return "", "", fmt.Errorf("安装基础版本 %s: %w", mcVersion, err)
	}

	loaderID := mcVersion
	if kind != "vanilla" {
		var err error
		loaderID, err = i.installLoaderInto(ctx, target, kind, mcVersion, opts.LoaderVersion, opts.JavaPath, opts.AutoJava)
		if err != nil {
			cleanup()
			return "", "", err
		}
	}

	target.report(Progress{Stage: "补全游戏文件", Current: loaderID}, true)
	if err := target.EnsureInstalled(ctx, loaderID); err != nil {
		cleanup()
		return "", "", fmt.Errorf("补全实例文件: %w", err)
	}

	marker, err := json.Marshal(map[string]any{
		"name":      name,
		"versionId": loaderID,
		"mcVersion": mcVersion,
		"loader":    loaderID,
		"source":    "wizard",
	})
	if err != nil {
		cleanup()
		return "", "", err
	}
	if err := os.WriteFile(filepath.Join(VersionPath(root, loaderID), "mctl-pack.json"), marker, 0o644); err != nil {
		cleanup()
		return "", "", err
	}
	settings := &InstanceSettings{Name: name, JavaPath: strings.TrimSpace(opts.JavaPath)}
	if err := SaveInstanceSettings(root, loaderID, settings); err != nil {
		cleanup()
		return "", "", err
	}
	target.report(Progress{Stage: "实例创建完成", Done: 1, Total: 1, Current: loaderID}, true)
	i.logf("实例已创建: %s", root)
	return loaderID, root, nil
}

func wizardLoaderAllowed(kind string) bool {
	for _, k := range WizardLoaders {
		if kind == k {
			return true
		}
	}
	return false
}

// installLoaderInto installs a loader profile or runs a loader installer in
// target's game directory. An empty loaderVersion picks the newest stable
// build. It returns the resulting version ID.
func (i *Installer) installLoaderInto(ctx context.Context, target *Installer, kind, mcVersion, loaderVersion, javaPath string, autoJava bool) (string, error) {
	version := strings.TrimSpace(loaderVersion)
	switch kind {
	case "fabric", "quilt", "legacyfabric":
		if version == "" {
			var err error
			version, err = i.resolveLoaderVersion(ctx, kind, mcVersion)
			if err != nil {
				return "", err
			}
			i.logf("已选择 %s %s", kind, version)
		}
		profile, err := fetchLoaderProfile(ctx, i.HTTP, kind, mcVersion, version)
		if err != nil {
			return "", err
		}
		var p VersionJSON
		if err := json.Unmarshal(profile, &p); err != nil {
			return "", fmt.Errorf("解析 %s 配置: %w", kind, err)
		}
		if p.ID == "" {
			return "", fmt.Errorf("%s 配置缺少版本 ID", kind)
		}
		target.report(Progress{Stage: "安装 " + kind, Current: version}, true)
		if err := target.InstallProfile(ctx, p.ID, profile); err != nil {
			return "", err
		}
		return p.ID, nil
	case "forge", "neoforge":
		if version == "" {
			builds, err := FetchLoaderBuilds(ctx, i.HTTP, kind, mcVersion)
			if err != nil {
				return "", err
			}
			if len(builds) == 0 {
				return "", fmt.Errorf("%s 没有适用于 %s 的构建", kind, mcVersion)
			}
			version = builds[0].Version
			i.logf("已选择 %s %s", kind, version)
		}
		required := InferJavaMajor(mcVersion)
		var javaInfo *JavaInfo
		var err error
		if autoJava {
			javaInfo, err = i.EnsureJava(ctx, required, javaPath)
		} else {
			javaInfo, err = FindJavaFor(required, javaPath)
		}
		if err != nil {
			return "", fmt.Errorf("需要 Java %d 运行 %s 安装器: %w", required, kind, err)
		}
		return target.InstallLoader(ctx, kind, mcVersion, version, javaInfo)
	}
	return "", fmt.Errorf("不支持的加载器: %s", kind)
}

// AddLoader installs an additional loader into an existing instance, repairs
// its files and keeps the pack marker in sync.
func (i *Installer) AddLoader(ctx context.Context, kind, mcVersion, loaderVersion, javaPath string, autoJava bool) (string, error) {
	kind = strings.ToLower(strings.TrimSpace(kind))
	mcVersion = strings.TrimSpace(mcVersion)
	if mcVersion == "" {
		return "", fmt.Errorf("缺少 Minecraft 版本")
	}
	if kind == "" || kind == "vanilla" {
		return "", fmt.Errorf("请选择要安装的加载器")
	}
	if !wizardLoaderAllowed(kind) {
		return "", fmt.Errorf("不支持的加载器: %s", kind)
	}
	if _, err := ResolveVersion(i.GameDir, mcVersion); err != nil {
		return "", fmt.Errorf("实例缺少基础版本 %s: %w", mcVersion, err)
	}
	id, err := i.installLoaderInto(ctx, i, kind, mcVersion, loaderVersion, javaPath, autoJava)
	if err != nil {
		return "", err
	}
	if err := i.EnsureInstalled(ctx, id); err != nil {
		return "", fmt.Errorf("补全加载器文件: %w", err)
	}
	i.updatePackMarker(mcVersion, id)
	if settings, err := LoadInstanceSettings(i.GameDir, id); err == nil && settings.Name == "" {
		if marker, _, err := findPackMarker(i.GameDir); err == nil && marker != nil && marker.Name != "" {
			settings.Name = marker.Name
			_ = SaveInstanceSettings(i.GameDir, id, settings)
		}
	}
	i.logf("加载器安装完成: %s", id)
	return id, nil
}

// updatePackMarker rewrites the pack marker so the new loader profile stays
// visible under instances/, and removes the stale marker.
func (i *Installer) updatePackMarker(mcVersion, loaderID string) {
	marker, oldID, err := findPackMarker(i.GameDir)
	if err != nil || marker == nil {
		return
	}
	marker.VersionID = loaderID
	marker.MCVersion = mcVersion
	marker.Loader = loaderID
	data, err := json.Marshal(marker)
	if err != nil {
		return
	}
	if err := os.WriteFile(filepath.Join(VersionPath(i.GameDir, loaderID), "mctl-pack.json"), data, 0o644); err != nil {
		return
	}
	if oldID != "" && oldID != loaderID {
		_ = os.Remove(filepath.Join(VersionPath(i.GameDir, oldID), "mctl-pack.json"))
	}
}

func findPackMarker(root string) (*packMarkerFile, string, error) {
	entries, err := os.ReadDir(filepath.Join(root, "versions"))
	if err != nil {
		return nil, "", err
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(root, "versions", e.Name(), "mctl-pack.json"))
		if err != nil {
			continue
		}
		var marker packMarkerFile
		if err := json.Unmarshal(data, &marker); err != nil {
			continue
		}
		return &marker, e.Name(), nil
	}
	return nil, "", nil
}

func fetchLoaderProfile(ctx context.Context, hc *HTTPClient, kind, mcVersion, loaderVersion string) ([]byte, error) {
	switch kind {
	case "fabric":
		return FetchFabricProfile(ctx, hc, mcVersion, loaderVersion)
	case "quilt":
		return FetchQuiltProfile(ctx, hc, mcVersion, loaderVersion)
	case "legacyfabric":
		return FetchLegacyFabricProfile(ctx, hc, mcVersion, loaderVersion)
	}
	return nil, fmt.Errorf("不支持的加载器: %s", kind)
}

func (i *Installer) resolveLoaderVersion(ctx context.Context, kind, mcVersion string) (string, error) {
	switch kind {
	case "fabric":
		loaders, err := FetchFabricLoaders(ctx, i.HTTP, mcVersion)
		if err != nil {
			return "", err
		}
		for _, l := range loaders {
			if l.Stable {
				return l.Version, nil
			}
		}
		if len(loaders) > 0 {
			return loaders[0].Version, nil
		}
	case "quilt":
		builds, err := FetchQuiltLoaders(ctx, i.HTTP, mcVersion)
		if err != nil {
			return "", err
		}
		if len(builds) > 0 {
			return builds[0].Version, nil
		}
	case "legacyfabric":
		builds, err := FetchLegacyFabricLoaders(ctx, i.HTTP, mcVersion)
		if err != nil {
			return "", err
		}
		if len(builds) > 0 {
			return builds[0].Version, nil
		}
	case "forge", "neoforge":
		builds, err := FetchLoaderBuilds(ctx, i.HTTP, kind, mcVersion)
		if err != nil {
			return "", err
		}
		if len(builds) > 0 {
			return builds[0].Version, nil
		}
	}
	return "", fmt.Errorf("%s 没有适用于 %s 的构建", kind, mcVersion)
}
