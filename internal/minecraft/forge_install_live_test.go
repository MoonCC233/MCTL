//go:build live

package minecraft

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestLiveForgeInstall exercises the real Forge installer against a temporary
// game directory. It downloads the installer and the vanilla profile/client,
// then runs "java -jar forge-installer.jar --installClient" and checks the
// generated launch profile. Assets are intentionally skipped.
func TestLiveForgeInstall(t *testing.T) {
	if strings.TrimSpace(os.Getenv("MCTL_REAL_INSTALL")) == "" {
		t.Skip("set MCTL_REAL_INSTALL=1 to run a real loader installation")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()

	dir := t.TempDir()
	inst := NewInstaller(dir, GetSource("bmclapi"), func(s string) { t.Log(s) }, nil)
	inst.HTTP.Client.Timeout = 3 * time.Minute
	inst.HTTP.Retries = 2

	java, err := FindJavaFor(17, "")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Java: %s", java.String())

	m, err := inst.manifestFor(ctx)
	if err != nil {
		t.Fatal(err)
	}
	entry, ok := m.Find("1.20.1")
	if !ok {
		t.Fatal("1.20.1 missing from the manifest")
	}
	data, err := inst.HTTP.Get(ctx, entry.URL)
	if err != nil {
		t.Fatal(err)
	}
	var profile VersionJSON
	if err := json.Unmarshal(data, &profile); err != nil {
		t.Fatal(err)
	}
	if err := writeVersionJSON(dir, "1.20.1", data); err != nil {
		t.Fatal(err)
	}
	client, ok := profile.Downloads["client"]
	if !ok || client.URL == "" {
		t.Fatalf("vanilla profile has no client download: %#v", profile.Downloads)
	}
	jar := filepath.Join(VersionPath(dir, "1.20.1"), "1.20.1.jar")
	if err := inst.HTTP.DownloadFile(ctx, client.URL, jar, client.SHA1, nil); err != nil {
		t.Fatal(err)
	}
	t.Log("vanilla profile and client jar prepared")

	build := LoaderBuild{Kind: "forge", Version: "47.4.28", MCVersion: "1.20.1"}
	installerDir := filepath.Join(dir, "mctl-installers")
	if err := os.MkdirAll(installerDir, 0o755); err != nil {
		t.Fatal(err)
	}
	installer := filepath.Join(installerDir, "forge-1.20.1-47.4.28-installer.jar")
	if err := inst.HTTP.DownloadFile(ctx, build.InstallerURL(), installer, "", nil); err != nil {
		t.Fatal(err)
	}
	t.Log("forge installer downloaded")

	if err := inst.runInstaller(ctx, java.Path, installer, "forge", "47.4.28"); err != nil {
		t.Fatal(err)
	}
	t.Log("forge installer finished")

	id, err := FindLoaderVersion(dir, "forge", "1.20.1", "47.4.28")
	if err != nil {
		t.Fatal(err)
	}
	rv, err := ResolveVersion(dir, id)
	if err != nil {
		t.Fatal(err)
	}
	if rv.MainClass == "" || len(rv.Libraries) == 0 {
		t.Fatalf("resolved version = %#v", rv)
	}
	t.Logf("generated profile: %s", id)
	t.Logf("main class: %s", rv.MainClass)
	t.Logf("chain: %v, libraries: %d", rv.Chain, len(rv.Libraries))
}
