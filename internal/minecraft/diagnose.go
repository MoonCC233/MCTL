package minecraft

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// DiagnoseStatus is the severity of one environment check.
type DiagnoseStatus string

const (
	DiagnoseOK   DiagnoseStatus = "ok"
	DiagnoseWarn DiagnoseStatus = "warn"
	DiagnoseFail DiagnoseStatus = "fail"
)

// DiagnoseCheck is one result of the environment self-test.
type DiagnoseCheck struct {
	Name   string
	Status DiagnoseStatus
	Detail string
}

// DiagnoseOptions configures the environment self-test.
type DiagnoseOptions struct {
	GameDir   string
	JavaPath  string
	VersionID string
}

// Diagnose checks the game directory, Java runtime, selected version, assets
// and download source. A nil HTTP client skips the network check.
func Diagnose(ctx context.Context, hc *HTTPClient, opts DiagnoseOptions) []DiagnoseCheck {
	var checks []DiagnoseCheck
	add := func(name string, status DiagnoseStatus, detail string) {
		checks = append(checks, DiagnoseCheck{Name: name, Status: status, Detail: detail})
	}

	gameDir := strings.TrimSpace(opts.GameDir)
	if gameDir == "" {
		add("游戏目录", DiagnoseFail, "尚未配置游戏目录")
	} else if err := os.MkdirAll(gameDir, 0o755); err != nil {
		add("游戏目录", DiagnoseFail, "无法创建目录: "+err.Error())
	} else {
		f, err := os.CreateTemp(gameDir, ".mctl-check-*")
		if err != nil {
			add("游戏目录", DiagnoseFail, "目录不可写: "+err.Error())
		} else {
			name := f.Name()
			f.Close()
			os.Remove(name)
			add("游戏目录", DiagnoseOK, gameDir)
		}
	}

	if info, err := FindJavaFor(8, opts.JavaPath); err != nil {
		add("Java 运行时", DiagnoseWarn, "未找到可用的 Java，可在设置中指定路径")
	} else {
		add("Java 运行时", DiagnoseOK, info.String()+" · "+info.Arch)
	}

	if strings.TrimSpace(opts.VersionID) == "" {
		add("游戏版本", DiagnoseWarn, "尚未选择版本")
	} else if gameDir != "" {
		rv, err := ResolveVersion(gameDir, opts.VersionID)
		if err != nil {
			add("游戏版本", DiagnoseFail, "无法解析 "+opts.VersionID+": "+err.Error())
		} else {
			var missing []string
			if rv.JarPath == "" {
				missing = append(missing, "客户端 jar")
			}
			if rv.MainClass == "" {
				missing = append(missing, "主类")
			}
			if len(missing) > 0 {
				add("游戏版本", DiagnoseFail, "缺少 "+strings.Join(missing, "、"))
			} else {
				loader := detectLoader(rv.ID, rv.MainClass)
				add("游戏版本", DiagnoseOK, rv.ID+" · "+loader)
			}
		}
	}

	if gameDir != "" {
		if st, err := os.Stat(filepath.Join(gameDir, "assets", "indexes")); err != nil || !st.IsDir() {
			add("资源文件", DiagnoseWarn, "资源索引尚未下载，首次启动会自动补全")
		} else {
			add("资源文件", DiagnoseOK, "资源索引已就绪")
		}
	}

	if hc != nil {
		cctx, cancel := context.WithTimeout(ctx, 15*time.Second)
		defer cancel()
		if data, err := hc.Get(cctx, hc.Source.ManifestURL); err != nil {
			add("下载源", DiagnoseWarn, "无法连接: "+err.Error())
		} else {
			add("下载源", DiagnoseOK, fmt.Sprintf("%s · %.1f KB", hc.Source.Name, float64(len(data))/1024))
		}
	}
	return checks
}
