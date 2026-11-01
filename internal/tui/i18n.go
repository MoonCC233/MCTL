package tui

import "fmt"

var language = "zh"

var english = map[string]string{
	// Navigation and chrome.
	"启动":                     "Launch",
	"版本":                     "Versions",
	"下载":                     "Download",
	"设置":                     "Settings",
	"工具":                     "Tools",
	"关于":                     "About",
	"日志":                     "Logs",
	"工作区":                    "Workspace",
	"账户":                     "Account",
	"当前版本":                   "Current version",
	"未安装":                    "Not installed",
	"离线账户":                   "Offline account",
	"Microsoft 账户":           "Microsoft account",
	"离线":                     "Offline",
	"游戏中":                    "Running",
	"已停止":                    "Stopped",
	"选择版本并开始游戏":              "Pick a version and start playing",
	"管理已安装的版本与实例":            "Manage installed versions and instances",
	"安装、更新与导入整合包":            "Install, update and import modpacks",
	"修改会立即保存":                "Changes are saved immediately",
	"偏好设置":                   "Preferences",
	"维护、目录与常用操作":             "Maintenance, folders and tools",
	"MCTL · Minecraft 终端启动器": "MCTL · Minecraft terminal launcher",
	"Minecraft 终端启动器":        "Minecraft terminal launcher",
	"关于 MCTL":                "About MCTL",
	"运行日志":                   "Runtime logs",
	"启动器与游戏实时输出":             "Live launcher and game output",
	"提示":                     "Tip",
	"帮助":                     "Help",
	"命令面板":                   "Command palette",
	"操作":                     "Actions",
	"导航":                     "Navigation",
	"选择":                     "Select",
	"确定":                     "Confirm",
	"取消":                     "Cancel",
	"关闭":                     "Close",
	"返回":                     "Back",
	"执行":                     "Run",
	"编辑":                     "Edit",
	"删除":                     "Delete",
	"打开":                     "Open",
	"刷新":                     "Refresh",
	"重试":                     "Retry",
	"添加":                     "Add",
	"保存":                     "Save",
	"详情":                     "Details",
	"名称":                     "Name",
	"类型":                     "Type",
	"大小":                     "Size",
	"模组":                     "Mods",
	"发布":                     "Release",
	"主页":                     "Homepage",
	"路径":                     "Path",
	"状态":                     "Status",
	"版本路径":                   "Version path",
	"实例路径":                   "Instance path",
	"描述":                     "Description",
	"加载器":                    "Loader",
	"文件":                     "File",
	"已启用":                    "Enabled",
	"已禁用":                    "Disabled",
	"未设置":                    "Not set",
	"图标":                     "Icon",
	"实例图标":                   "Instance icon",
	"未配置":                    "Not set",
	"开启":                     "On",
	"跟随全局":                   "Use global",
	"自动":                     "Auto",
	"直连":                     "Direct",
	"清空":                     "Clear",
	"跟随":                     "Follow",
	"滚动":                     "Scroll",

	// Settings.
	"游戏目录":           "Game directory",
	"下载源":            "Download source",
	"原版":             "Vanilla",
	"主题":             "Theme",
	"语言":             "Language",
	"Java 路径":        "Java path",
	"自动下载 Java":      "Auto-download Java",
	"检测 Java":        "Detect Java",
	"最大内存":           "Max memory",
	"最小内存":           "Min memory",
	"窗口宽度":           "Window width",
	"窗口高度":           "Window height",
	"全屏启动":           "Start fullscreen",
	"JVM 参数":         "JVM arguments",
	"游戏参数":           "Game arguments",
	"微软客户端 ID":       "Microsoft client ID",
	"Microsoft 登录":   "Microsoft sign-in",
	"CurseForge Key": "CurseForge key",
	"HTTP 代理":        "HTTP proxy",
	"打开游戏目录":         "Open game folder",
	"打开配置目录":         "Open config folder",
	"游戏":             "Game",
	"界面":             "Interface",
	"网络":             "Network",
	"整合包":            "Modpacks",
	"目录":             "Folders",
	"修改会立即保存 · ":     "Changes are saved immediately · ",
	"Minecraft 数据目录，版本、模组和存档都保存在这里。":     "Minecraft data folder for versions, mods and worlds.",
	"BMCLAPI 镜像通常在国内更快，Mojang 官方源为原始地址。": "BMCLAPI is usually faster in China; Mojang is the upstream source.",
	"切换终端配色：OpenCode 暗色、午夜蓝或浅色。":         "Switch the terminal palette: OpenCode dark, midnight or light.",
	"中文 / English": "Chinese / English",
	"留空则自动检测。也可以填写 java 可执行文件或 JAVA_HOME 目录。":              "Leave empty to auto-detect, or point to java or JAVA_HOME.",
	"缺少兼容的 Java 时，从 Adoptium 自动下载所需 JRE。":                  "Download the required JRE from Adoptium when Java is missing.",
	"重新扫描系统与常见安装目录中的 Java 运行时。":                            "Rescan the system and common install locations for Java.",
	"游戏可使用的最大堆内存（MB）。大型整合包建议 4096 以上。":                     "Maximum heap in MB. Large modpacks usually need 4096 or more.",
	"JVM 初始堆内存（MB），通常保持 512 即可。":                           "Initial heap in MB. 512 is usually enough.",
	"游戏窗口宽度。全屏模式下此值会被忽略。":                                  "Game window width; ignored in fullscreen.",
	"游戏窗口高度。全屏模式下此值会被忽略。":                                  "Game window height; ignored in fullscreen.",
	"启动游戏时直接进入全屏模式。":                                       "Start the game in fullscreen.",
	"追加到 Java 命令行的参数，例如 -XX:+UseG1GC。":                     "Extra JVM flags, for example -XX:+UseG1GC.",
	"追加到 Minecraft 主类的参数，例如 --demo。":                       "Extra game arguments, for example --demo.",
	"Azure 应用的客户端 ID，需要启用公共客户端设备代码流。":                      "Azure app client ID with public client device-code flow enabled.",
	"使用设备代码在浏览器中登录 Microsoft 账户。":                          "Sign in to a Microsoft account with a device code.",
	"导入 CurseForge 整合包所需的 API Key，可在 CurseForge 开发者控制台申请。": "API key required for CurseForge modpacks.",
	"例如 http://127.0.0.1:7890；留空表示直连并使用环境变量代理。":            "For example http://127.0.0.1:7890; empty means direct or environment proxy.",
	"在系统文件管理器中打开当前游戏目录。":                                   "Open the game folder in the file manager.",
	"打开 config.json 与日志文件所在目录。":                            "Open the folder containing config.json and logs.",

	// Tools.
	"导入整合包":          "Import modpack",
	"在线整合包":          "Online modpacks",
	"社区资源":           "Community resources",
	"整合包来源":          "Modpack source",
	"需要 API Key":     "API key required",
	"CurseForge 整合包": "CurseForge modpacks",
	"收藏":             "Favorites",
	"在线资源":           "Online resources",
	"存档管理":           "Worlds",
	"本地资源":           "Local resources",
	"模组信息":           "Mod info",
	"截图管理":           "Screenshots",
	"服务器列表":          "Servers",
	"皮肤 / 披风":        "Skin / cape",
	"皮肤":             "Skin",
	"披风":             "Cape",
	"模型":             "Model",
	"无":              "None",
	"导出实例":           "Export instance",
	"导入实例":           "Import instance",
	"新建实例":           "New instance",
	"实例设置":           "Instance settings",
	"环境自检":           "Self-test",
	"检查更新":           "Check for updates",
	"新闻 / 更新日志":      "News / changelog",
	"网络测速":           "Speed test",
	"刷新版本与清单":        "Refresh versions",
	"打开日志":           "Open logs",
	"清空日志":           "Clear logs",
	"前往下载页":          "Go to download",
	"导入本地 Modrinth .mrpack 或 CurseForge 整合包到独立实例。": "Import a local Modrinth or CurseForge modpack into an isolated instance.",
	"按游戏版本与加载器筛选，搜索并安装 Modrinth 整合包。":              "Filter by game version and loader, then install a Modrinth modpack.",
	"使用 CurseForge API Key 搜索并安装整合包。":              "Search and install CurseForge modpacks with an API key.",
	"查看已收藏的整合包、资源包、光影和模组。":                         "Browse saved modpacks, resource packs, shaders and mods.",
	"搜索 Modrinth 资源包、光影、模组或数据包并安装到当前实例。":           "Search Modrinth for resource packs, shaders, mods or data packs.",
	"查看、打开、备份、导入或删除当前实例的存档。":                       "List, open, back up, import or delete worlds.",
	"启用、停用或删除当前实例的模组、资源包和光影。":                      "Enable, disable or delete mods, packs and shaders.",
	"读取模组元数据、加载器、启用状态与内嵌 jar。":                     "Read mod metadata, loaders, state and nested jars.",
	"查看或删除当前实例的 Minecraft 截图。":                     "Browse or delete screenshots.",
	"添加、编辑或删除当前实例保存的多人服务器。":                        "Add, edit or delete saved servers.",
	"查询当前账户的公开皮肤与披风纹理。":                            "Look up public skin and cape textures for the current account.",
	"把当前版本或整合包连同模组、存档与截图打包为 zip。":                  "Bundle a version or pack with mods, worlds and screenshots.",
	"从 MCTL 实例 zip 创建独立实例并补全缺失文件。":                 "Create an instance from an MCTL zip and fill missing files.",
	"按名称、目录、版本、加载器与 Java 创建独立实例。":                  "Create an instance with a name, folder, version, loader and Java.",
	"为当前实例设置名称、主页、内存、JVM 参数与自动进入服务器。":              "Set name, homepage, memory, JVM args and auto-join server.",
	"检查游戏目录、Java、版本、资源与下载源。":                       "Check game folder, Java, version, assets and source.",
	"查询最新发布版本，有新版本时打开下载页面。":                        "Check the latest release and open its page.",
	"查看最近的版本发布说明。":                                 "Read recent release notes.",
	"测试下载源、Modrinth、GitHub 与 Adoptium 的延迟和吞吐。":     "Measure latency and throughput of common services.",
	"重新扫描本地版本并刷新下载版本清单。":                           "Rescan local versions and refresh the catalogue.",
	"查看启动器与游戏实时输出。":                                "View live launcher and game output.",
	"清空当前内存日志，不影响已写入的文件。":                          "Clear the in-memory log buffer.",
	"安装原版、Fabric、Quilt、Forge 或 NeoForge。":          "Install vanilla, Fabric, Quilt, Forge or NeoForge.",
	"查看版本、配置路径与项目信息。":                              "View version, paths and project info.",

	// About.
	"用键盘完成 Minecraft 的安装、版本管理、账户登录与启动。界面使用 Bubble Tea 与 Lip Gloss 构建，支持 Windows、Linux 与 macOS。": "Install, manage, sign in and launch Minecraft from the keyboard. Built with Bubble Tea and Lip Gloss for Windows, Linux and macOS.",
	"支持": "Supported",
	"项目": "Project",
	"仓库": "Repository",
	"许可": "License",
	"配置": "Config",
	"原版 / Fabric / Quilt / Legacy / LiteLoader / Forge / NeoForge / OptiFine / Cleanroom / LabyMod": "Vanilla / Fabric / Quilt / Legacy / LiteLoader / Forge / NeoForge / OptiFine / Cleanroom / LabyMod",
	"本地整合包导入、在线整合包与资源搜索":                                                                            "Local modpack import and online modpack/resource search",
	"离线账户与 Microsoft 设备代码登录":                                                                        "Offline accounts and Microsoft device-code sign-in",
	"Java 检测、Adoptium 自动下载与启动参数":                                                                    "Java detection, Adoptium download and launch options",
	"实时日志、命令面板、存档与本地资源管理":                                                                           "Live logs, command palette, worlds and local resources",
	"截图浏览与多人服务器列表编辑":                                                                                "Screenshot browser and server list editor",
	"实例 zip 导出与导入":                                                                                  "Instance zip export and import",
	"环境自检、更新检查与下载源连通性测试":                                                                            "Self-test, update check and source connectivity",
	"模组元数据与内嵌 jar 查看":                                                                               "Mod metadata and nested jar inspection",
	"新建实例向导（名称 / 目录 / 版本 / 加载器 / Java）":                                                             "New instance wizard (name / folder / version / loader / Java)",
	"在线整合包筛选与收藏列表":                                                                                  "Modpack filters and favorites",
	"三套终端主题与 HTTP 代理设置":                                                                             "Three palettes and HTTP proxy",
	"中英双语界面切换":                                                                                      "Chinese and English interface",
	"实例图标自定义":                                                                                       "Custom instance icons",
	"CurseForge 在线整合包浏览与安装":                                                                         "CurseForge online modpack browsing",
	"皮肤 / 披风纹理查询":                                                                                   "Skin and cape texture lookup",
	"数据包按存档安装":                                                                                      "Data packs installed into a chosen world",
	"世界 zip 下载与投影管理":                                                                                "World zip downloads and schematic management",
	"实例加载器安装与文件修复":                                                                                  "Instance loader install and file repair",
	"网络测速与延迟/吞吐展示":                                                                                  "Network speed test with latency and throughput",

	// Launch.
	"启动游戏": "Launch game",
	"内存":   "Memory",
	"停止游戏": "Stop game",
	"切换版本": "Change version",
	"切换账户": "Change account",
	"已选择":  "Selected",

	// Download / versions.
	"版本管理":          "Versions",
	"已安装":           "Installed",
	"游戏版本":          "Game version",
	"选择游戏版本":        "Choose a game version",
	"版本列表":          "Versions",
	"安装此版本":         "Install this version",
	"修复 / 重新安装":     "Repair / reinstall",
	"启动此版本":         "Launch this version",
	"删除版本":          "Delete version",
	"打开版本目录":        "Open version folder",
	"打开模组目录":        "Open mods folder",
	"打开资源包目录":       "Open resource packs folder",
	"打开存档目录":        "Open saves folder",
	"打开截图目录":        "Open screenshots folder",
	"按名称、加载器或版本号筛选": "Filter by name, loader or version",
	"选择版本":          "Select version",
	"加载器版本":         "Loader version",
	"Minecraft 版本":  "Minecraft version",
	"实例目录":          "Instance folder",
	"实例名称":          "Instance name",
	"自动进入服务器":       "Auto-join server",
	"单独 Java":       "Custom Java",
	"创建实例":          "Create instance",
	"开始安装并创建独立实例":   "Install and create an isolated instance",

	// Logs.
	"运行日志与游戏输出": "Launcher and game output",

	// Misc.
	"数据包":   "Data packs",
	"在线数据包": "Online data packs",
	"世界":    "Worlds",
	"在线世界":  "Online worlds",
	"投影管理":  "Schematics",
	"投影":    "Schematics",
	"查看、打开或删除当前实例的投影文件。": "Browse or delete schematic files in this instance.",
	"实例安装": "Instance install",
	"为当前实例安装加载器或补全游戏文件。":     "Install a loader or repair files for this instance.",
	"补全 / 修复文件":              "Repair files",
	"选择存档":                   "Choose world",
	"请先在存档管理中创建或导入一个存档":      "Create or import a world first",
	"已安装 %d 个版本":             "Installed %d versions",
	" · 筛选：%s":               " · filter: %s",
	"已安装 · %d":               "Installed · %d",
	"正在获取版本列表…":              "Loading versions…",
	"%s · %d 个版本 · 下载源 %s":   "%s · %d versions · source %s",
	"常用工具 · %d":              "Tools · %d",
	"%d 行 · %s · 日志同时写入配置目录": "%d lines · %s · logs are also written to the config folder",
	"跟随中": "Following",
	"已暂停": "Paused",
	"选择版本与账户，然后开始游戏":                 "Pick a version and account, then start",
	"还没有安装任何版本 · 前往「下载」获取 Minecraft": "No versions installed · get Minecraft from Download",
	"已安装 %d 个版本 · %s":                "Installed %d versions · %s",
	"没有可选项":                          "No options",
	"正在测速…":                          "Testing network…",
	"重新测速":                           "Run again",
	"正在检测…":                          "Checking…",
	"重新检测":                           "Run again",
	"新闻":                             "News",
	"当前实例没有模组":                       "No mods in this instance",
	"当前实例没有截图":                       "No screenshots in this instance",
	"当前实例没有保存的服务器":                   "No saved servers",
	"没有找到存档":                         "No worlds found",
	"当前实例没有本地资源":                     "No local resources",
}

// SetLanguage switches the interface language. Unknown values fall back to
// Chinese.
func SetLanguage(id string) {
	if id == "en" {
		language = "en"
		return
	}
	language = "zh"
}

// Language returns the active language identifier.
func Language() string { return language }

// NormalizeLanguage maps unknown identifiers onto Chinese.
func NormalizeLanguage(id string) string {
	if id == "en" {
		return "en"
	}
	return "zh"
}

func pageLabel(i int) string {
	if language == "en" {
		switch i {
		case 0:
			return "Launch"
		case 1:
			return "Versions"
		case 2:
			return "Download"
		case 3:
			return "Settings"
		case 4:
			return "Tools"
		case 5:
			return "About"
		case 6:
			return "Logs"
		}
	}
	if i >= 0 && i < len(pageNames) {
		return pageNames[i]
	}
	return ""
}

// tr translates a static interface string.
func tr(s string) string {
	if language != "en" {
		return s
	}
	if v, ok := english[s]; ok {
		return v
	}
	return s
}

// trf translates a format string before formatting it.
func trf(format string, args ...any) string {
	return fmt.Sprintf(tr(format), args...)
}

// LanguageName returns the display name for a language id.
func LanguageName(id string) string {
	if NormalizeLanguage(id) == "en" {
		return "English"
	}
	return "中文"
}
