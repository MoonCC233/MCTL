package tui

import (
	"fmt"
	"strings"

	"github.com/MoonCC233/MCTL/internal/minecraft"
)

type instanceSettingsData struct {
	GameDir   string
	VersionID string
	Settings  minecraft.InstanceSettings
	GlobalMax int
	GlobalMin int
}

type instanceSettingEdit struct {
	GameDir   string
	VersionID string
	Key       string
}

func (a *App) openInstanceSettings() {
	cfg := a.store.Snapshot()
	versionID := cfg.SelectedVersion
	if strings.TrimSpace(versionID) == "" {
		a.setToast("请先在版本页选择一个版本", 3)
		return
	}
	gameDir := cfg.GameDir
	for _, v := range a.launch.versions {
		if v.ID == versionID && v.GameDir != "" {
			gameDir = v.GameDir
			break
		}
	}
	a.showInstanceSettings(gameDir, versionID, "")
}

func (a *App) showInstanceSettings(gameDir, versionID, focus string) {
	settings, err := minecraft.LoadInstanceSettings(gameDir, versionID)
	if err != nil {
		a.openError("读取实例设置失败", err)
		return
	}
	cfg := a.store.Snapshot()
	valueOr := func(v, fallback string) string {
		if strings.TrimSpace(v) == "" {
			return fallback
		}
		return middleEllipsis(v, 44)
	}
	memory := func(v, global int) string {
		if v > 0 {
			return fmt.Sprintf("%d MB", v)
		}
		return fmt.Sprintf("跟随全局 (%d MB)", global)
	}
	items := []pickerItem{
		{Label: "实例名称", Desc: valueOr(settings.Name, "未设置"), Value: "name"},
		{Label: "主页链接", Desc: valueOr(settings.Link, "未设置"), Value: "link"},
		{Label: "最大内存", Desc: memory(settings.MaxMemory, cfg.MaxMemory), Value: "max"},
		{Label: "最小内存", Desc: memory(settings.MinMemory, cfg.MinMemory), Value: "min"},
		{Label: "JVM 参数", Desc: valueOr(settings.JVMArgs, "跟随全局"), Value: "jvm"},
		{Label: "游戏参数", Desc: valueOr(settings.GameArgs, "跟随全局"), Value: "game"},
		{Label: "自动进入服务器", Desc: valueOr(settings.Server, "未设置"), Value: "server"},
		{Label: "单独 Java", Desc: valueOr(settings.JavaPath, "跟随全局"), Value: "java"},
	}
	sel := instanceSettingIndex(focus)
	a.modal = modalState{
		Kind:     modalInstanceSettings,
		Title:    "实例设置 · " + instanceDisplayName(settings.Name, versionID),
		Items:    items,
		Selected: sel,
		Data: instanceSettingsData{
			GameDir:   gameDir,
			VersionID: versionID,
			Settings:  *settings,
			GlobalMax: cfg.MaxMemory,
			GlobalMin: cfg.MinMemory,
		},
	}
}

func instanceDisplayName(name, versionID string) string {
	if strings.TrimSpace(name) == "" {
		return versionID
	}
	return name
}

func instanceSettingIndex(key string) int {
	for i, k := range []string{"name", "link", "max", "min", "jvm", "game", "server", "java"} {
		if k == key {
			return i
		}
	}
	return 0
}

func instanceSettingLabel(key string) string {
	switch key {
	case "name":
		return "实例名称"
	case "link":
		return "主页链接"
	case "max":
		return "最大内存（MB）"
	case "min":
		return "最小内存（MB）"
	case "jvm":
		return "JVM 参数"
	case "game":
		return "游戏参数"
	case "server":
		return "自动进入服务器（host:port）"
	case "java":
		return "单独 Java 路径"
	}
	return key
}

func instanceSettingValue(settings minecraft.InstanceSettings, key string) string {
	switch key {
	case "name":
		return settings.Name
	case "link":
		return settings.Link
	case "max":
		if settings.MaxMemory > 0 {
			return fmt.Sprintf("%d", settings.MaxMemory)
		}
	case "min":
		if settings.MinMemory > 0 {
			return fmt.Sprintf("%d", settings.MinMemory)
		}
	case "jvm":
		return settings.JVMArgs
	case "game":
		return settings.GameArgs
	case "server":
		return settings.Server
	case "java":
		return settings.JavaPath
	}
	return ""
}

func instanceSettingPlaceholder(key string) string {
	switch key {
	case "name":
		return "例如 生存整合包"
	case "link":
		return "https://example.com/pack"
	case "max", "min":
		return "留空跟随全局"
	case "jvm":
		return "-XX:+UseG1GC"
	case "game":
		return "--demo"
	case "server":
		return "play.example.com:25565"
	case "java":
		return `C:\Java\bin\java.exe`
	}
	return ""
}
