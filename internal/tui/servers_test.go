package tui

import (
	"strings"
	"testing"

	"github.com/MoonCC233/MCTL/internal/config"
	"github.com/MoonCC233/MCTL/internal/logbuf"
	"github.com/MoonCC233/MCTL/internal/minecraft"
)

func TestServerModalListsServers(t *testing.T) {
	dir := t.TempDir()
	if err := minecraft.AddServer(dir, minecraft.Server{Name: "SMP", IP: "play.example.com"}); err != nil {
		t.Fatal(err)
	}
	store := &config.Store{Cfg: config.Config{GameDir: dir}, Path: "test"}
	rt := NewRuntime(store, logbuf.New(50))
	a := New(store, rt)
	a.width, a.height = 100, 30

	a.openServers()
	if a.modal.Kind != modalServers {
		t.Fatalf("modal kind = %v", a.modal.Kind)
	}
	if len(a.modal.Items) != 1 || a.modal.Items[0].Label != "SMP" {
		t.Fatalf("items = %#v", a.modal.Items)
	}
	out := a.View()
	for _, want := range []string{"服务器列表", "SMP", "play.example.com"} {
		if !strings.Contains(out, want) {
			t.Fatalf("modal output missing %q", want)
		}
	}
}

func TestServerModalActionsAddAndEdit(t *testing.T) {
	dir := t.TempDir()
	store := &config.Store{Cfg: config.Config{GameDir: dir}, Path: "test"}
	rt := NewRuntime(store, logbuf.New(50))
	a := New(store, rt)
	a.width, a.height = 100, 30

	a.runAction("add-server-name", "My Server", pickerItem{}, nil)
	if a.modal.Kind != modalInput || a.modal.Data != "My Server" {
		t.Fatalf("name step = %#v", a.modal)
	}
	a.runAction("add-server-address", "example.com:25565", pickerItem{}, a.modal.Data)

	list, err := minecraft.ListServers(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].Name != "My Server" || list[0].IP != "example.com:25565" {
		t.Fatalf("servers = %#v", list)
	}
	if a.modal.Kind != modalServers {
		t.Fatalf("expected server list after add, got %v", a.modal.Kind)
	}

	edit := serverEditData{GameDir: dir, Index: 0, Name: "My Server", Server: list[0]}
	a.runAction("edit-server", "new.example.com", pickerItem{}, edit)
	list, err = minecraft.ListServers(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].IP != "new.example.com" || !list[0].AcceptTextures {
		t.Fatalf("updated servers = %#v", list)
	}

	a.runAction("delete-server", "", pickerItem{}, serverEditData{GameDir: dir, Index: 0, Name: "My Server"})
	list, err = minecraft.ListServers(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 0 {
		t.Fatalf("servers after delete = %#v", list)
	}
}
