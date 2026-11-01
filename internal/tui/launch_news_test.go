package tui

import (
	"errors"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/MoonCC233/MCTL/internal/config"
	"github.com/MoonCC233/MCTL/internal/minecraft"
)

func newsTestApp(t *testing.T) *App {
	t.Helper()
	a := testApp(t, config.Config{GameDir: t.TempDir(), SelectedVersion: "1.20.1"})
	a.launch.versions = []minecraft.InstalledVersion{
		{ID: "1.20.1", MCVersion: "1.20.1", Loader: "Vanilla", Type: "release", Complete: true},
	}
	a.launch.news = []minecraft.ReleaseInfo{
		{TagName: "v1.0.32", Name: "MCTL v1.0.32", PublishedAt: "2026-11-01T09:00:00Z"},
		{TagName: "v1.0.31", Name: "MCTL v1.0.31", PublishedAt: "2026-10-30T09:00:00Z"},
		{TagName: "v1.0.30", Name: "MCTL v1.0.30", PublishedAt: "2026-10-28T09:00:00Z"},
	}
	a.launch.newsAt = time.Now()
	a.page = pageLaunch
	return a
}

func TestLaunchNewsPanelRenders(t *testing.T) {
	a := newsTestApp(t)
	out := a.View()
	for _, want := range []string{"新闻 / 公告", "MCTL v1.0.32", "MCTL v1.0.31", "11-01"} {
		if !strings.Contains(out, want) {
			t.Fatalf("launch news panel missing %q", want)
		}
	}
	if lines := strings.Count(out, "\n") + 1; lines != 30 {
		t.Fatalf("launch page returned %d lines", lines)
	}
}

func TestLaunchNewsMessageUpdatesState(t *testing.T) {
	a := testApp(t, config.Config{GameDir: t.TempDir()})
	a.launch.newsLoad = true
	a.handleEvent(launchNewsMsg{Releases: []minecraft.ReleaseInfo{
		{TagName: "v1.0.32", Name: "MCTL v1.0.32", PublishedAt: "2026-11-01T09:00:00Z"},
		{TagName: "v1.0.31", Name: "MCTL v1.0.31", PublishedAt: "2026-10-30T09:00:00Z"},
	}})
	if a.modal.Kind != modalNone {
		t.Fatalf("background news should not open a modal: %#v", a.modal)
	}
	if a.launch.newsLoad {
		t.Fatal("newsLoad should be cleared")
	}
	if len(a.launch.news) != 2 || a.launch.newsAt.IsZero() {
		t.Fatalf("news state = %#v", a.launch)
	}
	if !strings.Contains(a.View(), "新闻 / 公告") {
		t.Fatal("launch page should render the news panel")
	}
}

func TestLaunchNewsFailure(t *testing.T) {
	a := testApp(t, config.Config{GameDir: t.TempDir()})
	a.launch.newsLoad = true
	a.handleEvent(launchNewsMsg{Err: errors.New("offline")})
	out := a.View()
	if !strings.Contains(out, "新闻获取失败") {
		t.Fatalf("launch page should show the news error")
	}
}

func TestLaunchNewsKeyOpensChangelog(t *testing.T) {
	a := newsTestApp(t)
	a.launchKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	if !strings.Contains(a.toast.Text, "更新日志") {
		t.Fatalf("toast = %q", a.toast.Text)
	}
}
