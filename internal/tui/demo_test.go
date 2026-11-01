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
	for _, want := range []string{"服务器列表", "截图管理", "本地资源", "存档管理", "导出实例", "导入实例", "实例设置", "环境自检", "检查更新"} {
		if !strings.Contains(out, want) {
			t.Fatalf("tools demo is missing %q", want)
		}
	}
}

func TestDemoUsesExpectedTheme(t *testing.T) {
	out := RenderDemo("launch", 118, 34)
	if !strings.Contains(out, "MCTL") {
		t.Fatal("demo should show the product name")
	}
}
