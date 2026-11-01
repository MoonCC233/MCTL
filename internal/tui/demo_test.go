package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestRenderDemoPagesFillTerminal(t *testing.T) {
	for _, size := range []struct {
		w, h int
	}{
		{118, 34},
		{90, 24},
		{160, 44},
	} {
		for _, page := range []string{"launch", "versions", "download", "settings", "tools", "about", "logs"} {
			out := RenderDemo(page, size.w, size.h)
			lines := strings.Split(out, "\n")
			if len(lines) != size.h {
				t.Fatalf("%s at %dx%d: got %d lines", page, size.w, size.h, len(lines))
			}
			for i, line := range lines {
				if got := lipgloss.Width(line); got != size.w {
					t.Fatalf("%s at %dx%d: line %d width %d", page, size.w, size.h, i+1, got)
				}
			}
		}
	}
}

func TestRenderDemoAllPages(t *testing.T) {
	out := RenderDemo("all", 118, 34)
	for _, name := range pageNames {
		if !strings.Contains(out, name) {
			t.Fatalf("demo is missing page %q", name)
		}
	}
}

func TestVersionSubdirMapping(t *testing.T) {
	cases := map[string]string{
		"m": "mods",
		"p": "resourcepacks",
		"s": "saves",
		"h": "screenshots",
		"f": "config",
	}
	for key, want := range cases {
		if got := versionSubdir(key); got != want {
			t.Fatalf("versionSubdir(%q) = %q, want %q", key, got, want)
		}
	}
	if versionSubdir("x") != "" {
		t.Fatal("unknown key should not map to a directory")
	}
}

func TestToolsDemoListsInstanceTools(t *testing.T) {
	out := RenderDemo("tools", 118, 34)
	for _, want := range []string{"服务器列表", "截图管理", "本地资源", "模组信息", "存档管理", "导出实例", "导入实例", "新建实例", "实例设置", "收藏", "CurseForge 整合包", "环境自检", "检查更新", "新闻", "网络测速"} {
		if !strings.Contains(out, want) {
			t.Fatalf("tools demo is missing %q", want)
		}
	}
}

func TestDemoPageContentMap(t *testing.T) {
	pages := map[string][]string{
		"launch":   {"启动", "账户", "版本"},
		"versions": {"已安装", "详情", "实例设置"},
		"download": {"版本列表", "Fabric", "Forge"},
		"settings": {"设置", "Java", "主题"},
		"tools":    {"常用工具", "新建实例", "新闻"},
		"about":    {"关于", "支持", "MCTL"},
		"logs":     {"日志", "跟随", "清空"},
	}
	nav := []string{"LAUNCH", "VERSIONS", "DOWNLOAD", "SETTINGS", "TOOLS", "ABOUT", "LOGS"}
	for page, markers := range pages {
		out := RenderDemo(page, 118, 34)
		for _, m := range markers {
			if !strings.Contains(out, m) {
				t.Fatalf("%s page is missing %q", page, m)
			}
		}
	}
	// A wider terminal shows the full sidebar with English subtitles.
	for page := range pages {
		out := RenderDemo(page, 160, 44)
		for _, label := range nav {
			if !strings.Contains(out, label) {
				t.Fatalf("%s page is missing navigation label %q", page, label)
			}
		}
	}
}

func TestDemoUsesExpectedTheme(t *testing.T) {
	out := RenderDemo("launch", 118, 34)
	if !strings.Contains(out, "MCTL") {
		t.Fatal("demo should show the product name")
	}
}
