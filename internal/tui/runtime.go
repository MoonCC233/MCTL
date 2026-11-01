package tui

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/MoonCC233/MCTL/internal/config"
	"github.com/MoonCC233/MCTL/internal/logbuf"
	"github.com/MoonCC233/MCTL/internal/minecraft"
)

// Version is the launcher version. Release builds override it with -ldflags.
var Version = "1.0.5"

// Messages flowing from background work into the Bubble Tea loop.
type eventMsg struct{ Msg tea.Msg }

type logEventMsg struct{}

type versionsMsg struct {
	List []minecraft.InstalledVersion
	Err  error
}

type manifestMsg struct {
	Manifest *minecraft.VersionManifest
	Err      error
}

type javaMsg struct {
	Info   *minecraft.JavaInfo
	Err    error
	Source string
}

type progressMsg struct {
	Scope string
	P     minecraft.Progress
}

type installDoneMsg struct {
	Kind     string
	ID       string
	Err      error
	Canceled bool
}

type fabricLoadersMsg struct {
	MC      string
	Loaders []minecraft.FabricLoader
	Err     error
}

type loaderVersionsMsg struct {
	Kind   string
	MC     string
	Builds []minecraft.LoaderBuild
	Err    error
}

type launchStageMsg struct {
	Stage   string
	Percent float64
	Detail  string
}

type launchStartedMsg struct {
	Plan *minecraft.LaunchPlan
	PID  int
}

type launchExitMsg struct {
	Err     error
	Stopped bool
}

type microsoftCodeMsg struct {
	Code minecraft.MicrosoftDeviceCode
}

type microsoftDoneMsg struct {
	Account  config.Account
	Err      error
	Canceled bool
}

// Runtime owns background state shared by the pages.
type Runtime struct {
	Store  *config.Store
	Logs   *logbuf.Buffer
	Events chan tea.Msg

	mu       sync.Mutex
	manifest *minecraft.VersionManifest
	manErr   error
	manLoad  bool

	java    *minecraft.JavaInfo
	javaErr error

	installing    bool
	installCancel context.CancelFunc
	authing       bool
	authCancel    context.CancelFunc
	launching     bool
	stopping      bool
	cmd           *exec.Cmd
}

// NewRuntime creates the shared runtime.
func NewRuntime(store *config.Store, logs *logbuf.Buffer) *Runtime {
	return &Runtime{
		Store:  store,
		Logs:   logs,
		Events: make(chan tea.Msg, 4096),
	}
}

// Emit queues a message for the UI loop.
func (r *Runtime) Emit(msg tea.Msg) {
	r.Events <- msg
}

// Logf writes a launcher log line and notifies the UI.
func (r *Runtime) Logf(format string, args ...any) {
	line := "[MCTL] " + fmt.Sprintf(format, args...)
	r.Logs.Add(line)
	r.Emit(logEventMsg{})
}

// LogRaw appends raw game output without a launcher prefix.
func (r *Runtime) LogRaw(line string) {
	r.Logs.AddRaw(line)
}

// Java returns the currently detected runtime.
func (r *Runtime) Java() *minecraft.JavaInfo {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.java
}

// Manifest returns the cached version catalogue.
func (r *Runtime) Manifest() *minecraft.VersionManifest {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.manifest
}

// Busy reports whether an install or launch is in progress.
func (r *Runtime) Busy() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.installing || r.launching || r.authing
}

// Installing reports whether an install is running.
func (r *Runtime) Installing() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.installing
}

// Launching reports whether the game process is alive.
func (r *Runtime) Launching() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.launching
}

// ScanVersions rescans the game directory.
func (r *Runtime) ScanVersions() {
	cfg := r.Store.Snapshot()
	go func() {
		list, err := minecraft.ScanVersions(cfg.GameDir)
		r.Emit(versionsMsg{List: list, Err: err})
	}()
}

// LoadManifest fetches the version catalogue, using the cache unless forced.
func (r *Runtime) LoadManifest(force bool) {
	r.mu.Lock()
	if r.manLoad {
		r.mu.Unlock()
		return
	}
	if r.manifest != nil && !force {
		m := r.manifest
		r.mu.Unlock()
		r.Emit(manifestMsg{Manifest: m})
		return
	}
	r.manLoad = true
	r.mu.Unlock()

	cfg := r.Store.Snapshot()
	src := minecraft.GetSource(cfg.DownloadSource)
	go func() {
		hc := minecraft.NewHTTPClient(src)
		m, err := minecraft.FetchManifest(context.Background(), hc)
		r.mu.Lock()
		r.manLoad = false
		if err != nil {
			r.manErr = err
		} else {
			r.manifest = m
			r.manErr = nil
		}
		r.mu.Unlock()
		if err != nil {
			r.Logf("获取版本列表失败: %v", err)
		} else {
			r.Logf("版本列表已更新（%d 个版本）", len(m.Versions))
		}
		r.Emit(manifestMsg{Manifest: m, Err: err})
	}()
}

// DetectJava probes the configured or automatic Java runtime.
func (r *Runtime) DetectJava(preferred string) {
	go func() {
		var info *minecraft.JavaInfo
		var err error
		source := "auto"
		if preferred != "" {
			source = "config"
			info, err = minecraft.DetectJava(preferred)
			if err != nil {
				info, err = minecraft.FindJava("")
				source = "auto"
			}
		} else {
			info, err = minecraft.FindJava("")
		}
		r.mu.Lock()
		r.java = info
		r.javaErr = err
		r.mu.Unlock()
		if err != nil {
			r.Logf("Java 检测失败: %v", err)
		} else if info != nil {
			r.Logf("已检测到 Java %s（%s）", info.Version, info.Path)
		}
		r.Emit(javaMsg{Info: info, Err: err, Source: source})
	}()
}

// Install starts an asynchronous vanilla or Fabric installation.
func (r *Runtime) Install(kind, id, mcVersion, loader string, entry *minecraft.VersionEntry) {
	r.mu.Lock()
	if r.installing || r.launching || r.authing {
		r.mu.Unlock()
		r.Emit(installDoneMsg{Kind: kind, ID: id, Err: fmt.Errorf("已有任务正在进行")})
		return
	}
	r.installing = true
	ctx, cancel := context.WithCancel(context.Background())
	r.installCancel = cancel
	r.mu.Unlock()

	go func() {
		cfg := r.Store.Snapshot()
		src := minecraft.GetSource(cfg.DownloadSource)
		inst := minecraft.NewInstaller(cfg.GameDir, src, func(s string) { r.Logf("%s", s) }, func(p minecraft.Progress) {
			r.Emit(progressMsg{Scope: "install", P: p})
		})
		actualID := id
		var err error
		if kind == "fabric" {
			r.Logf("获取 Fabric %s 配置", loader)
			var profile []byte
			profile, err = minecraft.FetchFabricProfile(ctx, inst.HTTP, mcVersion, loader)
			if err == nil {
				var v minecraft.VersionJSON
				if json.Unmarshal(profile, &v) == nil && v.ID != "" {
					actualID = v.ID
				}
				err = inst.InstallProfile(ctx, actualID, profile)
			}
		} else if kind == "quilt" {
			r.Logf("获取 Quilt %s 配置", loader)
			var profile []byte
			profile, err = minecraft.FetchQuiltProfile(ctx, inst.HTTP, mcVersion, loader)
			if err == nil {
				var v minecraft.VersionJSON
				if json.Unmarshal(profile, &v) == nil && v.ID != "" {
					actualID = v.ID
				}
				err = inst.InstallProfile(ctx, actualID, profile)
			}
		} else if kind == "forge" || kind == "neoforge" {
			required := minecraft.InferJavaMajor(mcVersion)
			r.Logf("准备 %s 安装器（Java %d）", kind, required)
			var javaInfo *minecraft.JavaInfo
			if cfg.AutoJava {
				javaInfo, err = inst.EnsureJava(ctx, required, cfg.JavaPath)
			} else {
				javaInfo, err = minecraft.FindJavaFor(required, cfg.JavaPath)
			}
			if err == nil {
				r.Logf("先安装原版 %s", mcVersion)
				err = inst.InstallVersion(ctx, mcVersion, nil)
			}
			if err == nil {
				actualID, err = inst.InstallLoader(ctx, kind, mcVersion, loader, javaInfo)
			}
		} else {
			err = inst.InstallVersion(ctx, id, entry)
		}
		canceled := ctx.Err() != nil
		if err == nil {
			_ = r.Store.Update(func(c *config.Config) {
				c.SelectedVersion = actualID
			})
			r.Logf("安装完成: %s", actualID)
		} else if canceled {
			r.Logf("安装已取消: %s", actualID)
		} else {
			r.Logf("安装失败 %s: %v", actualID, err)
		}
		r.mu.Lock()
		r.installing = false
		r.installCancel = nil
		r.mu.Unlock()
		r.Emit(installDoneMsg{Kind: kind, ID: actualID, Err: err, Canceled: canceled})
		r.ScanVersions()
	}()
}

// ImportModpack installs a Modrinth pack into an isolated instance.
func (r *Runtime) ImportModpack(path string) {
	r.mu.Lock()
	if r.installing || r.launching || r.authing {
		r.mu.Unlock()
		r.Emit(installDoneMsg{Kind: "modpack", Err: fmt.Errorf("已有任务正在进行")})
		return
	}
	r.installing = true
	ctx, cancel := context.WithCancel(context.Background())
	r.installCancel = cancel
	r.mu.Unlock()

	go func() {
		cfg := r.Store.Snapshot()
		src := minecraft.GetSource(cfg.DownloadSource)
		inst := minecraft.NewInstaller(cfg.GameDir, src, func(s string) { r.Logf("%s", s) }, func(p minecraft.Progress) {
			r.Emit(progressMsg{Scope: "install", P: p})
		})
		id, err := inst.ImportModpack(ctx, path, cfg.JavaPath, cfg.AutoJava, cfg.CurseForgeAPIKey)
		canceled := ctx.Err() != nil
		if err == nil {
			_ = r.Store.Update(func(c *config.Config) {
				c.SelectedVersion = id
			})
			r.Logf("整合包导入完成: %s", id)
		} else if canceled {
			r.Logf("整合包导入已取消")
		} else {
			r.Logf("整合包导入失败: %v", err)
		}
		r.mu.Lock()
		r.installing = false
		r.installCancel = nil
		r.mu.Unlock()
		r.Emit(installDoneMsg{Kind: "modpack", ID: id, Err: err, Canceled: canceled})
		r.ScanVersions()
	}()
}

// CancelInstall cancels a running installation.
func (r *Runtime) CancelInstall() {
	r.mu.Lock()
	cancel := r.installCancel
	r.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

// FetchFabricLoaders lists loader builds for a game version.
func (r *Runtime) FetchFabricLoaders(mcVersion string) {
	cfg := r.Store.Snapshot()
	src := minecraft.GetSource(cfg.DownloadSource)
	go func() {
		hc := minecraft.NewHTTPClient(src)
		loaders, err := minecraft.FetchFabricLoaders(context.Background(), hc, mcVersion)
		if err != nil {
			r.Logf("获取 Fabric 版本失败: %v", err)
		}
		r.Emit(fabricLoadersMsg{MC: mcVersion, Loaders: loaders, Err: err})
	}()
}

// FetchLoaderVersions lists Forge or NeoForge builds for a game version.
func (r *Runtime) FetchLoaderVersions(kind, mcVersion string) {
	cfg := r.Store.Snapshot()
	src := minecraft.GetSource(cfg.DownloadSource)
	go func() {
		hc := minecraft.NewHTTPClient(src)
		var builds []minecraft.LoaderBuild
		var err error
		if kind == "quilt" {
			builds, err = minecraft.FetchQuiltLoaders(context.Background(), hc, mcVersion)
		} else {
			builds, err = minecraft.FetchLoaderBuilds(context.Background(), hc, kind, mcVersion)
		}
		if err != nil {
			r.Logf("获取 %s 版本失败: %v", kind, err)
		}
		r.Emit(loaderVersionsMsg{Kind: kind, MC: mcVersion, Builds: builds, Err: err})
	}()
}

// StartMicrosoftLogin starts the device-code login flow.
func (r *Runtime) StartMicrosoftLogin(clientID string) {
	r.mu.Lock()
	if r.authing || r.installing || r.launching {
		r.mu.Unlock()
		r.Emit(microsoftDoneMsg{Err: fmt.Errorf("已有任务正在进行")})
		return
	}
	r.authing = true
	ctx, cancel := context.WithCancel(context.Background())
	r.authCancel = cancel
	r.mu.Unlock()

	go func() {
		res, err := minecraft.LoginMicrosoft(ctx, clientID, func(code minecraft.MicrosoftDeviceCode) {
			r.Emit(microsoftCodeMsg{Code: code})
		})
		canceled := ctx.Err() != nil
		var account config.Account
		if err == nil {
			account = config.Account{
				Name:         res.Name,
				UUID:         res.UUID,
				Type:         "microsoft",
				AccessToken:  res.AccessToken,
				RefreshToken: res.RefreshToken,
				ExpiresAt:    res.ExpiresAt,
				XUID:         res.XUID,
				ClientID:     clientID,
			}
			if saveErr := r.Store.SetAccount(account); saveErr != nil {
				err = saveErr
			}
		}
		r.mu.Lock()
		r.authing = false
		r.authCancel = nil
		r.mu.Unlock()
		if canceled {
			r.Logf("Microsoft 登录已取消")
		} else if err != nil {
			r.Logf("Microsoft 登录失败: %v", err)
		} else {
			r.Logf("Microsoft 账户已登录: %s", account.Name)
		}
		r.Emit(microsoftDoneMsg{Account: account, Err: err, Canceled: canceled})
	}()
}

// CancelMicrosoftLogin cancels a pending device-code login.
func (r *Runtime) CancelMicrosoftLogin() {
	r.mu.Lock()
	cancel := r.authCancel
	r.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

// Launch prepares files and starts the game.
func (r *Runtime) Launch(versionID, gameDir string) {
	r.mu.Lock()
	if r.launching || r.installing || r.authing {
		r.mu.Unlock()
		r.Emit(launchExitMsg{Err: fmt.Errorf("已有任务正在进行")})
		return
	}
	r.launching = true
	r.stopping = false
	r.mu.Unlock()

	go func() {
		cfg := r.Store.Snapshot()
		account := cfg.Account()
		if gameDir == "" {
			gameDir = cfg.GameDir
		}
		src := minecraft.GetSource(cfg.DownloadSource)
		inst := minecraft.NewInstaller(gameDir, src, func(s string) { r.Logf("%s", s) }, func(p minecraft.Progress) {
			r.Emit(progressMsg{Scope: "launch", P: p})
		})
		fail := func(err error) {
			r.mu.Lock()
			r.launching = false
			r.cmd = nil
			r.mu.Unlock()
			r.Logf("启动失败: %v", err)
			r.Emit(launchExitMsg{Err: err})
		}
		if account.Type == "microsoft" && (account.AccessToken == "" || time.Now().Unix() >= account.ExpiresAt-120) {
			clientID := account.ClientID
			if clientID == "" {
				clientID = cfg.MicrosoftClientID
			}
			r.Logf("刷新 Microsoft 登录令牌")
			res, rerr := minecraft.RefreshMicrosoft(context.Background(), clientID, account.RefreshToken)
			if rerr != nil {
				fail(fmt.Errorf("Microsoft 登录已失效: %w", rerr))
				return
			}
			account.AccessToken = res.AccessToken
			account.RefreshToken = res.RefreshToken
			account.ExpiresAt = res.ExpiresAt
			if res.Name != "" {
				account.Name = res.Name
			}
			if res.UUID != "" {
				account.UUID = res.UUID
			}
			if res.XUID != "" {
				account.XUID = res.XUID
			}
			if err := r.Store.SetAccount(account); err != nil {
				fail(fmt.Errorf("保存 Microsoft 登录信息: %w", err))
				return
			}
		}
		r.Logf("准备启动 %s（账户 %s）", versionID, account.Name)
		rv, err := minecraft.ResolveVersion(gameDir, versionID)
		if err != nil {
			fail(err)
			return
		}
		required := 8
		if rv.JavaVersion != nil && rv.JavaVersion.MajorVersion > 0 {
			required = rv.JavaVersion.MajorVersion
		} else {
			required = minecraft.InferJavaMajor(versionID)
		}
		r.Emit(launchStageMsg{Stage: fmt.Sprintf("检查 Java %d", required)})
		var javaInfo *minecraft.JavaInfo
		if cfg.AutoJava {
			javaInfo, err = inst.EnsureJava(context.Background(), required, cfg.JavaPath)
		} else {
			javaInfo, err = minecraft.FindJavaFor(required, cfg.JavaPath)
		}
		if err != nil {
			fail(fmt.Errorf("需要 Java %d：%w", required, err))
			return
		}
		r.mu.Lock()
		r.java = javaInfo
		r.mu.Unlock()
		r.Emit(javaMsg{Info: javaInfo, Source: "launch"})
		r.Emit(launchStageMsg{Stage: "检查游戏文件"})
		if err := inst.EnsureInstalled(context.Background(), versionID); err != nil {
			fail(err)
			return
		}
		uuid := account.UUID
		if uuid == "" {
			uuid = minecraft.OfflineUUID(account.Name)
		}
		userType := "legacy"
		if account.Type == "microsoft" {
			userType = "msa"
		}
		plan, err := minecraft.BuildLaunchPlan(minecraft.LaunchOptions{
			GameDir:         gameDir,
			VersionID:       versionID,
			PlayerName:      account.Name,
			UUID:            uuid,
			AccessToken:     account.AccessToken,
			XUID:            account.XUID,
			UserType:        userType,
			Java:            javaInfo,
			MinMemory:       cfg.MinMemory,
			MaxMemory:       cfg.MaxMemory,
			Width:           cfg.Width,
			Height:          cfg.Height,
			Fullscreen:      cfg.Fullscreen,
			JVMArgs:         cfg.JVMArgs,
			GameArgs:        cfg.GameArgs,
			LauncherName:    "MCTL",
			LauncherVersion: Version,
		})
		if err != nil {
			fail(err)
			return
		}
		r.Emit(launchStageMsg{Stage: "启动游戏进程", Percent: 1})
		exitCh := make(chan error, 1)
		cmd, err := plan.Start(func(stream, line string) {
			r.LogRaw(line)
		}, func(err error) {
			exitCh <- err
		})
		if err != nil {
			fail(err)
			return
		}
		r.mu.Lock()
		r.cmd = cmd
		pid := 0
		if cmd.Process != nil {
			pid = cmd.Process.Pid
		}
		r.mu.Unlock()
		r.Logf("游戏已启动（PID %d）", pid)
		r.Emit(launchStartedMsg{Plan: plan, PID: pid})
		err = <-exitCh
		r.mu.Lock()
		stopped := r.stopping
		r.stopping = false
		r.launching = false
		r.cmd = nil
		r.mu.Unlock()
		if stopped {
			r.Logf("游戏已停止")
		} else if err != nil {
			r.Logf("游戏异常退出: %v", err)
		} else {
			r.Logf("游戏已退出")
		}
		r.Emit(launchExitMsg{Err: err, Stopped: stopped})
	}()
}

// Stop terminates the game process.
func (r *Runtime) Stop() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.cmd == nil || r.cmd.Process == nil {
		return
	}
	r.stopping = true
	_ = r.cmd.Process.Kill()
}

// ProgressPercent converts a progress update into a 0..1 ratio.
func ProgressPercent(p minecraft.Progress) float64 {
	if p.Total > 0 {
		return float64(p.Done) / float64(p.Total)
	}
	if p.TotalBytes > 0 {
		return float64(p.Bytes) / float64(p.TotalBytes)
	}
	return 0
}

// RefreshDelay is a small helper for UI animations.
const RefreshDelay = 180 * time.Millisecond
