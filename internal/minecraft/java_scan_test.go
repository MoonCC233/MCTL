package minecraft

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestListJavaSortedAndUnique verifies the scan result is ordered newest first
// and contains no duplicate executables.
func TestListJavaSortedAndUnique(t *testing.T) {
	list := ListJava("")
	if len(list) == 0 {
		t.Skip("no Java runtime installed on this machine")
	}
	seen := map[string]bool{}
	for i, j := range list {
		t.Logf("java[%d] = %d %s %s", i, j.Major, j.Version, j.Path)
		if j.Major <= 0 {
			t.Fatalf("java[%d] has no major version: %#v", i, j)
		}
		if seen[j.Path] {
			t.Fatalf("duplicate runtime %s", j.Path)
		}
		seen[j.Path] = true
		if i > 0 && list[i-1].Major < j.Major {
			t.Fatalf("list not sorted: %d before %d", list[i-1].Major, j.Major)
		}
	}
}

// TestJavaCandidatesIncludesToolHome checks that a custom toolchain home is
// picked up from the environment.
func TestJavaCandidatesIncludesToolHome(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Setenv("JAVA_HOME", "")
	}
	home := t.TempDir()
	bin := filepath.Join(home, "bin")
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	exe := filepath.Join(bin, javaBinary())
	if err := os.WriteFile(exe, []byte{}, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("JDK_HOME", home)
	found := false
	for _, p := range JavaCandidates("") {
		if p == exe {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("JDK_HOME candidate %s missing from %v", exe, JavaCandidates(""))
	}
}

// TestWindowsJavaPathsCoverExtraDrives documents the multi-drive scan: the
// glob table must stay non-empty and include the standard JDK locations.
func TestWindowsJavaPathsCoverExtraDrives(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("windows only")
	}
	if len(windowsJavaGlobs) == 0 {
		t.Fatal("no windows java globs")
	}
	joined := ""
	for _, g := range windowsJavaGlobs {
		joined += g + "\n"
	}
	if !strings.Contains(joined, `Program Files\Java`) {
		t.Fatalf("missing Program Files\\Java pattern:\n%s", joined)
	}
	for _, root := range windowsDriveRoots() {
		t.Logf("drive root scanned: %s", root)
	}
}
