// 博客文章存储于文件系统（PostsDir），运行时读取并解析 YAML frontmatter。
// 设计参照 fuwari 的 src/content/config.ts 集合定义。
package handlers

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"fuwari-server/config"
	"fuwari-server/models"
	"fuwari-server/utils"

	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"
)

// Frontmatter 与 fuwari 集合 schema 对齐
type Frontmatter struct {
	Title       string   `yaml:"title" json:"title"`
	Published   string   `yaml:"published" json:"published"`
	Updated     string   `yaml:"updated,omitempty" json:"updated"`
	Draft       bool     `yaml:"draft" json:"draft"`
	Description string   `yaml:"description" json:"description"`
	Image       string   `yaml:"image" json:"image"`
	Tags        []string `yaml:"tags" json:"tags"`
	Category    string   `yaml:"category" json:"category"`
	Lang        string   `yaml:"lang,omitempty" json:"lang"`
}

// Post 列表项
type Post struct {
	Slug         string   `json:"slug"`
	Title        string   `json:"title"`
	Published    string   `json:"published"`
	Updated      string   `json:"updated,omitempty"`
	Description  string   `json:"description"`
	Image        string   `json:"image"`
	Tags         []string `json:"tags"`
	Category     string   `json:"category"`
	Draft        bool     `json:"draft"`
	CommentCount int64    `json:"comment_count"`
}

// PostDetail 含正文
type PostDetail struct {
	Post
	Body string `json:"body"`
}

// parseFrontmatter 从文件内容解析 frontmatter 与正文
func parseFrontmatter(data []byte) (Frontmatter, string, error) {
	var fm Frontmatter
	s := string(data)
	if !strings.HasPrefix(s, "---") {
		return fm, s, fmt.Errorf("missing frontmatter")
	}
	rest := s[3:]
	// 兼容 ---\r\n 与 ---\n
	idx := strings.Index(rest, "\n---")
	if idx < 0 {
		return fm, s, fmt.Errorf("frontmatter not closed")
	}
	yamlPart := rest[:idx]
	body := rest[idx+4:]
	if err := yaml.Unmarshal([]byte(yamlPart), &fm); err != nil {
		return fm, s, fmt.Errorf("invalid frontmatter: %w", err)
	}
	return fm, strings.TrimLeft(body, "\r\n"), nil
}

// slugFromPath 由文件路径计算 slug：
//   - foo.md        -> foo
//   - guide/index.md -> guide
//   - guide/bar.md  -> guide/bar
func slugFromPath(rel string) string {
	rel = strings.ReplaceAll(rel, "\\", "/")
	dir, file := filepath.Split(rel)
	base := strings.TrimSuffix(file, ".md")
	if base == "index" {
		dir = strings.TrimSuffix(dir, "/")
		if dir == "" {
			return base
		}
		return dir
	}
	if dir == "" {
		return base
	}
	return dir + "/" + base
}

// listPosts 扫描内容目录返回文章列表（默认倒序，排除草稿）
func listPosts() ([]Post, error) {
	var posts []Post
	root := config.PostsDir
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(strings.ToLower(d.Name()), ".md") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		fm, _, perr := parseFrontmatter(data)
		if perr != nil {
			return nil // 跳过无法解析的文件
		}
		if fm.Draft {
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		slug := slugFromPath(rel)
		posts = append(posts, Post{
			Slug:         slug,
			Title:        fm.Title,
			Published:    fm.Published,
			Updated:      fm.Updated,
			Description:  fm.Description,
			Image:        fm.Image,
			Tags:         fm.Tags,
			Category:     fm.Category,
			Draft:        fm.Draft,
			CommentCount: models.CountCommentsByTarget("post", slug),
		})
		return nil
	})
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	// 按发布时间倒序
	sort.Slice(posts, func(i, j int) bool {
		return posts[i].Published > posts[j].Published
	})
	return posts, nil
}

// resolvePostFile 根据 slug 定位文件路径（防路径穿越）
func resolvePostFile(slug string) (string, error) {
	slug = strings.Trim(slug, "/")
	if strings.Contains(slug, "..") {
		return "", fmt.Errorf("invalid slug")
	}
	root := config.PostsDir
	// 优先 slug/index.md，其次 slug.md
	candidates := []string{
		filepath.Join(root, filepath.FromSlash(slug), "index.md"),
		filepath.Join(root, filepath.FromSlash(slug+".md")),
	}
	for _, p := range candidates {
		abs, _ := filepath.Abs(p)
		rootAbs, _ := filepath.Abs(root)
		if !strings.HasPrefix(abs, rootAbs) {
			continue
		}
		if _, err := os.Stat(abs); err == nil {
			return abs, nil
		}
	}
	return "", fmt.Errorf("post not found")
}

// ListPosts GET /api/posts
func ListPosts(c *gin.Context) {
	posts, err := listPosts()
	if err != nil {
		utils.InternalError(c, "读取文章列表失败")
		return
	}
	utils.Success(c, gin.H{"list": posts, "total": len(posts)})
}

// GetPost GET /api/posts/:slug
func GetPost(c *gin.Context) {
	slug := c.Param("slug")
	path, err := resolvePostFile(slug)
	if err != nil {
		utils.NotFound(c, "文章不存在")
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		utils.InternalError(c, "读取文章失败")
		return
	}
	fm, body, perr := parseFrontmatter(data)
	if perr != nil {
		utils.InternalError(c, "解析文章失败")
		return
	}
	rel, _ := filepath.Rel(config.PostsDir, path)
	utils.Success(c, PostDetail{
		Post: Post{
			Slug:         slugFromPath(rel),
			Title:        fm.Title,
			Published:    fm.Published,
			Updated:      fm.Updated,
			Description:  fm.Description,
			Image:        fm.Image,
			Tags:         fm.Tags,
			Category:     fm.Category,
			Draft:        fm.Draft,
			CommentCount: models.CountCommentsByTarget("post", slugFromPath(rel)),
		},
		Body: body,
	})
}

// GetPostRaw GET /api/posts/:slug/raw 返回原始 Markdown（编辑器加载用）
func GetPostRaw(c *gin.Context) {
	slug := c.Param("slug")
	path, err := resolvePostFile(slug)
	if err != nil {
		utils.NotFound(c, "文章不存在")
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		utils.InternalError(c, "读取文章失败")
		return
	}
	c.Data(200, "text/markdown; charset=utf-8", data)
}

// createOrUpdatePost 写入文章文件（创建或更新）
func writePost(c *gin.Context, slug string) {
	var req struct {
		Title       string   `json:"title" binding:"required"`
		Description string   `json:"description"`
		Body        string   `json:"body"`
		Tags        []string `json:"tags"`
		Category    string   `json:"category"`
		Draft       bool     `json:"draft"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "请填写文章标题")
		return
	}

	if slug == "" {
		// 新建：由标题生成 slug
		slug = utils.Slugify(req.Title, fmt.Sprintf("post-%d", time.Now().Unix()))
	}
	if strings.Contains(slug, "..") || strings.ContainsAny(slug, "\x00") {
		utils.BadRequest(c, "无效的文章标识")
		return
	}

	published := time.Now().Format("2006-01-02")
	// 若已存在，保留原 published
	if existing, err := resolvePostFile(slug); err == nil {
		if data, e := os.ReadFile(existing); e == nil {
			if fm, _, pe := parseFrontmatter(data); pe == nil && fm.Published != "" {
				published = fm.Published
			}
		}
	}

	// 组装 frontmatter
	var fmBuf strings.Builder
	fmBuf.WriteString("---\n")
	fmBuf.WriteString(fmt.Sprintf("title: %s\n", yamlQuote(req.Title)))
	fmBuf.WriteString(fmt.Sprintf("published: %s\n", published))
	if req.Description != "" {
		fmBuf.WriteString(fmt.Sprintf("description: %s\n", yamlQuote(req.Description)))
	}
	if len(req.Tags) > 0 {
		fmBuf.WriteString("tags: [")
		for i, t := range req.Tags {
			if i > 0 {
				fmBuf.WriteString(", ")
			}
			fmBuf.WriteString(yamlQuote(t))
		}
		fmBuf.WriteString("]\n")
	}
	if req.Category != "" {
		fmBuf.WriteString(fmt.Sprintf("category: %s\n", yamlQuote(req.Category)))
	}
	if req.Draft {
		fmBuf.WriteString("draft: true\n")
	}
	fmBuf.WriteString("---\n\n")
	fmBuf.WriteString(strings.TrimSpace(req.Body))
	fmBuf.WriteString("\n")

	// 写入文件
	target := filepath.Join(config.PostsDir, filepath.FromSlash(slug)+".md")
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		utils.InternalError(c, "写入文章失败")
		return
	}
	if err := os.WriteFile(target, []byte(fmBuf.String()), 0644); err != nil {
		utils.InternalError(c, "写入文章失败")
		return
	}

	utils.Success(c, gin.H{"slug": slug, "path": target})
}

// yamlQuote 为 YAML 标量加双引号并转义
func yamlQuote(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	return "\"" + s + "\""
}

// CreatePost POST /api/posts
func CreatePost(c *gin.Context) {
	writePost(c, "")
}

// UpdatePost PUT /api/posts/:slug
func UpdatePost(c *gin.Context) {
	slug := c.Param("slug")
	if _, err := resolvePostFile(slug); err != nil {
		utils.NotFound(c, "文章不存在")
		return
	}
	writePost(c, slug)
}

// DeletePost DELETE /api/posts/:slug
func DeletePost(c *gin.Context) {
	slug := c.Param("slug")
	path, err := resolvePostFile(slug)
	if err != nil {
		utils.NotFound(c, "文章不存在")
		return
	}
	if err := os.Remove(path); err != nil {
		utils.InternalError(c, "删除文章失败")
		return
	}
	utils.Success(c, gin.H{"message": "文章已删除"})
}
