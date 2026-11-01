package minecraft

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// InstanceFormatVersion is the archive layout written by ExportInstance.
const InstanceFormatVersion = 1

// InstanceArchive describes the metadata stored in mctl-instance.json.
type InstanceArchive struct {
	FormatVersion int      `json:"formatVersion"`
	Name          string   `json:"name"`
	VersionID     string   `json:"versionId"`
	MCVersion     string   `json:"mcVersion"`
	Loader        string   `json:"loader"`
	Source        string   `json:"source"`
	Chain         []string `json:"chain,omitempty"`
	Created       string   `json:"created"`
}

// DefaultInstanceExportPath returns a timestamped export path for a version.
func DefaultInstanceExportPath(gameDir, versionID string) string {
	name := sanitizeFilePart(versionID)
	if name == "" {
		name = "instance"
	}
	return filepath.Join(gameDir, "exports", fmt.Sprintf("%s-%s.zip", name, time.Now().Format("20060102-150405")))
}

func sanitizeFilePart(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_', r == '.':
			b.WriteRune(r)
		default:
			b.WriteByte('-')
		}
	}
	return strings.Trim(b.String(), "-.")
}

// ExportInstance writes a portable zip for one installed version or pack
// instance. Player data (saves and screenshots) is included when includeSaves
// is true.
func ExportInstance(gameDir, versionID, dest string, includeSaves bool) (*InstanceArchive, error) {
	versionID = strings.TrimSpace(versionID)
	if versionID == "" {
		return nil, fmt.Errorf("没有选择要导出的实例")
	}
	if strings.TrimSpace(dest) == "" {
		return nil, fmt.Errorf("导出路径不能为空")
	}
	list, err := ScanVersions(gameDir)
	if err != nil {
		return nil, err
	}
	var installed *InstalledVersion
	for i := range list {
		if list[i].ID == versionID {
			installed = &list[i]
			break
		}
	}
	if installed == nil {
		return nil, fmt.Errorf("找不到版本 %s", versionID)
	}

	archive := &InstanceArchive{
		FormatVersion: InstanceFormatVersion,
		Name:          versionID,
		VersionID:     versionID,
		MCVersion:     installed.MCVersion,
		Loader:        installed.Loader,
		Source:        "version",
		Created:       time.Now().UTC().Format(time.RFC3339),
	}
	root := gameDir
	if installed.GameDir != "" && filepath.Clean(installed.GameDir) != filepath.Clean(gameDir) {
		root = installed.GameDir
		archive.Source = "pack"
		if marker, err := readPackMarker(root, versionID); err == nil && marker.Name != "" {
			archive.Name = marker.Name
		}
	} else {
		rv, err := ResolveVersion(gameDir, versionID)
		if err != nil {
			return nil, fmt.Errorf("解析版本 %s: %w", versionID, err)
		}
		archive.Chain = append([]string(nil), rv.Chain...)
		if len(archive.Chain) == 0 {
			archive.Chain = []string{versionID}
		}
	}

	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return nil, err
	}
	tmp, err := os.CreateTemp(filepath.Dir(dest), "mctl-export-*.zip")
	if err != nil {
		return nil, err
	}
	tmpName := tmp.Name()
	zw := zip.NewWriter(tmp)
	fail := func(err error) (*InstanceArchive, error) {
		_ = zw.Close()
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return nil, err
	}

	if err := writeZipJSON(zw, "mctl-instance.json", archive); err != nil {
		return fail(err)
	}
	if archive.Source == "version" {
		if err := writeZipVersionDirs(zw, root, archive.Chain); err != nil {
			return fail(err)
		}
	}
	if err := writeZipTree(zw, root, archive.Source, includeSaves); err != nil {
		return fail(err)
	}
	if err := zw.Close(); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return nil, err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return nil, err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return nil, err
	}
	if err := os.Rename(tmpName, dest); err != nil {
		_ = os.Remove(tmpName)
		return nil, err
	}
	return archive, nil
}

type packMarkerFile struct {
	Name      string `json:"name"`
	VersionID string `json:"versionId"`
	MCVersion string `json:"mcVersion"`
	Loader    string `json:"loader"`
	Source    string `json:"source"`
}

func readPackMarker(root, versionID string) (*packMarkerFile, error) {
	data, err := os.ReadFile(filepath.Join(VersionPath(root, versionID), "mctl-pack.json"))
	if err != nil {
		return nil, err
	}
	var marker packMarkerFile
	if err := json.Unmarshal(data, &marker); err != nil {
		return nil, err
	}
	return &marker, nil
}

// writeZipVersionDirs adds the version directories of the inheritance chain.
func writeZipVersionDirs(zw *zip.Writer, root string, ids []string) error {
	seen := map[string]bool{}
	for _, id := range ids {
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		dir := VersionPath(root, id)
		if st, err := os.Stat(dir); err != nil || !st.IsDir() {
			continue
		}
		if err := filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			zipPath := filepath.ToSlash(filepath.Join("versions", id, strings.TrimPrefix(p, dir+string(filepath.Separator))))
			return addZipFile(zw, p, zipPath)
		}); err != nil {
			return err
		}
	}
	return nil
}

// writeZipTree adds the files of root that belong to an instance archive.
func writeZipTree(zw *zip.Writer, root, source string, includeSaves bool) error {
	skipDir := func(name string) bool {
		switch name {
		case "mctl-installers", "logs", "exports", "backups", "crash-reports":
			return true
		case "saves", "screenshots":
			return !includeSaves
		}
		return false
	}
	switch source {
	case "pack":
		return filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			rel, err := filepath.Rel(root, p)
			if err != nil {
				return err
			}
			if rel == "." {
				return nil
			}
			if d.IsDir() {
				if skipDir(d.Name()) {
					return filepath.SkipDir
				}
				return nil
			}
			return addZipFile(zw, p, filepath.ToSlash(rel))
		})
	default:
		for _, name := range []string{"mods", "resourcepacks", "shaderpacks", "config"} {
			dir := filepath.Join(root, name)
			if st, err := os.Stat(dir); err != nil || !st.IsDir() {
				continue
			}
			if err := filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if d.IsDir() {
					return nil
				}
				rel, err := filepath.Rel(root, p)
				if err != nil {
					return err
				}
				return addZipFile(zw, p, filepath.ToSlash(rel))
			}); err != nil {
				return err
			}
		}
		if includeSaves {
			for _, name := range []string{"saves", "screenshots"} {
				dir := filepath.Join(root, name)
				if st, err := os.Stat(dir); err != nil || !st.IsDir() {
					continue
				}
				if err := filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
					if err != nil {
						return err
					}
					if d.IsDir() {
						return nil
					}
					rel, err := filepath.Rel(root, p)
					if err != nil {
						return err
					}
					return addZipFile(zw, p, filepath.ToSlash(rel))
				}); err != nil {
					return err
				}
			}
		}
		for _, name := range []string{"options.txt", "optionsof.txt", "servers.dat", "servers.dat_old"} {
			p := filepath.Join(root, name)
			if st, err := os.Stat(p); err == nil && !st.IsDir() {
				if err := addZipFile(zw, p, name); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func addZipFile(zw *zip.Writer, diskPath, zipPath string) error {
	st, err := os.Stat(diskPath)
	if err != nil || st.IsDir() {
		return err
	}
	if st.Mode()&os.ModeSymlink != 0 {
		return nil
	}
	in, err := os.Open(diskPath)
	if err != nil {
		return err
	}
	defer in.Close()
	w, err := zw.Create(zipPath)
	if err != nil {
		return err
	}
	_, err = io.Copy(w, in)
	return err
}

func writeZipJSON(zw *zip.Writer, name string, v any) error {
	w, err := zw.Create(name)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(w)
	return enc.Encode(v)
}

// ImportInstance extracts an MCTL instance archive into a new isolated
// instance under <gameDir>/instances and returns its metadata and directory.
func ImportInstance(gameDir, zipPath string) (*InstanceArchive, string, error) {
	zipPath = strings.TrimSpace(zipPath)
	if zipPath == "" {
		return nil, "", fmt.Errorf("请填写实例压缩包路径")
	}
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, "", fmt.Errorf("打开实例压缩包: %w", err)
	}
	defer zr.Close()

	var archive *InstanceArchive
	for _, f := range zr.File {
		if f.Name != "mctl-instance.json" {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return nil, "", err
		}
		data, err := io.ReadAll(io.LimitReader(rc, 1<<20))
		rc.Close()
		if err != nil {
			return nil, "", err
		}
		var a InstanceArchive
		if err := json.Unmarshal(data, &a); err != nil {
			return nil, "", fmt.Errorf("解析 mctl-instance.json: %w", err)
		}
		archive = &a
		break
	}
	if archive == nil {
		return nil, "", fmt.Errorf("不是 MCTL 实例压缩包：缺少 mctl-instance.json")
	}
	if archive.FormatVersion != InstanceFormatVersion {
		return nil, "", fmt.Errorf("不支持的实例格式版本: %d", archive.FormatVersion)
	}
	if archive.VersionID == "" {
		return nil, "", fmt.Errorf("实例压缩包缺少版本 ID")
	}
	if archive.Name == "" {
		archive.Name = archive.VersionID
	}
	if archive.Source != "pack" {
		archive.Source = "version"
	}

	slug := packSlug(archive.Name, archive.VersionID)
	instancesDir := filepath.Join(gameDir, "instances")
	if err := os.MkdirAll(instancesDir, 0o755); err != nil {
		return nil, "", err
	}
	root := filepath.Join(instancesDir, slug)
	for i := 2; ; i++ {
		if _, err := os.Stat(root); os.IsNotExist(err) {
			break
		}
		root = filepath.Join(instancesDir, fmt.Sprintf("%s-%d", slug, i))
	}
	stage, err := os.MkdirTemp(instancesDir, ".mctl-import-*")
	if err != nil {
		return nil, "", err
	}
	cleanup := func() {
		_ = os.RemoveAll(stage)
		_ = os.Remove(instancesDir)
	}
	if err := extractInstanceZip(zr, stage); err != nil {
		cleanup()
		return nil, "", err
	}
	if archive.Source == "version" {
		versionDir := VersionPath(stage, archive.VersionID)
		if st, err := os.Stat(versionDir); err != nil || !st.IsDir() {
			cleanup()
			return nil, "", fmt.Errorf("实例压缩包缺少版本目录 versions/%s", archive.VersionID)
		}
		marker := packMarkerFile{
			Name:      archive.Name,
			VersionID: archive.VersionID,
			MCVersion: archive.MCVersion,
			Loader:    archive.Loader,
			Source:    "export",
		}
		data, err := json.Marshal(marker)
		if err != nil {
			cleanup()
			return nil, "", err
		}
		if err := os.WriteFile(filepath.Join(versionDir, "mctl-pack.json"), data, 0o644); err != nil {
			cleanup()
			return nil, "", err
		}
	}
	if err := os.Rename(stage, root); err != nil {
		cleanup()
		return nil, "", err
	}
	return archive, root, nil
}

func extractInstanceZip(zr *zip.ReadCloser, root string) error {
	for _, f := range zr.File {
		if f.Name == "mctl-instance.json" || f.FileInfo().IsDir() {
			continue
		}
		rel := strings.TrimPrefix(filepath.ToSlash(f.Name), "/")
		if rel == "" {
			continue
		}
		dest, err := safeJoin(root, rel)
		if err != nil {
			return fmt.Errorf("实例文件路径无效 %q: %w", f.Name, err)
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
