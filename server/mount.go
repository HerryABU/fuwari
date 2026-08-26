// 反代挂载前缀自适应（mount-aware path handling）。
//
// 场景：用户通过自建反向代理将站点挂载到任意子路径下，例如
//
//	https://host:8088/{name}/  →  fuwari-server   （{name} 仅示例，任意前缀）
//
// 反代【保留】前缀（不重写路径），fuwari 收到 /{name}/api/...、/{name}/posts/... 等。
// 本模块保证：
//
//  1. 内部路由自适应 —— 剥离前缀后递归重路由（帽子对用户不可见，URL 不变）；
//  2. 页面 HTML 中所有根绝对路径引用（js/css/图片/站内链接）改写为带前缀的路径；
//  3. 内联脚本中 pagefind 的硬编码路径修复；
//  4. 挂件/编辑器 JS 通过 window.FUWARI_BASE 动态拼接 API 与资源路径（无前缀时回退 "/"）。
//
// 全程动态探测，严禁硬编码任何前缀名。
package main

import (
	"io/fs"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
)

// knownRootDirs 站点根级路径集合：
// 请求路径第一段命中该集合 → 视为正常站点路径（如 /posts/...、/api/...）；
// 否则第一段视为反代挂载前缀（如 /{name}/...）。
var knownRootDirs = map[string]bool{}

// InitKnownRoots 初始化根级路径集合：后端保留段 + 嵌入前端根目录/文件。
// 在 frontendFS 就绪后调用一次。
func InitKnownRoots(frontendFS fs.FS) {
	for _, s := range []string{"api", "assets", "themes", "extensions", "editor"} {
		knownRootDirs[s] = true
	}
	if entries, err := fs.ReadDir(frontendFS, "."); err == nil {
		for _, e := range entries {
			name := e.Name()
			if i := strings.IndexByte(name, '.'); i >= 0 {
				name = name[:i]
			}
			if name != "" {
				knownRootDirs[name] = true
			}
		}
	}
}

// stripMountPrefix 尝试剥离反代挂载前缀。
// 判定：第一段不是已知根级路径、且第二段命中已知根级路径 → 剥离第一段。
// 例：/name/api/comments → /api/comments；/name/posts/guide/ → /posts/guide/。
// 返回剥离后的路径与探测到的前缀（"/<name>/"）；无需剥离时 ok=false。
// 单段 /name/ 与全未知多段路径保留原样（由 NoRoute fallback 与 detectBasePath 兜底）。
func stripMountPrefix(path string) (newPath, prefix string, ok bool) {
	p := strings.Trim(path, "/")
	if p == "" {
		return path, "", false
	}
	segs := strings.Split(p, "/")
	if knownRootDirs[segs[0]] {
		return path, "", false
	}
	if len(segs) >= 2 && knownRootDirs[segs[1]] {
		return "/" + strings.Join(segs[1:], "/"), "/" + segs[0] + "/", true
	}
	return path, "", false
}

// basePathKey request context 键：外层前缀处理器探测到的挂载前缀
type basePathKey struct{}

// detectBasePath 计算当前请求的站点前缀（反代帽子）。
// 外层前缀处理器已把探测结果写入 context（strip 改写路径前）；
// 直接访问（无前缀）或单段前缀（未被 strip）时按路径推断。
func detectBasePath(c *gin.Context) string {
	if bp, ok := c.Request.Context().Value(basePathKey{}).(string); ok && bp != "" {
		return bp
	}
	p := strings.Trim(c.Request.URL.Path, "/")
	if p == "" {
		return ""
	}
	segs := strings.Split(p, "/")
	if knownRootDirs[segs[0]] {
		return ""
	}
	return "/" + segs[0] + "/"
}

// absAttrRe 匹配 HTML 属性中的根绝对路径（/xxx 开头）
var absAttrRe = regexp.MustCompile(`\b(href|src|action|poster|data-src|data-href)\s*=\s*("|')(/[^"']*)("|')`)

// pagefindScriptOld 构建产物内联脚本中的硬编码 pagefind 路径（fuwari 原版 Search 集成）
const pagefindScriptOld = `"/pagefind/pagefind.js"`

// RewriteHTML 将 HTML 中的根绝对路径引用改写为带站点前缀的路径：
//
//	href="/assets/x.css" + basePath="/name/" → href="/name/assets/x.css"
//	href="/"                                → href="/name/"
//
// 同时修复内联脚本中 pagefind 的硬编码路径。
// basePath 为空（正常挂载）时属性不改写，pagefind 保持原绝对路径。
func RewriteHTML(html []byte, basePath string) []byte {
	s := string(html)
	if basePath != "" {
		s = absAttrRe.ReplaceAllStringFunc(s, func(m string) string {
			sub := absAttrRe.FindStringSubmatch(m)
			if len(sub) != 5 {
				return m
			}
			attr, quote, path := sub[1], sub[2], sub[3]
			if strings.HasPrefix(path, "//") {
				return m // 协议相对 URL（//cdn.xxx），不改
			}
			if path == "/" {
				return attr + "=" + quote + basePath + quote
			}
			return attr + "=" + quote + basePath + strings.TrimPrefix(path, "/") + quote
		})
	}
	var neu string
	if basePath == "" {
		neu = pagefindScriptOld
	} else {
		neu = `"` + basePath + `pagefind/pagefind.js"`
	}
	s = strings.ReplaceAll(s, pagefindScriptOld, neu)
	return []byte(s)
}
