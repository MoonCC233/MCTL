package minecraft

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var (
	optiFineAPI       = "https://bmclapi2.bangbang93.com/optifine"
	optiFineMavenBase = "https://bmclapi2.bangbang93.com/maven/com/optifine"
)

type optiFineEntry struct {
	MCVersion string `json:"mcversion"`
	Patch     string `json:"patch"`
	Type      string `json:"type"`
	Filename  string `json:"filename"`
	Forge     string `json:"forge"`
}

// FetchOptiFineBuilds lists OptiFine builds for a Minecraft version. The
// BMCLAPI catalogue is used because optifine.net has no stable JSON API.
func FetchOptiFineBuilds(ctx context.Context, hc *HTTPClient, mcVersion string) ([]LoaderBuild, error) {
	u := fmt.Sprintf("%s/%s", strings.TrimRight(optiFineAPI, "/"), url.PathEscape(mcVersion))
	data, err := hc.Get(ctx, u)
	if err != nil {
		return nil, fmt.Errorf("获取 OptiFine 版本列表: %w", err)
	}
	var raw []optiFineEntry
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("解析 OptiFine 版本列表: %w", err)
	}
	var out []LoaderBuild
	for _, e := range raw {
		if e.Filename == "" {
			continue
		}
		label := strings.TrimSpace(e.Type + " " + e.Patch)
		if label == "" {
			label = strings.TrimSuffix(e.Filename, ".jar")
		}
		out = append(out, LoaderBuild{
			Version:   label,
			MCVersion: mcVersion,
			Kind:      "optifine",
			Stable:    !strings.HasPrefix(strings.ToLower(e.Patch), "pre"),
			Filename:  e.Filename,
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Stable != out[j].Stable {
			return out[i].Stable
		}
		return out[i].Filename > out[j].Filename
	})
	return out, nil
}

// InstallOptiFine downloads an OptiFine installer jar, extracts its embedded
// version profile, and places the jar at the Maven path referenced by it.
func (i *Installer) InstallOptiFine(ctx context.Context, mcVersion, filename string) (string, error) {
	if strings.TrimSpace(filename) == "" {
		return "", fmt.Errorf("缺少 OptiFine 文件名")
	}
	cacheDir := filepath.Join(i.GameDir, "mctl-installers")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return "", err
	}
	jarPath := filepath.Join(cacheDir, filepath.Base(filename))
	if st, err := os.Stat(jarPath); err != nil || st.Size() == 0 {
		u := fmt.Sprintf("%s/%s/%s", strings.TrimRight(optiFineMavenBase, "/"), url.PathEscape(mcVersion), url.PathEscape(filename))
		i.report(Progress{Stage: "下载 OptiFine", Current: filename}, true)
		i.logf("下载 OptiFine %s", filename)
		if err := i.HTTP.DownloadFile(ctx, u, jarPath, "", nil); err != nil {
			return "", fmt.Errorf("下载 OptiFine: %w", err)
		}
	}

	i.report(Progress{Stage: "读取 OptiFine 配置", Current: filename}, true)
	profile, optiID, err := readOptiFineProfile(jarPath, mcVersion, filename)
	if err != nil {
		return "", err
	}
	id := strings.TrimSpace(optiID)
	if id == "" {
		id = mcVersion + "-" + strings.TrimSuffix(filepath.Base(filename), ".jar")
	}
	libVersion := strings.TrimPrefix(strings.TrimSuffix(filepath.Base(filename), ".jar"), "OptiFine_")
	coord := MavenCoord{Group: "optifine", Artifact: "OptiFine", Version: libVersion, Ext: "jar"}
	libPath := filepath.Join(i.GameDir, "libraries", filepath.FromSlash(coord.Path()))
	if err := os.MkdirAll(filepath.Dir(libPath), 0o755); err != nil {
		return "", err
	}
	if err := copyFile(jarPath, libPath); err != nil {
		return "", fmt.Errorf("写入 OptiFine 库: %w", err)
	}

	profile, err = ensureOptiFineLibrary(profile, coord)
	if err != nil {
		return "", err
	}
	if err := i.InstallProfile(ctx, id, profile); err != nil {
		return "", err
	}
	return id, nil
}

func readOptiFineProfile(jarPath, mcVersion, filename string) ([]byte, string, error) {
	zr, err := zip.OpenReader(jarPath)
	if err != nil {
		return nil, "", fmt.Errorf("打开 OptiFine 安装器: %w", err)
	}
	defer zr.Close()
	for _, f := range zr.File {
		if f.Name != "version.json" {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return nil, "", err
		}
		data, err := io.ReadAll(io.LimitReader(rc, 16<<20))
		rc.Close()
		if err != nil {
			return nil, "", err
		}
		var v VersionJSON
		if err := json.Unmarshal(data, &v); err != nil {
			return nil, "", fmt.Errorf("解析 OptiFine version.json: %w", err)
		}
		id := v.ID
		if id == "" {
			id = mcVersion + "-" + strings.TrimSuffix(filepath.Base(filename), ".jar")
		}
		if v.InheritsFrom == "" {
			v.InheritsFrom = mcVersion
		}
		if v.ID == "" {
			v.ID = id
		}
		if v.Type == "" {
			v.Type = "release"
		}
		normalized, err := json.Marshal(v)
		if err != nil {
			return nil, "", err
		}
		return normalized, id, nil
	}
	return nil, "", fmt.Errorf("OptiFine 安装器缺少 version.json")
}

func ensureOptiFineLibrary(profile []byte, coord MavenCoord) ([]byte, error) {
	var raw map[string]any
	if err := json.Unmarshal(profile, &raw); err != nil {
		return nil, err
	}
	libs, _ := raw["libraries"].([]any)
	for _, item := range libs {
		lib, _ := item.(map[string]any)
		name, _ := lib["name"].(string)
		if strings.HasPrefix(strings.ToLower(name), "optifine:optifine:") {
			return profile, nil
		}
	}
	lib := map[string]any{
		"name": coord.Group + ":" + coord.Artifact + ":" + coord.Version,
	}
	raw["libraries"] = append(libs, lib)
	return json.Marshal(raw)
}
