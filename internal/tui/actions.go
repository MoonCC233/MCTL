package tui

import (
	"fmt"
	"regexp"
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
		a.openInput("导入整合包", "Modrinth .mrpack 文件路径", "", `C:\Users\Alex\Downloads\pack.mrpack`, "import-modpack")
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
	case pageLogs:
		return a.logsKey(msg)
	}
	return nil
}
