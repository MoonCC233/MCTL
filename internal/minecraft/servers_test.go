package minecraft

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestServersRoundTrip(t *testing.T) {
	dir := t.TempDir()

	list, err := ListServers(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 0 {
		t.Fatalf("missing file should yield no servers: %#v", list)
	}

	if err := AddServer(dir, Server{Name: "SMP", IP: "play.example.com", AcceptTextures: true}); err != nil {
		t.Fatal(err)
	}
	if err := AddServer(dir, Server{Name: "  Creative  ", IP: "creative.example.com:25565"}); err != nil {
		t.Fatal(err)
	}

	raw, err := os.ReadFile(ServersPath(dir))
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) < 2 || raw[0] != 0x1f || raw[1] != 0x8b {
		t.Fatalf("servers.dat should be gzip compressed, got % x", raw[:min(len(raw), 8)])
	}

	list, err = ListServers(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Fatalf("servers = %#v", list)
	}
	if list[0].Name != "SMP" || list[0].IP != "play.example.com" || !list[0].AcceptTextures {
		t.Fatalf("first server = %#v", list[0])
	}
	if list[1].Name != "Creative" || list[1].IP != "creative.example.com:25565" || list[1].AcceptTextures {
		t.Fatalf("second server = %#v", list[1])
	}

	if err := UpdateServer(dir, 1, Server{Name: "Creative 2", IP: "new.example.com", Hidden: true}); err != nil {
		t.Fatal(err)
	}
	list, err = ListServers(dir)
	if err != nil {
		t.Fatal(err)
	}
	if list[1].Name != "Creative 2" || list[1].IP != "new.example.com" || !list[1].Hidden {
		t.Fatalf("updated server = %#v", list[1])
	}
	if !list[0].AcceptTextures {
		t.Fatalf("unrelated entry should keep its flag: %#v", list[0])
	}

	if err := DeleteServer(dir, 0); err != nil {
		t.Fatal(err)
	}
	list, err = ListServers(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].Name != "Creative 2" {
		t.Fatalf("servers after delete = %#v", list)
	}
}

func TestServersValidation(t *testing.T) {
	dir := t.TempDir()
	if err := AddServer(dir, Server{Name: "No Address"}); err == nil {
		t.Fatal("expected empty address error")
	}
	if err := UpdateServer(dir, 0, Server{Name: "Missing", IP: "example.com"}); err == nil {
		t.Fatal("expected invalid index error")
	}
	if err := DeleteServer(dir, 0); err == nil {
		t.Fatal("expected invalid index error")
	}
	if err := AddServer(dir, Server{Name: "   ", IP: "example.com"}); err != nil {
		t.Fatal(err)
	}
	list, err := ListServers(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].Name != "Minecraft Server" {
		t.Fatalf("fallback name = %#v", list)
	}
}

func TestDecodeServersAcceptsUncompressedNBT(t *testing.T) {
	root := map[string]any{
		"servers": nbtListValue{
			ElementType: nbtCompound,
			Values: []any{
				map[string]any{"name": "Local", "ip": "127.0.0.1:25565", "acceptTextures": int8(0)},
			},
		},
	}
	var buf bytes.Buffer
	if err := writeNBT(&buf, "", root); err != nil {
		t.Fatal(err)
	}
	list, err := decodeServers(buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].Name != "Local" || list[0].IP != "127.0.0.1:25565" {
		t.Fatalf("servers = %#v", list)
	}
}

func TestNBTGenericRoundTrip(t *testing.T) {
	root := map[string]any{
		"byte":   int8(-4),
		"short":  int16(300),
		"int":    int32(70000),
		"long":   int64(1 << 40),
		"float":  float32(1.5),
		"double": 2.25,
		"string": "测试",
		"bytes":  []byte{1, 2, 3},
		"ints":   []int32{4, 5},
		"longs":  []int64{6, 7},
		"nested": map[string]any{
			"flag": int8(1),
			"list": nbtListValue{ElementType: nbtString, Values: []any{"a", "b"}},
		},
	}
	var buf bytes.Buffer
	if err := writeNBT(&buf, "root", root); err != nil {
		t.Fatal(err)
	}
	name, got, err := readNBT(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	if name != "root" {
		t.Fatalf("root name = %q", name)
	}
	if got["byte"].(int8) != -4 || got["string"].(string) != "测试" {
		t.Fatalf("roundtrip = %#v", got)
	}
	nested := got["nested"].(map[string]any)
	list := nested["list"].(nbtListValue)
	if list.ElementType != nbtString || len(list.Values) != 2 || list.Values[1] != "b" {
		t.Fatalf("nested list = %#v", list)
	}
}

func TestServersPathIsPerInstance(t *testing.T) {
	root := t.TempDir()
	instance := filepath.Join(root, "instances", "pack")
	if ServersPath(instance) == ServersPath(root) {
		t.Fatal("instances must store their own servers.dat")
	}
	if filepath.Base(ServersPath(instance)) != "servers.dat" {
		t.Fatalf("path = %q", ServersPath(instance))
	}
}
