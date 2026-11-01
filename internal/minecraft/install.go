package minecraft

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Progress is one installer status update.
type Progress struct {
	Stage      string
	Done       int
	Total      int
	Current    string
	Bytes      int64
	TotalBytes int64
	Detail     string
}

// Installer downloads and verifies everything a version needs.
type Installer struct {
	GameDir  string
	Source   Source
	HTTP     *HTTPClient
	Workers  int
	Log      func(string)
	Progress func(Progress)

	manifest   *VersionManifest
	manifestMu sync.Mutex

	progressMu sync.Mutex
	lastReport time.Time
	lastStage  string
}

// NewInstaller constructs an installer for a game directory.
func NewInstaller(gameDir string, source Source, logf func(string), progress func(Progress)) *Installer {
	if logf == nil {
		logf = func(string) {}
	}
	return &Installer{
		GameDir:  gameDir,
		Source:   source,
		HTTP:     NewHTTPClient(source),
		Workers:  8,
		Log:      logf,
		Progress: progress,
	}
}

func (i *Installer) logf(format string, args ...any) {
	if i.Log != nil {
		i.Log(fmt.Sprintf(format, args...))
	}
}

func (i *Installer) report(p Progress, force bool) {
	if i.Progress == nil {
		return
	}
	i.progressMu.Lock()
	if !force && p.Stage == i.lastStage && time.Since(i.lastReport) < 90*time.Millisecond {
		i.progressMu.Unlock()
		return
	}
	i.lastReport = time.Now()
	i.lastStage = p.Stage
	i.progressMu.Unlock()
	i.Progress(p)
}

func (i *Installer) manifestFor(ctx context.Context) (*VersionManifest, error) {
	i.manifestMu.Lock()
	defer i.manifestMu.Unlock()
	if i.manifest != nil {
		return i.manifest, nil
	}
	m, err := FetchManifest(ctx, i.HTTP)
	if err != nil {
		return nil, err
	}
	i.manifest = m
	return m, nil
}

// InstallVersion downloads a version from the manifest (including parents).
func (i *Installer) InstallVersion(ctx context.Context, id string, entry *VersionEntry) error {
	return i.installVersion(ctx, id, entry, 0)
}

func (i *Installer) installVersion(ctx context.Context, id string, entry *VersionEntry, depth int) error {
	if depth > 8 {
		return fmt.Errorf("版本继承层级过深")
	}
	if entry == nil {
		m, err := i.manifestFor(ctx)
		if err != nil {
			return err
		}
		e, ok := m.Find(id)
		if !ok {
			return fmt.Errorf("版本 %s 不在版本列表中", id)
		}
		entry = &e
	}
	if entry.URL == "" {
		return fmt.Errorf("版本 %s 缺少下载地址", id)
	}
	i.report(Progress{Stage: "获取版本信息", Current: id}, true)
	data, err := i.HTTP.Get(ctx, entry.URL)
	if err != nil {
		return fmt.Errorf("下载版本 %s 信息: %w", id, err)
	}
	var v VersionJSON
	if err := json.Unmarshal(data, &v); err != nil {
		return fmt.Errorf("解析版本 %s 信息: %w", id, err)
	}
	if v.ID == "" {
		v.ID = id
	}
	if v.InheritsFrom != "" {
		if _, err := ResolveVersion(i.GameDir, v.InheritsFrom); err != nil {
			i.logf("先安装父版本 %s", v.InheritsFrom)
			if err := i.installVersion(ctx, v.InheritsFrom, nil, depth+1); err != nil {
				return fmt.Errorf("安装父版本 %s: %w", v.InheritsFrom, err)
			}
		}
	}
	if err := writeVersionJSON(i.GameDir, v.ID, data); err != nil {
		return err
	}
	i.logf("版本文件已写入 versions/%s", v.ID)
	return i.EnsureInstalled(ctx, id)
}

// InstallProfile writes a launcher-provided version profile (Fabric/Quilt).
func (i *Installer) InstallProfile(ctx context.Context, id string, data []byte) error {
	var v VersionJSON
	if err := json.Unmarshal(data, &v); err != nil {
		return fmt.Errorf("解析版本配置: %w", err)
	}
	if v.ID == "" {
		v.ID = id
	}
	if v.InheritsFrom != "" {
		if _, err := ResolveVersion(i.GameDir, v.InheritsFrom); err != nil {
			i.logf("先安装父版本 %s", v.InheritsFrom)
			if err := i.InstallVersion(ctx, v.InheritsFrom, nil); err != nil {
				return fmt.Errorf("安装父版本 %s: %w", v.InheritsFrom, err)
			}
		}
	}
	if err := writeVersionJSON(i.GameDir, v.ID, data); err != nil {
		return err
	}
	i.logf("版本配置已写入 versions/%s", v.ID)
	return i.EnsureInstalled(ctx, v.ID)
}

func writeVersionJSON(gameDir, id string, data []byte) error {
	dir := VersionPath(gameDir, id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	var buf bytes.Buffer
	pretty := data
	if err := json.Indent(&buf, data, "", "  "); err == nil {
		pretty = append(buf.Bytes(), '\n')
	}
	tmp := filepath.Join(dir, id+".json.tmp")
	if err := os.WriteFile(tmp, pretty, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, filepath.Join(dir, id+".json"))
}

// EnsureInstalled repairs every missing file for an already-known version.
func (i *Installer) EnsureInstalled(ctx context.Context, id string) error {
	rv, err := ResolveVersion(i.GameDir, id)
	if err != nil {
		return err
	}
	features := Features(0, 0)

	// Client jar(s) across the inheritance chain.
	clients, err := ClientDownloads(i.GameDir, id)
	if err != nil {
		return err
	}
	ids := make([]string, 0, len(clients))
	for k := range clients {
		ids = append(ids, k)
	}
	sort.Strings(ids)
	var clientTasks []fileTask
	for _, vid := range ids {
		d := clients[vid]
		dest := filepath.Join(VersionPath(i.GameDir, vid), vid+".jar")
		clientTasks = append(clientTasks, fileTask{URL: d.URL, Dest: dest, SHA1: d.SHA1, Size: d.Size, Label: vid + ".jar"})
	}
	i.report(Progress{Stage: "检查客户端文件"}, true)
	if err := i.downloadAll(ctx, clientTasks, "下载客户端"); err != nil {
		return err
	}

	// Libraries and native jars.
	var libTasks []fileTask
	var natives []nativeSpec
	for _, lib := range rv.LibrariesFor(features) {
		if art, err := lib.Artifact(); err == nil && art != nil && art.Path != "" && art.URL != "" {
			libTasks = append(libTasks, fileTask{
				URL:   art.URL,
				Dest:  filepath.Join(i.GameDir, "libraries", filepath.FromSlash(art.Path)),
				SHA1:  art.SHA1,
				Size:  art.Size,
				Label: path.Base(art.Path),
			})
		}
		if nat, _, err := lib.NativeArtifact(); err == nil && nat != nil && nat.Path != "" && nat.URL != "" {
			dest := filepath.Join(i.GameDir, "libraries", filepath.FromSlash(nat.Path))
			libTasks = append(libTasks, fileTask{
				URL:   nat.URL,
				Dest:  dest,
				SHA1:  nat.SHA1,
				Size:  nat.Size,
				Label: path.Base(nat.Path),
			})
			var ex []string
			if lib.Extract != nil {
				ex = lib.Extract.Exclude
			}
			natives = append(natives, nativeSpec{jar: dest, excludes: ex})
		}
	}
	i.logf("检查 %d 个依赖库", len(libTasks))
	if err := i.downloadAll(ctx, libTasks, "下载依赖库"); err != nil {
		return err
	}

	// Assets.
	if err := i.installAssets(ctx, rv); err != nil {
		return err
	}

	// Log4j configuration.
	if rv.Logging != nil && rv.Logging.Client != nil && rv.Logging.Client.File.URL != "" {
		p := LoggingConfigPath(i.GameDir, rv.Logging.Client)
		if !fileMatches(p, rv.Logging.Client.File.Size) {
			i.logf("下载日志配置")
			if err := i.HTTP.DownloadFile(ctx, rv.Logging.Client.File.URL, p, rv.Logging.Client.File.SHA1, nil); err != nil {
				return fmt.Errorf("下载日志配置: %w", err)
			}
		}
	}

	// Natives must be unpacked for every launch.
	if len(natives) > 0 {
		i.report(Progress{Stage: "解压运行库", Current: id}, true)
		if err := extractNatives(natives, filepath.Join(VersionPath(i.GameDir, id), "natives")); err != nil {
			return err
		}
	}
	i.report(Progress{Stage: "准备完成", Done: 1, Total: 1}, true)
	return nil
}

type nativeSpec struct {
	jar      string
	excludes []string
}

// LoggingConfigPath is the on-disk location of a log4j config.
func LoggingConfigPath(gameDir string, lc *LoggingClient) string {
	id := lc.File.ID
	if id == "" {
		id = lc.Type
	}
	if id == "" {
		id = "client"
	}
	return filepath.Join(gameDir, "assets", "log_configs", id)
}

type fileTask struct {
	URL   string
	Dest  string
	SHA1  string
	Size  int64
	Label string
}

func (i *Installer) downloadAll(ctx context.Context, tasks []fileTask, stage string) error {
	seen := map[string]bool{}
	pending := make([]fileTask, 0, len(tasks))
	for _, t := range tasks {
		if t.URL == "" || t.Dest == "" || seen[t.Dest] {
			continue
		}
		seen[t.Dest] = true
		if fileMatches(t.Dest, t.Size) {
			continue
		}
		pending = append(pending, t)
	}
	if len(pending) == 0 {
		i.report(Progress{Stage: stage, Done: 0, Total: 0, Detail: "已是最新"}, true)
		return nil
	}
	total := len(pending)
	i.logf("%s：%d 个文件", stage, total)
	i.report(Progress{Stage: stage, Done: 0, Total: total, Current: pending[0].Label}, true)

	workers := i.Workers
	if workers <= 0 {
		workers = 8
	}
	if workers > total {
		workers = total
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	jobs := make(chan fileTask)
	var wg sync.WaitGroup
	var done atomic.Int64
	var firstErr error
	var errMu sync.Mutex

	worker := func() {
		defer wg.Done()
		for t := range jobs {
			if ctx.Err() != nil {
				return
			}
			err := i.HTTP.DownloadFile(ctx, t.URL, t.Dest, t.SHA1, nil)
			if err != nil {
				errMu.Lock()
				if firstErr == nil {
					firstErr = fmt.Errorf("下载 %s: %w", t.Label, err)
					cancel()
				}
				errMu.Unlock()
				return
			}
			n := done.Add(1)
			i.report(Progress{Stage: stage, Done: int(n), Total: total, Current: t.Label}, false)
		}
	}
	wg.Add(workers)
	for k := 0; k < workers; k++ {
		go worker()
	}
	for _, t := range pending {
		select {
		case <-ctx.Done():
		case jobs <- t:
		}
	}
	close(jobs)
	wg.Wait()
	if firstErr != nil {
		return firstErr
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	i.report(Progress{Stage: stage, Done: total, Total: total}, true)
	return nil
}

func fileMatches(p string, size int64) bool {
	st, err := os.Stat(p)
	if err != nil || st.IsDir() {
		return false
	}
	if size > 0 && st.Size() != size {
		return false
	}
	return st.Size() > 0
}

func extractNatives(specs []nativeSpec, dest string) error {
	tmp := dest + ".tmp"
	_ = os.RemoveAll(tmp)
	if err := os.MkdirAll(tmp, 0o755); err != nil {
		return err
	}
	for _, spec := range specs {
		if err := extractZip(spec.jar, tmp, spec.excludes); err != nil {
			_ = os.RemoveAll(tmp)
			return fmt.Errorf("解压 %s: %w", filepath.Base(spec.jar), err)
		}
	}
	_ = os.RemoveAll(dest)
	if err := os.Rename(tmp, dest); err != nil {
		_ = os.RemoveAll(tmp)
		return err
	}
	return nil
}

func extractZip(jar, dest string, excludes []string) error {
	zr, err := zip.OpenReader(jar)
	if err != nil {
		return err
	}
	defer zr.Close()
	for _, f := range zr.File {
		name := f.Name
		if f.FileInfo().IsDir() {
			continue
		}
		if excluded(name, excludes) {
			continue
		}
		target := filepath.Join(dest, filepath.FromSlash(name))
		rel, err := filepath.Rel(dest, target)
		if err != nil || strings.HasPrefix(rel, "..") {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		out, err := os.Create(target)
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

func excluded(name string, patterns []string) bool {
	if strings.HasPrefix(name, "META-INF/") {
		return true
	}
	for _, p := range patterns {
		if p == "" {
			continue
		}
		if strings.HasSuffix(p, "/") && strings.HasPrefix(name, p) {
			return true
		}
		if ok, _ := path.Match(p, name); ok {
			return true
		}
	}
	return false
}

// CheckResult counts missing files for an installed version.
type CheckResult struct {
	Client    int
	Libraries int
	Assets    int
	Index     int
	Logging   int
	Natives   int
	Complete  bool
}

func (c CheckResult) Missing() int {
	return c.Client + c.Libraries + c.Assets + c.Index + c.Logging + c.Natives
}

// CheckVersion verifies that all required files exist.
func CheckVersion(gameDir, id string) (*CheckResult, error) {
	rv, err := ResolveVersion(gameDir, id)
	if err != nil {
		return nil, err
	}
	features := Features(0, 0)
	res := &CheckResult{}
	clients, err := ClientDownloads(gameDir, id)
	if err != nil {
		return nil, err
	}
	for vid, d := range clients {
		if !fileMatches(filepath.Join(VersionPath(gameDir, vid), vid+".jar"), d.Size) {
			res.Client++
		}
	}
	for _, lib := range rv.LibrariesFor(features) {
		if art, err := lib.Artifact(); err == nil && art != nil && art.Path != "" {
			if !fileMatches(filepath.Join(gameDir, "libraries", filepath.FromSlash(art.Path)), art.Size) {
				res.Libraries++
			}
		}
		if nat, _, err := lib.NativeArtifact(); err == nil && nat != nil && nat.Path != "" {
			if !fileMatches(filepath.Join(gameDir, "libraries", filepath.FromSlash(nat.Path)), nat.Size) {
				res.Natives++
			}
		}
	}
	if rv.AssetIndex != nil && rv.AssetIndex.URL != "" {
		idxID := rv.AssetIndex.ID
		if idxID == "" {
			idxID = rv.Assets
		}
		idxPath := filepath.Join(gameDir, "assets", "indexes", idxID+".json")
		if !fileMatches(idxPath, rv.AssetIndex.Size) {
			res.Index++
		} else if data, err := os.ReadFile(idxPath); err == nil {
			var idx AssetIndex
			if json.Unmarshal(data, &idx) == nil {
				for _, obj := range idx.Objects {
					if obj.Hash == "" {
						continue
					}
					p := filepath.Join(gameDir, "assets", "objects", obj.Hash[:2], obj.Hash)
					if !fileMatches(p, obj.Size) {
						res.Assets++
					}
				}
			}
		}
	}
	if rv.Logging != nil && rv.Logging.Client != nil && rv.Logging.Client.File.URL != "" {
		if !fileMatches(LoggingConfigPath(gameDir, rv.Logging.Client), rv.Logging.Client.File.Size) {
			res.Logging++
		}
	}
	res.Complete = res.Missing() == 0
	return res, nil
}
