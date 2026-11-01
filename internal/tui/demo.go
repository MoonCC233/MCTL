package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/MoonCC233/MCTL/internal/config"
	"github.com/MoonCC233/MCTL/internal/logbuf"
	"github.com/MoonCC233/MCTL/internal/minecraft"
)

// RenderDemo renders the interface with representative data. It is used by
// the --demo flag for previews and documentation screenshots.
func RenderDemo(page string, width, height int) string {
	now := time.Now()
	cfg := config.Config{
		GameDir:         `C:\Users\Alex\AppData\Roaming\.minecraft`,
		JavaPath:        "",
		MinMemory:       512,
		MaxMemory:       6144,
		Width:           1280,
		Height:          720,
		DownloadSource:  "bmclapi",
		SelectedVersion: "fabric-loader-0.16.9-1.21.4",
		SelectedAccount: "Alex",
		AutoJava:        true,
		Accounts: []config.Account{
			{Name: "Alex", Type: "offline", UUID: minecraft.OfflineUUID("Alex")},
			{Name: "Steve", Type: "offline", UUID: minecraft.OfflineUUID("Steve")},
		},
	}
	store := &config.Store{Cfg: cfg, Path: "demo"}
	logs := logbuf.New(500)
	logs.AddRaw("[MCTL] 已检测到 Java 21.0.11（C:\\Program Files\\Eclipse Adoptium\\jdk-21\\bin\\java.exe）")
	logs.AddRaw("[MCTL] 版本列表已更新（912 个版本）")
	logs.AddRaw("[12:01:04] [main/INFO]: Loading Minecraft 1.21.4 with Fabric Loader 0.16.9")
	logs.AddRaw("[12:01:05] [main/INFO]: Loading 48 mods:")
	logs.AddRaw("[12:01:05] [main/INFO]: - fabric-api 0.110.5+1.21.4")
	logs.AddRaw("[12:01:05] [main/INFO]: - sodium 0.6.9+mc1.21.4")
	logs.AddRaw("[12:01:06] [Render thread/INFO]: OpenGL debug message: Using hardware acceleration")
	logs.AddRaw("[12:01:07] [Render thread/INFO]: Backend library: LWJGL version 3.3.3+5")
	logs.AddRaw("[12:01:08] [Render thread/INFO]: Narrator library for x64 successfully loaded")
	rt := NewRuntime(store, logs)
	rt.java = &minecraft.JavaInfo{Path: `C:\Program Files\Eclipse Adoptium\jdk-21\bin\java.exe`, Version: "21.0.11", Major: 21, Vendor: "Eclipse Adoptium", Arch: "amd64"}
	a := New(store, rt)
	a.width, a.height = width, height
	a.demo = true
	a.launch.java = rt.java
	a.settings.detected = rt.java
	a.launch.versions = []minecraft.InstalledVersion{
		{ID: "fabric-loader-0.16.9-1.21.4", Type: "release", MCVersion: "1.21.4", Loader: "Fabric", JavaMajor: 21, Size: 186 << 20, Mods: 48, Complete: true, Dir: cfg.GameDir + `\versions\fabric-loader-0.16.9-1.21.4`, ReleaseTime: now.AddDate(0, -1, 0).Format(time.RFC3339)},
		{ID: "1.20.1-forge-47.3.0", Type: "release", MCVersion: "1.20.1", Loader: "Forge", JavaMajor: 17, Size: 154 << 20, Mods: 48, Complete: true, Dir: cfg.GameDir + `\versions\1.20.1-forge-47.3.0`, ReleaseTime: now.AddDate(0, -8, 0).Format(time.RFC3339)},
		{ID: "1.21.1", Type: "release", MCVersion: "1.21.1", Loader: "Vanilla", JavaMajor: 21, Size: 92 << 20, Mods: 48, Complete: true, Dir: cfg.GameDir + `\versions\1.21.1`, ReleaseTime: now.AddDate(0, -3, 0).Format(time.RFC3339)},
		{ID: "1.8.9", Type: "release", MCVersion: "1.8.9", Loader: "Vanilla", JavaMajor: 8, Size: 41 << 20, Mods: 48, Complete: false, Dir: cfg.GameDir + `\versions\1.8.9`, ReleaseTime: now.AddDate(-9, 0, 0).Format(time.RFC3339)},
	}
	a.versions.list = a.launch.versions
	a.launch.versionIdx = 0
	a.launch.focus = 2
	a.launch.stage = ""
	a.download.installed = map[string]bool{}
	for _, v := range a.launch.versions {
		a.download.installed[v.ID] = true
	}
	a.download.manifest = demoManifest(now)
	a.download.idx = 0

	render := func(p pageID) string {
		a.page = p
		return a.View()
	}
	if page == "all" {
		var b strings.Builder
		for _, p := range []pageID{pageLaunch, pageVersions, pageDownload, pageSettings, pageLogs} {
			fmt.Fprintf(&b, "── %s ──\n", pageNames[p])
			b.WriteString(render(p))
			b.WriteString("\n\n")
		}
		return strings.TrimRight(b.String(), "\n")
	}
	switch page {
	case "versions":
		return render(pageVersions)
	case "download":
		return render(pageDownload)
	case "settings":
		return render(pageSettings)
	case "logs":
		return render(pageLogs)
	default:
		return render(pageLaunch)
	}
}

func demoManifest(now time.Time) *minecraft.VersionManifest {
	ids := []struct {
		id, typ string
		age     time.Duration
	}{
		{"1.21.4", "release", 35 * 24 * time.Hour},
		{"1.21.4-rc1", "snapshot", 38 * 24 * time.Hour},
		{"1.21.3", "release", 70 * 24 * time.Hour},
		{"1.21.2", "release", 95 * 24 * time.Hour},
		{"1.21.1", "release", 130 * 24 * time.Hour},
		{"24w40a", "snapshot", 150 * 24 * time.Hour},
		{"1.21", "release", 180 * 24 * time.Hour},
		{"1.20.6", "release", 210 * 24 * time.Hour},
		{"1.20.1", "release", 500 * 24 * time.Hour},
		{"1.8.9", "release", 3000 * 24 * time.Hour},
	}
	m := &minecraft.VersionManifest{}
	for _, v := range ids {
		t := now.Add(-v.age)
		m.Versions = append(m.Versions, minecraft.VersionEntry{ID: v.id, Type: v.typ, ReleaseTime: t, Time: t, URL: "https://example.invalid/" + v.id + ".json"})
	}
	m.Latest.Release = "1.21.4"
	m.Latest.Snapshot = "1.21.4-rc1"
	return m
}
