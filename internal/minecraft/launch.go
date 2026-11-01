package minecraft

import (
	"bufio"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

// LaunchOptions configures a game launch.
type LaunchOptions struct {
	GameDir         string
	VersionID       string
	PlayerName      string
	UUID            string
	AccessToken     string
	XUID            string
	UserType        string
	Java            *JavaInfo
	MinMemory       int
	MaxMemory       int
	Width           int
	Height          int
	Fullscreen      bool
	JVMArgs         string
	GameArgs        string
	LauncherName    string
	LauncherVersion string
}

// LaunchPlan is a fully resolved java command line.
type LaunchPlan struct {
	Java        string
	Args        []string
	Dir         string
	Version     *ResolvedVersion
	Classpath   []string
	NativeDir   string
	MainClass   string
	CommandLine string
}

// IncompleteError reports missing game files.
type IncompleteError struct{ What string }

func (e *IncompleteError) Error() string { return "游戏文件不完整: " + e.What }

// BuildLaunchPlan resolves arguments without starting a process.
func BuildLaunchPlan(opts LaunchOptions) (*LaunchPlan, error) {
	if opts.VersionID == "" {
		return nil, fmt.Errorf("未选择游戏版本")
	}
	if opts.Java == nil {
		return nil, fmt.Errorf("未配置 Java")
	}
	gameDir, err := filepath.Abs(opts.GameDir)
	if err != nil {
		return nil, err
	}
	rv, err := ResolveVersion(gameDir, opts.VersionID)
	if err != nil {
		return nil, err
	}
	if rv.MainClass == "" {
		return nil, fmt.Errorf("版本 %s 缺少主类", opts.VersionID)
	}
	required := 8
	if rv.JavaVersion != nil && rv.JavaVersion.MajorVersion > 0 {
		required = rv.JavaVersion.MajorVersion
	} else {
		required = InferJavaMajor(opts.VersionID)
	}
	if opts.Java.Major < required {
		return nil, fmt.Errorf("该版本需要 Java %d，当前为 Java %d", required, opts.Java.Major)
	}

	features := Features(opts.Width, opts.Height)
	if _, err := os.Stat(rv.JarPath); err != nil {
		return nil, &IncompleteError{What: filepath.Base(rv.JarPath)}
	}
	classpath := make([]string, 0, len(rv.Libraries)+1)
	seenCP := map[string]bool{}
	for _, lib := range rv.LibrariesFor(features) {
		art, err := lib.Artifact()
		if err != nil || art == nil || art.Path == "" {
			continue
		}
		p := filepath.Join(gameDir, "libraries", filepath.FromSlash(art.Path))
		if _, err := os.Stat(p); err != nil {
			return nil, &IncompleteError{What: filepath.Base(art.Path)}
		}
		if seenCP[p] {
			continue
		}
		seenCP[p] = true
		classpath = append(classpath, p)
	}
	classpath = append(classpath, rv.JarPath)
	nativeDir := filepath.Join(VersionPath(gameDir, opts.VersionID), "natives")
	hasNatives := false
	for _, lib := range rv.LibrariesFor(features) {
		if _, ok := lib.NativeClassifier(); ok {
			hasNatives = true
			break
		}
	}
	if hasNatives {
		if entries, err := os.ReadDir(nativeDir); err != nil || len(entries) == 0 {
			return nil, &IncompleteError{What: "运行库 natives"}
		}
	}
	if rv.AssetIndex != nil && rv.AssetIndex.URL != "" {
		idxID := rv.AssetIndex.ID
		if idxID == "" {
			idxID = rv.Assets
		}
		if _, err := os.Stat(filepath.Join(gameDir, "assets", "indexes", idxID+".json")); err != nil {
			return nil, &IncompleteError{What: "资源索引"}
		}
	}

	assetIndexName := rv.Assets
	if rv.AssetIndex != nil && rv.AssetIndex.ID != "" {
		assetIndexName = rv.AssetIndex.ID
	}
	gameAssets := filepath.Join(gameDir, "assets")
	if rv.AssetIndex != nil && rv.AssetIndex.Virtual {
		gameAssets = filepath.Join(gameDir, "assets", "virtual", assetIndexName)
	}
	uuid := strings.ReplaceAll(opts.UUID, "-", "")
	if uuid == "" {
		uuid = strings.ReplaceAll(OfflineUUID(opts.PlayerName), "-", "")
	}
	token := opts.AccessToken
	if token == "" {
		token = "0"
	}
	userType := opts.UserType
	if userType == "" {
		userType = "legacy"
	}
	xuid := opts.XUID
	if xuid == "" {
		xuid = "0"
	}
	launcherName := opts.LauncherName
	if launcherName == "" {
		launcherName = "MCTL"
	}
	launcherVersion := opts.LauncherVersion
	if launcherVersion == "" {
		launcherVersion = "1.0.0"
	}
	placeholders := map[string]string{
		"auth_player_name":    opts.PlayerName,
		"version_name":        opts.VersionID,
		"game_directory":      gameDir,
		"assets_root":         filepath.Join(gameDir, "assets"),
		"assets_index_name":   assetIndexName,
		"auth_uuid":           uuid,
		"auth_access_token":   token,
		"clientid":            "mctl",
		"auth_xuid":           xuid,
		"user_type":           userType,
		"version_type":        rv.Type,
		"natives_directory":   nativeDir,
		"launcher_name":       launcherName,
		"launcher_version":    launcherVersion,
		"classpath":           strings.Join(classpath, string(os.PathListSeparator)),
		"classpath_separator": string(os.PathListSeparator),
		"library_directory":   filepath.Join(gameDir, "libraries"),
		"resolution_width":    fmt.Sprint(opts.Width),
		"resolution_height":   fmt.Sprint(opts.Height),
		"game_assets":         gameAssets,
		"user_properties":     "{}",
		"auth_session":        "token:" + token + ":" + uuid,
	}

	jvm := rv.JVMArguments(features)
	jvm = replaceAll(jvm, placeholders)
	game := rv.GameArguments(features)
	if len(game) == 0 && rv.MinecraftArguments != "" {
		game = SplitArgs(rv.MinecraftArguments)
	}
	game = replaceAll(game, placeholders)

	jvm = append([]string{
		fmt.Sprintf("-Xms%dM", maxInt(opts.MinMemory, 256)),
		fmt.Sprintf("-Xmx%dM", maxInt(opts.MaxMemory, 512)),
	}, jvm...)
	if !hasFlag(jvm, "-cp") && !hasFlag(jvm, "-classpath") {
		jvm = append(jvm, "-cp", placeholders["classpath"])
	}
	if !hasPrefixFlag(jvm, "-Djava.library.path") {
		jvm = append(jvm, "-Djava.library.path="+nativeDir)
	}
	if !hasPrefixFlag(jvm, "-Dminecraft.launcher.brand") {
		jvm = append(jvm, "-Dminecraft.launcher.brand="+launcherName)
	}
	if !hasPrefixFlag(jvm, "-Dminecraft.launcher.version") {
		jvm = append(jvm, "-Dminecraft.launcher.version="+launcherVersion)
	}
	if rv.Logging != nil && rv.Logging.Client != nil {
		arg := rv.Logging.Client.Argument
		if arg == "" {
			arg = "-Dlog4j.configurationFile=${path}"
		}
		arg = strings.ReplaceAll(arg, "${path}", LoggingConfigPath(gameDir, rv.Logging.Client))
		if !hasPrefixFlag(jvm, "-Dlog4j.configurationFile") {
			jvm = append(jvm, arg)
		}
	}
	if runtime.GOOS == "darwin" && !hasFlag(jvm, "-XstartOnFirstThread") {
		jvm = append(jvm, "-XstartOnFirstThread")
	}
	jvm = append(jvm, SplitArgs(opts.JVMArgs)...)

	if opts.Fullscreen && !hasFlag(game, "--fullscreen") {
		game = append(game, "--fullscreen")
	}
	if opts.Width > 0 && opts.Height > 0 && !opts.Fullscreen {
		if !hasFlag(game, "--width") {
			game = append(game, "--width", fmt.Sprint(opts.Width), "--height", fmt.Sprint(opts.Height))
		}
	}
	game = append(game, SplitArgs(opts.GameArgs)...)

	args := append(jvm, rv.MainClass)
	args = append(args, game...)
	plan := &LaunchPlan{
		Java:      opts.Java.Path,
		Args:      args,
		Dir:       gameDir,
		Version:   rv,
		Classpath: classpath,
		NativeDir: nativeDir,
		MainClass: rv.MainClass,
	}
	plan.CommandLine = opts.Java.Path + " " + strings.Join(args, " ")
	return plan, nil
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func replaceAll(in []string, vars map[string]string) []string {
	out := make([]string, len(in))
	for i, s := range in {
		for k, v := range vars {
			s = strings.ReplaceAll(s, "${"+k+"}", v)
		}
		out[i] = s
	}
	return out
}

func hasFlag(args []string, flag string) bool {
	for _, a := range args {
		if a == flag {
			return true
		}
	}
	return false
}

func hasPrefixFlag(args []string, prefix string) bool {
	for _, a := range args {
		if a == prefix || strings.HasPrefix(a, prefix+"=") {
			return true
		}
	}
	return false
}

// SplitArgs tokenizes a command-line fragment with quote support.
func SplitArgs(s string) []string {
	var out []string
	var cur strings.Builder
	inSingle, inDouble := false, false
	flush := func() {
		if cur.Len() > 0 {
			out = append(out, cur.String())
			cur.Reset()
		}
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '\\' && i+1 < len(s) && (s[i+1] == '"' || s[i+1] == '\\'):
			cur.WriteByte(s[i+1])
			i++
		case c == '"' && !inSingle:
			inDouble = !inDouble
		case c == '\'' && !inDouble:
			inSingle = !inSingle
		case (c == ' ' || c == '\t' || c == '\n') && !inSingle && !inDouble:
			flush()
		default:
			cur.WriteByte(c)
		}
	}
	flush()
	return out
}

// Start launches the game, streaming stdout/stderr line by line.
func (p *LaunchPlan) Start(onLine func(stream, line string), onExit func(error)) (*exec.Cmd, error) {
	cmd := exec.Command(p.Java, p.Args...)
	cmd.Dir = p.Dir
	cmd.Env = append(os.Environ(), "MCTL=1")
	hideWindow(cmd)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	var wg sync.WaitGroup
	scan := func(r interface{ Read([]byte) (int, error) }, stream string) {
		defer wg.Done()
		sc := bufio.NewScanner(r)
		sc.Buffer(make([]byte, 64*1024), 2*1024*1024)
		for sc.Scan() {
			line := strings.TrimRight(sc.Text(), "\r")
			if onLine != nil {
				onLine(stream, line)
			}
		}
	}
	wg.Add(2)
	go scan(stdout, "stdout")
	go scan(stderr, "stderr")
	go func() {
		wg.Wait()
		err := cmd.Wait()
		if onExit != nil {
			onExit(err)
		}
	}()
	return cmd, nil
}

// OfflineUUID derives the same UUID Minecraft uses for offline players.
func OfflineUUID(name string) string {
	sum := md5.Sum([]byte("OfflinePlayer:" + name))
	sum[6] = (sum[6] & 0x0f) | 0x30
	sum[8] = (sum[8] & 0x3f) | 0x80
	h := hex.EncodeToString(sum[:])
	return h[0:8] + "-" + h[8:12] + "-" + h[12:16] + "-" + h[16:20] + "-" + h[20:32]
}
