package minecraft

import (
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

// windowsJavaGlobs lists Java install layouts relative to a drive root. The
// launcher scans every fixed drive with these patterns so runtimes installed
// outside C: are still discovered.
var windowsJavaGlobs = []string{
	`Program Files\Java\*\bin\java.exe`,
	`Program Files\Java\*\*\bin\java.exe`,
	`Program Files\Eclipse Adoptium\*\bin\java.exe`,
	`Program Files\Eclipse Foundation\*\bin\java.exe`,
	`Program Files\Microsoft\jdk-*\bin\java.exe`,
	`Program Files\Zulu\*\bin\java.exe`,
	`Program Files\BellSoft\*\bin\java.exe`,
	`Program Files\Amazon Corretto\*\bin\java.exe`,
	`Program Files\Semeru\*\bin\java.exe`,
	`Program Files\AdoptOpenJDK\*\bin\java.exe`,
	`Program Files\Temurin\*\bin\java.exe`,
	`Program Files\GraalVM\*\bin\java.exe`,
	`Program Files\Android\Android Studio\jbr\bin\java.exe`,
	`Program Files\JetBrains\*\jbr\bin\java.exe`,
	`Program Files (x86)\Java\*\bin\java.exe`,
	`Program Files (x86)\Eclipse Adoptium\*\bin\java.exe`,
	`Program Files (x86)\Microsoft\jdk-*\bin\java.exe`,
	`Program Files (x86)\Zulu\*\bin\java.exe`,
	`Java\*\bin\java.exe`,
	`Java\*\*\bin\java.exe`,
	`jdk*\bin\java.exe`,
	`jdk-*\bin\java.exe`,
	`jre*\bin\java.exe`,
	`Tools\jdk*\bin\java.exe`,
	`Dev\Java\*\bin\java.exe`,
	`Environment\Java\*\bin\java.exe`,
}

// windowsDriveRoots returns the mounted drive roots (C: through Z:).
func windowsDriveRoots() []string {
	var roots []string
	for c := 'C'; c <= 'Z'; c++ {
		root := string(c) + `:\`
		if st, err := os.Stat(root); err == nil && st.IsDir() {
			roots = append(roots, root)
		}
	}
	return roots
}

// windowsJavaPaths expands the per-drive Java install patterns.
func windowsJavaPaths() []string {
	var out []string
	for _, root := range windowsDriveRoots() {
		for _, pattern := range windowsJavaGlobs {
			matches, _ := filepath.Glob(filepath.Join(root, pattern))
			out = append(out, matches...)
		}
	}
	return out
}

// ListJava detects every Java runtime found on this system, newest first.
func ListJava(preferred string) []JavaInfo {
	seen := map[string]bool{}
	var out []JavaInfo
	for _, p := range JavaCandidates(preferred) {
		info, err := DetectJava(p)
		if err != nil {
			continue
		}
		key := info.Path
		if runtime.GOOS == "windows" {
			key = strings.ToLower(key)
		}
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, *info)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Major != out[j].Major {
			return out[i].Major > out[j].Major
		}
		return strings.ToLower(out[i].Path) < strings.ToLower(out[j].Path)
	})
	return out
}
