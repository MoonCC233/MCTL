package minecraft

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Screenshot describes an image in <gameDir>/screenshots.
type Screenshot struct {
	Name    string
	Path    string
	Size    int64
	Updated time.Time
}

// ListScreenshots lists PNG/JPEG screenshots newest first.
func ListScreenshots(gameDir string) ([]Screenshot, error) {
	dir := filepath.Join(gameDir, "screenshots")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []Screenshot
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(e.Name()))
		if ext != ".png" && ext != ".jpg" && ext != ".jpeg" {
			continue
		}
		st, err := e.Info()
		if err != nil {
			continue
		}
		out = append(out, Screenshot{
			Name:    e.Name(),
			Path:    filepath.Join(dir, e.Name()),
			Size:    st.Size(),
			Updated: st.ModTime(),
		})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Updated.After(out[j].Updated) })
	return out, nil
}

// DeleteScreenshot removes a screenshot file safely.
func DeleteScreenshot(gameDir, name string) error {
	if name == "" || name == "." || name == ".." || strings.ContainsAny(name, `/\`) {
		return fmt.Errorf("非法截图名: %q", name)
	}
	dir := filepath.Join(gameDir, "screenshots")
	p := filepath.Join(dir, name)
	rel, err := filepath.Rel(filepath.Clean(dir), p)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("截图路径越界")
	}
	return os.Remove(p)
}
