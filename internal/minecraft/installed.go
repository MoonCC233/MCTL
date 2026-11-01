package minecraft

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// InstalledVersion summarises a version found on disk.
type InstalledVersion struct {
	ID          string
	Dir         string
	GameDir     string
	Type        string
	MCVersion   string
	Loader      string
	JavaMajor   int
	ReleaseTime string
	Size        int64
	Mods        int
	Complete    bool
	MainClass   string
	Name        string
	Icon        string
	Link        string
}

// ScanVersions lists every version directory with a JSON document. Pack
// instances are discovered under instances/<name> and only expose versions
// marked with mctl-pack.json, so their internal base/loader profiles stay
// hidden.
func ScanVersions(gameDir string) ([]InstalledVersion, error) {
	out, err := scanVersionRoot(gameDir, gameDir, false)
	if err != nil {
		return nil, err
	}
	instances, err := os.ReadDir(filepath.Join(gameDir, "instances"))
	if err == nil {
		for _, e := range instances {
			if !e.IsDir() {
				continue
			}
			root := filepath.Join(gameDir, "instances", e.Name())
			list, err := scanVersionRoot(root, root, true)
			if err != nil {
				continue
			}
			out = append(out, list...)
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	sortInstalledVersions(out)
	return out, nil
}

func scanVersionRoot(root, gameRoot string, packOnly bool) ([]InstalledVersion, error) {
	entries, err := os.ReadDir(filepath.Join(root, "versions"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	mods := CountMods(gameRoot)
	var out []InstalledVersion
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		id := e.Name()
		if packOnly {
			if _, err := os.Stat(filepath.Join(VersionPath(root, id), "mctl-pack.json")); err != nil {
				continue
			}
		}
		jp := VersionJSONPath(root, id)
		raw, err := ReadVersionJSON(jp)
		if err != nil {
			continue
		}
		iv := InstalledVersion{
			ID:          id,
			Dir:         VersionPath(root, id),
			GameDir:     gameRoot,
			Type:        raw.Type,
			Mods:        mods,
			ReleaseTime: raw.ReleaseTime,
			MainClass:   raw.MainClass,
		}
		if iv.Type == "" {
			iv.Type = "release"
		}
		if settings, err := LoadInstanceSettings(root, id); err == nil {
			iv.Name = settings.Name
			iv.Icon = settings.Icon
			iv.Link = settings.Link
		}
		if rv, err := ResolveVersion(root, id); err == nil {
			iv.Loader = detectLoader(id, rv.MainClass)
			iv.MainClass = rv.MainClass
			if rv.JavaVersion != nil {
				iv.JavaMajor = rv.JavaVersion.MajorVersion
			}
			iv.MCVersion = mcVersionFromChain(rv.Chain, id)
			if iv.ReleaseTime == "" {
				iv.ReleaseTime = rv.ReleaseTime
			}
		} else {
			iv.Loader = detectLoader(id, raw.MainClass)
			iv.MCVersion = mcVersionFallback(id, iv.Loader)
		}
		if raw.MinecraftVersion != "" {
			iv.MCVersion = raw.MinecraftVersion
		}
		if iv.MCVersion == "" {
			iv.MCVersion = mcVersionFallback(id, iv.Loader)
		}
		if st, err := os.Stat(filepath.Join(iv.Dir, id+".jar")); err == nil && st.Size() > 0 {
			iv.Complete = true
		}
		if !iv.Complete {
			if clients, err := ClientDownloads(root, id); err == nil {
				for vid := range clients {
					st, err := os.Stat(filepath.Join(VersionPath(root, vid), vid+".jar"))
					if err == nil && st.Size() > 0 {
						iv.Complete = true
						break
					}
				}
			}
		}
		if packOnly {
			iv.Size = dirSize(gameRoot)
		} else {
			iv.Size = dirSize(iv.Dir)
		}
		out = append(out, iv)
	}
	return out, nil
}

func sortInstalledVersions(out []InstalledVersion) {
	sort.SliceStable(out, func(i, j int) bool {
		ti, ei := time.Parse(time.RFC3339, out[i].ReleaseTime)
		tj, ej := time.Parse(time.RFC3339, out[j].ReleaseTime)
		if ei == nil && ej == nil && !ti.Equal(tj) {
			return ti.After(tj)
		}
		return strings.ToLower(out[i].ID) > strings.ToLower(out[j].ID)
	})
}

func detectLoader(id, mainClass string) string {
	s := strings.ToLower(id + " " + mainClass)
	switch {
	case strings.Contains(s, "labymod"):
		return "LabyMod"
	case strings.Contains(s, "neoforge"):
		return "NeoForge"
	case strings.Contains(s, "fabric"):
		return "Fabric"
	case strings.Contains(s, "quilt"):
		return "Quilt"
	case strings.Contains(s, "optifine"):
		return "OptiFine"
	case strings.Contains(s, "forge"):
		return "Forge"
	default:
		return "Vanilla"
	}
}

func mcVersionFromChain(chain []string, id string) string {
	for _, c := range chain {
		if LooksLikeMCVersion(c) {
			return c
		}
	}
	for _, c := range chain {
		if m := mcVersionRe.FindString(c); m != "" {
			return m
		}
	}
	return mcVersionFallback(id, detectLoader(id, ""))
}

var mcVersionRe = regexp.MustCompile(`\d+\.\d+(?:\.\d+)?`)

func mcVersionFallback(id, loader string) string {
	if LooksLikeMCVersion(id) {
		return id
	}
	all := mcVersionRe.FindAllString(id, -1)
	if len(all) == 0 {
		return ""
	}
	if loader == "Fabric" || loader == "Quilt" {
		return all[len(all)-1]
	}
	return all[0]
}

// CountMods counts enabled mod jars in the shared mods directory.
func CountMods(gameDir string) int {
	entries, err := os.ReadDir(filepath.Join(gameDir, "mods"))
	if err != nil {
		return 0
	}
	n := 0
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(strings.ToLower(e.Name()), ".jar") {
			n++
		}
	}
	return n
}

func dirSize(root string) int64 {
	var total int64
	_ = filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if info, err := d.Info(); err == nil {
			total += info.Size()
		}
		return nil
	})
	return total
}

// DeleteVersion removes a version directory safely.
func DeleteVersion(gameDir, id string) error {
	if id == "" || strings.ContainsAny(id, `/\`) || id == "." || id == ".." {
		return fmt.Errorf("非法版本名: %q", id)
	}
	root := filepath.Clean(filepath.Join(gameDir, "versions"))
	target := filepath.Clean(VersionPath(gameDir, id))
	rel, err := filepath.Rel(root, target)
	if err != nil || strings.HasPrefix(rel, "..") {
		return fmt.Errorf("拒绝删除版本目录之外的路径")
	}
	return os.RemoveAll(target)
}

// IsInstalled reports whether a version document exists.
func IsInstalled(gameDir, id string) bool {
	_, err := os.Stat(VersionJSONPath(gameDir, id))
	return err == nil
}
