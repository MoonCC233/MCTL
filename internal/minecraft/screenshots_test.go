package minecraft

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScreenshotListAndDelete(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "screenshots", "2026-01-01_12.00.00.png"), "png")
	mustWrite(t, filepath.Join(dir, "screenshots", "notes.txt"), "txt")
	list, err := ListScreenshots(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].Name != "2026-01-01_12.00.00.png" {
		t.Fatalf("list = %#v", list)
	}
	if err := DeleteScreenshot(dir, list[0].Name); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(list[0].Path); !os.IsNotExist(err) {
		t.Fatalf("screenshot should be deleted: %v", err)
	}
	if err := DeleteScreenshot(dir, `..\outside.png`); err == nil {
		t.Fatal("expected traversal rejection")
	}
}
