package minecraft

import "strings"

// Source describes a download endpoint set.
type Source struct {
	ID          string
	Name        string
	ManifestURL string
}

// Sources are the supported download origins.
var Sources = []Source{
	{
		ID:          "bmclapi",
		Name:        "BMCLAPI 镜像",
		ManifestURL: "https://bmclapi2.bangbang93.com/mc/game/version_manifest_v2.json",
	},
	{
		ID:          "official",
		Name:        "Mojang 官方",
		ManifestURL: "https://piston-meta.mojang.com/mc/game/version_manifest_v2.json",
	},
}

// GetSource resolves a source id, defaulting to BMCLAPI.
func GetSource(id string) Source {
	for _, s := range Sources {
		if s.ID == id {
			return s
		}
	}
	return Sources[0]
}

var rewrites = []struct{ from, to string }{
	{"https://piston-meta.mojang.com", "https://bmclapi2.bangbang93.com"},
	{"https://launchermeta.mojang.com", "https://bmclapi2.bangbang93.com"},
	{"https://launcher.mojang.com", "https://bmclapi2.bangbang93.com"},
	{"https://piston-data.mojang.com", "https://bmclapi2.bangbang93.com"},
	{"https://libraries.minecraft.net", "https://bmclapi2.bangbang93.com/maven"},
	{"https://resources.download.minecraft.net", "https://bmclapi2.bangbang93.com/assets"},
}

// Rewrite maps an upstream URL onto this source when a mirror is selected.
func (s Source) Rewrite(raw string) string {
	if s.ID != "bmclapi" || raw == "" {
		return raw
	}
	for _, r := range rewrites {
		if strings.HasPrefix(raw, r.from) {
			return r.to + strings.TrimPrefix(raw, r.from)
		}
	}
	return raw
}
