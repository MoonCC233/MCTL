package minecraft

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildLaunchPlan(t *testing.T) {
	dir := t.TempDir()
	writeJSON(t, VersionJSONPath(dir, "1.20.1"), map[string]any{
		"id":        "1.20.1",
		"type":      "release",
		"mainClass": "net.minecraft.client.main.Main",
		"assets":    "5",
		"assetIndex": map[string]any{
			"id": "5", "url": "https://example.invalid/5.json",
		},
		"downloads": map[string]any{
			"client": map[string]any{"url": "https://example.invalid/client.jar", "size": 1},
		},
		"javaVersion": map[string]any{"majorVersion": 17},
		"libraries": []any{
			map[string]any{"name": "com.example:lib:1.0"},
		},
		"arguments": map[string]any{
			"game": []any{
				"--username", "${auth_player_name}",
				"--gameDir", "${game_directory}",
				"--assetsDir", "${assets_root}",
				"--assetIndex", "${assets_index_name}",
				"--uuid", "${auth_uuid}",
				"--accessToken", "${auth_access_token}",
				"--version", "${version_name}",
				"--width", "${resolution_width}",
				"--height", "${resolution_height}",
			},
			"jvm": []any{"-Ddemo=true", "-Djava.library.path=${natives_directory}"},
		},
	})
	mustWrite(t, filepath.Join(dir, "versions", "1.20.1", "1.20.1.jar"), "jar")
	mustWrite(t, filepath.Join(dir, "libraries", "com", "example", "lib", "1.0", "lib-1.0.jar"), "jar")
	mustWrite(t, filepath.Join(dir, "assets", "indexes", "5.json"), "{}")

	plan, err := BuildLaunchPlan(LaunchOptions{
		GameDir:      dir,
		VersionID:    "1.20.1",
		PlayerName:   "Alex",
		Java:         &JavaInfo{Path: "java", Major: 17},
		MinMemory:    512,
		MaxMemory:    2048,
		Width:        1280,
		Height:       720,
		JVMArgs:      "-XX:+UseG1GC",
		GameArgs:     "--demo",
		LauncherName: "MCTL",
	})
	if err != nil {
		t.Fatal(err)
	}
	if plan.MainClass != "net.minecraft.client.main.Main" {
		t.Fatalf("main class = %q", plan.MainClass)
	}
	joined := strings.Join(plan.Args, "\x00")
	for _, want := range []string{
		"-Xms512M", "-Xmx2048M", "-Ddemo=true", "-Djava.library.path=",
		"lib-1.0.jar", "1.20.1.jar", "-XX:+UseG1GC", "--username", "Alex",
		"--width", "1280", "--height", "720", "--demo", "--assetIndex", "5",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("args missing %q:\n%s", want, strings.Join(plan.Args, "\n"))
		}
	}
	if strings.Contains(joined, "${") {
		t.Fatalf("unresolved placeholder in args:\n%s", strings.Join(plan.Args, "\n"))
	}
	if len(plan.Classpath) != 2 {
		t.Fatalf("classpath = %#v", plan.Classpath)
	}
	if plan.NativeDir == "" || plan.CommandLine == "" {
		t.Fatal("plan metadata should be populated")
	}
}

func TestBuildLaunchPlanRejectsOldJava(t *testing.T) {
	dir := t.TempDir()
	writeJSON(t, VersionJSONPath(dir, "1.20.1"), map[string]any{
		"id":          "1.20.1",
		"mainClass":   "net.minecraft.client.main.Main",
		"javaVersion": map[string]any{"majorVersion": 17},
	})
	_, err := BuildLaunchPlan(LaunchOptions{
		GameDir:   dir,
		VersionID: "1.20.1",
		Java:      &JavaInfo{Path: "java", Major: 8},
	})
	if err == nil || !strings.Contains(err.Error(), "Java 17") {
		t.Fatalf("expected Java requirement error, got %v", err)
	}
}

func TestDeleteVersionRejectsTraversal(t *testing.T) {
	dir := t.TempDir()
	if err := DeleteVersion(dir, `..\outside`); err == nil {
		t.Fatal("expected traversal rejection")
	}
	if err := DeleteVersion(dir, "1.20.1"); err != nil {
		t.Fatalf("normal delete failed: %v", err)
	}
}

func mustWrite(t *testing.T, p, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
