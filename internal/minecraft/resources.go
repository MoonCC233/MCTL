package minecraft

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// LocalResource is a mod, resource pack or shader file in an instance.
type LocalResource struct {
	Name     string
	Path     string
	Size     int64
	Disabled bool
}

// ListResources lists files in the requested resource directory. Files ending
// in .disabled are reported as disabled.
func ListResources(gameDir, kind string) ([]LocalResource, error) {
	dir, err := resourceDir(gameDir, kind)
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []LocalResource
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		st, err := e.Info()
		if err != nil {
			continue
		}
		disabled := strings.HasSuffix(strings.ToLower(e.Name()), ".disabled")
		out = append(out, LocalResource{
			Name:     e.Name(),
			Path:     filepath.Join(dir, e.Name()),
			Size:     st.Size(),
			Disabled: disabled,
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Disabled != out[j].Disabled {
			return !out[i].Disabled
		}
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	return out, nil
}

// ToggleResource enables or disables a file by adding or removing .disabled.
func ToggleResource(gameDir, kind, name string) (string, error) {
	p, err := resourcePath(gameDir, kind, name)
	if err != nil {
		return "", err
	}
	next := name + ".disabled"
	if strings.HasSuffix(strings.ToLower(name), ".disabled") {
		next = name[:len(name)-len(".disabled")]
	}
	dest, err := resourcePath(gameDir, kind, next)
	if err != nil {
		return "", err
	}
	if err := os.Rename(p, dest); err != nil {
		return "", err
	}
	return next, nil
}

// DeleteResource removes a local resource file.
func DeleteResource(gameDir, kind, name string) error {
	p, err := resourcePath(gameDir, kind, name)
	if err != nil {
		return err
	}
	return os.Remove(p)
}

func resourceDir(gameDir, kind string) (string, error) {
	sub := ResourceSubdir(kind)
	if sub == "" {
		return "", fmt.Errorf("未知资源类型: %s", kind)
	}
	return filepath.Join(gameDir, sub), nil
}

func resourcePath(gameDir, kind, name string) (string, error) {
	if name == "" || name == "." || name == ".." || strings.ContainsAny(name, `/\`) {
		return "", fmt.Errorf("非法资源名: %q", name)
	}
	dir, err := resourceDir(gameDir, kind)
	if err != nil {
		return "", err
	}
	root := filepath.Clean(dir)
	p := filepath.Clean(filepath.Join(root, name))
	rel, err := filepath.Rel(root, p)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("资源路径越界")
	}
	return p, nil
}
