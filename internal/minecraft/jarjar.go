package minecraft

import (
	"archive/zip"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ModInfo describes a mod jar in an instance.
type ModInfo struct {
	File        string
	Path        string
	ID          string
	Name        string
	Version     string
	Loader      string
	Description string
	Disabled    bool
	Nested      []string
	Err         string
}

// InspectMods lists and inspects jars (including disabled ones) in the mods
// directory.
func InspectMods(gameDir string) ([]ModInfo, error) {
	dir := filepath.Join(gameDir, "mods")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []ModInfo
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		low := strings.ToLower(e.Name())
		if !strings.HasSuffix(low, ".jar") && !strings.HasSuffix(low, ".jar.disabled") {
			continue
		}
		out = append(out, InspectModJar(filepath.Join(dir, e.Name())))
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Disabled != out[j].Disabled {
			return !out[i].Disabled
		}
		return strings.ToLower(out[i].File) < strings.ToLower(out[j].File)
	})
	return out, nil
}

// InspectModJar reads mod metadata and counts nested jars.
func InspectModJar(path string) ModInfo {
	info := ModInfo{
		File:     filepath.Base(path),
		Path:     path,
		Disabled: strings.HasSuffix(strings.ToLower(path), ".disabled"),
	}
	zr, err := zip.OpenReader(path)
	if err != nil {
		info.Err = "无法读取 jar: " + err.Error()
		return info
	}
	defer zr.Close()

	for _, f := range zr.File {
		low := strings.ToLower(f.Name)
		if strings.HasSuffix(low, ".jar") {
			if len(info.Nested) < 24 {
				info.Nested = append(info.Nested, f.Name)
			}
			continue
		}
	}
	if len(info.Nested) > 0 {
		// Nested jars are surfaced separately by the UI.
	}

	for _, f := range zr.File {
		switch strings.ToLower(f.Name) {
		case "fabric.mod.json":
			data, err := readZipEntry(f)
			if err != nil {
				info.Err = err.Error()
				return info
			}
			var meta struct {
				ID          string `json:"id"`
				Name        string `json:"name"`
				Version     string `json:"version"`
				Description string `json:"description"`
			}
			if err := json.Unmarshal(data, &meta); err != nil {
				info.Err = "解析 fabric.mod.json: " + err.Error()
				return info
			}
			info.ID, info.Name, info.Version = meta.ID, meta.Name, meta.Version
			info.Description = joinDescription(info.Description, meta.Description)
			info.Loader = "Fabric"
			return info
		case "quilt.mod.json":
			data, err := readZipEntry(f)
			if err != nil {
				info.Err = err.Error()
				return info
			}
			var meta struct {
				QuiltLoader struct {
					ID       string `json:"id"`
					Version  string `json:"version"`
					Metadata struct {
						Name        string `json:"name"`
						Description string `json:"description"`
					} `json:"metadata"`
				} `json:"quilt_loader"`
			}
			if err := json.Unmarshal(data, &meta); err != nil {
				info.Err = "解析 quilt.mod.json: " + err.Error()
				return info
			}
			info.ID = meta.QuiltLoader.ID
			info.Version = meta.QuiltLoader.Version
			info.Name = meta.QuiltLoader.Metadata.Name
			info.Description = joinDescription(info.Description, meta.QuiltLoader.Metadata.Description)
			info.Loader = "Quilt"
			return info
		case "meta-inf/neoforge.mods.toml", "meta-inf/mods.toml":
			data, err := readZipEntry(f)
			if err != nil {
				info.Err = err.Error()
				return info
			}
			meta := parseModsTOML(string(data))
			info.ID, info.Name, info.Version = meta.ID, meta.Name, meta.Version
			info.Description = joinDescription(info.Description, meta.Description)
			if strings.EqualFold(f.Name, "META-INF/neoforge.mods.toml") {
				info.Loader = "NeoForge"
			} else {
				info.Loader = "Forge"
			}
			return info
		case "mcmod.info":
			data, err := readZipEntry(f)
			if err != nil {
				info.Err = err.Error()
				return info
			}
			meta, err := parseMCMODInfo(data)
			if err != nil {
				info.Err = "解析 mcmod.info: " + err.Error()
				return info
			}
			info.ID, info.Name, info.Version = meta.ID, meta.Name, meta.Version
			info.Description = joinDescription(info.Description, meta.Description)
			info.Loader = "Forge"
			return info
		}
	}
	if info.Loader == "" {
		info.Loader = "未知"
	}
	return info
}

func joinDescription(existing, extra string) string {
	extra = strings.TrimSpace(extra)
	if extra == "" {
		return existing
	}
	if existing == "" {
		return extra
	}
	return existing + " · " + extra
}

type modMetadata struct {
	ID          string
	Name        string
	Version     string
	Description string
}

// parseModsTOML extracts the first mod entry from a Forge/NeoForge mods.toml.
// The format uses simple key="value" lines; full TOML is not required here.
func parseModsTOML(data string) modMetadata {
	var meta modMetadata
	parseLine := func(line string) {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "[") {
			return
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			return
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		value = strings.TrimPrefix(value, `"`)
		value = strings.TrimSuffix(value, `"`)
		value = strings.Trim(value, "'")
		value = strings.TrimSpace(value)
		switch key {
		case "modId":
			if meta.ID == "" {
				meta.ID = value
			}
		case "displayName":
			if meta.Name == "" {
				meta.Name = value
			}
		case "version":
			if meta.Version == "" {
				meta.Version = value
			}
		case "description":
			if meta.Description == "" {
				meta.Description = value
			}
		}
	}
	for _, line := range strings.Split(data, "\n") {
		parseLine(line)
	}
	return meta
}

func parseMCMODInfo(data []byte) (modMetadata, error) {
	type legacyMod struct {
		ModID       string `json:"modid"`
		Name        string `json:"name"`
		Version     string `json:"version"`
		Description string `json:"description"`
	}
	var list []legacyMod
	if err := json.Unmarshal(data, &list); err != nil {
		var wrapper struct {
			ModList []legacyMod `json:"modList"`
		}
		if err2 := json.Unmarshal(data, &wrapper); err2 != nil {
			return modMetadata{}, err
		}
		list = wrapper.ModList
	}
	if len(list) == 0 {
		return modMetadata{}, nil
	}
	return modMetadata{
		ID:          list[0].ModID,
		Name:        list[0].Name,
		Version:     list[0].Version,
		Description: list[0].Description,
	}, nil
}
