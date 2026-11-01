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

// TestLiveLabyModProfile builds a real LabyMod profile from the official
// manifest and downloads its main artifact. A full game launch needs a desktop
// session, so this stops at a resolvable, downloaded profile.
func TestLiveLabyModProfile(t *testing.T) {
	if strings.TrimSpace(os.Getenv("MCTL_REAL_INSTALL")) == "" {
		t.Skip("set MCTL_REAL_INSTALL=1 to run a real loader installation")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	dir := t.TempDir()
	inst := NewInstaller(dir, GetSource("bmclapi"), func(s string) { t.Log(s) }, nil)
	inst.HTTP.Client.Timeout = 2 * time.Minute
	inst.HTTP.Retries = 2

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
	if err := writeVersionJSON(dir, "1.20.1", data); err != nil {
		t.Fatal(err)
	}

	id, err := inst.InstallLabyModProfile(ctx, "1.20.1", "production")
	if err != nil {
		skipIfUnreachable(t, err)
	}
	rv, err := ResolveVersion(dir, id)
	if err != nil {
		t.Fatal(err)
	}
	if rv.MainClass == "" || len(rv.Libraries) == 0 {
		t.Fatalf("resolved version = %#v", rv)
	}
	var laby *Library
	for i := range rv.Libraries {
		if strings.Contains(strings.ToLower(rv.Libraries[i].Name), "labymod") {
			laby = &rv.Libraries[i]
			break
		}
	}
	if laby == nil || laby.Downloads == nil || laby.Downloads.Artifact == nil {
		t.Fatal("LabyMod artifact missing from the profile")
	}
	art := laby.Downloads.Artifact
	dest := filepath.Join(dir, "libraries", filepath.FromSlash(art.Path))
	if err := inst.HTTP.DownloadFile(ctx, art.URL, dest, art.SHA1, nil); err != nil {
		t.Fatal(err)
	}
	st, err := os.Stat(dest)
	if err != nil {
		t.Fatal(err)
	}
	if st.Size() == 0 {
		t.Fatal("downloaded LabyMod artifact is empty")
	}
	t.Logf("generated profile: %s", id)
	t.Logf("main class: %s", rv.MainClass)
	t.Logf("chain: %v, libraries: %d", rv.Chain, len(rv.Libraries))
	t.Logf("LabyMod artifact: %s (%d bytes)", art.Path, st.Size())
}
