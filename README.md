# MCTL

MCTL 是一个用 Go 编写的 Minecraft 终端启动器。它把版本管理、游戏安装、账户切换、Java 检测和启动日志放进同一个键盘驱动的 TUI 界面，适合喜欢在终端里完成一切的用户。

界面使用 Bubble Tea、Lip Gloss 和 ANSI 颜色构建，默认采用深色背景、桃色强调色与圆角面板。所有页面都可以只用键盘操作，并在窄终端中自动调整布局。

## 特性

- **完整的终端界面**：启动、版本、下载、设置、工具、关于、日志七个页面，支持响应式布局。
- **真实安装流程**：读取 Mojang 版本清单，下载客户端、依赖库、natives 和资源文件，校验 SHA-1。
- **版本继承**：解析 `inheritsFrom`，合并父版本与子版本的库、参数和主类。
- **加载器支持**：Fabric、Quilt、Legacy Fabric、LiteLoader、Forge、NeoForge、OptiFine、Cleanroom 与 LabyMod；Forge/NeoForge/Cleanroom 使用官方安装器，LiteLoader/OptiFine 使用嵌入或生成的版本配置安装，LabyMod 使用官方自定义清单与依赖库生成可启动版本。
- **整合包导入**：导入 Modrinth `.mrpack` 或 CurseForge 整合包，创建独立实例目录，下载模组与覆盖文件，并复用已安装的基础版本。CurseForge 下载需要在设置中填写 API Key。
- **在线整合包**：搜索 Modrinth 整合包目录，选择项目后自动获取最新稳定版本、下载 `.mrpack` 并创建独立实例。
- **收藏与筛选**：在线整合包可按游戏版本与加载器筛选；搜索结果可收藏，随时从收藏列表直接安装。
- **主题与代理**：内置 OpenCode 暗色、午夜蓝、浅色三套主题；支持 HTTP / HTTPS / SOCKS5 代理。
- **中英双语界面**：导航、页面标题、动态副标题、设置项、工具列表与命令面板支持中文 / English 即时切换。
- **实例图标**：为实例设置最多 4 个字符的图标，显示在版本列表与详情中。
- **CurseForge 在线浏览**：使用设置中的 API Key 搜索整合包，下载后直接导入独立实例。
- **皮肤 / 披风**：查询当前账户的公开皮肤与披风纹理，显示模型类型并可在浏览器中打开；离线 UUID 通常没有公开纹理。
- **新闻 / 更新日志**：读取发布源并展示最近的版本说明，可直接打开发布页面。
- **启动页新闻**：启动页读取最近发布记录并显示在侧栏，单击新闻条目打开完整更新日志。
- **网络测速**：测试 BMCLAPI、Mojang、Modrinth、GitHub 与 Adoptium 的延迟和吞吐，显示 MB/s。
- **在线资源**：搜索 Modrinth 资源包、光影、模组和数据包，按当前实例的 Minecraft 版本与加载器匹配后下载到 `resourcepacks/`、`shaderpacks/`、`mods/`，数据包会先选择存档再安装到该存档的 `datapacks/`。
- **世界下载**：填入世界 zip 链接，下载后解压到当前实例的 `saves/`，并自动避免重名。
- **投影管理**：列出当前实例 `schematics/` 下的投影文件，支持打开目录和删除。
- **Microsoft 登录**：设备代码流登录 Microsoft 账户，自动完成 Xbox Live、XSTS 与 Minecraft Services 认证；令牌过期时自动刷新。
- **工具与关于页面**：集中提供整合包导入、在线资源、存档管理、目录打开、Java 检测、刷新、日志清理和版本信息。
- **存档管理**：列出当前实例存档，支持打开目录、备份为 zip、导入 zip 和删除。
- **本地资源管理**：列出当前实例的模组、资源包与光影，支持启用/停用（`.disabled`）和删除。
- **截图管理**：列出当前实例的 Minecraft 截图，支持打开目录和删除。
- **服务器列表**：读写 `servers.dat`，支持添加、编辑、删除多人服务器。
- **实例导出/导入**：把版本或整合包连同模组、配置、存档与截图打包为 zip；导入时创建独立实例并补全缺失文件。
- **环境自检与更新检查**：检查游戏目录、Java、版本、资源与下载源；查询最新发布版本并在有更新时打开下载页面。设置页也提供检查更新与问题反馈入口。
- **单实例启动覆盖**：为实例单独设置名称、主页链接、最小/最大内存、JVM/游戏参数、Java 路径和启动后自动进入的服务器。
- **模组信息**：读取 Fabric、Quilt、Forge、NeoForge 与旧版模组元数据，显示版本、加载器、启用状态和内嵌 jar。
- **新建实例向导**：自定义实例名称、目录、Minecraft 版本、加载器、加载器版本与 Java 路径，一步安装到独立实例。
- **离线账户**：添加、切换和删除离线账户，自动生成与 Minecraft 一致的离线 UUID。
- **Java 管理**：扫描常见安装位置与 `JAVA_HOME`，检查版本兼容性；缺少 Java 时可从 Adoptium 下载 JRE。
- **双下载源**：BMCLAPI 镜像与 Mojang 官方源，可在设置中即时切换。
- **启动参数**：自定义最小/最大内存、窗口尺寸、全屏、JVM 参数和游戏参数。
- **日志系统**：启动器日志与游戏 stdout/stderr 实时进入日志页，并同步写入日志文件。
- **鼠标操作**：导航、列表、按钮与弹窗全部支持单击、双击与滚轮；键盘仅用于文本输入框。
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

## 鼠标操作

界面默认使用鼠标操作，键盘只用于文本输入框。

### 全局

| 操作 | 功能 |
| --- | --- |
| 单击左侧导航 | 切换页面 |
| 单击顶部 `[?]` | 打开帮助 |
| 单击顶部 `[×]` | 退出 |
| 滚轮 | 滚动列表或页面 |
| 右键列表项 | 更多操作（版本、下载） |
| 悬停 | 底部显示当前可执行操作，行会高亮 |
| 单击弹窗外部 | 关闭弹窗 |
| 单击弹窗按钮 | 确认 / 取消 / 收藏 |

### 启动页

- 单击版本行：选择版本。
- 单击账户行：管理账户（Microsoft 登录、离线账户、皮肤）。
- 单击绿色按钮：启动或停止游戏。
- 单击新闻条目：打开完整更新日志。

### 版本页

- 单击列表项选中，双击启动。
- 单击右侧操作行：删除版本、打开版本 / 模组 / 资源包 / 存档 / 截图 / 配置目录、实例设置、实例安装 / 修复、导出实例、新建实例。

### 下载页

- 单击页签切换原版与各类加载器。
- 单击版本选中，双击安装。
- 单击右侧操作行安装版本或选择加载器版本。
- 整合包导入、社区资源、在线整合包等入口在工具页与下载页底部提示中。

### 设置页

- 单击条目：编辑文本、切换开关、循环选项或执行操作（检查更新 / 问题反馈）。
- 文本输入框仍需要键盘输入。

在账户弹窗中单击「Microsoft 登录」即可开始设备代码登录。首次使用需要准备一个 Azure 应用：

1. 在 Azure Portal 的「应用注册」中新建应用，账户类型选择支持个人 Microsoft 账户。
2. 在「身份验证」中启用「允许公共客户端流」。
3. 复制「应用程序（客户端）ID」，填入设置页的「微软客户端 ID」。
4. 打开账户弹窗，单击「Microsoft 登录」，按提示打开浏览器并输入设备代码。

登录成功后，访问令牌、刷新令牌和过期时间保存在本机 `config.json` 中；启动器会在令牌过期前自动刷新。若刷新失败，需要重新执行 Microsoft 登录。

### 整合包

在工具页单击「导入整合包」，输入本地 `.mrpack` 或 CurseForge 压缩包路径即可导入。MCTL 会：

1. 读取 `modrinth.index.json` 或 CurseForge `manifest.json`。
2. 在 `instances/<名称>-<短哈希>` 下创建独立实例。
3. 安装声明的 Minecraft 版本与加载器。
4. 下载模组和资源文件，并解压 `overrides/` 与 `client-overrides/`。
5. 生成 `pack-*` 版本，出现在版本页和启动页中。

CurseForge 整合包使用其 API 解析文件下载地址，需要先在设置页填写「CurseForge Key」。Modrinth 格式不需要额外密钥。

### 工具页

单击条目即可执行整合包、在线资源、收藏、存档、本地资源、模组信息、截图、服务器、实例、自检、更新、新闻、测速等操作；滚轮滚动列表。

### 关于页 / 日志页

- 滚轮滚动内容；日志滚到底部会自动恢复跟随最新输出。

## 配置与日志

配置目录：

- Windows：`%APPDATA%\MCTL`
- macOS：`~/Library/Application Support/MCTL`
- Linux：`$XDG_CONFIG_HOME/MCTL` 或 `~/.config/MCTL`

主要文件：

- `config.json`：游戏目录、Java 路径、内存、窗口、下载源、Microsoft 客户端 ID、账户与令牌等。Microsoft 刷新令牌只保存在本机配置文件中，请勿分享该文件。
- `versions/<版本>/mctl-settings.json`：实例级名称、主页链接、内存、JVM/游戏参数、自动进入的服务器与单独 Java 路径。
- `logs/mctl-YYYYMMDD.log`：启动器与游戏日志。

配置采用原子写入；如果 `config.json` 损坏，会保留为 `config.json.broken` 并回退到默认配置。

## 开发

```bash
go test ./...
go vet ./...
go build ./...
```

加载器端点实时检查（需要能访问对应的官方服务）：

```bash
go test -tags live -v ./internal/minecraft -run TestLiveLoaderMetadata -count=1
```

检查覆盖 Forge、NeoForge、Cleanroom、LabyMod、Quilt、Legacy Fabric、LiteLoader、OptiFine、Modrinth 在线整合包与资源接口，以及启动器发布源与更新日志。若在仓库 Secrets 中配置 `CURSEFORGE_API_KEY`，还会验证 CurseForge 搜索接口。

真实加载器安装（会下载安装器与依赖，需数十 MB 到数百 MB 流量，默认跳过）：

```bash
MCTL_REAL_INSTALL=1 go test -tags live -v ./internal/minecraft -run TestLiveForgeInstall -count=1
```

该测试会准备原版 profile 与客户端 jar，下载官方 Forge 安装器并实际执行 `java -jar ... --installClient`，然后解析生成的版本配置。需要可访问 Forge Maven 与 Java 17+。

GitHub Actions 中也提供了手动触发的 `Live loader check` 工作流。

项目结构：

```text
.
├── main.go                     CLI 入口与演示模式
├── internal/
│   ├── config/                 配置持久化与账户
│   ├── logbuf/                 内存日志环形缓冲区与文件镜像
│   ├── minecraft/              清单、版本解析、安装、Java、启动、认证与整合包
│   └── tui/                    Bubble Tea 界面、页面、弹窗与运行时
└── .github/workflows/          CI 与 Release 工作流
```

## 发布

推送形如 `v1.0.36` 的标签会触发 Release 工作流：

```bash
git tag v1.0.36
git push origin v1.0.36
```

工作流会为 Windows、Linux、macOS 的 amd64 与 arm64 构建二进制，生成压缩包和 `checksums.txt`，然后自动创建 GitHub Release。也可以在 Actions 页面手动运行 `Release` 工作流并填写版本号。

## 许可证

[MIT](LICENSE)
