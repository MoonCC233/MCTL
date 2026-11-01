package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/MoonCC233/MCTL/internal/config"
)

func (a *App) viewAbout(w, h int) []string {
	cfg := a.store.Snapshot()
	confDir, _ := config.Dir()
	lines := pageHeader("关于", "MCTL · Minecraft 终端启动器", w)
	contentH := h - len(lines)
	if contentH < 5 {
		return fitLines(lines, h, w)
	}
	inner := panelContentWidth(w - 4)
	desc := "用键盘完成 Minecraft 的安装、版本管理、账户登录与启动。界面使用 Bubble Tea 与 Lip Gloss 构建，支持 Windows、Linux 与 macOS。"
	var rows []string
	rows = append(rows, "")
	rows = append(rows, "  "+styleAccent.Render("◆")+" "+styleBold.Render("MCTL")+" "+styleFaint.Render("v"+Version))
	rows = append(rows, "")
	for _, l := range wrapText(desc, max(inner-4, 20)) {
		rows = append(rows, "  "+styleMuted.Render(l))
	}
	rows = append(rows, "", "  "+sectionLabel("支持"))
	features := []string{
		"原版 / Fabric / Quilt / Forge / NeoForge 安装",
		"Modrinth 与 CurseForge 整合包导入",
		"离线账户与 Microsoft 设备代码登录",
		"Java 检测、Adoptium 自动下载与启动参数",
		"实时日志、命令面板与演示模式",
	}
	for _, f := range features {
		rows = append(rows, "  "+styleGreen.Render("•")+" "+styleText.Render(f))
	}
	rows = append(rows, "", "  "+sectionLabel("路径"))
	rows = append(rows, "  "+styleMuted.Render("配置  ")+styleFaint.Render(middleEllipsis(confDir, max(inner-12, 12))))
	rows = append(rows, "  "+styleMuted.Render("游戏  ")+styleFaint.Render(middleEllipsis(cfg.GameDir, max(inner-12, 12))))
	rows = append(rows, "", "  "+sectionLabel("项目"))
	rows = append(rows, "  "+styleMuted.Render("仓库  ")+styleBlue.Render("github.com/MoonCC233/MCTL"))
	rows = append(rows, "  "+styleMuted.Render("许可  ")+styleText.Render("MIT"))
	rows = append(rows, "", "  "+keyHint("o", "打开游戏目录")+"  "+keyHint("g", "打开配置目录")+"  "+keyHint("esc", "返回启动页"))
	for len(rows) < panelContentHeight(contentH) {
		rows = append(rows, "")
	}
	if len(rows) > panelContentHeight(contentH) {
		rows = rows[:panelContentHeight(contentH)]
	}
	p := panel("MCTL", w-4, contentH, a.page == pageAbout, rows)
	for _, l := range strings.Split(p, "\n") {
		lines = append(lines, "  "+l)
	}
	return fitLines(lines, h, w)
}

func (a *App) aboutKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "o":
		a.openGameDir()
	case "g":
		if dir, err := config.Dir(); err == nil {
			if err := openPath(dir); err != nil {
				a.setToast("打开配置目录失败: "+err.Error(), 2)
			}
		}
	}
	return nil
}
