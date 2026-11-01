package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/MoonCC233/MCTL/internal/config"
	"github.com/MoonCC233/MCTL/internal/logbuf"
	"github.com/MoonCC233/MCTL/internal/tui"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "MCTL:", err)
		os.Exit(1)
	}
}

func run() error {
	var (
		showVersion = flag.Bool("version", false, "显示版本号")
		gameDir     = flag.String("dir", "", "指定游戏目录")
		source      = flag.String("source", "", "下载源：bmclapi 或 official")
		demo        = flag.Bool("demo", false, "渲染演示界面并退出")
		demoPage    = flag.String("demo-page", "launch", "演示页面：launch, versions, download, settings, logs, all")
		demoW       = flag.Int("demo-width", 118, "演示界面宽度")
		demoH       = flag.Int("demo-height", 34, "演示界面高度")
		printConfig = flag.Bool("print-config", false, "打印配置路径与内容")
	)
	flag.Usage = printHelp
	args, positionalPage := splitDemoArgs(os.Args[1:])
	if err := flag.CommandLine.Parse(args); err != nil {
		return err
	}
	if positionalPage != "" {
		*demoPage = positionalPage
	} else if *demo && flag.NArg() > 0 {
		*demoPage = flag.Arg(0)
	}

	if *showVersion {
		fmt.Printf("MCTL %s\n", tui.Version)
		return nil
	}
	if *demo {
		fmt.Println(tui.RenderDemo(*demoPage, *demoW, *demoH))
		return nil
	}

	store, err := config.Load()
	if err != nil {
		return fmt.Errorf("加载配置: %w", err)
	}
	if *gameDir != "" {
		abs, err := filepath.Abs(*gameDir)
		if err != nil {
			return fmt.Errorf("无效的游戏目录: %w", err)
		}
		store.Cfg.GameDir = abs
	}
	if *source != "" {
		if *source != "bmclapi" && *source != "official" {
			return fmt.Errorf("未知下载源 %q（可选 bmclapi / official）", *source)
		}
		store.Cfg.DownloadSource = *source
	}
	if *printConfig {
		cfg := store.Snapshot()
		data, _ := json.MarshalIndent(cfg, "", "  ")
		fmt.Printf("配置文件: %s\n%s\n", store.Path, data)
		return nil
	}

	logs := logbuf.New(6000)
	if dir, err := config.LogDir(); err == nil {
		if err := os.MkdirAll(dir, 0o755); err == nil {
			name := "mctl-" + time.Now().Format("20060102") + ".log"
			_ = logs.AttachFile(filepath.Join(dir, name))
		}
	}
	defer logs.Close()
	logs.Add(fmt.Sprintf("MCTL %s 启动 · %s/%s", tui.Version, runtime.GOOS, runtime.GOARCH))
	logs.Add("游戏目录: " + store.Cfg.GameDir)

	rt := tui.NewRuntime(store, logs)
	app := tui.New(store, rt)
	program := tea.NewProgram(app, tea.WithAltScreen())
	if _, err := program.Run(); err != nil {
		return err
	}
	rt.Stop()
	rt.CancelInstall()
	rt.CancelMicrosoftLogin()
	return nil
}

// splitDemoArgs allows a page name to appear between flags, for example
// "mctl --demo versions --demo-width 120".
func splitDemoArgs(args []string) ([]string, string) {
	hasDemo := false
	for _, a := range args {
		if a == "--demo" || a == "-demo" {
			hasDemo = true
			break
		}
	}
	if !hasDemo {
		return args, ""
	}
	out := make([]string, 0, len(args))
	page := ""
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--demo" || a == "-demo" {
			out = append(out, a)
			continue
		}
		if strings.HasPrefix(a, "-") {
			out = append(out, a)
			if !strings.Contains(a, "=") && demoFlagTakesValue(a) && i+1 < len(args) {
				i++
				out = append(out, args[i])
			}
			continue
		}
		if page == "" && isDemoPage(a) {
			page = a
			continue
		}
		out = append(out, a)
	}
	return out, page
}

func demoFlagTakesValue(flag string) bool {
	switch flag {
	case "--dir", "-dir", "--source", "-source", "--demo-page", "-demo-page", "--demo-width", "-demo-width", "--demo-height", "-demo-height":
		return true
	}
	return false
}

func isDemoPage(s string) bool {
	switch s {
	case "launch", "versions", "download", "settings", "tools", "about", "logs", "all":
		return true
	}
	return false
}

func printHelp() {
	fmt.Fprintf(os.Stderr, `MCTL %s — Minecraft 终端启动器

用法:
  mctl [选项]

选项:
  --dir <路径>          指定游戏目录（仅本次运行）
  --source <名称>       下载源: bmclapi | official
  --print-config        打印配置路径与内容
  --demo                渲染演示界面并退出
  --demo-page <页面>    launch | versions | download | settings | tools | about | logs | all
  --demo-width <列数>   演示界面宽度（默认 118）
  --demo-height <行数>  演示界面高度（默认 34）
  --version             显示版本号
  -h, --help            显示本帮助

快捷键:
  tab 切换页面 · 1-7 快速跳转 · : 命令面板 · ? 帮助 · q 退出
`, tui.Version)
}
