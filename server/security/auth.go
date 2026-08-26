// Package security 提供管理员密码认证与用户输入净化。
package security

import (
	"fuwari-server/models"
	"fuwari-server/utils"

	"github.com/gin-gonic/gin"
)

// AdminAuth 管理员密码中间件：内容写操作（文章创建/保存/删除、评论删除、
// 修改密码）须携带有效管理员密码。
// 支持两种携带方式：
//   - Authorization: Bearer <password>
//   - X-Admin-Token: <password>
// 密码以 bcrypt 哈希存于数据库 settings 表；首次启动由 ADMIN_TOKEN 或随机生成引导，
// 之后可用 /editor 页面修改，忘记密码可用 ./fuwari-server -re pwd 命令行重置。
func AdminAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := extractToken(c)
		if token == "" {
			utils.Unauthorized(c, "需要管理员密码")
			c.Abort()
			return
		}
		if !models.VerifyAdminPassword(token) {
			utils.Forbidden(c, "管理员密码无效")
			c.Abort()
			return
		}
		c.Next()
	}
}

// extractToken 从请求头提取管理员凭据
func extractToken(c *gin.Context) string {
	if h := c.GetHeader("Authorization"); len(h) > 7 && h[:7] == "Bearer " {
		return h[7:]
	}
	return c.GetHeader("X-Admin-Token")
}

// VerifyPassword 校验给定明文是否为当前管理员密码（供 handler 复用）
func VerifyPassword(plain string) bool {
	return models.VerifyAdminPassword(plain)
}
