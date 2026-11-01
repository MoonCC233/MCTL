package tui

import (
	"fmt"
	"strings"

	"github.com/MoonCC233/MCTL/internal/minecraft"
)

type instanceWizardState struct {
	Name          string
	Dir           string
	MCVersion     string
	Loader        string
	LoaderVersion string
	JavaPath      string
}

type instanceWizardEdit struct {
	State instanceWizardState
	Key   string
}

func wizardLoaderLabel(kind string) string {
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
	return "原版"
}

func (a *App) openInstanceWizard() {
	cfg := a.store.Snapshot()
	state := instanceWizardState{Loader: "vanilla"}
	mcVersion := ""
	for _, v := range a.launch.versions {
		if v.ID == cfg.SelectedVersion && v.MCVersion != "" {
			mcVersion = v.MCVersion
			break
		}
	}
	if mcVersion == "" && cfg.SelectedVersion != "" {
		if list, err := minecraft.ScanVersions(cfg.GameDir); err == nil {
			for _, v := range list {
				if v.ID == cfg.SelectedVersion && v.MCVersion != "" {
					mcVersion = v.MCVersion
					break
				}
			}
		}
	}
	if mcVersion == "" && a.download.manifest != nil {
		mcVersion = a.download.manifest.Latest.Release
	}
	if mcVersion == "" {
		mcVersion = "1.20.1"
	}
	state.MCVersion = mcVersion
	a.showInstanceWizard(state, 0)
}

func (a *App) showInstanceWizard(state instanceWizardState, selected int) {
	name := state.Name
	if strings.TrimSpace(name) == "" {
		name = "未命名（使用版本号）"
	}
	dir := state.Dir
	if strings.TrimSpace(dir) == "" {
		dir = "自动：instances/<名称>-<短哈希>"
	}
	loaderVersion := state.LoaderVersion
	if strings.TrimSpace(loaderVersion) == "" {
		loaderVersion = "自动选择最新稳定版"
	}
	javaPath := state.JavaPath
	if strings.TrimSpace(javaPath) == "" {
		javaPath = "跟随全局设置 / 自动下载"
	}
	items := []pickerItem{
		{Label: "实例名称", Desc: middleEllipsis(name, 42), Value: "name"},
		{Label: "实例目录", Desc: middleEllipsis(dir, 42), Value: "dir"},
		{Label: "Minecraft 版本", Desc: state.MCVersion, Value: "mc"},
		{Label: "加载器", Desc: wizardLoaderLabel(state.Loader), Value: "loader"},
		{Label: "加载器版本", Desc: middleEllipsis(loaderVersion, 42), Value: "loaderVersion"},
		{Label: "Java 路径", Desc: middleEllipsis(javaPath, 42), Value: "java"},
		{Label: "创建实例", Desc: "开始安装并创建独立实例", Value: "create"},
	}
	a.modal = modalState{
		Kind:     modalInstanceWizard,
		Title:    "新建实例",
		Items:    items,
		Selected: min(max(selected, 0), len(items)-1),
		Data:     state,
	}
}

func wizardLoaderOptions() []pickerItem {
	out := make([]pickerItem, 0, len(minecraft.WizardLoaders))
	for _, kind := range minecraft.WizardLoaders {
		out = append(out, pickerItem{Label: wizardLoaderLabel(kind), Desc: kind, Value: kind})
	}
	return out
}

func (a *App) wizardStart(state instanceWizardState) {
	cfg := a.store.Snapshot()
	name := strings.TrimSpace(state.Name)
	mcVersion := strings.TrimSpace(state.MCVersion)
	if mcVersion == "" {
		a.setToast("请填写 Minecraft 版本", 3)
		return
	}
	if _, err := minecraft.ResolveInstanceDir(cfg.GameDir, name, mcVersion, state.Dir); err != nil {
		a.setToast(err.Error(), 2)
		return
	}
	opts := minecraft.InstanceWizardOptions{
		Name:          name,
		Dir:           strings.TrimSpace(state.Dir),
		MCVersion:     mcVersion,
		Loader:        state.Loader,
		LoaderVersion: strings.TrimSpace(state.LoaderVersion),
		JavaPath:      strings.TrimSpace(state.JavaPath),
		AutoJava:      cfg.AutoJava,
	}
	title := name
	if title == "" {
		title = mcVersion
	}
	a.modal = modalState{Kind: modalProgress, Title: "新建实例 · " + title, Stage: "准备安装", Action: "cancel-install"}
	a.rt.CreateInstance(opts)
}

func wizardPlaceholder(key string) string {
	switch key {
	case "name":
		return "例如 生存整合包"
	case "dir":
		return "留空自动创建；可填 instances 下的目录名"
	case "mc":
		return "例如 1.20.1"
	case "loaderVersion":
		return "留空自动选择最新稳定版"
	case "java":
		return `C:\Java\bin\java.exe`
	}
	return ""
}

func wizardEditLabel(key string) string {
	switch key {
	case "name":
		return "实例名称"
	case "dir":
		return "实例目录"
	case "mc":
		return "Minecraft 版本"
	case "loaderVersion":
		return "加载器版本"
	case "java":
		return "Java 路径"
	}
	return key
}

func wizardEditValue(state instanceWizardState, key string) string {
	switch key {
	case "name":
		return state.Name
	case "dir":
		return state.Dir
	case "mc":
		return state.MCVersion
	case "loaderVersion":
		return state.LoaderVersion
	case "java":
		return state.JavaPath
	}
	return ""
}

func (a *App) buildInstanceWizard(edit instanceWizardEdit, value string) {
	state := edit.State
	value = strings.TrimSpace(value)
	switch edit.Key {
	case "name":
		state.Name = value
	case "dir":
		state.Dir = value
	case "mc":
		if value != "" {
			state.MCVersion = value
		}
	case "loaderVersion":
		state.LoaderVersion = value
	case "java":
		state.JavaPath = value
	}
	index := instanceWizardIndex(edit.Key)
	a.showInstanceWizard(state, index)
}

func instanceWizardIndex(key string) int {
	for i, k := range []string{"name", "dir", "mc", "loader", "loaderVersion", "java", "create"} {
		if k == key {
			return i
		}
	}
	return 0
}

func (a *App) wizardError(msg string) {
	a.openError("新建实例", fmt.Errorf("%s", msg))
}
