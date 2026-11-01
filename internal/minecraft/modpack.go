package minecraft

import (
	"archive/zip"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

type modpackIndex struct {
	FormatVersion int               `json:"formatVersion"`
	Game          string            `json:"game"`
	VersionID     string            `json:"versionId"`
	Name          string            `json:"name"`
	Files         []modpackFile     `json:"files"`
	Dependencies  map[string]string `json:"dependencies"`
}

type modpackFile struct {
	Path      string            `json:"path"`
	Hashes    map[string]string `json:"hashes"`
	Env       map[string]string `json:"env"`
	Downloads []string          `json:"downloads"`
	FileSize  int64             `json:"fileSize"`
}

type curseLoader struct {
	ID      string `json:"id"`
	Primary bool   `json:"primary"`
}

type curseFile struct {
	ProjectID int  `json:"projectID"`
	FileID    int  `json:"fileID"`
	Required  bool `json:"required"`
}

type curseForgeManifest struct {
	Minecraft struct {
		Version    string        `json:"version"`
		ModLoaders []curseLoader `json:"modLoaders"`
	} `json:"minecraft"`
	Name    string      `json:"name"`
	Version string      `json:"version"`
	Files   []curseFile `json:"files"`
}

var (
	curseForgeAPIBase    = "https://api.curseforge.com/v1"
	curseForgeHTTPClient = &http.Client{Timeout: 45 * time.Second}
)

// ImportModpack installs a Modrinth .mrpack or CurseForge zip into an isolated
// instance under <gameDir>/instances/<pack>. It returns the generated version ID.
func (i *Installer) ImportModpack(ctx context.Context, mrpackPath, javaPath string, autoJava bool, curseForgeAPIKey string) (string, error) {
	zr, err := zip.OpenReader(mrpackPath)
	if err != nil {
		return "", fmt.Errorf("打开整合包: %w", err)
	}
	defer zr.Close()

	var modrinthData, curseData []byte
	for _, f := range zr.File {
		switch f.Name {
		case "modrinth.index.json":
			modrinthData, err = readZipEntry(f)
		case "manifest.json":
			curseData, err = readZipEntry(f)
		}
		if err != nil {
			return "", err
		}
	}
	if modrinthData == nil && curseData != nil {
		return i.importCurseForge(ctx, zr, curseData, javaPath, autoJava, curseForgeAPIKey)
	}
	if modrinthData == nil {
		return "", fmt.Errorf("无法识别的整合包：缺少 modrinth.index.json 或 manifest.json")
	}
	var index modpackIndex
	if err := json.Unmarshal(modrinthData, &index); err != nil {
		return "", fmt.Errorf("解析 modrinth.index.json: %w", err)
	}
	if index.Game != "" && index.Game != "minecraft" {
		return "", fmt.Errorf("不支持的整合包类型: %s", index.Game)
	}
	mcVersion := strings.TrimSpace(index.Dependencies["minecraft"])
	if mcVersion == "" {
		return "", fmt.Errorf("整合包没有声明 Minecraft 版本")
	}

	slug := packSlug(index.Name, index.VersionID)
	packID := "pack-" + slug
	instanceRoot := filepath.Join(i.GameDir, "instances", slug)
	if err := os.MkdirAll(instanceRoot, 0o755); err != nil {
		return "", err
	}
	i.logf("导入整合包 %s（Minecraft %s）到 instances/%s", index.Name, mcVersion, slug)
	target := i.forGameDir(instanceRoot)

	target.report(Progress{Stage: "安装基础版本", Current: mcVersion}, true)
	if err := target.InstallVersion(ctx, mcVersion, nil); err != nil {
		return "", fmt.Errorf("安装基础版本 %s: %w", mcVersion, err)
	}

	loaderID, err := target.installPackLoader(ctx, index.Dependencies, mcVersion, javaPath, autoJava)
	if err != nil {
		return "", err
	}
	if loaderID == "" {
		loaderID = mcVersion
	}

	target.report(Progress{Stage: "写入整合包配置", Current: packID}, true)
	profile, _ := json.Marshal(map[string]any{
		"id":           packID,
		"inheritsFrom": loaderID,
		"type":         "release",
	})
	if err := writeVersionJSON(instanceRoot, packID, profile); err != nil {
		return "", err
	}

	tasks, err := modpackTasks(&index, instanceRoot)
	if err != nil {
		return "", err
	}
	target.report(Progress{Stage: "下载整合包文件", Done: 0, Total: len(tasks)}, true)
	if err := target.downloadAll(ctx, tasks, "下载整合包文件"); err != nil {
		return "", err
	}
	target.report(Progress{Stage: "解压整合包覆盖文件", Current: slug}, true)
	if err := extractModpackOverrides(zr, instanceRoot); err != nil {
		return "", err
	}
	if err := target.EnsureInstalled(ctx, packID); err != nil {
		return "", fmt.Errorf("补全整合包版本: %w", err)
	}

	marker, _ := json.Marshal(map[string]any{
		"name":      index.Name,
		"versionId": index.VersionID,
		"mcVersion": mcVersion,
		"loader":    loaderID,
	})
	if err := os.WriteFile(filepath.Join(VersionPath(instanceRoot, packID), "mctl-pack.json"), marker, 0o644); err != nil {
		return "", err
	}
	i.report(Progress{Stage: "整合包安装完成", Done: 1, Total: 1, Current: packID}, true)
	i.logf("整合包安装完成: %s", packID)
	return packID, nil
}

func readZipEntry(f *zip.File) ([]byte, error) {
	rc, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return io.ReadAll(io.LimitReader(rc, 32<<20))
}

func (i *Installer) importCurseForge(ctx context.Context, zr *zip.ReadCloser, data []byte, javaPath string, autoJava bool, apiKey string) (string, error) {
	if strings.TrimSpace(apiKey) == "" {
		return "", fmt.Errorf("导入 CurseForge 整合包需要 API Key，请先在设置中填写")
	}
	var m curseForgeManifest
	if err := json.Unmarshal(data, &m); err != nil {
		return "", fmt.Errorf("解析 CurseForge manifest.json: %w", err)
	}
	mcVersion := strings.TrimSpace(m.Minecraft.Version)
	if mcVersion == "" {
		return "", fmt.Errorf("CurseForge 整合包没有声明 Minecraft 版本")
	}
	kind, loaderVersion := parseCurseLoader(m.Minecraft.ModLoaders)
	slug := packSlug(m.Name, m.Version)
	packID := "pack-" + slug
	instanceRoot := filepath.Join(i.GameDir, "instances", slug)
	if err := os.MkdirAll(instanceRoot, 0o755); err != nil {
		return "", err
	}
	i.logf("导入 CurseForge 整合包 %s（Minecraft %s）到 instances/%s", m.Name, mcVersion, slug)
	target := i.forGameDir(instanceRoot)

	target.report(Progress{Stage: "安装基础版本", Current: mcVersion}, true)
	if err := target.InstallVersion(ctx, mcVersion, nil); err != nil {
		return "", fmt.Errorf("安装基础版本 %s: %w", mcVersion, err)
	}
	loaderID := mcVersion
	if kind != "" {
		deps := map[string]string{}
		switch kind {
		case "fabric":
			deps["fabric-loader"] = loaderVersion
		case "quilt":
			deps["quilt-loader"] = loaderVersion
		case "forge":
			deps["forge"] = loaderVersion
		case "neoforge":
			deps["neoforge"] = loaderVersion
		}
		var err error
		loaderID, err = target.installPackLoader(ctx, deps, mcVersion, javaPath, autoJava)
		if err != nil {
			return "", err
		}
		if loaderID == "" {
			loaderID = mcVersion
		}
	}

	target.report(Progress{Stage: "写入整合包配置", Current: packID}, true)
	profile, _ := json.Marshal(map[string]any{
		"id":           packID,
		"inheritsFrom": loaderID,
		"type":         "release",
	})
	if err := writeVersionJSON(instanceRoot, packID, profile); err != nil {
		return "", err
	}

	tasks, err := target.curseForgeTasks(ctx, m.Files, apiKey, instanceRoot)
	if err != nil {
		return "", err
	}
	target.report(Progress{Stage: "下载 CurseForge 文件", Done: 0, Total: len(tasks)}, true)
	if err := target.downloadAll(ctx, tasks, "下载 CurseForge 文件"); err != nil {
		return "", err
	}
	target.report(Progress{Stage: "解压整合包覆盖文件", Current: slug}, true)
	if err := extractModpackOverrides(zr, instanceRoot); err != nil {
		return "", err
	}
	if err := target.EnsureInstalled(ctx, packID); err != nil {
		return "", fmt.Errorf("补全整合包版本: %w", err)
	}

	marker, _ := json.Marshal(map[string]any{
		"name":      m.Name,
		"versionId": m.Version,
		"mcVersion": mcVersion,
		"loader":    loaderID,
		"source":    "curseforge",
	})
	if err := os.WriteFile(filepath.Join(VersionPath(instanceRoot, packID), "mctl-pack.json"), marker, 0o644); err != nil {
		return "", err
	}
	i.report(Progress{Stage: "整合包安装完成", Done: 1, Total: 1, Current: packID}, true)
	i.logf("整合包安装完成: %s", packID)
	return packID, nil
}

func parseCurseLoader(mods []curseLoader) (kind, version string) {
	for _, m := range mods {
		id := strings.TrimSpace(m.ID)
		low := strings.ToLower(id)
		for _, k := range []string{"neoforge", "forge", "fabric", "quilt"} {
			if !strings.HasPrefix(low, k+"-") {
				continue
			}
			v := id[len(k)+1:]
			return k, v
		}
	}
	return "", ""
}

func (i *Installer) curseForgeTasks(ctx context.Context, files []curseFile, apiKey, root string) ([]fileTask, error) {
	var tasks []fileTask
	seen := map[string]bool{}
	for _, f := range files {
		raw, err := curseForgeDownloadURL(ctx, apiKey, f.ProjectID, f.FileID)
		if err != nil {
			return nil, err
		}
		name := fileNameFromURL(raw, f.ProjectID, f.FileID)
		dest, err := safeJoin(root, path.Join("mods", name))
		if err != nil {
			return nil, err
		}
		if seen[dest] {
			continue
		}
		seen[dest] = true
		tasks = append(tasks, fileTask{URL: raw, Dest: dest, Label: name})
	}
	return tasks, nil
}

func curseForgeDownloadURL(ctx context.Context, apiKey string, projectID, fileID int) (string, error) {
	endpoint := fmt.Sprintf("%s/mods/%d/files/%d/download-url", strings.TrimRight(curseForgeAPIBase, "/"), projectID, fileID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", UserAgent)
	resp, err := curseForgeHTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("获取 CurseForge 下载地址: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("获取 CurseForge 下载地址: HTTP %d %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var out struct {
		Data string `json:"data"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return "", fmt.Errorf("解析 CurseForge 下载地址: %w", err)
	}
	if strings.TrimSpace(out.Data) == "" {
		return "", fmt.Errorf("CurseForge 文件 %d/%d 没有可用下载地址", projectID, fileID)
	}
	return out.Data, nil
}

func fileNameFromURL(raw string, projectID, fileID int) string {
	if u, err := url.Parse(raw); err == nil && (u.Scheme == "http" || u.Scheme == "https") {
		name := path.Base(u.Path)
		if name != "" && name != "." && name != "/" {
			return name
		}
	}
	return fmt.Sprintf("%d-%d.jar", projectID, fileID)
}

func (i *Installer) forGameDir(dir string) *Installer {
	return &Installer{
		GameDir:  dir,
		Source:   i.Source,
		HTTP:     i.HTTP,
		Workers:  i.Workers,
		Log:      i.Log,
		Progress: i.Progress,
	}
}

func (i *Installer) installPackLoader(ctx context.Context, deps map[string]string, mcVersion, javaPath string, autoJava bool) (string, error) {
	if v := strings.TrimSpace(deps["fabric-loader"]); v != "" {
		i.report(Progress{Stage: "安装 Fabric", Current: v}, true)
		profile, err := FetchFabricProfile(ctx, i.HTTP, mcVersion, v)
		if err != nil {
			return "", err
		}
		var p VersionJSON
		if err := json.Unmarshal(profile, &p); err != nil {
			return "", fmt.Errorf("解析 Fabric 配置: %w", err)
		}
		if p.ID == "" {
			return "", fmt.Errorf("Fabric 配置缺少版本 ID")
		}
		if err := i.InstallProfile(ctx, p.ID, profile); err != nil {
			return "", err
		}
		return p.ID, nil
	}

	if v := strings.TrimSpace(deps["quilt-loader"]); v != "" {
		i.report(Progress{Stage: "安装 Quilt", Current: v}, true)
		profile, err := FetchQuiltProfile(ctx, i.HTTP, mcVersion, v)
		if err != nil {
			return "", err
		}
		var p VersionJSON
		if err := json.Unmarshal(profile, &p); err != nil {
			return "", fmt.Errorf("解析 Quilt 配置: %w", err)
		}
		if p.ID == "" {
			return "", fmt.Errorf("Quilt 配置缺少版本 ID")
		}
		if err := i.InstallProfile(ctx, p.ID, profile); err != nil {
			return "", err
		}
		return p.ID, nil
	}

	kind := ""
	loaderVersion := ""
	if v := strings.TrimSpace(deps["forge"]); v != "" {
		kind = "forge"
		loaderVersion = strings.TrimPrefix(v, mcVersion+"-")
	} else if v := strings.TrimSpace(deps["neoforge"]); v != "" {
		kind = "neoforge"
		loaderVersion = v
	}
	if kind == "" {
		return "", nil
	}

	i.report(Progress{Stage: "准备 " + kind + " 安装器", Current: loaderVersion}, true)
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
	return i.InstallLoader(ctx, kind, mcVersion, loaderVersion, javaInfo)
}

func modpackTasks(index *modpackIndex, root string) ([]fileTask, error) {
	var tasks []fileTask
	for _, f := range index.Files {
		if strings.EqualFold(f.Env["client"], "unsupported") {
			continue
		}
		if len(f.Downloads) == 0 || f.Downloads[0] == "" {
			continue
		}
		dest, err := safeJoin(root, f.Path)
		if err != nil {
			return nil, fmt.Errorf("整合包文件路径无效 %q: %w", f.Path, err)
		}
		sha := ""
		if f.Hashes != nil {
			sha = f.Hashes["sha1"]
		}
		tasks = append(tasks, fileTask{
			URL:   f.Downloads[0],
			Dest:  dest,
			SHA1:  sha,
			Size:  f.FileSize,
			Label: filepath.Base(f.Path),
		})
	}
	return tasks, nil
}

func extractModpackOverrides(zr *zip.ReadCloser, root string) error {
	for _, f := range zr.File {
		name := f.Name
		prefix := ""
		switch {
		case strings.HasPrefix(name, "overrides/"):
			prefix = "overrides/"
		case strings.HasPrefix(name, "client-overrides/"):
			prefix = "client-overrides/"
		default:
			continue
		}
		rel := strings.TrimPrefix(name, prefix)
		if rel == "" || f.FileInfo().IsDir() {
			continue
		}
		dest, err := safeJoin(root, rel)
		if err != nil {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		out, err := os.Create(dest)
		if err != nil {
			rc.Close()
			return err
		}
		_, cerr := io.Copy(out, rc)
		rc.Close()
		out.Close()
		if cerr != nil {
			return cerr
		}
	}
	return nil
}

func safeJoin(root, rel string) (string, error) {
	rel = filepath.FromSlash(rel)
	if rel == "" || filepath.IsAbs(rel) || strings.HasPrefix(rel, "/") || strings.HasPrefix(rel, `\`) {
		return "", fmt.Errorf("空路径或绝对路径")
	}
	p := filepath.Join(root, rel)
	rp, err := filepath.Rel(root, p)
	if err != nil || rp == ".." || strings.HasPrefix(rp, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("路径越界")
	}
	return p, nil
}

func packSlug(name, versionID string) string {
	var b strings.Builder
	lastDash := false
	for _, r := range strings.ToLower(name) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '_':
			b.WriteRune(r)
			lastDash = false
		case r == '-' || r == ' ' || r == '.':
			if !lastDash && b.Len() > 0 {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}
	slug := strings.Trim(b.String(), "-")
	if slug == "" {
		slug = "pack"
	}
	if len(slug) > 32 {
		slug = slug[:32]
	}
	sum := sha1.Sum([]byte(name + "\x00" + versionID))
	return slug + "-" + hex.EncodeToString(sum[:4])
}
