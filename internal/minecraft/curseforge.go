package minecraft

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const curseForgeModpackClassID = 4471

// CurseForgeMod is a mod or modpack project from the CurseForge API.
type CurseForgeMod struct {
	ID            int    `json:"id"`
	Name          string `json:"name"`
	Slug          string `json:"slug"`
	Summary       string `json:"summary"`
	DownloadCount int64  `json:"downloadCount"`
	ClassID       int    `json:"classId"`
	Authors       []struct {
		Name string `json:"name"`
	} `json:"authors"`
	LatestFiles []CurseForgeFile `json:"latestFiles"`
}

// CurseForgeFile is one downloadable file of a CurseForge project.
type CurseForgeFile struct {
	ID          int    `json:"id"`
	DisplayName string `json:"displayName"`
	FileName    string `json:"fileName"`
	ReleaseType int    `json:"releaseType"` // 1 release, 2 beta, 3 alpha
	FileDate    string `json:"fileDate"`
	DownloadURL string `json:"downloadUrl"`
	FileLength  int64  `json:"fileLength"`
}

// SearchCurseForgeModpacks searches the CurseForge modpack catalogue.
func SearchCurseForgeModpacks(ctx context.Context, hc *HTTPClient, apiKey, query, gameVersion string, limit int) ([]CurseForgeMod, error) {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return nil, fmt.Errorf("使用 CurseForge 需要在设置中填写 API Key")
	}
	if limit <= 0 {
		limit = 20
	}
	q := url.Values{}
	q.Set("gameId", "432")
	q.Set("classId", strconv.Itoa(curseForgeModpackClassID))
	q.Set("searchFilter", strings.TrimSpace(query))
	q.Set("sortField", "2")
	q.Set("sortOrder", "desc")
	q.Set("pageSize", strconv.Itoa(limit))
	if v := strings.TrimSpace(gameVersion); v != "" {
		q.Set("gameVersion", v)
	}
	u := strings.TrimRight(curseForgeAPIBase, "/") + "/mods/search?" + q.Encode()
	data, err := curseForgeGet(ctx, hc, apiKey, u)
	if err != nil {
		return nil, fmt.Errorf("搜索 CurseForge 整合包: %w", err)
	}
	var resp struct {
		Data []CurseForgeMod `json:"data"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("解析 CurseForge 搜索结果: %w", err)
	}
	return resp.Data, nil
}

// LatestCurseForgeFile picks a downloadable release from a project, falling
// back to beta/alpha files when no release is available.
func LatestCurseForgeFile(mod CurseForgeMod) (CurseForgeFile, error) {
	best := -1
	for i, f := range mod.LatestFiles {
		if f.DownloadURL == "" {
			continue
		}
		if best == -1 {
			best = i
			continue
		}
		if f.ReleaseType < mod.LatestFiles[best].ReleaseType {
			best = i
		}
	}
	if best == -1 {
		return CurseForgeFile{}, fmt.Errorf("%s 没有通过 API 开放的下载文件", mod.Name)
	}
	return mod.LatestFiles[best], nil
}

// DownloadCurseForgeModpack downloads the newest modpack archive to destDir.
func DownloadCurseForgeModpack(ctx context.Context, hc *HTTPClient, apiKey string, mod CurseForgeMod, destDir string, onBytes func(done, total int64)) (string, CurseForgeFile, error) {
	file, err := LatestCurseForgeFile(mod)
	if err != nil {
		return "", CurseForgeFile{}, err
	}
	raw, err := curseForgeDownloadURL(ctx, apiKey, mod.ID, file.ID)
	if err != nil {
		return "", CurseForgeFile{}, err
	}
	name := file.FileName
	if name == "" {
		name = fmt.Sprintf("%d-%d.zip", mod.ID, file.ID)
	}
	name = filepath.Base(name)
	if !strings.HasSuffix(strings.ToLower(name), ".zip") {
		name += ".zip"
	}
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return "", CurseForgeFile{}, err
	}
	dest := filepath.Join(destDir, name)
	if err := hc.DownloadFile(ctx, raw, dest, "", onBytes); err != nil {
		return "", CurseForgeFile{}, fmt.Errorf("下载 CurseForge 整合包: %w", err)
	}
	return dest, file, nil
}

// GetCurseForgeMod fetches a project by numeric ID.
func GetCurseForgeMod(ctx context.Context, hc *HTTPClient, apiKey string, id int) (CurseForgeMod, error) {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return CurseForgeMod{}, fmt.Errorf("使用 CurseForge 需要在设置中填写 API Key")
	}
	if id <= 0 {
		return CurseForgeMod{}, fmt.Errorf("CurseForge 项目 ID 无效")
	}
	u := fmt.Sprintf("%s/mods/%d", strings.TrimRight(curseForgeAPIBase, "/"), id)
	data, err := curseForgeGet(ctx, hc, apiKey, u)
	if err != nil {
		return CurseForgeMod{}, fmt.Errorf("读取 CurseForge 项目: %w", err)
	}
	var resp struct {
		Data CurseForgeMod `json:"data"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return CurseForgeMod{}, fmt.Errorf("解析 CurseForge 项目: %w", err)
	}
	return resp.Data, nil
}

// CurseForgeModFiles lists the downloadable files of a project, newest first.
func CurseForgeModFiles(ctx context.Context, hc *HTTPClient, apiKey string, id int, gameVersion string) ([]CurseForgeFile, error) {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return nil, fmt.Errorf("使用 CurseForge 需要在设置中填写 API Key")
	}
	q := url.Values{}
	q.Set("pageSize", "50")
	if v := strings.TrimSpace(gameVersion); v != "" {
		q.Set("gameVersion", v)
	}
	u := fmt.Sprintf("%s/mods/%d/files?%s", strings.TrimRight(curseForgeAPIBase, "/"), id, q.Encode())
	data, err := curseForgeGet(ctx, hc, apiKey, u)
	if err != nil {
		return nil, fmt.Errorf("读取 CurseForge 文件列表: %w", err)
	}
	var resp struct {
		Data []CurseForgeFile `json:"data"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("解析 CurseForge 文件列表: %w", err)
	}
	return resp.Data, nil
}

// ParseCurseForgeID extracts a project ID from a number or project URL.
func ParseCurseForgeID(raw string) (int, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return 0, fmt.Errorf("请输入 CurseForge 项目 ID")
	}
	if i, err := strconv.Atoi(s); err == nil && i > 0 {
		return i, nil
	}
	segments := strings.Split(strings.TrimRight(s, "/"), "/")
	for i, seg := range segments {
		switch seg {
		case "modpacks", "mc-mods", "texture-packs", "worlds", "customization":
			if i+1 < len(segments) {
				slug := segments[i+1]
				if j := strings.LastIndex(slug, "-"); j > 0 {
					if n, err := strconv.Atoi(slug[j+1:]); err == nil && n > 0 {
						return n, nil
					}
				}
				return 0, fmt.Errorf("链接中的项目名无法解析为 ID，请使用带数字后缀的项目链接或直接填写项目 ID")
			}
		case "files":
			return 0, fmt.Errorf("这是文件链接，请改用项目页链接或项目 ID")
		}
	}
	if last := segments[len(segments)-1]; last != "" {
		if j := strings.LastIndex(last, "-"); j > 0 {
			if n, err := strconv.Atoi(last[j+1:]); err == nil && n > 0 {
				return n, nil
			}
		}
	}
	return 0, fmt.Errorf("无法从 %q 解析 CurseForge 项目 ID，请使用项目页链接或数字 ID", s)
}

// DownloadCurseForgeFile downloads one specific file of a project.
func DownloadCurseForgeFile(ctx context.Context, hc *HTTPClient, apiKey string, mod CurseForgeMod, file CurseForgeFile, destDir string, onBytes func(done, total int64)) (string, error) {
	raw := file.DownloadURL
	if raw == "" {
		var err error
		raw, err = curseForgeDownloadURL(ctx, apiKey, mod.ID, file.ID)
		if err != nil {
			return "", err
		}
	}
	name := file.FileName
	if name == "" {
		name = fmt.Sprintf("%d-%d.zip", mod.ID, file.ID)
	}
	name = filepath.Base(name)
	if !strings.HasSuffix(strings.ToLower(name), ".zip") {
		name += ".zip"
	}
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return "", err
	}
	dest := filepath.Join(destDir, name)
	if err := hc.DownloadFile(ctx, raw, dest, "", onBytes); err != nil {
		return "", fmt.Errorf("下载 CurseForge 文件: %w", err)
	}
	return dest, nil
}

func curseForgeGet(ctx context.Context, hc *HTTPClient, apiKey, endpoint string) ([]byte, error) {
	if hc == nil || hc.Client == nil {
		return nil, fmt.Errorf("缺少 HTTP 客户端")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", UserAgent)
	resp, err := hc.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusUnauthorized {
		msg := strings.TrimSpace(string(data))
		if msg == "" {
			msg = "无响应内容"
		}
		return nil, fmt.Errorf("CurseForge 拒绝了请求（API Key 无此接口权限或已过期）: %s", msg)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}
	return data, nil
}
