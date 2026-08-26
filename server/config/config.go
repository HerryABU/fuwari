// Package config 集中管理 Fuwari 服务端配置。
// 首次运行自动生成 .env 模板，之后从 .env 与进程环境变量加载配置。
package config

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
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

# ==================== 管理令牌 ====================
# 用于内容写操作（创建/保存/删除文章）与评论删除的简单令牌。
# 评论发表对读者开放（带限流与内容净化），仅删除评论需要此令牌。
# 留空 = 首次启动自动生成随机令牌并打印到启动日志（请自行记录并回填此处）。
ADMIN_TOKEN=

# ==================== 评论 ====================
# 单页评论条数默认值
COMMENT_PAGE_SIZE=20
# 单条评论最大长度（字符）
COMMENT_MAX_LENGTH=4000
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

	AdminToken = strings.TrimSpace(os.Getenv("ADMIN_TOKEN"))
	if AdminToken == "" {
		AdminToken = randomToken(24)
		log.Println("========================================")
		log.Printf("  未配置 ADMIN_TOKEN，已生成随机令牌（仅显示一次）: %s", AdminToken)
		log.Println("  请将其写入 .env 的 ADMIN_TOKEN 后重启，否则下次启动令牌会变化")
		log.Println("========================================")
	}

	CommentPageSize = envIntOr("COMMENT_PAGE_SIZE", 20)
	CommentMaxLength = envIntOr("COMMENT_MAX_LENGTH", 4000)
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

// randomToken 生成 n 字节的十六进制随机令牌
func randomToken(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "change-me-" + strconv.FormatInt(int64(os.Getpid()), 10)
	}
	return hex.EncodeToString(b)
}
