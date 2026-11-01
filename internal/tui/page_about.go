package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/MoonCC233/MCTL/internal/config"
)

type aboutPage struct {
	offset int
}

func (a *App) viewAbout(w, h int) []string {
	cfg := a.store.Snapshot()
	confDir, _ := config.Dir()
	lines := pageHeader("关于", "MCTL · Minecraft 终端启动器", w)
	a.addHit(2, len(lines), w-2, h-1, "wheel", "about")
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
	for _, l := range wrapText(tr(desc), max(inner-4, 20)) {
		rows = append(rows, "  "+styleMuted.Render(l))
	}
	rows = append(rows, "", "  "+sectionLabel("支持"))
	features := []string{
		"原版 / Fabric / Quilt / Legacy / LiteLoader / Forge / NeoForge / OptiFine / Cleanroom / LabyMod",
		"本地整合包导入、在线整合包与资源搜索",
		"离线账户与 Microsoft 设备代码登录",
		"Java 检测、Adoptium 自动下载与启动参数",
		"实时日志、命令面板、存档与本地资源管理",
		"截图浏览与多人服务器列表编辑",
		"实例 zip 导出与导入",
		"环境自检、更新检查与下载源连通性测试",
		"模组元数据与内嵌 jar 查看",
		"新建实例向导（名称 / 目录 / 版本 / 加载器 / Java）",
		"在线整合包筛选与收藏列表",
		"三套终端主题与 HTTP 代理设置",
		"中英双语界面切换",
		"实例图标自定义",
		"CurseForge 在线整合包浏览与安装",
		"皮肤 / 披风纹理查询",
		"数据包按存档安装",
		"世界 zip 下载与投影管理",
		"实例加载器安装与文件修复",
		"网络测速与延迟/吞吐展示",
		"启动页新闻与更新日志",
		"鼠标优先的列表、按钮与弹窗交互",
	}
	for _, f := range features {
		rows = append(rows, "  "+styleGreen.Render("•")+" "+styleText.Render(tr(f)))
	}
	rows = append(rows, "", "  "+sectionLabel("路径"))
	rows = append(rows, "  "+styleMuted.Render(tr("配置")+"  ")+styleFaint.Render(middleEllipsis(confDir, max(inner-12, 12))))
	rows = append(rows, "  "+styleMuted.Render(tr("游戏")+"  ")+styleFaint.Render(middleEllipsis(cfg.GameDir, max(inner-12, 12))))
	rows = append(rows, "", "  "+sectionLabel("项目"))
	rows = append(rows, "  "+styleMuted.Render(tr("仓库")+"  ")+styleBlue.Render("github.com/MoonCC233/MCTL"))
	rows = append(rows, "  "+styleMuted.Render(tr("许可")+"  ")+styleText.Render("MIT"))
	rows = append(rows, "", "  "+keyHint("o", "打开游戏目录")+"  "+keyHint("g", "打开配置目录")+"  "+keyHint("esc", "返回启动页"))
	viewH := panelContentHeight(contentH)
	total := len(rows)
	if viewH > 0 {
		a.about.offset = clamp(a.about.offset, 0, max(total-viewH, 0))
		rows = rows[a.about.offset:]
		for len(rows) < viewH {
			rows = append(rows, "")
		}
		if len(rows) > viewH {
			rows = rows[:viewH]
		}
	}
	title := tr("MCTL")
	if total > viewH && viewH > 0 {
		title += fmt.Sprintf(" · %d-%d/%d", a.about.offset+1, min(a.about.offset+viewH, total), total)
	}
	p := panel(title, w-4, contentH, a.page == pageAbout, rows)
	for _, l := range strings.Split(p, "\n") {
		lines = append(lines, "  "+l)
	}
	return fitLines(lines, h, w)
}

func (a *App) aboutKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "up", "k":
		if a.about.offset > 0 {
			a.about.offset--
		}
	case "down", "j":
		a.about.offset++
	case "pgup":
		a.about.offset = max(a.about.offset-8, 0)
	case "pgdown":
		a.about.offset += 8
	case "home":
		a.about.offset = 0
	case "end":
		a.about.offset = 1 << 20
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
