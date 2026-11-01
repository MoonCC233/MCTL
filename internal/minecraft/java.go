package minecraft

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

// JavaInfo describes a detected Java runtime.
type JavaInfo struct {
	Path    string
	Version string
	Major   int
	Vendor  string
	Arch    string
}

func (j *JavaInfo) String() string {
	if j == nil {
		return "未检测到"
	}
	v := j.Version
	if v == "" {
		v = strconv.Itoa(j.Major)
	}
	return fmt.Sprintf("Java %s", v)
}

var (
	javaCacheMu sync.Mutex
	javaCache   = map[string]*JavaInfo{}
)

// DetectJava probes a java executable.
func DetectJava(p string) (*JavaInfo, error) {
	if p == "" {
		return nil, errors.New("空 Java 路径")
	}
	if !strings.ContainsAny(p, `/\`) {
		if found, err := exec.LookPath(p); err == nil {
			p = found
		}
	}
	st, err := os.Stat(p)
	if err != nil {
		return nil, err
	}
	if st.IsDir() {
		candidate := filepath.Join(p, "bin", javaBinary())
		st, err = os.Stat(candidate)
		if err != nil {
			return nil, fmt.Errorf("%s 不是 Java 可执行文件", p)
		}
		p = candidate
	}
	if abs, err := filepath.Abs(p); err == nil {
		p = abs
	}
	javaCacheMu.Lock()
	if c, ok := javaCache[p]; ok {
		cp := *c
		javaCacheMu.Unlock()
		return &cp, nil
	}
	javaCacheMu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, p, "-XshowSettings:properties", "-version")
	hideWindow(cmd)
	out, err := cmd.CombinedOutput()
	if err != nil {
		cmd = exec.CommandContext(ctx, p, "-version")
		hideWindow(cmd)
		out, err = cmd.CombinedOutput()
	}
	text := string(out)
	if strings.TrimSpace(text) == "" {
		return nil, fmt.Errorf("无法运行 Java: %v", err)
	}
	info := &JavaInfo{Path: p}
	propRe := regexp.MustCompile(`(?m)^\s{2,}([A-Za-z0-9_.]+)\s*=\s*(.+?)\s*$`)
	props := map[string]string{}
	for _, m := range propRe.FindAllStringSubmatch(text, -1) {
		props[m[1]] = m[2]
	}
	info.Version = props["java.version"]
	info.Vendor = props["java.vendor"]
	info.Arch = props["os.arch"]
	if info.Version == "" {
		vr := regexp.MustCompile(`(?:java|openjdk) version "([^"]+)"`)
		if m := vr.FindStringSubmatch(text); len(m) == 2 {
			info.Version = m[1]
		}
	}
	if info.Version == "" {
		return nil, fmt.Errorf("无法解析 Java 版本")
	}
	spec := props["java.specification.version"]
	if spec == "" {
		spec = info.Version
	}
	info.Major = parseJavaMajor(spec)
	if info.Major == 0 {
		info.Major = parseJavaMajor(info.Version)
	}
	if info.Major == 0 {
		return nil, fmt.Errorf("无法解析 Java 主版本: %s", info.Version)
	}
	javaCacheMu.Lock()
	javaCache[p] = info
	javaCacheMu.Unlock()
	cp := *info
	return &cp, nil
}

func javaBinary() string {
	if runtime.GOOS == "windows" {
		return "java.exe"
	}
	return "java"
}

func parseJavaMajor(v string) int {
	v = strings.TrimSpace(v)
	if v == "" {
		return 0
	}
	v = strings.TrimPrefix(v, "1.")
	if i := strings.IndexAny(v, "._-+"); i >= 0 {
		v = v[:i]
	}
	n, _ := strconv.Atoi(v)
	return n
}

// JavaCandidates returns likely Java executable paths on this system.
func JavaCandidates(preferred string) []string {
	var out []string
	add := func(p string) {
		if p == "" {
			return
		}
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			out = append(out, p)
		}
	}
	if preferred != "" {
		add(preferred)
		if st, err := os.Stat(filepath.Join(preferred, "bin", javaBinary())); err == nil && !st.IsDir() {
			add(filepath.Join(preferred, "bin", javaBinary()))
		}
	}
	if home := os.Getenv("JAVA_HOME"); home != "" {
		add(filepath.Join(home, "bin", javaBinary()))
	}
	if p, err := exec.LookPath(javaBinary()); err == nil {
		add(p)
	}
	if runtime.GOOS == "windows" {
		globs := []string{
			`C:\Program Files\Java\*\bin\java.exe`,
			`C:\Program Files\Eclipse Adoptium\*\bin\java.exe`,
			`C:\Program Files\Microsoft\jdk-*\bin\java.exe`,
			`C:\Program Files\Zulu\*\bin\java.exe`,
			`C:\Program Files\BellSoft\*\bin\java.exe`,
			`C:\Program Files\Amazon Corretto\*\bin\java.exe`,
			`C:\Program Files\Semeru\*\bin\java.exe`,
			`C:\Program Files\Common Files\Oracle\Java\javapath\java.exe`,
		}
		if local := os.Getenv("LOCALAPPDATA"); local != "" {
			globs = append(globs,
				filepath.Join(local, "Programs", "Eclipse Adoptium", "*", "bin", "java.exe"),
				filepath.Join(local, "Programs", "Microsoft", "jdk-*", "bin", "java.exe"),
				filepath.Join(local, "Packages", "Microsoft.4297127D64EC6_8wekyb3d8bbwe", "LocalCache", "Local", "runtime", "*", "*", "*", "bin", "java.exe"),
			)
		}
		if appdata := os.Getenv("APPDATA"); appdata != "" {
			globs = append(globs, filepath.Join(appdata, ".minecraft", "runtime", "*", "*", "*", "bin", "java.exe"))
		}
		if home, err := os.UserHomeDir(); err == nil {
			globs = append(globs, filepath.Join(home, ".jdks", "*", "bin", "java.exe"))
		}
		globs = append(globs, `C:\Program Files (x86)\Java\*\bin\java.exe`)
		for _, g := range globs {
			matches, _ := filepath.Glob(g)
			for _, m := range matches {
				add(m)
			}
		}
	} else if runtime.GOOS == "darwin" {
		for _, g := range []string{
			"/Library/Java/JavaVirtualMachines/*/Contents/Home/bin/java",
			"/System/Library/Java/JavaVirtualMachines/*/Contents/Home/bin/java",
		} {
			matches, _ := filepath.Glob(g)
			for _, m := range matches {
				add(m)
			}
		}
	} else {
		for _, g := range []string{
			"/usr/lib/jvm/*/bin/java",
			"/usr/java/*/bin/java",
			"/opt/java/*/bin/java",
			"/opt/*/bin/java",
		} {
			matches, _ := filepath.Glob(g)
			for _, m := range matches {
				add(m)
			}
		}
	}
	seen := map[string]bool{}
	uniq := out[:0]
	for _, p := range out {
		key := p
		if runtime.GOOS == "windows" {
			key = strings.ToLower(p)
		}
		if seen[key] {
			continue
		}
		seen[key] = true
		uniq = append(uniq, p)
	}
	return uniq
}

// FindJava returns the best available Java runtime.
func FindJava(preferred string) (*JavaInfo, error) {
	return FindJavaFor(0, preferred)
}

// FindJavaFor picks the best runtime meeting the required major version.
func FindJavaFor(required int, preferred string) (*JavaInfo, error) {
	var best *JavaInfo
	for _, p := range JavaCandidates(preferred) {
		info, err := DetectJava(p)
		if err != nil {
			continue
		}
		if required > 0 && info.Major < required {
			continue
		}
		if best == nil {
			best = info
			continue
		}
		preferredPath := preferred != "" && strings.EqualFold(info.Path, preferred)
		bestPreferred := preferred != "" && strings.EqualFold(best.Path, preferred)
		if preferredPath && !bestPreferred {
			best = info
			continue
		}
		if info.Major == required && best.Major != required {
			best = info
			continue
		}
		if best.Major != required && info.Major > best.Major {
			best = info
		}
	}
	if best == nil {
		if required > 0 {
			return nil, fmt.Errorf("未找到 Java %d 或更高版本", required)
		}
		return nil, fmt.Errorf("未找到可用的 Java 运行时")
	}
	return best, nil
}

// InferJavaMajor guesses the Java requirement from a Minecraft version.
func InferJavaMajor(mcVersion string) int {
	if mcVersion == "" {
		return 8
	}
	base := mcVersion
	if i := strings.IndexAny(base, "- "); i >= 0 {
		base = base[:i]
	}
	parts := strings.Split(base, ".")
	if len(parts) < 2 {
		return 21
	}
	minor, _ := strconv.Atoi(parts[1])
	patch := 0
	if len(parts) > 2 {
		patch, _ = strconv.Atoi(parts[2])
	}
	switch {
	case minor > 20 || (minor == 20 && patch >= 5):
		return 21
	case minor >= 18:
		return 17
	case minor == 17:
		return 16
	default:
		return 8
	}
}

// EnsureJava finds a compatible Java or downloads a JRE from Adoptium.
func (i *Installer) EnsureJava(ctx context.Context, required int, preferred string) (*JavaInfo, error) {
	if required <= 0 {
		required = 8
	}
	if info, err := FindJavaFor(required, preferred); err == nil {
		return info, nil
	}
	target := filepath.Join(i.GameDir, "runtime", fmt.Sprintf("java-%d-%s-%s", required, runtime.GOOS, runtime.GOARCH))
	bin := filepath.Join(target, "bin", javaBinary())
	if info, err := DetectJava(bin); err == nil {
		return info, nil
	}

	osName, arch := adoptiumPlatform()
	url := fmt.Sprintf("https://api.adoptium.net/v3/binary/latest/%d/ga/%s/%s/jre/hotspot/normal/eclipse?project=jdk", required, osName, arch)
	ext := ".zip"
	if runtime.GOOS != "windows" {
		ext = ".tar.gz"
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return nil, err
	}
	archive := filepath.Join(filepath.Dir(target), fmt.Sprintf("java-%d%s", required, ext))
	i.logf("未找到 Java %d，正在从 Adoptium 下载", required)
	i.report(Progress{Stage: fmt.Sprintf("下载 Java %d", required), Current: "Adoptium JRE"}, true)
	err := i.HTTP.DownloadFile(ctx, url, archive, "", func(done, total int64) {
		i.report(Progress{
			Stage:      fmt.Sprintf("下载 Java %d", required),
			Bytes:      done,
			TotalBytes: total,
			Current:    "Adoptium JRE",
		}, false)
	})
	if err != nil {
		return nil, fmt.Errorf("下载 Java %d: %w", required, err)
	}
	defer os.Remove(archive)

	tmp := target + ".tmp"
	_ = os.RemoveAll(tmp)
	if err := os.MkdirAll(tmp, 0o755); err != nil {
		return nil, err
	}
	i.report(Progress{Stage: "解压 Java", Current: "Adoptium JRE"}, true)
	if ext == ".zip" {
		err = unzipTo(archive, tmp)
	} else {
		err = untarGzTo(archive, tmp)
	}
	if err != nil {
		_ = os.RemoveAll(tmp)
		return nil, fmt.Errorf("解压 Java: %w", err)
	}
	home := findJavaHome(tmp)
	if home == "" {
		_ = os.RemoveAll(tmp)
		return nil, fmt.Errorf("Java 压缩包结构异常")
	}
	_ = os.RemoveAll(target)
	if home != tmp {
		if err := os.Rename(home, target); err != nil {
			_ = os.RemoveAll(tmp)
			return nil, err
		}
		_ = os.RemoveAll(tmp)
	} else {
		if err := os.Rename(tmp, target); err != nil {
			return nil, err
		}
	}
	info, err := DetectJava(filepath.Join(target, "bin", javaBinary()))
	if err != nil {
		return nil, fmt.Errorf("安装的 Java 无法运行: %w", err)
	}
	i.logf("Java %s 已安装到 %s", info.Version, target)
	return info, nil
}

func adoptiumPlatform() (string, string) {
	osName := "linux"
	switch runtime.GOOS {
	case "windows":
		osName = "windows"
	case "darwin":
		osName = "mac"
	}
	arch := "x64"
	switch runtime.GOARCH {
	case "386":
		arch = "x86"
	case "arm64":
		arch = "aarch64"
	case "arm":
		arch = "arm"
	}
	return osName, arch
}

func findJavaHome(root string) string {
	var found string
	_ = filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil || found != "" {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		name := strings.ToLower(d.Name())
		if name == "java" || name == "java.exe" {
			if strings.EqualFold(filepath.Base(filepath.Dir(p)), "bin") {
				found = filepath.Dir(filepath.Dir(p))
			}
		}
		return nil
	})
	return found
}

func unzipTo(archive, dest string) error {
	zr, err := zip.OpenReader(archive)
	if err != nil {
		return err
	}
	defer zr.Close()
	for _, f := range zr.File {
		target := filepath.Join(dest, filepath.FromSlash(f.Name))
		rel, err := filepath.Rel(dest, target)
		if err != nil || strings.HasPrefix(rel, "..") {
			continue
		}
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		mode := f.Mode()
		if mode == 0 {
			mode = 0o644
		}
		out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode.Perm())
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

func untarGzTo(archive, dest string) error {
	f, err := os.Open(archive)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		target := filepath.Join(dest, filepath.FromSlash(h.Name))
		rel, err := filepath.Rel(dest, target)
		if err != nil || strings.HasPrefix(rel, "..") {
			continue
		}
		switch h.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeSymlink:
			_ = os.MkdirAll(filepath.Dir(target), 0o755)
			_ = os.Symlink(h.Linkname, target)
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(h.Mode)&0o777)
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return err
			}
			out.Close()
		}
	}
	return nil
}
