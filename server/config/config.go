// Package config 集中管理 Fuwari 服务端配置。
// 首次运行自动生成 .env 模板，之后从 .env 与进程环境变量加载配置。
package config

import (
	"bufio"
	"log"
	"os"
	"strconv"
	"strings"
)

// defaultEnvTemplate 首次运行生成的 .env 模板
const defaultEnvTemplate = `# ==================== Fuwari Server 配置 ====================

# 服务端口
SERVER_PORT=9000

# ==================== 数据库（评论存储） ====================
# DB_DRIVER: sqlite (默认，零配置) | mysql (暂未启用，预留)
DB_DRIVER=sqlite
# SQLite 文件路径
DB_PATH=./data/fuwari.db

# ==================== 文件系统（博客内容存储） ====================
# 运行时内容目录：服务启动后所有读写都发生在这里
POSTS_DIR=./content/posts
# 种子目录（只读）：运行时内容目录为空时，从该目录复制初始文章
# 默认指向 fuwari 模板自带的示例文章
SRC_POSTS_DIR=./src/content/posts

# ==================== 管理员密码 ====================
# 管理员密码以 bcrypt 哈希存于数据库（settings 表），用于内容写操作
# （创建/保存/删除文章）、评论删除与修改密码。
# 首次启动时：
#   - 已设置 ADMIN_TOKEN  → 将其作为初始管理员密码写入数据库；
#   - 留空               → 自动生成随机密码并打印到启动日志（仅显示一次）。
# 之后一律以数据库中的密码为准，可通过：
#   - /editor 页面修改（知道当前密码）；
#   - 命令行重置：./fuwari-server -re pwd（忘记密码时）。
ADMIN_TOKEN=

# ==================== 评论 ====================
# 单页评论条数默认值
COMMENT_PAGE_SIZE=20
# 单条评论最大长度（字符）
COMMENT_MAX_LENGTH=4000

# ==================== 主题系统（运行时热加载，无需重编译） ====================
# 主题目录：每个子目录为一个主题，包含 theme.css / background.* / custom.js / manifest.json。
# 修改后刷新页面即可生效；URL ?theme=<name> 或 Cookie fuwari_theme 切换。
THEMES_DIR=./themes
# 默认主题（themes 下不存在同名目录时回退到内嵌默认样式）
DEFAULT_THEME=default

# ==================== 扩展（看板娘等，运行时热加载） ====================
# 扩展目录：每个子目录为一个扩展，其中的 *.js / *.css 会被注入到所有页面。
# 例：extensions/live2d/ 放置看板娘资源，无需重新编译即可生效。
EXTENSIONS_DIR=./extensions

# ==================== 管理后台 UI（运行时热加载，无需重编译） ====================
# admin/ui.css 与 admin/ui.js 由 /admin 页面动态注入；
# 修改样式/逻辑后刷新页面即生效。首次启动从内嵌默认资源复制。
ADMIN_DIR=./admin

# ==================== 网络 / IPv6 ====================
# 绑定 IPv4 地址（0.0.0.0 = 所有网卡）
BIND_IPV4=0.0.0.0
# 是否启用 IPv6 双栈监听（true/false，true 时监听 [::] 同时接受 IPv4/IPv6）
ENABLE_IPV6=false
# IPv6 绑定地址
BIND_IPV6=::
`

var (
	// 服务端口
	ServerPort string

	// 数据库
	DBDriver string
	DBPath   string

	// 文件系统内容
	PostsDir    string
	SrcPostsDir string

	// 管理令牌
	AdminToken string

	// 评论
	CommentPageSize  int
	CommentMaxLength int

	// 主题系统
	ThemesDir    string
	DefaultTheme string

	// 扩展目录
	ExtensionsDir string

	// 管理后台 UI 目录（运行时热加载：ui.css / ui.js，无需重编译）
	AdminDir string

	// 数据目录（站点设置等运行时配置）
	DataDir string

	// 网络 / IPv6
	BindIPv4   string
	EnableIPv6 bool
	BindIPv6   string
)

// Init 初始化配置：缺 .env 则生成模板，然后加载配置
func Init() {
	ensureEnvFile()
	loadEnvFile(".env")

	ServerPort = envOr("SERVER_PORT", "9000")

	DBDriver = envOr("DB_DRIVER", "sqlite")
	DBPath = envOr("DB_PATH", "./data/fuwari.db")

	PostsDir = envOr("POSTS_DIR", "./content/posts")
	SrcPostsDir = envOr("SRC_POSTS_DIR", "./src/content/posts")

	// ADMIN_TOKEN 仅作为「首次启动的初始管理员密码」种子（由 main.ensureAdminPassword
	// 写入数据库 bcrypt 哈希）。之后一律以数据库中的密码为准，可通过
	// /editor 页面修改，或 ./fuwari-server -re pwd 命令行重置。
	AdminToken = strings.TrimSpace(os.Getenv("ADMIN_TOKEN"))

	CommentPageSize = envIntOr("COMMENT_PAGE_SIZE", 20)
	CommentMaxLength = envIntOr("COMMENT_MAX_LENGTH", 4000)

	ThemesDir = envOr("THEMES_DIR", "./themes")
	DefaultTheme = envOr("DEFAULT_THEME", "default")

	ExtensionsDir = envOr("EXTENSIONS_DIR", "./extensions")

	AdminDir = envOr("ADMIN_DIR", "./admin")

	DataDir = envOr("DATA_DIR", "./data")

	BindIPv4 = envOr("BIND_IPV4", "0.0.0.0")
	EnableIPv6 = envOr("ENABLE_IPV6", "false") == "true"
	BindIPv6 = envOr("BIND_IPV6", "::")
}

// ensureEnvFile 不存在则根据模板生成 .env
func ensureEnvFile() {
	if _, err := os.Stat(".env"); err == nil {
		return
	}
	if err := os.WriteFile(".env", []byte(defaultEnvTemplate), 0644); err != nil {
		log.Printf("警告: 生成 .env 模板失败: %v", err)
		return
	}
	log.Println("已生成 .env 配置模板，可编辑后重启生效")
}

// loadEnvFile 解析 .env（KEY=VALUE，# 注释），不覆盖已有环境变量
func loadEnvFile(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		k = strings.TrimSpace(k)
		v = strings.TrimSpace(v)
		v = strings.Trim(v, `"'`)
		if _, exists := os.LookupEnv(k); !exists {
			os.Setenv(k, v)
		}
	}
}

// envOr 读取环境变量，缺省返回 fallback
func envOr(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

// envIntOr 读取整数环境变量，缺省/非法返回 fallback
func envIntOr(key string, fallback int) int {
	if v, err := strconv.Atoi(strings.TrimSpace(os.Getenv(key))); err == nil && v > 0 {
		return v
	}
	return fallback
}
