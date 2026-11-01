package minecraft

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// InstanceSettings are per-instance launch overrides. Zero values fall back to
// the global launcher configuration.
type InstanceSettings struct {
	Name      string `json:"name,omitempty"`
	Icon      string `json:"icon,omitempty"`
	Link      string `json:"link,omitempty"`
	MinMemory int    `json:"minMemory,omitempty"`
	MaxMemory int    `json:"maxMemory,omitempty"`
	JVMArgs   string `json:"jvmArgs,omitempty"`
	GameArgs  string `json:"gameArgs,omitempty"`
	Server    string `json:"server,omitempty"`
	JavaPath  string `json:"javaPath,omitempty"`
}

// InstanceSettingsPath returns the settings file for a version directory.
func InstanceSettingsPath(gameDir, versionID string) string {
	return filepath.Join(VersionPath(gameDir, versionID), "mctl-settings.json")
}

// LoadInstanceSettings reads overrides. A missing file yields empty settings.
func LoadInstanceSettings(gameDir, versionID string) (*InstanceSettings, error) {
	settings := &InstanceSettings{}
	if strings.TrimSpace(versionID) == "" {
		return settings, nil
	}
	data, err := os.ReadFile(InstanceSettingsPath(gameDir, versionID))
	if err != nil {
		if os.IsNotExist(err) {
			return settings, nil
		}
		return nil, err
	}
	if err := json.Unmarshal(data, settings); err != nil {
		return nil, fmt.Errorf("解析实例设置: %w", err)
	}
	return settings, nil
}

// SaveInstanceSettings replaces the settings file atomically.
func SaveInstanceSettings(gameDir, versionID string, settings *InstanceSettings) error {
	if strings.TrimSpace(versionID) == "" {
		return fmt.Errorf("缺少版本 ID")
	}
	dir := VersionPath(gameDir, versionID)
	if st, err := os.Stat(dir); err != nil || !st.IsDir() {
		return fmt.Errorf("版本目录不存在: %s", versionID)
	}
	if settings == nil {
		settings = &InstanceSettings{}
	}
	settings.Name = strings.TrimSpace(settings.Name)
	settings.Icon = strings.TrimSpace(settings.Icon)
	settings.Link = strings.TrimSpace(settings.Link)
	settings.Server = strings.TrimSpace(settings.Server)
	settings.JVMArgs = strings.TrimSpace(settings.JVMArgs)
	settings.GameArgs = strings.TrimSpace(settings.GameArgs)
	settings.JavaPath = strings.TrimSpace(settings.JavaPath)
	if settings.MinMemory < 0 || settings.MaxMemory < 0 {
		return fmt.Errorf("内存不能为负数")
	}
	if settings.MinMemory > 0 && settings.MaxMemory > 0 && settings.MinMemory > settings.MaxMemory {
		return fmt.Errorf("最小内存不能大于最大内存")
	}
	if runes := []rune(settings.Icon); len(runes) > 4 {
		return fmt.Errorf("实例图标最多 4 个字符")
	}
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, "mctl-settings-*.json")
	if err != nil {
		return err
	}
	name := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(name)
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		os.Remove(name)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(name)
		return err
	}
	if err := os.Rename(name, InstanceSettingsPath(gameDir, versionID)); err != nil {
		os.Remove(name)
		return err
	}
	return nil
}

// SplitServerAddress splits host and port, defaulting the port to 25565.
func SplitServerAddress(address string) (host, port string) {
	address = strings.TrimSpace(address)
	if address == "" {
		return "", ""
	}
	if h, p, err := net.SplitHostPort(address); err == nil && h != "" && p != "" {
		if _, err := strconv.Atoi(p); err == nil {
			return h, p
		}
	}
	if i := strings.LastIndex(address, ":"); i > 0 && !strings.Contains(address[:i], ":") {
		if p := address[i+1:]; p != "" {
			if _, err := strconv.Atoi(p); err == nil {
				return address[:i], p
			}
		}
	}
	return strings.Trim(address, "[]"), "25565"
}
