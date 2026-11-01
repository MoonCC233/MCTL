//go:build live

package minecraft

import (
	"archive/zip"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestLiveCurseForgeDownload downloads a real CurseForge project file and
// verifies the archive is a readable modpack. It is opt-in because it writes
// network traffic and a temporary archive.
//
//	MCTL_REAL_DOWNLOAD=1 CURSEFORGE_API_KEY=... go test -tags live -run TestLiveCurseForgeDownload -v ./internal/minecraft
func TestLiveCurseForgeDownload(t *testing.T) {
	if strings.TrimSpace(os.Getenv("MCTL_REAL_DOWNLOAD")) == "" {
		t.Skip("set MCTL_REAL_DOWNLOAD=1 to download a real project file")
	}
	key := strings.TrimSpace(os.Getenv("CURSEFORGE_API_KEY"))
	if key == "" {
		t.Skip("CURSEFORGE_API_KEY not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	hc := NewHTTPClient(GetSource("bmclapi"))
	hc.Client.Timeout = 2 * time.Minute
	hc.Retries = 2

	mod, err := GetCurseForgeMod(ctx, hc, key, 285109)
	if err != nil {
		t.Fatal(err)
	}
	files, err := CurseForgeModFiles(ctx, hc, key, mod.ID, "")
	if err != nil {
		t.Fatal(err)
	}
	var pick CurseForgeFile
	for _, f := range files {
		if f.DownloadURL == "" {
			continue
		}
		if pick.ID == 0 || f.FileLength < pick.FileLength {
			pick = f
		}
	}
	if pick.ID == 0 {
		t.Fatal("no downloadable file")
	}
	dir := t.TempDir()
	path, err := DownloadCurseForgeFile(ctx, hc, key, mod, pick, dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	st, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if st.Size() == 0 {
		t.Fatal("downloaded archive is empty")
	}
	zr, err := zip.OpenReader(path)
	if err != nil {
		t.Fatalf("archive is not a zip: %v", err)
	}
	defer zr.Close()
	hasManifest := false
	for _, f := range zr.File {
		if filepath.Base(f.Name) == "manifest.json" {
			hasManifest = true
			break
		}
	}
	if !hasManifest {
		t.Fatalf("archive %s has no manifest.json", path)
	}
	t.Logf("downloaded %s (%d bytes, %d entries) from %s",
		filepath.Base(path), st.Size(), len(zr.File), mod.Name)
}
