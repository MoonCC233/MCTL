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
	switch kind {
	case "vanilla":
	case "fabric", "quilt", "legacyfabric":
		version := strings.TrimSpace(opts.LoaderVersion)
		if version == "" {
			version, err = i.resolveLoaderVersion(ctx, kind, mcVersion)
			if err != nil {
				cleanup()
				return "", "", err
			}
			i.logf("已选择 %s %s", kind, version)
		}
		profile, err := fetchLoaderProfile(ctx, i.HTTP, kind, mcVersion, version)
		if err != nil {
			cleanup()
			return "", "", err
		}
		var p VersionJSON
		if err := json.Unmarshal(profile, &p); err != nil {
			cleanup()
			return "", "", fmt.Errorf("解析 %s 配置: %w", kind, err)
		}
		if p.ID == "" {
			cleanup()
			return "", "", fmt.Errorf("%s 配置缺少版本 ID", kind)
		}
		target.report(Progress{Stage: "安装 " + kind, Current: version}, true)
		if err := target.InstallProfile(ctx, p.ID, profile); err != nil {
			cleanup()
			return "", "", err
		}
		loaderID = p.ID
	case "forge", "neoforge":
		version := strings.TrimSpace(opts.LoaderVersion)
		if version == "" {
			builds, err := FetchLoaderBuilds(ctx, i.HTTP, kind, mcVersion)
			if err != nil {
				cleanup()
				return "", "", err
			}
			if len(builds) == 0 {
				cleanup()
				return "", "", fmt.Errorf("%s 没有适用于 %s 的构建", kind, mcVersion)
			}
			version = builds[0].Version
			i.logf("已选择 %s %s", kind, version)
		}
		required := InferJavaMajor(mcVersion)
		var javaInfo *JavaInfo
		if opts.AutoJava {
			javaInfo, err = i.EnsureJava(ctx, required, opts.JavaPath)
		} else {
			javaInfo, err = FindJavaFor(required, opts.JavaPath)
		}
		if err != nil {
			cleanup()
			return "", "", fmt.Errorf("需要 Java %d 运行 %s 安装器: %w", required, kind, err)
		}
		loaderID, err = target.InstallLoader(ctx, kind, mcVersion, version, javaInfo)
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
