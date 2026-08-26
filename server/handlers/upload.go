// 编辑器图片上传。
// POST /api/admin/upload（AdminAuth + 限流）：multipart 文件 → 保存到
// 运行时内容目录 content/posts/uploads/，返回可被 ResolveContentAsset
// 直出的相对 URL（/uploads/xxx.png）。正文 Markdown 用绝对路径引用，
// 反代挂载下由 RewriteHTML 自动加前缀。
package handlers

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"fuwari-server/config"
	"fuwari-server/utils"

	"github.com/gin-gonic/gin"
)

// 允许上传的图片扩展名（SVG 含脚本风险，禁止）
var uploadImageExt = map[string]bool{
	".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".webp": true,
	".avif": true, ".bmp": true,
}

const maxUploadBytes = 5 << 20 // 5MB

// UploadImage POST /api/admin/upload  multipart/form-data 字段名: file
// 返回 {url: "/uploads/<name>.<ext>"}
func UploadImage(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		utils.BadRequest(c, "缺少 file 字段")
		return
	}
	defer file.Close()

	if header.Size > maxUploadBytes {
		utils.BadRequest(c, "图片不能超过 5MB")
		return
	}

	ext := strings.ToLower(filepath.Ext(header.Filename))
	if !uploadImageExt[ext] {
		utils.BadRequest(c, "不支持的图片类型: "+ext)
		return
	}

	dir := filepath.Join(config.PostsDir, "uploads")
	if err := os.MkdirAll(dir, 0755); err != nil {
		utils.InternalError(c, "创建上传目录失败")
		return
	}

	name := fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)
	target := filepath.Join(dir, name)

	out, err := os.Create(target)
	if err != nil {
		utils.InternalError(c, "保存图片失败")
		return
	}
	defer out.Close()
	if _, err := io.Copy(out, file); err != nil {
		utils.InternalError(c, "写入图片失败")
		return
	}

	utils.Success(c, gin.H{"url": "/uploads/" + name})
}
