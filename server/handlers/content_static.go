package handlers

import (
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"fuwari-server/config"

	"github.com/gin-gonic/gin"
)

// allowedAssetExt 运行时内容目录允许直出的静态资源扩展名
// （.md 永不直出——文章内容只通过 API 暴露）
var allowedAssetExt = map[string]bool{
	".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".webp": true,
	".svg": true, ".avif": true, ".ico": true, ".bmp": true,
	".mp4": true, ".webm": true, ".ogg": true, ".mp3": true, ".wav": true,
	".pdf": true, ".txt": true, ".json": true,
}

// ResolveContentAsset 将 URL 路径解析为运行时内容目录下的静态资源文件。
// 返回 (绝对路径, true) 表示命中；否则 ("" , false)。
// 路径穿越防护：Clean 后不允许跳出内容根目录。
func ResolveContentAsset(urlPath string) (string, bool) {
	root, err := filepath.Abs(config.PostsDir)
	if err != nil {
		return "", false
	}

	clean := path.Clean("/" + strings.TrimPrefix(urlPath, "/"))
	target := filepath.Join(root, filepath.FromSlash(clean))

	abs, err := filepath.Abs(target)
	if err != nil || !strings.HasPrefix(abs, root+string(os.PathSeparator)) {
		return "", false
	}

	info, err := os.Stat(abs)
	if err != nil || info.IsDir() {
		return "", false
	}
	if !allowedAssetExt[strings.ToLower(filepath.Ext(abs))] {
		return "", false
	}
	return abs, true
}

// ServeContentAsset 运行时内容资源直出（供 NoRoute 前置检查命中的路径调用）
func ServeContentAsset(c *gin.Context) {
	filePath, ok := ResolveContentAsset(c.Request.URL.Path)
	if !ok {
		c.Status(http.StatusNotFound)
		return
	}
	c.File(filePath)
}
