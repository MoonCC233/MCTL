package tui

import (
	"strings"
	"testing"
)

// uiChromeChinese lists interface strings that must be translated in English
// mode. Game output, version IDs and paths are excluded on purpose.
var uiChromeChinese = []string{
	"当前实例", "运行环境", "最近日志", "已安装", "版本列表", "偏好设置", "常用工具",
	"启动游戏", "实例安装", "版本路径", "实例路径", "发布时间", "未安装", "正式", "快照",
	"本地资源", "服务器列表", "截图管理", "新建实例", "环境自检", "检查更新", "网络测速",
	"就绪", "切换", "退出", "调整", "翻页", "修改会立即保存", "自动 ·", "（无）",
	"打开模组目录", "打开版本目录", "下载源", "账户", "帮助", "关于",
}

func englishChromeLine(line string) bool {
	for _, marker := range []string{"[MCTL]", "[main/", "fabric-api", "Loading 48", "已检测到"} {
		if strings.Contains(line, marker) {
			return false
		}
	}
	return true
}

func TestEnglishChromeIsTranslated(t *testing.T) {
	t.Cleanup(func() { SetLanguage("zh") })
	SetLanguage("en")
	for _, page := range []string{"launch", "versions", "download", "settings", "tools", "about", "logs"} {
		out := RenderDemo(page, 160, 40)
		lines := strings.Split(out, "\n")
		if page == "logs" {
			// Log rows are game output and stay in their original language.
			lines = lines[:8]
		}
		for i, line := range lines {
			if !englishChromeLine(line) {
				continue
			}
			for _, token := range uiChromeChinese {
				if strings.Contains(line, token) {
					t.Fatalf("%s:%d still shows Chinese UI text %q: %s", page, i+1, token, strings.TrimSpace(line))
				}
			}
		}
	}
}

func TestLayoutRegions(t *testing.T) {
	regions := map[string][]string{
		"launch":   {"当前实例", "运行环境", "最近日志"},
		"versions": {"已安装 ·", "详情"},
		"download": {"版本列表 ·", "详情"},
		"settings": {"偏好设置 ·"},
		"tools":    {"常用工具 ·"},
		"about":    {"MCTL ·"},
		"logs":     {"日志"},
	}
	for page, want := range regions {
		out := RenderDemo(page, 118, 34)
		for _, token := range want {
			if !strings.Contains(out, token) {
				t.Fatalf("%s page is missing layout region %q", page, token)
			}
		}
	}
}
