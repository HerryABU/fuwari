// fuwari-server 入口：
//  1. 数据库（SQLite）存储评论；
//  2. 文件系统存储博客内容（运行时内容目录 + 种子目录）；
//  3. 内嵌 Astro 前端构建产物，单二进制交付完整博客站点；
//  4. 集成 Cherry Markdown 编辑器（/editor）与文章页评论挂件。
package main

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"fuwari-server/config"
	"fuwari-server/handlers"
	"fuwari-server/models"
	"fuwari-server/security"
	"fuwari-server/version"

	"github.com/gin-gonic/gin"
)

func main() {
	// 确保工作目录为可执行文件所在目录（双击启动时工作目录可能不是程序目录）
	if exePath, err := os.Executable(); err == nil {
		if dir := filepath.Dir(exePath); dir != "" {
			_ = os.Chdir(dir)
		}
	}

	config.Init()

	// 命令行重置管理员密码模式：./fuwari-server -re pwd（忘记密码场景）
	if isResetPwdMode() {
		runResetPassword()
		return
	}

	// 确保内容目录存在，并从种子目录初始化
	if err := ensureContentDirs(); err != nil {
		log.Fatalf("内容目录初始化失败: %v", err)
	}

	// 确保主题/扩展运行时目录存在，并补充默认主题模板
	_ = os.MkdirAll(config.ThemesDir, 0755)
	_ = os.MkdirAll(config.ExtensionsDir, 0755)
	seedThemesAndExtensions()

	// 确保管理后台 UI 运行时目录存在，并从内嵌默认资源种子（ui.css / ui.js 热加载）
	_ = os.MkdirAll(config.AdminDir, 0755)
	seedAdminUI()

	// 初始化数据库
	if err := models.InitDatabase(); err != nil {
		log.Fatalf("数据库初始化失败: %v", err)
	}
	log.Println("数据库初始化完成")

	// 首次启动引导初始管理员密码（数据库无密码哈希时）
	ensureAdminPassword()

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	// 注入内嵌 assets FS（主题默认兜底资源）
	handlers.EmbeddedAssetsFS = assetsFS

	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		if strings.Contains(c.GetHeader("Accept"), "application/json") {
			c.JSON(http.StatusOK, handlers.CollectHealth())
			return
		}
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(healthHTML()))
	})
	r.GET("/api/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, handlers.CollectHealth())
	})

	// ==================== API 路由 ====================

	// 公开读取
	r.GET("/api/comments", handlers.GetComments)
	r.GET("/api/posts", handlers.ListPosts)
	r.GET("/api/posts/:slug", handlers.GetPost)
	r.GET("/api/posts/:slug/raw", handlers.GetPostRaw)
	r.GET("/api/themes", handlers.ListThemes)

	// 评论发表对读者开放（限流 + 内容净化，见 security.SanitizeMarkdown）
	r.POST("/api/comments", security.RateLimit(10, 60), handlers.CreateComment)

	// 主题切换（写入 Cookie，持久化）
	r.POST("/api/theme", handlers.SetTheme)

	// 管理令牌保护
	admin := r.Group("/api")
	admin.Use(security.AdminAuth())
	{
		admin.DELETE("/comments/:id", handlers.DeleteComment)
		admin.POST("/posts", handlers.CreatePost)
		admin.PUT("/posts/:slug", handlers.UpdatePost)
		admin.DELETE("/posts/:slug", handlers.DeletePost)
		// 修改管理员密码（知道当前密码场景，AdminAuth 校验请求头）
		admin.POST("/admin/password", security.RateLimit(5, 60), handlers.ChangeAdminPassword)
		// 编辑器图片上传
		admin.POST("/admin/upload", security.RateLimit(30, 60), handlers.UploadImage)
	}

	// 主题静态资源（运行时优先，嵌入默认兜底）
	r.GET("/themes/:name/*filepath", handlers.ServeThemeAsset)
	// 扩展静态资源（运行时热加载）
	r.GET("/extensions/:name/*filepath", handlers.ServeExtensionAsset)

	// 静态资源挂载：/static/* -> /_astro 等构建产物别名（见 NoRoute）

	// ==================== 前端静态文件服务 ====================

	frontendFS, err := fs.Sub(distFS, "dist")
	if err != nil {
		log.Fatalf("前端资源加载失败（dist 目录不存在，请先构建前端并同步到 server/dist）: %v", err)
	}

	// 初始化根级路径集合（反代前缀探测用）：后端段 + 嵌入前端根目录
	InitKnownRoots(frontendFS)

	// 解析前台构建产物样式表（后台 /admin 复用，保证前后台 UI 逐字节一致）
	loadFrontendStyles(frontendFS)

	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path

		// Cherry Markdown 静态资源（编辑器 + 评论挂件共用，含 Apache-2.0 LICENSE 随附）
		if strings.HasPrefix(path, "/assets/cherry/") {
			serveEmbeddedAsset(c, strings.TrimPrefix(path, "/assets/cherry/"), "cherry")
			return
		}
		// 评论挂件脚本/样式（服务端注入用）
		if path == "/assets/comments.js" {
			serveEmbeddedAsset(c, "comments.js", "")
			return
		}
		if path == "/assets/comments.css" {
			serveEmbeddedAsset(c, "comments.css", "")
			return
		}
		// 后台 UI 动态资源（运行时 admin/ 优先热加载，内嵌 admin-default 兜底）
		if path == "/admin/ui.css" || path == "/admin/ui.js" {
			serveAdminAsset(c, filepath.Base(path))
			return
		}
		// 动态文章查看器（SSG 下运行时文章前台渲染 + 列表补全）
		if path == "/assets/post-viewer.js" {
			serveEmbeddedAsset(c, "post-viewer.js", "")
			return
		}
		// 管理后台统一入口（/admin 主入口；/editor 兼容保留，二者渲染同一后台页）。
		// 注入前台构建样式 + 统一主题注入，后台与前台 UI 完全一致。
		if strings.HasPrefix(path, "/admin") || strings.HasPrefix(path, "/editor") {
			data, err := fs.ReadFile(assetsFS, "assets/admin.html")
			if err != nil {
				c.Status(http.StatusNotFound)
				return
			}
			s := string(data)
			if strings.Contains(s, "<!-- FRONTEND_STYLES -->") {
				var b strings.Builder
				for _, href := range frontendStyles {
					b.WriteString(`<link rel="stylesheet" href="` + href + `">` + "\n")
				}
				s = strings.Replace(s, "<!-- FRONTEND_STYLES -->", b.String(), 1)
			}
			// 前台 <html> 内联变量（--hue 等）复制到后台，保证 oklch 主题色与前台一致
			if frontendHtmlStyle != "" {
				s = strings.Replace(s, `<html lang="zh-CN">`,
					`<html lang="zh-CN" style="`+frontendHtmlStyle+`">`, 1)
			}
			injected := injectPageAssets([]byte(s), c)
			c.Header("Cache-Control", "no-cache")
			c.Data(http.StatusOK, "text/html; charset=utf-8", injected)
			return
		}

		// /api/* 未匹配 → 404（JSON），不落回前端
		if strings.HasPrefix(path, "/api/") {
			c.JSON(http.StatusNotFound, gin.H{
				"code":    4,
				"message": "接口不存在",
				"data":    nil,
			})
			return
		}

		// 路径穿越防护
		if strings.Contains(path, "..") {
			c.Status(http.StatusNotFound)
			return
		}

		// 运行时内容目录静态资源（编辑器创建的图文文章的相对路径附件）
		if filePath, ok := handlers.ResolveContentAsset(path); ok {
			c.Header("Cache-Control", "public, max-age=3600")
			c.File(filePath)
			return
		}

		// 尝试匹配嵌入的静态资源
		cleanPath := strings.TrimPrefix(path, "/")
		if cleanPath == "" {
			cleanPath = "index.html"
		}

		// HTML 页面：解析出实际文件并注入页面资产（评论挂件 + 主题 + 扩展）
		if resolved, ok := resolveFrontendFile(frontendFS, cleanPath); ok && strings.HasSuffix(resolved, ".html") {
			data, err := fs.ReadFile(frontendFS, resolved)
			if err == nil {
				injected := injectPageAssets(data, c)
				if strings.HasPrefix(resolved, "_astro/") {
					c.Header("Cache-Control", "public, max-age=31536000, immutable")
				} else {
					c.Header("Cache-Control", "public, max-age=3600")
				}
				c.Data(http.StatusOK, "text/html; charset=utf-8", injected)
				return
			}
		}

		if f, err := frontendFS.Open(cleanPath); err == nil {
			f.Close()
			if strings.HasPrefix(cleanPath, "_astro/") {
				// Astro 产物带内容 hash，可长期缓存
				c.Header("Cache-Control", "public, max-age=31536000, immutable")
			} else {
				c.Header("Cache-Control", "public, max-age=3600")
			}
			http.FileServer(http.FS(frontendFS)).ServeHTTP(c.Writer, c.Request)
			return
		}

		// 非资源请求 → 返回 index.html（由 swup 接管页面过渡）
		if indexContent, err := fs.ReadFile(frontendFS, "index.html"); err == nil {
			injected := injectPageAssets(indexContent, c)
			c.Header("Cache-Control", "no-cache")
			c.Header("Content-Type", "text/html; charset=utf-8")
			c.Data(http.StatusOK, "text/html; charset=utf-8", injected)
			return
		}

		c.Status(http.StatusNotFound)
	})

	// IPv4/IPv6 双栈监听（ENABLE_IPV6=true 时监听 [::] 同时接受 IPv4 与 IPv6）
	var listenAddr string
	if config.EnableIPv6 {
		listenAddr = fmt.Sprintf("[%s]:%s", config.BindIPv6, config.ServerPort)
	} else {
		listenAddr = fmt.Sprintf("%s:%s", config.BindIPv4, config.ServerPort)
	}
	log.Println("========================================")
	log.Printf("  Fuwari Server v%s", version.AppVersion)
	log.Printf("  监听地址: %s", listenAddr)
	log.Println("  博客内容目录:", config.PostsDir)
	log.Println("  主题目录:", config.ThemesDir)
	log.Printf("  编辑器入口: http://localhost:%s/editor", config.ServerPort)
	log.Println("========================================")

	// 反代挂载前缀兼容：/{name}/api/... → /api/...（递归重路由，帽子对用户保留不可见）
	// 无前缀的普通请求直接走 gin 路由，零开销。
	siteHandler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if newPath, prefix, ok := stripMountPrefix(req.URL.Path); ok && newPath != req.URL.Path {
			req2 := req.Clone(req.Context())
			req2.URL.Path = newPath
			req2.URL.RawPath = ""
			if prefix != "" {
				req2 = req2.WithContext(context.WithValue(req2.Context(), basePathKey{}, prefix))
			}
			r.ServeHTTP(w, req2)
			return
		}
		r.ServeHTTP(w, req)
	})

	if err := http.ListenAndServe(listenAddr, siteHandler); err != nil {
		log.Fatalf("服务器启动失败: %v", err)
	}
}

// frontendStyles 前台构建产物样式表（后台 /admin 复用，保证前后台视觉逐字节一致）
var frontendStyles []string

// frontendHtmlStyle 前台 <html> 内联 CSS 变量（--hue 等；oklch 颜色依赖 --hue）
var frontendHtmlStyle string

// loadFrontendStyles 从构建产物首页解析样式表链接（/_astro/*.css）与 html 内联变量
func loadFrontendStyles(fsys fs.FS) {
	data, err := fs.ReadFile(fsys, "index.html")
	if err != nil {
		return
	}
	re := regexp.MustCompile(`<link rel="stylesheet"[^>]*href="([^"]+)"`)
	for _, m := range re.FindAllSubmatch(data, -1) {
		href := string(m[1])
		if strings.HasPrefix(href, "/_astro/") && strings.HasSuffix(href, ".css") {
			frontendStyles = append(frontendStyles, href)
		}
	}
	if m := regexp.MustCompile(`<html[^>]*\sstyle="([^"]+)"`).FindSubmatch(data); len(m) == 2 {
		frontendHtmlStyle = string(m[1])
	}
}

// seedAdminUI 确保运行时 admin/ 目录含 ui.css 与 ui.js（首次从内嵌 admin-default 复制）。
// 之后用户可直接编辑运行时文件，刷新页面即热加载，无需重新编译。
func seedAdminUI() {
	for _, name := range []string{"ui.css", "ui.js"} {
		target := filepath.Join(config.AdminDir, name)
		if _, err := os.Stat(target); err == nil {
			continue // 已存在（用户可能已自定义），保留
		}
		data, err := fs.ReadFile(assetsFS, "assets/admin-default/"+name)
		if err != nil {
			continue
		}
		if werr := os.WriteFile(target, data, 0644); werr == nil {
			log.Printf("已生成后台 UI 默认资源: %s（修改后刷新即生效）", target)
		}
	}
}

// serveAdminAsset 提供运行时 admin/<file>（热加载，no-cache）；缺失时内嵌 admin-default 兜底。
func serveAdminAsset(c *gin.Context, file string) {
	// 运行时优先
	target := filepath.Join(config.AdminDir, file)
	if data, err := os.ReadFile(target); err == nil {
		c.Header("Cache-Control", "no-cache")
		c.Data(http.StatusOK, assetContentType(file), data)
		return
	}
	// 内嵌兜底
	if data, err := fs.ReadFile(assetsFS, "assets/admin-default/"+file); err == nil {
		c.Header("Cache-Control", "no-cache")
		c.Data(http.StatusOK, assetContentType(file), data)
		return
	}
	c.Status(http.StatusNotFound)
}

// seedThemesAndExtensions 从仓库根（exe 父目录）复制默认主题/扩展模板到运行时目录。
// 主题目录为空时，将 ../themes 下的示例主题复制进来（如 ocean）。
func seedThemesAndExtensions() {
	// 种子源：exe 父目录（仓库根）的 themes/ extensions/
	var repoRoot string
	if exePath, err := os.Executable(); err == nil {
		repoRoot = filepath.Dir(exePath)
	}

	seedTheme := func() {
		if repoRoot == "" {
			return
		}
		src := filepath.Join(repoRoot, "..", "themes")
		if info, err := os.Stat(src); err != nil || !info.IsDir() {
			return
		}
		// 仅当运行时主题目录为空时复制
		entries, err := os.ReadDir(config.ThemesDir)
		if err != nil || len(entries) > 0 {
			return
		}
		log.Printf("主题目录为空，从模板复制示例主题: %s -> %s", src, config.ThemesDir)
		_ = copyTree(src, config.ThemesDir)
	}

	seedExt := func() {
		if repoRoot == "" {
			return
		}
		src := filepath.Join(repoRoot, "..", "extensions")
		if info, err := os.Stat(src); err != nil || !info.IsDir() {
			return
		}
		entries, err := os.ReadDir(config.ExtensionsDir)
		if err != nil || len(entries) > 0 {
			return
		}
		log.Printf("扩展目录为空，从模板复制: %s -> %s", src, config.ExtensionsDir)
		_ = copyTree(src, config.ExtensionsDir)
	}

	seedTheme()
	seedExt()
}

// locateSrcPosts 定位种子内容目录。
// 由于 exe 启动时会 chdir 到自身所在目录（通常为 server/），
// 相对路径 ./src/content/posts 会失效，因此在多个候选基址中探测：
//  1. 配置值按当前工作目录解析；
//  2. 配置值按 exe 父目录解析（即仓库根，exe 在 server/ 时）；
//  3. 显式的 ../src/content/posts 兜底。
// 返回第一个存在且包含 .md 文件的目录；找不到返回空串。
func locateSrcPosts() string {
	candidates := []string{config.SrcPostsDir}
	if exePath, err := os.Executable(); err == nil {
		dir := filepath.Dir(exePath)
		candidates = append(candidates,
			filepath.Join(dir, "..", filepath.ToSlash(config.SrcPostsDir)),
			filepath.Join(dir, "..", "src", "content", "posts"),
		)
	}
	for _, c := range candidates {
		if c == "" {
			continue
		}
		abs, err := filepath.Abs(c)
		if err != nil {
			continue
		}
		if info, err := os.Stat(abs); err != nil || !info.IsDir() {
			continue
		}
		if mds, _ := filepath.Glob(filepath.Join(abs, "*.md")); len(mds) > 0 {
			return abs
		}
	}
	return ""
}

// ensureContentDirs 确保运行时内容目录存在；为空时从种子目录复制初始文章
func ensureContentDirs() error {
	if err := os.MkdirAll(config.PostsDir, 0755); err != nil {
		return err
	}

	// 检查运行时目录是否已有 .md 文件
	entries, err := os.ReadDir(config.PostsDir)
	if err != nil {
		return err
	}
	empty := true
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(strings.ToLower(e.Name()), ".md") {
			empty = false
			break
		}
		if e.IsDir() {
			// 子目录内可能有 index.md
			if sub, _ := filepath.Glob(filepath.Join(config.PostsDir, e.Name(), "*.md")); len(sub) > 0 {
				empty = false
				break
			}
		}
	}
	if !empty {
		return nil
	}

	// 从种子目录复制（若存在）
	srcPosts := locateSrcPosts()
	if srcPosts == "" {
		return nil
	}
	log.Printf("内容目录为空，从种子目录初始化: %s -> %s", srcPosts, config.PostsDir)
	return copyTree(srcPosts, config.PostsDir)
}

// copyTree 递归复制目录树
func copyTree(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0644)
	})
}

// serveEmbeddedAsset 从嵌入的 assets 目录提供静态资源。
// sub 为 assets/ 下的相对路径；root 指定子目录前缀（如 "cherry"），空表示 assets 根。
func serveEmbeddedAsset(c *gin.Context, sub, root string) {
	clean := filepath.ToSlash(filepath.Clean("/" + sub))
	full := strings.TrimPrefix(clean, "/")
	if root != "" {
		full = root + "/" + full
	}
	data, err := fs.ReadFile(assetsFS, "assets/"+full)
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}
	c.Header("Cache-Control", "public, max-age=86400")
	c.Data(http.StatusOK, assetContentType(full), data)
}

// resolveFrontendFile 将干净的 URL 路径解析为嵌入前端中的实际文件。
// Astro 使用 trailingSlash: always，页面为 /posts/xxx/index.html 或 /xxx/index.html 形式。
// 目录/无扩展名路径优先尝试 index.html（确保命中页面而非目录句柄）。
func resolveFrontendFile(fsys fs.FS, cleanPath string) (string, bool) {
	var candidates []string
	switch {
	case strings.HasSuffix(cleanPath, "/"):
		candidates = []string{cleanPath + "index.html", cleanPath}
	case !strings.Contains(filepath.Base(cleanPath), "."):
		candidates = []string{cleanPath + "/index.html", cleanPath + ".html", cleanPath}
	default:
		candidates = []string{cleanPath}
	}
	for _, c := range candidates {
		if f, err := fsys.Open(c); err == nil {
			f.Close()
			return c, true
		}
	}
	return "", false
}

// commentWidgetSnippet 评论挂件 + 动态文章查看器（服务端注入，不改前端源码）
var commentWidgetSnippet = `
<link rel="stylesheet" href="/assets/comments.css">
<script src="/assets/comments.js"></script>
<script src="/assets/post-viewer.js"></script>
`

// injectPageAssets 统一页面注入：评论挂件 + 主题（前后台一致）+ 扩展（看板娘等）
// + FUWARI_BASE（反代前缀自适应），最后统一把根绝对路径改写为带前缀路径。
// 对所有 HTML 页面注入，保证无刷新导航后资产持续存在。
func injectPageAssets(html []byte, c *gin.Context) []byte {
	basePath := detectBasePath(c)
	themeName := handlers.CurrentThemeName(c)
	themeCSS := handlers.ThemeHeadInjection(themeName)
	themeJS := handlers.ThemeBodyInjection(themeName)
	extInjection := handlers.BuildExtensionInjection()

	s := string(html)

	// FUWARI_BASE 全局：JS 内 API/资源路径动态拼接（反代前缀自适应，严禁硬编码）。
	// 必须注入 <head>，确保先于 body 中所有立即执行的脚本（编辑器/挂件 IIFE）就绪。
	// 注意：不能用 "window.FUWARI_BASE" 做防重复标记——editor.html 等脚本已引用该全局名。
	baseScript := fmt.Sprintf(`<script id="fuwari-base">window.FUWARI_BASE=%q;</script>`, basePath)

	// 主题 CSS 注入 <head>（data-fuwari-theme 标记防重复）
	var headInjection strings.Builder
	headInjection.WriteString(baseScript)
	if themeCSS != "" {
		headInjection.WriteString("\n")
		headInjection.WriteString(themeCSS)
	}
	if !strings.Contains(s, `id="fuwari-base"`) && headInjection.Len() > 0 {
		if idx := strings.LastIndex(s, "</head>"); idx >= 0 {
			s = s[:idx] + headInjection.String() + s[idx:]
		} else if idx := strings.LastIndex(s, "<head"); idx >= 0 {
			if bi := strings.Index(s[idx:], ">"); bi >= 0 {
				s = s[:idx+bi+1] + headInjection.String() + s[idx+bi+1:]
			}
		} else {
			// 无 head 结构（极端兜底）：插到最前，保证最先执行
			s = headInjection.String() + s
		}
	}

	// 主题 JS + 扩展 + 评论挂件 注入 </body> 前
	var bodyJS strings.Builder
	if themeJS != "" {
		bodyJS.WriteString(themeJS)
	}
	if extInjection != "" {
		if bodyJS.Len() > 0 {
			bodyJS.WriteString("\n")
		}
		bodyJS.WriteString(extInjection)
	}
	// 评论挂件 —— 已注入则跳过
	if !strings.Contains(s, "/assets/comments.js") {
		if bodyJS.Len() > 0 {
			bodyJS.WriteString("\n")
		}
		bodyJS.WriteString(commentWidgetSnippet)
	}
	if bodyJS.Len() > 0 {
		if idx := strings.LastIndex(s, "</body>"); idx >= 0 {
			s = s[:idx] + bodyJS.String() + s[idx:]
		} else {
			s += bodyJS.String()
		}
	}

	// 最后统一改写根绝对路径（页面原有引用 + 本次注入标签 + 内联脚本）
	return RewriteHTML([]byte(s), basePath)
}

// assetContentType 按扩展名返回 MIME 类型
func assetContentType(name string) string {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".js":
		return "application/javascript; charset=utf-8"
	case ".css":
		return "text/css; charset=utf-8"
	case ".svg":
		return "image/svg+xml"
	case ".woff":
		return "font/woff"
	case ".woff2":
		return "font/woff2"
	case ".ttf":
		return "font/ttf"
	case ".eot":
		return "application/vnd.ms-fontobject"
	case ".html":
		return "text/html; charset=utf-8"
	default:
		return "application/octet-stream"
	}
}

// healthHTML 浏览器访问 /health 时的简单状态页
func healthHTML() string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="zh-CN">
<head><meta charset="UTF-8"><title>Fuwari Server</title>
<style>body{font-family:system-ui,sans-serif;background:#f6f7fb;display:flex;justify-content:center;align-items:center;min-height:100vh;margin:0}.card{background:#fff;border-radius:12px;padding:40px 56px;text-align:center;box-shadow:0 2px 20px rgba(0,0,0,.08)}.dot{display:inline-block;width:12px;height:12px;border-radius:50%%;background:#22c55e;margin-right:8px}h1{font-size:1.3rem;color:#1a1a2e;margin:0 0 8px}p{color:#666}</style></head>
<body><div class="card"><h1><span class="dot"></span>Fuwari Server v%s</h1><p>服务运行中</p></div></body></html>`, version.AppVersion)
}
