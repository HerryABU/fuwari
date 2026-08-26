// 主题系统（alist 风格，运行时热加载，无需重新编译）。
//
// 目录结构：
//
//	themes/
//	  <theme-name>/
//	    theme.css        # CSS 变量覆盖（--primary/--card-bg/...）+ 自定义样式
//	    background.*     # 背景图（可选，theme.css 中引用 ./background.png）
//	    custom.js        # 自定义脚本（可选，如看板娘、统计代码）
//	    manifest.json    # 主题元信息（可选）：name/description/author/version
//
// 切换方式：
//   - URL 参数 ?theme=<name>
//   - Cookie: fuwari_theme=<name>
//   - 未指定时使用 config.DefaultTheme（default = 内嵌默认样式，无需主题目录）
//
// 注入机制：服务端在返回 HTML 时向 <head> 注入主题 CSS、向 </body> 前注入 custom.js，
// 前台（Astro 页面）与后台（/editor）统一走同一注入逻辑，保证前后台 UI 完全一致。
package handlers

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"fuwari-server/config"

	"github.com/gin-gonic/gin"
)

// EmbeddedAssetsFS 由 main 注入的内嵌 assets FS（go:embed all:assets）。
// 主题兜底资源位于 assets/themes/default/ 下。
var EmbeddedAssetsFS fs.FS

// themeCookieName 主题切换 Cookie 名
const themeCookieName = "fuwari_theme"

// themeNameRe 主题名合法性（仅允许安全字符，防路径穿越）
func validThemeName(name string) bool {
	if name == "" || name == "." || name == ".." {
		return false
	}
	if strings.ContainsAny(name, "/\\") {
		return false
	}
	return true
}

// ResolveThemeDir 解析主题目录绝对路径；不存在返回 ("", false)
func ResolveThemeDir(name string) (string, bool) {
	if !validThemeName(name) {
		return "", false
	}
	dir := filepath.Join(config.ThemesDir, name)
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", false
	}
	if info, err := os.Stat(abs); err == nil && info.IsDir() {
		return abs, true
	}
	return "", false
}

// currentThemeName 从请求中解析当前主题（URL 参数优先，其次 Cookie，最后默认）
func currentThemeName(c *gin.Context) string {
	if q := c.Query("theme"); q != "" && validThemeName(q) {
		return q
	}
	if cookie, err := c.Cookie(themeCookieName); err == nil && validThemeName(cookie) {
		return cookie
	}
	if validThemeName(config.DefaultTheme) {
		return config.DefaultTheme
	}
	return "default"
}

// CurrentThemeName 导出当前主题名（供 main 注入使用）
func CurrentThemeName(c *gin.Context) string {
	return currentThemeName(c)
}

// ThemeManifest 主题元信息（manifest.json）
type ThemeManifest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Author      string `json:"author"`
	Version     string `json:"version"`
}

// ListThemes GET /api/themes 列出可用主题（扫描 themes/ 目录 + 内嵌默认）
func ListThemes(c *gin.Context) {
	type themeEntry struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Author      string `json:"author"`
		Version     string `json:"version"`
		Background  string `json:"background,omitempty"` // 背景图相对 URL
		Active      bool   `json:"active"`
	}
	var list []themeEntry

	// 内嵌默认主题始终存在（default = 原版样式）
	list = append(list, themeEntry{
		Name:   "default",
		Active: currentThemeName(c) == "default",
	})

	entries, err := os.ReadDir(config.ThemesDir)
	if err == nil {
		for _, e := range entries {
			if !e.IsDir() || !validThemeName(e.Name()) {
				continue
			}
			entry := themeEntry{Name: e.Name(), Active: currentThemeName(c) == e.Name()}
			dir := filepath.Join(config.ThemesDir, e.Name())
			if data, err := os.ReadFile(filepath.Join(dir, "manifest.json")); err == nil {
				var m ThemeManifest
				if json.Unmarshal(data, &m) == nil {
					entry.Description = m.Description
					entry.Author = m.Author
					entry.Version = m.Version
				}
			}
			// 探测背景图
			for _, ext := range []string{".jpg", ".jpeg", ".png", ".webp", ".gif", ".svg", ".avif"} {
				if _, err := os.Stat(filepath.Join(dir, "background"+ext)); err == nil {
					entry.Background = "/themes/" + e.Name() + "/background" + ext
					break
				}
			}
			list = append(list, entry)
		}
	}

	utils_Success(c, gin.H{"list": list})
}

// utils_Success 兼容 utils 包响应（避免循环引用，直接内联响应）
func utils_Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": data})
}

// ServeThemeAsset 提供 /themes/<name>/<file> 静态资源。
// 双源策略：运行时 themes/<name>/ 优先（可热修改），嵌入的默认主题兜底。
func ServeThemeAsset(c *gin.Context) {
	parts := strings.Split(strings.Trim(c.Param("filepath"), "/"), "/")
	if len(parts) == 0 {
		c.Status(http.StatusNotFound)
		return
	}
	name := c.Param("name")
	rel := filepath.Join(parts...)

	// 运行时主题目录优先
	if dir, ok := ResolveThemeDir(name); ok {
		target := filepath.Join(dir, rel)
		abs, err := filepath.Abs(target)
		if err == nil && strings.HasPrefix(abs, filepath.Clean(dir)+string(os.PathSeparator)) {
			if data, err := os.ReadFile(abs); err == nil {
				c.Header("Cache-Control", "no-cache") // 热加载：每次校验
				c.Data(http.StatusOK, assetMIME(rel), data)
				return
			}
		}
	}

	// 嵌入的默认主题兜底（assets/themes/default/）
	if name == "default" && EmbeddedAssetsFS != nil {
		clean := filepath.ToSlash(filepath.Clean("/" + rel))
		if data, err := fs.ReadFile(EmbeddedAssetsFS, "assets/themes/default"+clean); err == nil {
			c.Header("Cache-Control", "no-cache")
			c.Data(http.StatusOK, assetMIME(rel), data)
			return
		}
	}

	c.Status(http.StatusNotFound)
}

// assetMIME 简易 MIME 推断（与 main.assetContentType 一致，独立实现避免跨包）
func assetMIME(name string) string {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".css":
		return "text/css; charset=utf-8"
	case ".js":
		return "application/javascript; charset=utf-8"
	case ".json":
		return "application/json; charset=utf-8"
	case ".svg":
		return "image/svg+xml"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".avif":
		return "image/avif"
	case ".woff":
		return "font/woff"
	case ".woff2":
		return "font/woff2"
	case ".ttf":
		return "font/ttf"
	default:
		return "application/octet-stream"
	}
}

// ThemeHeadInjection 返回注入 <head> 的主题 CSS 标签（默认主题且无自定义文件时返回空串）
func ThemeHeadInjection(themeName string) string {
	if !validThemeName(themeName) {
		themeName = "default"
	}
	// default 主题仅当存在自定义 theme.css 时才注入（内嵌 default 为空占位，跳过减少无效请求）
	if themeName == "default" {
		if _, ok := ResolveThemeDir("default"); !ok {
			// 检查内嵌兜底是否有实际内容
			if EmbeddedAssetsFS != nil {
				if data, err := fs.ReadFile(EmbeddedAssetsFS, "assets/themes/default/theme.css"); err == nil && len(strings.TrimSpace(string(data))) > 60 {
					return fmt.Sprintf(`<link rel="stylesheet" href="/themes/default/theme.css" data-fuwari-theme>`, )
				}
			}
			return ""
		}
	}
	return fmt.Sprintf(`<link rel="stylesheet" href="/themes/%s/theme.css" data-fuwari-theme>`, themeName)
}

// ThemeBodyInjection 返回注入 </body> 前的主题脚本（FUWARI_THEME 全局 + custom.js）
func ThemeBodyInjection(themeName string) string {
	if !validThemeName(themeName) {
		themeName = "default"
	}
	script := fmt.Sprintf(`<script>window.FUWARI_THEME="%s"</script>`, themeName)
	// default 主题：仅当 custom.js 存在时注入
	if themeName == "default" {
		if _, ok := ResolveThemeDir("default"); !ok {
			return script
		}
	}
	script += fmt.Sprintf(`<script src="/themes/%s/custom.js" defer data-fuwari-theme></script>`, themeName)
	return script
}

// InjectThemeHTML 向 HTML 注入主题样式与脚本（前后台统一入口）。
// 返回注入后的 HTML；非 HTML 或注入失败时原样返回。
// 注入内容：
//   <head>  : <link rel="stylesheet" href="/themes/<name>/theme.css">
//   </body> : <script src="/themes/<name>/custom.js" defer></script>
//             <script>window.FUWARI_THEME="<name>"</script>
func InjectThemeHTML(html []byte, themeName string) []byte {
	if !validThemeName(themeName) {
		themeName = "default"
	}
	// 已注入过则跳过
	if strings.Contains(string(html), "fuwari-theme") {
		return html
	}

	themeCSS := ThemeHeadInjection(themeName)
	themeJS := ThemeBodyInjection(themeName)

	s := string(html)
	// 注入 CSS 到 </head> 前
	if themeCSS != "" {
		if idx := strings.LastIndex(s, "</head>"); idx >= 0 {
			s = s[:idx] + themeCSS + s[idx:]
		} else if idx := strings.LastIndex(s, "<head"); idx >= 0 {
			if bi := strings.Index(s[idx:], ">"); bi >= 0 {
				s = s[:idx+bi+1] + themeCSS + s[idx+bi+1:]
			}
		}
	}
	// 注入 JS 到 </body> 前
	if idx := strings.LastIndex(s, "</body>"); idx >= 0 {
		s = s[:idx] + themeJS + s[idx:]
	} else {
		s += themeJS
	}
	return []byte(s)
}

// themeDirExists 检查运行时主题目录是否存在（供 main 判断是否启用注入）
func themeDirExists(name string) bool {
	_, ok := ResolveThemeDir(name)
	return ok
}

// SetTheme 设置主题切换 Cookie（POST /api/theme {theme: "xxx"}）
func SetTheme(c *gin.Context) {
	var req struct {
		Theme string `json:"theme"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || !validThemeName(req.Theme) {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "无效主题名", "data": nil})
		return
	}
	// 校验主题存在（default 恒存在；自定义主题需有目录）
	if req.Theme != "default" {
		if _, ok := ResolveThemeDir(req.Theme); !ok {
			c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "主题不存在: " + req.Theme, "data": nil})
			return
		}
	}
	c.SetCookie(themeCookieName, req.Theme, 365*24*3600, "/", "", false, true)
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": gin.H{"theme": req.Theme}})
}

// VerifyThemeToken 校验主题名（常数时间比较辅助，供测试/安全使用）
func VerifyThemeToken(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
