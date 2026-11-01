package tui

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/MoonCC233/MCTL/internal/config"
	"github.com/MoonCC233/MCTL/internal/minecraft"
)

var accountNameRe = regexp.MustCompile(`^[A-Za-z0-9_]{3,16}$`)

type loaderPickerData struct {
	Kind string
	MC   string
}

type worldDeleteData struct {
	GameDir string
	Name    string
}

type resourceModalData struct {
	Kind    string
	GameDir string
}

type resourceDeleteData struct {
	Kind    string
	GameDir string
	Name    string
}

type screenshotDeleteData struct {
	GameDir string
	Name    string
}

type serverListData struct {
	GameDir string
	Servers []minecraft.Server
}

type serverEditData struct {
	GameDir string
	Index   int
	Name    string
	Server  minecraft.Server
}

func localResourceLabel(kind string) string {
	switch kind {
	case "mod":
		return "模组"
	case "resourcepack":
		return "资源包"
	case "shader":
		return "光影"
	case "schematic":
		return "投影"
	}
	return kind
}

func (a *App) worldGameDir() string {
	cfg := a.store.Snapshot()
	for _, v := range a.launch.versions {
		if v.ID == cfg.SelectedVersion && v.GameDir != "" {
			return v.GameDir
		}
	}
	return cfg.GameDir
}

func (a *App) openLocalResourceTypePicker() {
	items := []pickerItem{
		{Label: "模组", Desc: "Mods", Value: "mod"},
		{Label: "资源包", Desc: "Resource Packs", Value: "resourcepack"},
		{Label: "光影", Desc: "Shaders", Value: "shader"},
	}
	a.modal = modalState{Kind: modalPicker, Title: "本地资源", Items: items, Action: "choose-local-resource-kind"}
}

func (a *App) openDatapackWorldPicker() {
	gameDir := a.worldGameDir()
	worlds, err := minecraft.ListWorlds(gameDir)
	if err != nil {
		a.openError("读取存档失败", err)
		return
	}
	if len(worlds) == 0 {
		a.setToast("请先在存档管理中创建或导入一个存档", 3)
		return
	}
	items := make([]pickerItem, 0, len(worlds))
	for _, w := range worlds {
		items = append(items, pickerItem{Label: w.Name, Desc: formatBytes(w.Size), Value: w.Dir})
	}
	a.modal = modalState{Kind: modalPicker, Title: "数据包 · 选择存档", Items: items, Action: "choose-datapack-world"}
}

func (a *App) openResourceFiles(kind string) {
	gameDir := a.worldGameDir()
	resources, err := minecraft.ListResources(gameDir, kind)
	if err != nil {
		a.openError("读取本地资源失败", err)
		return
	}
	items := make([]pickerItem, 0, len(resources))
	for _, r := range resources {
		status := "已启用"
		if r.Disabled {
			status = "已禁用"
		}
		items = append(items, pickerItem{
			Label: r.Name,
			Desc:  formatBytes(r.Size) + " · " + status,
			Value: r.Name,
		})
	}
	a.modal = modalState{
		Kind:  modalResources,
		Title: "本地资源 · " + localResourceLabel(kind),
		Items: items,
		Data:  resourceModalData{Kind: kind, GameDir: gameDir},
	}
}

type instanceInstallData struct {
	GameDir   string
	VersionID string
	MCVersion string
}

func (a *App) openInstanceInstall() {
	cfg := a.store.Snapshot()
	versionID := cfg.SelectedVersion
	if strings.TrimSpace(versionID) == "" {
		a.setToast("请先在版本页选择一个版本", 3)
		return
	}
	gameDir := cfg.GameDir
	for _, v := range a.launch.versions {
		if v.ID == versionID {
			if v.GameDir != "" {
				gameDir = v.GameDir
			}
			mcVersion := v.MCVersion
			if mcVersion == "" {
				mcVersion = versionID
			}
			a.showInstanceInstallPicker(gameDir, versionID, mcVersion)
			return
		}
	}
	if list, err := minecraft.ScanVersions(cfg.GameDir); err == nil {
		for _, v := range list {
			if v.ID != versionID {
				continue
			}
			if v.GameDir != "" {
				gameDir = v.GameDir
			}
			mcVersion := v.MCVersion
			if mcVersion == "" {
				mcVersion = versionID
			}
			a.showInstanceInstallPicker(gameDir, versionID, mcVersion)
			return
		}
	}
	a.setToast("请先在版本页选择一个版本", 3)
}

func (a *App) showInstanceInstallPicker(gameDir, versionID, mcVersion string) {
	label := func(kind string) string {
		switch kind {
		case "fabric":
			return "Fabric"
		case "quilt":
			return "Quilt"
		case "legacyfabric":
			return "Legacy Fabric"
		case "forge":
			return "Forge"
		case "neoforge":
			return "NeoForge"
		}
		return kind
	}
	items := []pickerItem{
		{Label: "补全 / 修复文件", Desc: "重新校验客户端、依赖库与资源文件", Value: "repair"},
	}
	prefix := "安装 "
	if language == "en" {
		prefix = "Install "
	}
	for _, kind := range []string{"fabric", "quilt", "legacyfabric", "forge", "neoforge"} {
		items = append(items, pickerItem{Label: prefix + label(kind), Desc: "自动选择最新构建", Value: kind})
	}
	a.modal = modalState{
		Kind:   modalPicker,
		Title:  "实例安装 · " + versionID,
		Items:  items,
		Data:   instanceInstallData{GameDir: gameDir, VersionID: versionID, MCVersion: mcVersion},
		Action: "instance-install",
	}
}

func (a *App) openScreenshots() {
	gameDir := a.worldGameDir()
	shots, err := minecraft.ListScreenshots(gameDir)
	if err != nil {
		a.openError("读取截图失败", err)
		return
	}
	items := make([]pickerItem, 0, len(shots))
	for _, s := range shots {
		items = append(items, pickerItem{
			Label: s.Name,
			Desc:  formatBytes(s.Size) + " · " + s.Updated.Format("2006-01-02 15:04"),
			Value: s.Name,
		})
	}
	a.modal = modalState{Kind: modalScreenshots, Title: "截图管理", Items: items, Data: gameDir}
}

func (a *App) openExportInstance() {
	cfg := a.store.Snapshot()
	versionID := cfg.SelectedVersion
	if strings.TrimSpace(versionID) == "" {
		a.setToast("请先在版本页选择一个版本", 3)
		return
	}
	dest := minecraft.DefaultInstanceExportPath(cfg.GameDir, versionID)
	a.openInput("导出实例", "导出到 zip 文件", dest, dest, "export-instance")
	a.modal.Data = versionID
}

func (a *App) openImportInstance() {
	a.openInput("导入实例", "MCTL 实例 zip 文件路径", "", `C:\Users\Alex\Downloads\instance.zip`, "import-instance")
}

func (a *App) openServers() {
	gameDir := a.worldGameDir()
	servers, err := minecraft.ListServers(gameDir)
	if err != nil {
		a.openError("读取服务器列表失败", err)
		return
	}
	items := make([]pickerItem, 0, len(servers))
	for _, s := range servers {
		items = append(items, pickerItem{Label: s.Name, Desc: s.IP, Value: s.IP})
	}
	a.modal = modalState{
		Kind:  modalServers,
		Title: "服务器列表",
		Items: items,
		Data:  serverListData{GameDir: gameDir, Servers: servers},
	}
}

func (a *App) openWorlds() {
	gameDir := a.worldGameDir()
	worlds, err := minecraft.ListWorlds(gameDir)
	if err != nil {
		a.openError("读取存档失败", err)
		return
	}
	items := make([]pickerItem, 0, len(worlds))
	for _, w := range worlds {
		items = append(items, pickerItem{
			Label: w.Name,
			Desc:  formatBytes(w.Size) + " · " + w.Updated.Format("2006-01-02 15:04"),
			Value: w.Name,
		})
	}
	a.modal = modalState{Kind: modalWorlds, Title: "存档管理", Items: items, Data: gameDir}
}

func (a *App) openCurseForgeSearch() {
	cfg := a.store.Snapshot()
	if strings.TrimSpace(cfg.CurseForgeAPIKey) == "" {
		a.openError("CurseForge", fmt.Errorf("请先在设置中填写 CurseForge API Key"))
		return
	}
	mcVersion := ""
	for _, v := range a.launch.versions {
		if v.ID == cfg.SelectedVersion {
			mcVersion = v.MCVersion
			break
		}
	}
	if mcVersion == "" && a.download.manifest != nil {
		mcVersion = a.download.manifest.Latest.Release
	}
	a.openInput("CurseForge 整合包", "搜索关键词", "", "例如 Better MC", "search-curseforge")
	a.modal.Data = mcVersion
}

func (a *App) openCurseForgeProject() {
	cfg := a.store.Snapshot()
	if strings.TrimSpace(cfg.CurseForgeAPIKey) == "" {
		a.openError("CurseForge", fmt.Errorf("请先在设置中填写 CurseForge API Key"))
		return
	}
	a.openInput("CurseForge 项目", "项目 ID 或链接", "", "例如 285109 或 .../modpacks/rlcraft", "load-curseforge-project")
}

func (a *App) openCommunityResources() {
	items := []pickerItem{
		{Label: "模组", Desc: "Modrinth", Value: "mod"},
		{Label: "资源包", Desc: "Resource Packs", Value: "resourcepack"},
		{Label: "光影", Desc: "Shaders", Value: "shader"},
		{Label: "数据包", Desc: "Data Packs", Value: "datapack"},
		{Label: "世界", Desc: "Worlds", Value: "world"},
		{Label: "整合包", Desc: "Modpacks", Value: "modpacks"},
		{Label: "收藏", Desc: "Favorites", Value: "favorites"},
	}
	a.modal = modalState{Kind: modalPicker, Title: "社区资源", Items: items, Action: "community-resource"}
}

func (a *App) openOnlineModpacks() {
	cfg := a.store.Snapshot()
	mcVersion := ""
	for _, v := range a.launch.versions {
		if v.ID == cfg.SelectedVersion {
			mcVersion = v.MCVersion
			break
		}
	}
	if mcVersion == "" {
		mcVersion = "1.20.1"
	}
	a.openInput("在线整合包 · 游戏版本", "游戏版本（留空表示全部）", mcVersion, "1.20.1", "online-modpack-version")
}

func (a *App) openModpackLoaderPicker(mcVersion string) {
	items := []pickerItem{
		{Label: "任意加载器", Desc: "不限制", Value: ""},
		{Label: "Fabric", Desc: "fabric", Value: "fabric"},
		{Label: "Quilt", Desc: "quilt", Value: "quilt"},
		{Label: "Forge", Desc: "forge", Value: "forge"},
		{Label: "NeoForge", Desc: "neoforge", Value: "neoforge"},
	}
	a.modal = modalState{Kind: modalPicker, Title: "在线整合包 · 加载器", Items: items, Action: "online-modpack-loader", Data: mcVersion}
}

func favoriteKindLabel(kind string) string {
	switch kind {
	case "modpack":
		return "整合包"
	case "resourcepack":
		return "资源包"
	case "shader":
		return "光影"
	case "mod":
		return "模组"
	case "datapack":
		return "数据包"
	}
	return kind
}

func (a *App) openFavorites() {
	items := []pickerItem{
		{Label: "整合包", Desc: "Modpacks", Value: "modpack"},
		{Label: "资源包", Desc: "Resource Packs", Value: "resourcepack"},
		{Label: "光影", Desc: "Shaders", Value: "shader"},
		{Label: "模组", Desc: "Mods", Value: "mod"},
		{Label: "数据包", Desc: "Data Packs", Value: "datapack"},
	}
	a.modal = modalState{Kind: modalPicker, Title: "收藏", Items: items, Action: "favorites-kind"}
}

func (a *App) openFavoriteList(kind string) {
	favs := a.store.Snapshot().FavoritesOf(kind)
	if len(favs) == 0 {
		a.setToast("没有收藏的"+favoriteKindLabel(kind), 3)
		return
	}
	items := make([]pickerItem, 0, len(favs))
	for _, f := range favs {
		items = append(items, pickerItem{Label: f.Name, Desc: favoriteKindLabel(kind), Value: f.ID})
	}
	a.modal = modalState{Kind: modalPicker, Title: "收藏 · " + favoriteKindLabel(kind), Items: items, Action: "install-favorite", Data: kind}
}

func (a *App) openResourceTypePicker() {
	items := []pickerItem{
		{Label: "资源包", Desc: "Resource Packs", Value: "resourcepack"},
		{Label: "光影", Desc: "Shaders", Value: "shader"},
		{Label: "模组", Desc: "Mods", Value: "mod"},
		{Label: "数据包", Desc: "Data Packs", Value: "datapack"},
	}
	a.modal = modalState{Kind: modalPicker, Title: "在线资源", Items: items, Action: "choose-resource-kind"}
}

func (a *App) openMicrosoftLogin() {
	cfg := a.store.Snapshot()
	if strings.TrimSpace(cfg.MicrosoftClientID) == "" {
		a.openError("Microsoft 登录", fmt.Errorf("请先在设置中填写 Microsoft 客户端 ID；Azure 应用需要启用公共客户端设备代码流"))
		return
	}
	a.modal = modalState{
		Kind:   modalProgress,
		Title:  "Microsoft 登录",
		Stage:  "正在请求设备代码…",
		Action: "cancel-auth",
	}
	a.rt.StartMicrosoftLogin(cfg.MicrosoftClientID)
}

func (a *App) runAction(action, value string, item pickerItem, data any) tea.Cmd {
	switch {
	case action == "quit":
		if a.rt.Launching() {
			a.rt.Stop()
		}
		if a.rt.Installing() {
			a.rt.CancelInstall()
		}
		a.rt.CancelMicrosoftLogin()
		a.quitting = true
		return tea.Quit
	case action == "stop-game":
		a.rt.Stop()
		a.setToast("正在停止游戏…", 3)
	case action == "delete-version":
		id, _ := data.(string)
		gameDir := a.store.Snapshot().GameDir
		for _, v := range a.launch.versions {
			if v.ID == id && v.GameDir != "" {
				gameDir = v.GameDir
				break
			}
		}
		if err := minecraft.DeleteVersion(gameDir, id); err != nil {
			a.openError("删除版本失败", err)
			return nil
		}
		if a.store.Snapshot().SelectedVersion == id {
			a.saveCfg(func(c *config.Config) { c.SelectedVersion = "" })
		}
		a.setToast("已删除版本 "+id, 1)
		a.rt.ScanVersions()
	case action == "install-version":
		id, _ := data.(string)
		a.installByID(id)
	case action == "install-fabric-loader":
		mc, _ := data.(string)
		loader := item.Value
		profileID := fmt.Sprintf("fabric-loader-%s-%s", loader, mc)
		a.startInstall("fabric", profileID, mc, loader, nil)
	case action == "install-loader":
		d, _ := data.(loaderPickerData)
		if d.Kind == "" || d.MC == "" || item.Value == "" {
			return nil
		}
		id := fmt.Sprintf("%s-%s-%s", d.Kind, d.MC, item.Value)
		a.startInstall(d.Kind, id, d.MC, item.Value, nil)
	case action == "import-modpack":
		a.startModpackImport(strings.TrimSpace(value))
	case action == "search-modpacks":
		query := strings.TrimSpace(value)
		filter, _ := data.(minecraft.ModpackFilter)
		a.setToast("正在搜索在线整合包…", 0)
		a.rt.SearchModpacks(query, filter)
	case action == "community-resource":
		switch item.Value {
		case "mod", "resourcepack", "shader":
			a.resources.kind = item.Value
			a.openInput("在线资源 · "+resourceKindLabel(item.Value), "搜索 Modrinth "+resourceKindLabel(item.Value), "", "输入关键词", "search-resources")
		case "world":
			a.openInput("下载世界", "世界压缩包（zip）地址", "", "https://example.com/world.zip", "download-world-url")
		case "datapack":
			a.openDatapackWorldPicker()
		case "modpacks":
			items := []pickerItem{
				{Label: "Modrinth", Desc: "在线整合包", Value: "modrinth"},
				{Label: "CurseForge", Desc: "需要 API Key", Value: "curseforge"},
			}
			a.modal = modalState{Kind: modalPicker, Title: "整合包来源", Items: items, Action: "community-pack-source"}
		case "favorites":
			a.openFavorites()
		}
	case action == "choose-datapack-world":
		a.resources.kind = "datapack"
		a.resources.worldDir = item.Value
		if a.resources.pendingID != "" {
			id, name := a.resources.pendingID, a.resources.pendingName
			a.resources.pendingID, a.resources.pendingName = "", ""
			a.startDatapackInstall(id, name, item.Value)
			return nil
		}
		a.openInput("在线数据包", "搜索 Modrinth 数据包", "", "输入关键词", "search-resources")
	case action == "install-datapack":
		worldDir, _ := data.(string)
		a.startDatapackInstall(item.Value, strings.TrimPrefix(item.Label, "★ "), worldDir)
	case action == "instance-install":
		d, ok := data.(instanceInstallData)
		if !ok {
			return nil
		}
		if item.Value == "repair" {
			a.modal = modalState{Kind: modalProgress, Title: "修复实例 · " + d.VersionID, Stage: "校验文件", Action: "cancel-install"}
			a.rt.RepairInstance(d.GameDir, d.VersionID)
			return nil
		}
		a.modal = modalState{Kind: modalProgress, Title: "安装加载器 · " + item.Label, Stage: "准备安装", Action: "cancel-install"}
		a.rt.AddLoader(d.GameDir, item.Value, d.MCVersion, d.VersionID)
	case action == "download-world-url":
		a.startWorldDownload(strings.TrimSpace(value))
	case action == "community-pack-source":
		if item.Value == "curseforge" {
			a.openCurseForgeSearch()
		} else {
			a.openOnlineModpacks()
		}
	case action == "online-modpack-version":
		a.openModpackLoaderPicker(strings.TrimSpace(value))
	case action == "search-curseforge":
		mcVersion, _ := data.(string)
		query := strings.TrimSpace(value)
		a.setToast("正在搜索 CurseForge…", 0)
		a.rt.SearchCurseForgeModpacks(query, mcVersion)
	case action == "load-curseforge-project":
		id, err := minecraft.ParseCurseForgeID(value)
		if err != nil {
			a.openError("CurseForge", err)
			return nil
		}
		a.setToast("正在读取 CurseForge 项目…", 0)
		a.rt.LoadCurseForgeProject(id)
	case action == "install-curseforge-file":
		d, ok := data.(curseForgeFileData)
		if !ok {
			a.setToast("无效的 CurseForge 文件", 3)
			return nil
		}
		idx, err := strconv.Atoi(item.Value)
		if err != nil || idx < 0 || idx >= len(d.Files) {
			a.setToast("无效的 CurseForge 文件", 3)
			return nil
		}
		mod := d.Mod
		mod.LatestFiles = []minecraft.CurseForgeFile{d.Files[idx]}
		a.startCurseForgeInstall(mod)
	case action == "install-curseforge":
		mods, _ := data.([]minecraft.CurseForgeMod)
		id := item.Value
		for _, mod := range mods {
			if fmt.Sprint(mod.ID) == id {
				a.startCurseForgeInstall(mod)
				return nil
			}
		}
		a.setToast("无效的 CurseForge 项目", 3)
	case action == "online-modpack-loader":
		mcVersion, _ := data.(string)
		filter := minecraft.ModpackFilter{GameVersion: mcVersion, Loader: item.Value}
		a.openInput("在线整合包", "搜索 Modrinth 整合包", "", "例如 Create / Cobblemon", "search-modpacks")
		a.modal.Data = filter
	case action == "favorites":
		a.openFavorites()
	case action == "favorites-kind":
		a.openFavoriteList(item.Value)
	case action == "install-favorite":
		kind, _ := data.(string)
		name := strings.TrimPrefix(item.Label, "★ ")
		if kind == "modpack" {
			a.startOnlineModpack(item.Value, name)
		} else if kind == "datapack" {
			a.resources.pendingID = item.Value
			a.resources.pendingName = name
			a.openDatapackWorldPicker()
		} else {
			a.startResourceInstall(kind, item.Value, name)
		}
	case action == "install-online-modpack":
		a.startOnlineModpack(item.Value, strings.TrimPrefix(item.Label, "★ "))
	case action == "choose-resource-kind":
		if item.Value == "datapack" {
			a.openDatapackWorldPicker()
			return nil
		}
		a.resources.kind = item.Value
		a.openInput("在线资源 · "+resourceKindLabel(item.Value), "搜索 Modrinth "+resourceKindLabel(item.Value), "", "输入关键词", "search-resources")
	case action == "search-resources":
		query := strings.TrimSpace(value)
		a.setToast("正在搜索在线资源…", 0)
		a.rt.SearchResources(a.resources.kind, query)
	case action == "install-resource":
		a.startResourceInstall(a.resources.kind, item.Value, strings.TrimPrefix(item.Label, "★ "))
	case action == "delete-world":
		d, _ := data.(worldDeleteData)
		if err := minecraft.DeleteWorld(d.GameDir, d.Name); err != nil {
			a.openError("删除存档失败", err)
			return nil
		}
		a.setToast("已删除存档 "+d.Name, 1)
		a.openWorlds()
	case action == "import-world":
		path := strings.TrimSpace(value)
		if path == "" {
			a.setToast("请填写存档压缩包路径", 3)
			return nil
		}
		a.rt.ImportWorld(a.worldGameDir(), path)
		a.setToast("正在导入存档…", 0)
	case action == "choose-local-resource-kind":
		a.openResourceFiles(item.Value)
	case action == "delete-resource":
		d, _ := data.(resourceDeleteData)
		if err := minecraft.DeleteResource(d.GameDir, d.Kind, d.Name); err != nil {
			a.openError("删除资源失败", err)
			return nil
		}
		a.setToast("已删除: "+d.Name, 1)
		a.openResourceFiles(d.Kind)
	case action == "wizard-edit":
		d, ok := data.(instanceWizardEdit)
		if ok {
			a.buildInstanceWizard(d, value)
		}
	case action == "wizard-loader":
		state, ok := data.(instanceWizardState)
		if ok {
			state.Loader = item.Value
			a.showInstanceWizard(state, instanceWizardIndex("loader"))
		}
	case action == "save-instance-setting":
		d, ok := data.(instanceSettingEdit)
		if !ok {
			return nil
		}
		settings, err := minecraft.LoadInstanceSettings(d.GameDir, d.VersionID)
		if err != nil {
			a.openError("读取实例设置失败", err)
			return nil
		}
		value = strings.TrimSpace(value)
		switch d.Key {
		case "name":
			if runes := []rune(value); len(runes) > 64 {
				value = string(runes[:64])
			}
			settings.Name = value
		case "icon":
			if runes := []rune(value); len(runes) > 4 {
				a.openError("图标无效", fmt.Errorf("实例图标最多 4 个字符"))
				return nil
			}
			settings.Icon = value
		case "link":
			if value != "" && !strings.HasPrefix(value, "http://") && !strings.HasPrefix(value, "https://") {
				a.openError("链接无效", fmt.Errorf("链接需要以 http:// 或 https:// 开头"))
				return nil
			}
			settings.Link = value
		case "max", "min":
			n := 0
			if value != "" {
				n, err = strconv.Atoi(value)
				if err != nil || n < 256 {
					a.openError("内存无效", fmt.Errorf("请输入不小于 256 的整数（MB），留空表示跟随全局"))
					return nil
				}
			}
			if d.Key == "max" {
				if n > 0 && settings.MinMemory > n {
					a.openError("内存无效", fmt.Errorf("最大内存不能小于最小内存（%d MB）", settings.MinMemory))
					return nil
				}
				settings.MaxMemory = n
			} else {
				effectiveMax := settings.MaxMemory
				if effectiveMax == 0 {
					effectiveMax = a.store.Snapshot().MaxMemory
				}
				if n > 0 && effectiveMax > 0 && n > effectiveMax {
					a.openError("内存无效", fmt.Errorf("最小内存不能大于最大内存（%d MB）", effectiveMax))
					return nil
				}
				settings.MinMemory = n
			}
		case "jvm":
			settings.JVMArgs = value
		case "game":
			settings.GameArgs = value
		case "server":
			if strings.ContainsAny(value, " \t") {
				a.openError("服务器地址无效", fmt.Errorf("服务器地址不能包含空格"))
				return nil
			}
			if value != "" {
				if host, _ := minecraft.SplitServerAddress(value); host == "" {
					a.openError("服务器地址无效", fmt.Errorf("请输入 host 或 host:port"))
					return nil
				}
			}
			settings.Server = value
		case "java":
			settings.JavaPath = value
		}
		if err := minecraft.SaveInstanceSettings(d.GameDir, d.VersionID, settings); err != nil {
			a.openError("保存实例设置失败", err)
			return nil
		}
		a.setToast("实例设置已保存", 1)
		a.showInstanceSettings(d.GameDir, d.VersionID, d.Key)
		a.rt.ScanVersions()
	case action == "diagnose":
		a.setToast("正在检测运行环境…", 0)
		a.rt.RunDiagnose()
	case action == "update":
		a.setToast("正在检查更新…", 0)
		a.rt.CheckUpdate()
	case action == "open-release":
		url, _ := data.(string)
		if url == "" {
			a.setToast("发布页面地址无效", 3)
			return nil
		}
		if err := openPath(url); err != nil {
			a.setToast("打开浏览器失败: "+err.Error(), 2)
		}
	case action == "open-news":
		if item.Value == "" {
			a.setToast("没有可打开的发布页面", 3)
			return nil
		}
		if err := openPath(item.Value); err != nil {
			a.setToast("打开浏览器失败: "+err.Error(), 2)
		}
	case action == "news":
		a.setToast("正在获取更新日志…", 0)
		a.rt.FetchNews()
	case action == "select-java":
		path := item.Value
		a.saveCfg(func(c *config.Config) { c.JavaPath = path })
		a.rt.DetectJava(path)
		if path == "" {
			a.setToast("已切换为自动检测 Java", 1)
		} else {
			a.setToast("已选择 "+path, 1)
		}
	case action == "modal-context":
		d, ok := data.(modalContextData)
		if !ok {
			return nil
		}
		a.contextReturn = nil
		a.modal = d.Origin
		_, cmd := a.handleModalKey(keyMsg(item.Value))
		return cmd
	case action == "context-action":
		d, ok := data.(contextMenuData)
		if !ok {
			return nil
		}
		switch d.Kind {
		case "versions":
			list := a.filteredVersions()
			if d.Index < 0 || d.Index >= len(list) {
				return nil
			}
			a.versions.idx = d.Index
			return a.versionsKey(keyMsg(item.Value))
		case "download":
			list := a.download.versions()
			if d.Index < 0 || d.Index >= len(list) {
				return nil
			}
			a.download.idx = d.Index
			if item.Value == "mode" {
				a.openDownloadModePicker()
				return nil
			}
			return a.downloadKey(keyMsg(item.Value))
		}
	case action == "context-download-mode":
		mode, err := strconv.Atoi(item.Value)
		if err != nil || mode < 0 || mode >= len(loaderModeNames) {
			return nil
		}
		a.download.mode = mode
		a.download.idx = 0
		a.download.loaderLoading = false
		if a.download.manifest == nil && !a.download.loading {
			a.rt.LoadManifest(false)
		}
	case action == "export-instance":
		versionID, _ := data.(string)
		dest := strings.TrimSpace(value)
		if dest == "" {
			a.setToast("请填写导出路径", 3)
			return nil
		}
		a.modal = modalState{Kind: modalProgress, Title: "导出实例", Stage: "正在打包", Action: "instance-op"}
		a.setToast("正在导出实例…", 0)
		a.rt.ExportInstance(versionID, dest)
	case action == "import-instance":
		path := strings.TrimSpace(value)
		if path == "" {
			a.setToast("请填写实例压缩包路径", 3)
			return nil
		}
		a.modal = modalState{Kind: modalProgress, Title: "导入实例", Stage: "正在解压", Action: "instance-op"}
		a.rt.ImportInstance(path)
	case action == "delete-screenshot":
		d, _ := data.(screenshotDeleteData)
		if err := minecraft.DeleteScreenshot(d.GameDir, d.Name); err != nil {
			a.openError("删除截图失败", err)
			return nil
		}
		a.setToast("已删除截图 "+d.Name, 1)
		a.openScreenshots()
	case action == "add-server-name":
		name := strings.TrimSpace(value)
		a.openInput("添加服务器 · "+name, "服务器地址", "", "play.example.com", "add-server-address")
		a.modal.Data = name
	case action == "add-server-address":
		name, _ := data.(string)
		gameDir := a.worldGameDir()
		address := strings.TrimSpace(value)
		if err := minecraft.AddServer(gameDir, minecraft.Server{Name: name, IP: address, AcceptTextures: true}); err != nil {
			a.openError("添加服务器失败", err)
			return nil
		}
		a.setToast("已添加服务器 "+address, 1)
		a.openServers()
	case action == "edit-server":
		d, _ := data.(serverEditData)
		address := strings.TrimSpace(value)
		server := d.Server
		server.IP = address
		if err := minecraft.UpdateServer(d.GameDir, d.Index, server); err != nil {
			a.openError("更新服务器失败", err)
			return nil
		}
		a.setToast("已更新服务器 "+d.Name, 1)
		a.openServers()
	case action == "delete-server":
		d, _ := data.(serverEditData)
		if err := minecraft.DeleteServer(d.GameDir, d.Index); err != nil {
			a.openError("删除服务器失败", err)
			return nil
		}
		a.setToast("已移除服务器 "+d.Name, 1)
		a.openServers()
	case action == "filter-versions":
		a.versions.filter = strings.TrimSpace(value)
		a.versions.idx = 0
	case action == "filter-download":
		a.download.filter = strings.TrimSpace(value)
		a.download.idx = 0
	case action == "add-account":
		name := strings.TrimSpace(value)
		if !accountNameRe.MatchString(name) {
			a.openError("账户名无效", fmt.Errorf("玩家名需要是 3-16 位字母、数字或下划线"))
			return nil
		}
		err := a.store.SetAccount(config.Account{Name: name, Type: "offline", UUID: minecraft.OfflineUUID(name)})
		if err != nil {
			a.openError("保存账户失败", err)
			return nil
		}
		a.setToast("已添加账户 "+name, 1)
		a.openAccounts()
	case action == "remove-account":
		name, _ := data.(string)
		if len(a.store.Snapshot().Accounts) <= 1 {
			a.setToast("至少需要保留一个账户", 3)
			return nil
		}
		if err := a.store.RemoveAccount(name); err != nil {
			a.openError("删除账户失败", err)
			return nil
		}
		a.setToast("已删除账户 "+name, 1)
		a.openAccounts()
	case action == "select-version":
		id := item.Value
		for i, v := range a.launch.versions {
			if v.ID == id {
				a.launch.versionIdx = i
				break
			}
		}
		a.saveCfg(func(c *config.Config) { c.SelectedVersion = id })
		a.setToast("已选择版本 "+id, 1)
	case action == "select-account":
		name := item.Value
		a.saveCfg(func(c *config.Config) { c.SelectedAccount = name })
		a.setToast("已切换账户 "+name, 1)
	case action == "cancel-install":
		a.rt.CancelInstall()
	case strings.HasPrefix(action, "setting:"):
		a.submitSetting(strings.TrimPrefix(action, "setting:"), value)
	case action == "command":
		return a.runCommand(item.Value)
	}
	return nil
}

func (a *App) runCommand(cmd string) tea.Cmd {
	cfg := a.store.Snapshot()
	switch cmd {
	case "cmd-launch":
		a.startLaunch(cfg.SelectedVersion)
	case "cmd-page-versions":
		a.switchPage(pageVersions)
	case "cmd-page-download":
		a.switchPage(pageDownload)
	case "cmd-page-settings":
		a.switchPage(pageSettings)
	case "cmd-page-tools":
		a.switchPage(pageTools)
	case "cmd-page-about":
		a.switchPage(pageAbout)
	case "cmd-page-logs":
		a.switchPage(pageLogs)
	case "cmd-account":
		a.openAccounts()
	case "cmd-ms-login":
		a.openMicrosoftLogin()
	case "cmd-java":
		a.rt.DetectJava(cfg.JavaPath)
		a.setToast("正在检测 Java…", 0)
	case "cmd-refresh":
		a.rt.ScanVersions()
		a.rt.LoadManifest(true)
		a.setToast("正在刷新版本与列表…", 0)
	case "cmd-import-modpack":
		a.openInput("导入整合包", "Modrinth .mrpack 或 CurseForge 压缩包路径", "", `C:\Users\Alex\Downloads\pack.mrpack`, "import-modpack")
	case "cmd-online-modpacks":
		a.openOnlineModpacks()
	case "cmd-curseforge":
		a.openCurseForgeSearch()
	case "cmd-favorites":
		a.openFavorites()
	case "cmd-online-resources":
		a.openResourceTypePicker()
	case "cmd-worlds":
		a.openWorlds()
	case "cmd-local-resources":
		a.openLocalResourceTypePicker()
	case "cmd-schematics":
		a.openResourceFiles("schematic")
	case "cmd-screenshots":
		a.openScreenshots()
	case "cmd-servers":
		a.openServers()
	case "cmd-skin":
		a.setToast("正在获取皮肤…", 0)
		a.rt.FetchSkin()
	case "cmd-export-instance":
		a.openExportInstance()
	case "cmd-import-instance":
		a.openImportInstance()
	case "cmd-diagnose":
		a.setToast("正在检测运行环境…", 0)
		a.rt.RunDiagnose()
	case "cmd-update":
		a.setToast("正在检查更新…", 0)
		a.rt.CheckUpdate()
	case "cmd-instance-settings":
		a.openInstanceSettings()
	case "cmd-instance-install":
		a.openInstanceInstall()
	case "cmd-mods-info":
		a.openModsInfo()
	case "cmd-new-instance":
		a.openInstanceWizard()
	case "cmd-news":
		a.setToast("正在获取更新日志…", 0)
		a.rt.FetchNews()
	case "cmd-speedtest":
		a.modal = modalState{Kind: modalSpeed, Title: "网络测速"}
		a.rt.RunSpeedTest()
	case "cmd-folder":
		a.openGameDir()
	case "cmd-help":
		a.openHelp()
	case "cmd-quit":
		if a.rt.Busy() {
			a.openConfirm("退出 MCTL", "仍有任务正在进行。退出会结束游戏或取消安装，确定继续吗？", "quit", nil)
			return nil
		}
		a.quitting = true
		return tea.Quit
	}
	return nil
}

func (a *App) handlePageKey(msg tea.KeyMsg) tea.Cmd {
	switch a.page {
	case pageLaunch:
		return a.launchKey(msg)
	case pageVersions:
		return a.versionsKey(msg)
	case pageDownload:
		return a.downloadKey(msg)
	case pageSettings:
		return a.settingsKey(msg)
	case pageTools:
		return a.toolsKey(msg)
	case pageAbout:
		return a.aboutKey(msg)
	case pageLogs:
		return a.logsKey(msg)
	}
	return nil
}
