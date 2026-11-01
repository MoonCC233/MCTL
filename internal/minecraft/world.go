package minecraft

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// World describes a save directory.
type World struct {
	Name    string
	Dir     string
	Size    int64
	Updated time.Time
}

// ListWorlds lists directories under <gameDir>/saves that contain level.dat.
func ListWorlds(gameDir string) ([]World, error) {
	root := filepath.Join(gameDir, "saves")
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []World
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		dir := filepath.Join(root, e.Name())
		st, err := os.Stat(filepath.Join(dir, "level.dat"))
		if err != nil {
			continue
		}
		out = append(out, World{
			Name:    e.Name(),
			Dir:     dir,
			Size:    dirSize(dir),
			Updated: st.ModTime(),
		})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Updated.After(out[j].Updated) })
	return out, nil
}

// WorldBackupPath returns a timestamped backup path for a world.
func WorldBackupPath(gameDir, name string) string {
	safe := strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			return r
		default:
			return '-'
		}
	}, name)
	safe = strings.Trim(safe, "-")
	if safe == "" {
		safe = "world"
	}
	return filepath.Join(gameDir, "backups", fmt.Sprintf("%s-%s.zip", safe, time.Now().Format("20060102-150405")))
}

// BackupWorld writes a world directory to a zip archive.
func BackupWorld(gameDir, name, dest string) error {
	worldDir, err := worldPath(gameDir, name)
	if err != nil {
		return err
	}
	if _, err := os.Stat(filepath.Join(worldDir, "level.dat")); err != nil {
		return fmt.Errorf("存档不存在或缺少 level.dat")
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	zw := zip.NewWriter(f)
	walkErr := filepath.WalkDir(worldDir, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, err := filepath.Rel(worldDir, p)
		if err != nil {
			return err
		}
		w, err := zw.Create(filepath.ToSlash(filepath.Join(name, rel)))
		if err != nil {
			return err
		}
		in, err := os.Open(p)
		if err != nil {
			return err
		}
		defer in.Close()
		_, err = io.Copy(w, in)
		return err
	})
	closeErr := zw.Close()
	fileErr := f.Close()
	if walkErr != nil {
		_ = os.Remove(dest)
		return walkErr
	}
	if closeErr != nil {
		_ = os.Remove(dest)
		return closeErr
	}
	if fileErr != nil {
		_ = os.Remove(dest)
		return fileErr
	}
	return nil
}

// ImportWorld extracts a zipped world into <gameDir>/saves and returns its name.
func ImportWorld(gameDir, zipPath string) (string, error) {
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return "", fmt.Errorf("打开存档压缩包: %w", err)
	}
	defer zr.Close()
	prefix := commonWorldPrefix(zr.File)
	name := strings.TrimSuffix(filepath.Base(zipPath), filepath.Ext(zipPath))
	if prefix != "" {
		name = strings.TrimSuffix(prefix, "/")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		name = "imported-world"
	}
	saves := filepath.Join(gameDir, "saves")
	if err := os.MkdirAll(saves, 0o755); err != nil {
		return "", err
	}
	target := filepath.Join(saves, name)
	for i := 2; ; i++ {
		if _, err := os.Stat(target); os.IsNotExist(err) {
			break
		}
		target = filepath.Join(saves, fmt.Sprintf("%s-%d", name, i))
	}
	if err := os.MkdirAll(target, 0o755); err != nil {
		return "", err
	}
	for _, f := range zr.File {
		if f.FileInfo().IsDir() {
			continue
		}
		rel := f.Name
		if prefix != "" && strings.HasPrefix(rel, prefix) {
			rel = strings.TrimPrefix(rel, prefix)
		}
		rel = strings.TrimPrefix(rel, "/")
		if rel == "" {
			continue
		}
		dest, err := safeJoin(target, rel)
		if err != nil {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return "", err
		}
		rc, err := f.Open()
		if err != nil {
			return "", err
		}
		out, err := os.Create(dest)
		if err != nil {
			rc.Close()
			return "", err
		}
		_, cerr := io.Copy(out, rc)
		rc.Close()
		out.Close()
		if cerr != nil {
			return "", cerr
		}
	}
	if _, err := os.Stat(filepath.Join(target, "level.dat")); err != nil {
		_ = os.RemoveAll(target)
		return "", fmt.Errorf("压缩包不是有效的 Minecraft 存档：缺少 level.dat")
	}
	return filepath.Base(target), nil
}

// DeleteWorld removes a world directory safely.
func DeleteWorld(gameDir, name string) error {
	dir, err := worldPath(gameDir, name)
	if err != nil {
		return err
	}
	return os.RemoveAll(dir)
}

func worldPath(gameDir, name string) (string, error) {
	if name == "" || name == "." || name == ".." || strings.ContainsAny(name, `/\`) {
		return "", fmt.Errorf("非法存档名: %q", name)
	}
	root := filepath.Clean(filepath.Join(gameDir, "saves"))
	dir := filepath.Clean(filepath.Join(root, name))
	rel, err := filepath.Rel(root, dir)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("存档路径越界")
	}
	return dir, nil
}

func commonWorldPrefix(files []*zip.File) string {
	prefix := ""
	for _, f := range files {
		if f.FileInfo().IsDir() {
			continue
		}
		parts := strings.SplitN(f.Name, "/", 2)
		if len(parts) < 2 {
			return ""
		}
		if prefix == "" {
			prefix = parts[0] + "/"
		} else if prefix != parts[0]+"/" {
			return ""
		}
	}
	return prefix
}
