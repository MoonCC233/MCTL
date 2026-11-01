package tui

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/MoonCC233/MCTL/internal/config"
	"github.com/MoonCC233/MCTL/internal/minecraft"
)

func TestDiagnoseModalRendersChecks(t *testing.T) {
	a := testApp(t, config.Config{GameDir: t.TempDir()})
	a.handleEvent(diagnoseDoneMsg{Checks: []minecraft.DiagnoseCheck{
		{Name: "Java 运行时", Status: minecraft.DiagnoseOK, Detail: "Java 21 · amd64"},
		{Name: "下载源", Status: minecraft.DiagnoseWarn, Detail: "无法连接"},
		{Name: "游戏版本", Status: minecraft.DiagnoseFail, Detail: "尚未选择版本"},
	}})
	if a.modal.Kind != modalDiagnostics {
		t.Fatalf("modal = %#v", a.modal)
	}
	if len(a.modal.Items) != 3 {
		t.Fatalf("items = %#v", a.modal.Items)
	}
	out := a.View()
	for _, want := range []string{"环境自检", "Java 21", "无法连接", "尚未选择版本"} {
		if !strings.Contains(out, want) {
			t.Fatalf("modal output missing %q", want)
		}
	}
}

func TestUpdateCheckModal(t *testing.T) {
	a := testApp(t, config.Config{GameDir: t.TempDir()})
	a.handleEvent(updateCheckMsg{
		Current: "1.0.16",
		Info: &minecraft.ReleaseInfo{
			TagName: "v1.0.17",
			HTMLURL: "https://example.invalid/releases/v1.0.17",
		},
	})
	if a.modal.Kind != modalConfirm || a.modal.Action != "open-release" {
		t.Fatalf("modal = %#v", a.modal)
	}
	if a.modal.Data != "https://example.invalid/releases/v1.0.17" {
		t.Fatalf("data = %#v", a.modal.Data)
	}

	b := testApp(t, config.Config{GameDir: t.TempDir()})
	b.handleEvent(updateCheckMsg{Current: "1.0.16", Info: &minecraft.ReleaseInfo{TagName: "v1.0.16"}})
	if b.modal.Kind != modalNone {
		t.Fatalf("up-to-date should not open a modal: %#v", b.modal)
	}
	if !strings.Contains(b.toast.Text, "已是最新版本") {
		t.Fatalf("toast = %q", b.toast.Text)
	}
}

func TestSpeedTestModal(t *testing.T) {
	a := testApp(t, config.Config{GameDir: t.TempDir()})
	a.handleEvent(speedTestMsg{Results: []minecraft.SpeedResult{
		{Target: minecraft.SpeedTarget{Name: "BMCLAPI 镜像"}, Latency: 120 * time.Millisecond, Bytes: 262144, MBps: 2.5},
		{Target: minecraft.SpeedTarget{Name: "Modrinth API"}, Err: errors.New("HTTP 502")},
	}})
	if a.modal.Kind != modalSpeed {
		t.Fatalf("modal = %#v", a.modal)
	}
	out := a.View()
	for _, want := range []string{"网络测速", "BMCLAPI", "2.50 MB/s", "Modrinth", "失败"} {
		if !strings.Contains(out, want) {
			t.Fatalf("speed modal missing %q", want)
		}
	}
}

func TestNewsModal(t *testing.T) {
	a := testApp(t, config.Config{GameDir: t.TempDir()})
	a.handleEvent(newsMsg{Releases: []minecraft.ReleaseInfo{
		{TagName: "v1.0.21", Name: "MCTL v1.0.21", HTMLURL: "https://example.invalid/rel", PublishedAt: "2026-11-01T09:00:00Z"},
	}})
	if a.modal.Kind != modalPicker || a.modal.Action != "open-news" {
		t.Fatalf("modal = %#v", a.modal)
	}
	if len(a.modal.Items) != 1 || a.modal.Items[0].Value != "https://example.invalid/rel" {
		t.Fatalf("items = %#v", a.modal.Items)
	}
	out := a.View()
	for _, want := range []string{"新闻", "MCTL v1.0.21", "2026-11-01"} {
		if !strings.Contains(out, want) {
			t.Fatalf("news modal missing %q", want)
		}
	}
}

func TestUpdateCheckError(t *testing.T) {
	a := testApp(t, config.Config{GameDir: t.TempDir()})
	a.handleEvent(updateCheckMsg{Current: "1.0.16", Err: errors.New("更新服务不可用")})
	if a.modal.Kind != modalError {
		t.Fatalf("modal = %#v", a.modal)
	}
	if !strings.Contains(a.modal.Body, "更新服务不可用") {
		t.Fatalf("body = %q", a.modal.Body)
	}
}
