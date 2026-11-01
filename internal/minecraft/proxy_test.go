package minecraft

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureLauncherProfilesCreatesFile(t *testing.T) {
	dir := t.TempDir()
	if err := ensureLauncherProfiles(dir); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "launcher_profiles.json"))
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if doc["profiles"] == nil {
		t.Fatalf("profiles missing: %s", data)
	}
}

func TestEnsureLauncherProfilesKeepsExisting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "launcher_profiles.json")
	if err := os.WriteFile(path, []byte(`{"profiles":{"keep":{}}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := ensureLauncherProfiles(dir); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != `{"profiles":{"keep":{}}}` {
		t.Fatalf("existing file rewritten: %s", data)
	}
}

func TestInstallerProxyFromConfig(t *testing.T) {
	if err := SetProxy("http://127.0.0.1:7897"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = SetProxy("") })
	host, port, ok := installerProxy()
	if !ok {
		t.Fatal("configured proxy not detected")
	}
	if host != "127.0.0.1" || port != "7897" {
		t.Fatalf("proxy = %s:%s", host, port)
	}
}
