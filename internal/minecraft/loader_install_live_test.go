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

// livePrepareVanilla writes a vanilla profile and client jar so official
// loader installers can find the base game.
func livePrepareVanilla(t *testing.T, inst *Installer, dir, mcVersion string) {
	t.Helper()
	m, err := inst.manifestFor(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	entry, ok := m.Find(mcVersion)
	if !ok {
		t.Fatalf("%s missing from the manifest", mcVersion)
	}
	data, err := inst.HTTP.Get(context.Background(), entry.URL)
	if err != nil {
		t.Fatal(err)
	}
	var profile VersionJSON
	if err := json.Unmarshal(data, &profile); err != nil {
		t.Fatal(err)
	}
	if err := writeVersionJSON(dir, mcVersion, data); err != nil {
		t.Fatal(err)
	}
	client, ok := profile.Downloads["client"]
	if !ok || client.URL == "" {
		t.Fatalf("vanilla profile has no client download: %#v", profile.Downloads)
	}
	jar := filepath.Join(VersionPath(dir, mcVersion), mcVersion+".jar")
	if err := inst.HTTP.DownloadFile(context.Background(), client.URL, jar, client.SHA1, nil); err != nil {
		t.Fatal(err)
	}
}

// liveInstallLoader downloads and runs a real loader installer, then checks the
// generated launch profile.
func liveInstallLoader(t *testing.T, kind, mcVersion, loaderVersion string, javaMajor int) {
	t.Helper()
	if strings.TrimSpace(os.Getenv("MCTL_REAL_INSTALL")) == "" {
		t.Skip("set MCTL_REAL_INSTALL=1 to run a real loader installation")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Minute)
	defer cancel()

	dir := t.TempDir()
	inst := NewInstaller(dir, GetSource("bmclapi"), func(s string) { t.Log(s) }, nil)
	inst.HTTP.Client.Timeout = 3 * time.Minute
	inst.HTTP.Retries = 2

	java, err := FindJavaFor(javaMajor, "")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Java: %s (%s)", java.String(), java.Path)

	livePrepareVanilla(t, inst, dir, mcVersion)
	t.Log("vanilla profile and client jar prepared")

	build := LoaderBuild{Kind: kind, Version: loaderVersion, MCVersion: mcVersion}
	installerDir := filepath.Join(dir, "mctl-installers")
	if err := os.MkdirAll(installerDir, 0o755); err != nil {
		t.Fatal(err)
	}
	installer := filepath.Join(installerDir, filepath.Base(build.InstallerURL()))
	if err := inst.HTTP.DownloadFile(ctx, build.InstallerURL(), installer, "", nil); err != nil {
		t.Fatal(err)
	}
	t.Logf("%s installer downloaded: %s", kind, filepath.Base(installer))

	if err := inst.runInstaller(ctx, java.Path, installer, kind, loaderVersion); err != nil {
		t.Fatal(err)
	}
	t.Logf("%s installer finished", kind)

	id, err := FindLoaderVersion(dir, kind, mcVersion, loaderVersion)
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

// TestLiveNeoForgeInstall runs the real NeoForge installer.
func TestLiveNeoForgeInstall(t *testing.T) {
	liveInstallLoader(t, "neoforge", "1.21.1", "21.1.265", 21)
}

// TestLiveCleanroomInstall runs the real Cleanroom installer, which requires
// Java 25.
func TestLiveCleanroomInstall(t *testing.T) {
	liveInstallLoader(t, "cleanroom", "1.12.2", "0.7.14-alpha", 25)
}
