package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/MoonCC233/MCTL/internal/config"
	"github.com/MoonCC233/MCTL/internal/minecraft"
)

type launchPage struct {
	versions   []minecraft.InstalledVersion
	versionIdx int
	accountIdx int
	focus      int
	java       *minecraft.JavaInfo
	javaErr    error
	stage      string
	percent    float64
	detail     string
}

func (p *launchPage) selectedVersion(cfg config.Config) string {
	if len(p.versions) == 0 {
		return ""
	}
	if p.versionIdx < 0 || p.versionIdx >= len(p.versions) {
		p.versionIdx = 0
	}
	return p.versions[p.versionIdx].ID
}

func (p *launchPage) selectedAccount(cfg config.Config) config.Account {
	if len(cfg.Accounts) == 0 {
		return config.Account{Name: "Steve", Type: "offline"}
	}
	for i, acc := range cfg.Accounts {
		if acc.Name == cfg.SelectedAccount {
			p.accountIdx = i
			return acc
		}
	}
	if p.accountIdx < 0 || p.accountIdx >= len(cfg.Accounts) {
		p.accountIdx = 0
	}
	return cfg.Accounts[p.accountIdx]
}

func (p *launchPage) versionByID(id string) (minecraft.InstalledVersion, bool) {
	for _, v := range p.versions {
		if v.ID == id {
			return v, true
		}
	}
	return minecraft.InstalledVersion{}, false
}

func (a *App) viewLaunch(w, h int) []string {
	cfg := a.store.Snapshot()
	p := &a.launch
	subtitle := tr("选择版本与账户，然后开始游戏")
	if len(p.versions) == 0 {
		subtitle = tr("还没有安装任何版本 · 前往「下载」获取 Minecraft")
	} else {
		subtitle = trf("已安装 %d 个版本 · %s", len(p.versions), middleEllipsis(cfg.GameDir, max(w-32, 20)))
	}
	lines := pageHeader("启动游戏", subtitle, w)
	contentH := h - len(lines)
	if contentH < 6 {
		return fitLines(lines, h, w)
	}

	heroH := min(12, max(contentH-7, 7))
	infoH := contentH - heroH
	heroW := w - 4
	hero := a.renderHero(heroW, heroH, cfg)
	for _, l := range strings.Split(hero, "\n") {
		lines = append(lines, "  "+l)
	}
	if infoH >= 7 {
		gap := 1
		leftW := (w - 4 - gap) / 2
		rightW := w - 4 - gap - leftW
		left := a.renderEnvironment(leftW, infoH, cfg)
		right := a.renderRecentLog(rightW, infoH)
		cols := joinColumns(left, right, gap)
		lines = append(lines, "")
		for _, l := range cols {
			lines = append(lines, "  "+l)
		}
	}
	return fitLines(lines, h, w)
}

func (a *App) renderHero(w, h int, cfg config.Config) string {
	p := &a.launch
	inner := panelContentWidth(w)
	var content []string
	if len(p.versions) == 0 {
		content = append(content,
			"",
			"  "+styleMuted.Render("尚未安装游戏版本"),
			"  "+styleFaint.Render("按 3 打开下载页，安装原版或 Fabric"),
			"",
			centerText(styleFaint.Render("— 安装后即可从这里启动 —"), inner),
		)
	} else {
		if p.versionIdx < 0 || p.versionIdx >= len(p.versions) {
			p.versionIdx = 0
		}
		v := p.versions[p.versionIdx]
		focus := a.page == pageLaunch
		prefix := "  "
		if focus && a.launchFocus() == 0 {
			prefix = styleAccent.Render("▸ ")
		}
		content = append(content,
			"",
			"  "+prefix+styleBold.Render(middleEllipsis(v.ID, max(inner-8, 10))),
		)
		meta := v.Loader
		if v.MCVersion != "" && v.MCVersion != v.ID {
			meta += " · Minecraft " + v.MCVersion
		}
		meta += " · " + v.Type
		content = append(content, "     "+styleMuted.Render(ansiTruncate(meta, max(inner-6, 10))))

		acc := p.selectedAccount(cfg)
		accPrefix := "  "
		if focus && a.launchFocus() == 1 {
			accPrefix = styleAccent.Render("▸ ")
		}
		accKind := "离线"
		if acc.Type == "microsoft" {
			accKind = "Microsoft"
		}
		content = append(content,
			"",
			"  "+accPrefix+styleMuted.Render(tr("账户")+"  ")+dot(colorGreen)+" "+styleText.Render(acc.Name)+" "+styleFaint.Render("· "+tr(accKind)),
		)
		content = append(content, "")

		button := "  启动游戏  "
		if a.rt.Launching() {
			button = "  游戏运行中  "
		}
		if focus && a.launchFocus() == 2 && !a.rt.Launching() {
			content = append(content, centerText(lipgloss.NewStyle().Foreground(colorBg).Background(colorGreen).Bold(true).Padding(0, 3).Render(button), inner))
		} else {
			style := lipgloss.NewStyle().Foreground(colorGreen).Border(lipgloss.RoundedBorder()).BorderForeground(colorGreen).Padding(0, 2)
			content = append(content, centerText(style.Render(button), inner))
		}
		if p.stage != "" {
			pct := int(p.percent * 100)
			detail := p.stage
			if p.detail != "" {
				detail += " · " + p.detail
			}
			content = append(content,
				"",
				"  "+spinner(a.frame, fmt.Sprintf("%s %d%%", detail, pct)),
				"  "+progressBar(p.percent, max(inner-4, 10)),
			)
		} else if a.rt.Launching() {
			content = append(content, "", "  "+dot(colorGreen)+" "+styleGreen.Render("游戏进程运行中 · 按 s 停止"))
		}
	}
	for len(content) < panelContentHeight(h) {
		content = append(content, "")
	}
	if len(content) > panelContentHeight(h) {
		content = content[:panelContentHeight(h)]
	}
	return panel("当前实例", w, h, true, content)
}

func (a *App) launchFocus() int {
	if a.launch.focus < 0 || a.launch.focus > 2 {
		return 2
	}
	return a.launch.focus
}

func (a *App) renderEnvironment(w, h int, cfg config.Config) string {
	inner := panelContentWidth(w)
	java := "未检测到"
	if a.launch.java != nil {
		java = fmt.Sprintf("%s · %d", a.launch.java.Version, a.launch.java.Major)
	} else if a.launch.javaErr != nil {
		java = "未检测到"
	}
	mods := 0
	if len(a.launch.versions) > 0 {
		mods = a.launch.versions[0].Mods
	}
	lines := []string{
		"  " + styleMuted.Render("Java    ") + styleText.Render(ansiTruncate(java, max(inner-12, 10))),
		"  " + styleMuted.Render(tr("内存")+"    ") + styleText.Render(fmt.Sprintf("%d MB", cfg.MaxMemory)),
		"  " + styleMuted.Render(tr("目录")+"    ") + styleFaint.Render(middleEllipsis(cfg.GameDir, max(inner-12, 10))),
		"  " + styleMuted.Render(tr("模组")+"    ") + styleText.Render(fmt.Sprintf("%d 个", mods)),
		"  " + styleMuted.Render(tr("下载源")+"  ") + styleText.Render(minecraft.GetSource(cfg.DownloadSource).Name),
	}
	for len(lines) < panelContentHeight(h) {
		lines = append(lines, "")
	}
	if len(lines) > panelContentHeight(h) {
		lines = lines[:panelContentHeight(h)]
	}
	return panel("运行环境", w, h, false, lines)
}

func (a *App) renderRecentLog(w, h int) string {
	inner := panelContentWidth(w)
	tail := a.rt.Logs.Tail(max(panelContentHeight(h), 1))
	var lines []string
	if len(tail) == 0 {
		lines = append(lines, "", "  "+styleFaint.Render("暂无日志 · 启动后可在此查看"))
	} else {
		for _, l := range tail {
			lines = append(lines, "  "+styleFaint.Render(ansiTruncate(l, max(inner-2, 8))))
		}
	}
	for len(lines) < panelContentHeight(h) {
		lines = append(lines, "")
	}
	if len(lines) > panelContentHeight(h) {
		lines = lines[:panelContentHeight(h)]
	}
	return panel("最近日志", w, h, false, lines)
}

func (a *App) launchKey(msg tea.KeyMsg) tea.Cmd {
	cfg := a.store.Snapshot()
	p := &a.launch
	switch msg.String() {
	case "up", "k":
		p.focus = (p.focus - 1 + 3) % 3
	case "down", "j":
		p.focus = (p.focus + 1) % 3
	case "left", "h":
		switch p.focus {
		case 0:
			if len(p.versions) > 0 {
				p.versionIdx = (p.versionIdx - 1 + len(p.versions)) % len(p.versions)
				a.saveCfg(func(c *config.Config) { c.SelectedVersion = p.versions[p.versionIdx].ID })
			}
		case 1:
			if len(cfg.Accounts) > 0 {
				p.accountIdx = (p.accountIdx - 1 + len(cfg.Accounts)) % len(cfg.Accounts)
				a.saveCfg(func(c *config.Config) { c.SelectedAccount = cfg.Accounts[p.accountIdx].Name })
			}
		}
	case "right":
		switch p.focus {
		case 0:
			if len(p.versions) > 0 {
				p.versionIdx = (p.versionIdx + 1) % len(p.versions)
				a.saveCfg(func(c *config.Config) { c.SelectedVersion = p.versions[p.versionIdx].ID })
			}
		case 1:
			if len(cfg.Accounts) > 0 {
				p.accountIdx = (p.accountIdx + 1) % len(cfg.Accounts)
				a.saveCfg(func(c *config.Config) { c.SelectedAccount = cfg.Accounts[p.accountIdx].Name })
			}
		}
	case "enter":
		switch p.focus {
		case 0:
			a.openVersionPicker()
		case 1:
			a.openAccountPicker()
		case 2:
			a.startLaunch(p.selectedVersion(cfg))
		}
	case "a":
		a.openAccounts()
	case "i":
		a.openInstanceSettings()
	case "p":
		a.setToast("正在获取皮肤…", 0)
		a.rt.FetchSkin()
	case "d":
		a.switchPage(pageDownload)
	case "o":
		a.openGameDir()
	case "r":
		a.rt.ScanVersions()
		a.setToast("正在刷新版本列表", 0)
	case "s":
		if a.rt.Launching() {
			a.openConfirm("停止游戏", "确定要结束当前游戏进程吗？未保存的游戏进度可能丢失。", "stop-game", nil)
		}
	case "l":
		a.switchPage(pageLogs)
	}
	return nil
}

func (a *App) openVersionPicker() {
	if len(a.launch.versions) == 0 {
		a.setToast("没有可启动的版本", 3)
		return
	}
	items := make([]pickerItem, 0, len(a.launch.versions))
	for _, v := range a.launch.versions {
		desc := v.Loader + " · " + v.Type
		if v.MCVersion != "" && v.MCVersion != v.ID {
			desc = v.Loader + " · " + v.MCVersion
		}
		if !v.Complete {
			desc += " · 文件不完整"
		}
		items = append(items, pickerItem{Label: v.ID, Desc: desc, Value: v.ID})
	}
	a.modal = modalState{Kind: modalPicker, Title: "选择游戏版本", Items: items, Action: "select-version"}
}

func (a *App) openAccountPicker() {
	cfg := a.store.Snapshot()
	items := make([]pickerItem, 0, len(cfg.Accounts))
	for _, acc := range cfg.Accounts {
		desc := "离线账户"
		if acc.Type == "microsoft" {
			desc = "Microsoft 账户"
		}
		if acc.Name == cfg.SelectedAccount {
			desc += " · 当前"
		}
		items = append(items, pickerItem{Label: acc.Name, Desc: desc, Value: acc.Name})
	}
	a.modal = modalState{Kind: modalPicker, Title: "选择账户", Items: items, Action: "select-account"}
}
