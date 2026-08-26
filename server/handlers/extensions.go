// 扩展系统（alist 风格，运行时热加载）。
//
// extensions/
//   <name>/
//     index.js    # 入口脚本（注入所有页面）
//     index.css   # 入口样式（注入所有页面）
//     ...         # 其他资源（模型、图片等），经 /extensions/<name>/ 静态服务
//
// 修改后刷新页面即可生效，无需重新编译。兼容看板娘等原生扩展。
package handlers

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"fuwari-server/config"

	"github.com/gin-gonic/gin"
)

// validExtName 扩展名合法性（防路径穿越）
func validExtName(name string) bool {
	if name == "" || name == "." || name == ".." {
		return false
	}
	return !strings.ContainsAny(name, "/\\")
}

// ListExtensions 扫描扩展目录，返回扩展名列表
func ListExtensions() []string {
	var names []string
	entries, err := os.ReadDir(config.ExtensionsDir)
	if err != nil {
		return names
	}
	for _, e := range entries {
		if e.IsDir() && validExtName(e.Name()) {
			names = append(names, e.Name())
		}
	}
	return names
}

// ServeExtensionAsset 提供 /extensions/<name>/<file> 静态资源
func ServeExtensionAsset(c *gin.Context) {
	name := c.Param("name")
	if !validExtName(name) {
		c.Status(http.StatusNotFound)
		return
	}
	rel := strings.Trim(c.Param("filepath"), "/")
	if rel == "" || strings.Contains(rel, "..") {
		c.Status(http.StatusNotFound)
		return
	}
	root, err := filepath.Abs(config.ExtensionsDir)
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}
	target, err := filepath.Abs(filepath.Join(root, name, filepath.FromSlash(rel)))
	if err != nil || !strings.HasPrefix(target, root+string(os.PathSeparator)) {
		c.Status(http.StatusNotFound)
		return
	}
	if data, err := os.ReadFile(target); err == nil {
		c.Header("Cache-Control", "no-cache") // 热加载
		c.Data(http.StatusOK, assetMIME(rel), data)
		return
	}
	c.Status(http.StatusNotFound)
}

// BuildExtensionInjection 构建扩展注入 HTML（<head> CSS + </body> JS），空扩展时返回空串
func BuildExtensionInjection() string {
	names := ListExtensions()
	if len(names) == 0 {
		return ""
	}
	var cssParts, jsParts []string
	for _, n := range names {
		if _, err := os.Stat(filepath.Join(config.ExtensionsDir, n, "index.css")); err == nil {
			cssParts = append(cssParts, `<link rel="stylesheet" href="/extensions/`+n+`/index.css" data-fuwari-ext>`)
		}
		if _, err := os.Stat(filepath.Join(config.ExtensionsDir, n, "index.js")); err == nil {
			jsParts = append(jsParts, `<script src="/extensions/`+n+`/index.js" defer data-fuwari-ext></script>`)
		}
	}
	var sb strings.Builder
	if len(cssParts) > 0 {
		sb.WriteString(strings.Join(cssParts, "\n"))
	}
	if len(jsParts) > 0 {
		sb.WriteString("\n")
		sb.WriteString(strings.Join(jsParts, "\n"))
	}
	return sb.String()
}
