package minecraft

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"
)

// VersionEntry is one item in Mojang's version manifest.
type VersionEntry struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"`
	URL         string    `json:"url"`
	Time        time.Time `json:"time"`
	ReleaseTime time.Time `json:"releaseTime"`
	SHA1        string    `json:"sha1"`
	Compliance  int       `json:"complianceLevel"`
}

// VersionManifest is the official version catalogue.
type VersionManifest struct {
	Latest struct {
		Release  string `json:"release"`
		Snapshot string `json:"snapshot"`
	} `json:"latest"`
	Versions []VersionEntry `json:"versions"`
}

// FetchManifest downloads and decodes the version catalogue.
func FetchManifest(ctx context.Context, hc *HTTPClient) (*VersionManifest, error) {
	data, err := hc.Get(ctx, hc.Source.ManifestURL)
	if err != nil {
		return nil, fmt.Errorf("获取版本列表失败: %w", err)
	}
	var m VersionManifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("解析版本列表: %w", err)
	}
	if len(m.Versions) == 0 {
		return nil, fmt.Errorf("版本列表为空")
	}
	return &m, nil
}

// Find returns the entry with the given id.
func (m *VersionManifest) Find(id string) (VersionEntry, bool) {
	for _, v := range m.Versions {
		if v.ID == id {
			return v, true
		}
	}
	return VersionEntry{}, false
}

// Releases returns only release versions, newest first.
func (m *VersionManifest) Releases() []VersionEntry {
	out := make([]VersionEntry, 0, len(m.Versions))
	for _, v := range m.Versions {
		if v.Type == "release" {
			out = append(out, v)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].ReleaseTime.After(out[j].ReleaseTime) })
	return out
}

// NewestStable returns the newest release entry.
func (m *VersionManifest) NewestStable() (VersionEntry, bool) {
	rel := m.Releases()
	if len(rel) == 0 {
		return VersionEntry{}, false
	}
	return rel[0], true
}
