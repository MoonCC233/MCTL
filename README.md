# MCTL

MCTL 是一个用 Go 编写的 Minecraft 终端启动器。它把版本管理、游戏安装、账户切换、Java 检测和启动日志放进同一个键盘驱动的 TUI 界面，适合喜欢在终端里完成一切的用户。

界面使用 Bubble Tea、Lip Gloss 和 ANSI 颜色构建，默认采用深色背景、桃色强调色与圆角面板。所有页面都可以只用键盘操作，并在窄终端中自动调整布局。

## 特性

- **完整的终端界面**：启动、版本、下载、设置、日志五个页面，支持响应式布局。
- **真实安装流程**：读取 Mojang 版本清单，下载客户端、依赖库、natives 和资源文件，校验 SHA-1。
- **版本继承**：解析 `inheritsFrom`，合并父版本与子版本的库、参数和主类。
- **Fabric 支持**：获取 Fabric Meta 加载器列表，安装原版后写入 Fabric 版本配置。
- **Java 管理**：扫描常见安装位置与 `JAVA_HOME`，检查版本兼容性；缺少 Java 时可从 Adoptium 下载 JRE。
- **离线账户**：添加、切换和删除离线账户，自动生成与 Minecraft 一致的离线 UUID。
- **双下载源**：BMCLAPI 镜像与 Mojang 官方源，可在设置中即时切换。
- **启动参数**：自定义最小/最大内存、窗口尺寸、全屏、JVM 参数和游戏参数。
- **日志系统**：启动器日志与游戏 stdout/stderr 实时进入日志页，并同步写入日志文件。
- **命令面板**：按 `:` 快速执行启动、刷新、打开目录、切换页面等操作。
- **演示模式**：无需真实配置即可渲染全部页面，便于预览和截图。

## 界面预览

```bash
# 渲染启动页
mctl --demo

# 渲染指定页面
mctl --demo versions
mctl --demo download
mctl --demo settings
mctl --demo logs

# 一次渲染全部页面
mctl --demo all
```

演示模式支持自定义尺寸：

```bash
mctl --demo launch --demo-width 120 --demo-height 36
```

## 安装

### 下载预编译版本

前往 [Releases](https://github.com/MoonCC233/MCTL/releases) 下载对应平台的压缩包，解压后运行 `mctl`（Windows 为 `mctl.exe`）。

### 使用 Go 安装

需要 Go 1.24 或更高版本：

```bash
go install github.com/MoonCC233/MCTL@latest
```

### 从源码构建

```bash
git clone https://github.com/MoonCC233/MCTL.git
cd MCTL
go build -trimpath -o mctl .
```

Windows：

```powershell
go build -trimpath -o mctl.exe .
```

## 使用

直接运行：

```bash
mctl
```

常用参数：

| 参数 | 说明 |
| --- | --- |
| `--dir <路径>` | 本次运行使用指定的游戏目录 |
| `--source <名称>` | 下载源：`bmclapi` 或 `official` |
| `--print-config` | 打印配置文件路径与当前配置 |
| `--demo [页面]` | 渲染演示界面并退出 |
| `--demo-page <页面>` | 演示页面：`launch`、`versions`、`download`、`settings`、`logs`、`all` |
| `--demo-width <列数>` | 演示界面宽度，默认 `118` |
| `--demo-height <行数>` | 演示界面高度，默认 `34` |
| `--version` | 显示版本号 |
| `-h`, `--help` | 显示帮助 |

首次启动会自动创建配置文件。默认游戏目录遵循各平台惯例；如果可执行文件旁存在 `.minecraft` 目录或 `mctl.portable` 标记，则优先使用便携目录。

## 快捷键

### 全局

| 按键 | 功能 |
| --- | --- |
| `tab` / `shift+tab` | 下一页 / 上一页 |
| `1` - `5` | 直接跳转到启动、版本、下载、设置、日志 |
| `:` | 打开命令面板 |
| `?` | 打开帮助 |
| `q` / `ctrl+c` | 退出 |

### 启动页

| 按键 | 功能 |
| --- | --- |
| `↑` `↓` / `k` `j` | 在版本、账户、启动按钮之间移动 |
| `←` `→` / `h` `l` | 切换版本或账户 |
| `enter` | 激活当前焦点 |
| `a` | 管理账户 |
| `s` | 停止游戏 |
| `d` | 前往下载页 |
| `l` | 打开日志页 |
| `o` | 打开游戏目录 |
| `r` | 刷新版本列表 |

### 版本页

| 按键 | 功能 |
| --- | --- |
| `↑` `↓` | 选择版本 |
| `enter` | 启动选中版本 |
| `d` | 删除选中版本 |
| `o` | 打开版本目录 |
| `/` | 按名称、加载器或版本号筛选 |
| `c` | 清除筛选 |
| `r` | 刷新 |
| `n` | 前往下载页 |

### 下载页

| 按键 | 功能 |
| --- | --- |
| `↑` `↓` | 选择版本 |
| `enter` | 安装原版，或在 Fabric 模式下选择加载器 |
| `←` `→` / `[` `]` | 切换原版 / Fabric |
| `m` | 切换正式版、快照、全部 |
| `/` | 搜索版本 |
| `r` | 刷新版本清单 |

### 设置页

| 按键 | 功能 |
| --- | --- |
| `↑` `↓` | 选择设置项 |
| `enter` | 编辑或执行 |
| `←` `→` | 调整数值或选项 |
| `o` | 打开游戏目录 |

### 日志页

| 按键 | 功能 |
| --- | --- |
| `↑` `↓` | 滚动 |
| `pgup` / `pgdn` | 翻页 |
| `home` / `end` | 跳到顶部 / 跟随最新 |
| `f` | 切换跟随 |
| `c` | 清空日志 |

## 配置与日志

配置目录：

- Windows：`%APPDATA%\MCTL`
- macOS：`~/Library/Application Support/MCTL`
- Linux：`$XDG_CONFIG_HOME/MCTL` 或 `~/.config/MCTL`

主要文件：

- `config.json`：游戏目录、Java 路径、内存、窗口、下载源、账户等。
- `logs/mctl-YYYYMMDD.log`：启动器与游戏日志。

配置采用原子写入；如果 `config.json` 损坏，会保留为 `config.json.broken` 并回退到默认配置。

## 开发

```bash
go test ./...
go vet ./...
go build ./...
```

项目结构：

```text
.
├── main.go                     CLI 入口与演示模式
├── internal/
│   ├── config/                 配置持久化与账户
│   ├── logbuf/                 内存日志环形缓冲区与文件镜像
│   ├── minecraft/              清单、版本解析、安装、Java、启动
│   └── tui/                    Bubble Tea 界面、页面、弹窗与运行时
└── .github/workflows/          CI 与 Release 工作流
```

## 发布

推送形如 `v1.0.0` 的标签会触发 Release 工作流：

```bash
git tag v1.0.0
git push origin v1.0.0
```

工作流会为 Windows、Linux、macOS 的 amd64 与 arm64 构建二进制，生成压缩包和 `checksums.txt`，然后自动创建 GitHub Release。也可以在 Actions 页面手动运行 `Release` 工作流并填写版本号。

## 许可证

[MIT](LICENSE)
