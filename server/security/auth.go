// Package security 提供管理令牌认证与用户输入净化。
package security

import (
	"crypto/subtle"

	"fuwari-server/config"
	"fuwari-server/utils"

	"github.com/gin-gonic/gin"
)

// AdminAuth 管理令牌中间件：内容写操作（文章创建/保存/删除、评论删除）须携带令牌。
// 支持两种携带方式：
//   - Authorization: Bearer <token>
//   - X-Admin-Token: <token>
func AdminAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := extractToken(c)
		if token == "" {
			utils.Unauthorized(c, "需要管理令牌")
			c.Abort()
			return
		}
		// 常数时间比较，防时序攻击
		if subtle.ConstantTimeCompare([]byte(token), []byte(config.AdminToken)) != 1 {
			utils.Forbidden(c, "管理令牌无效")
			c.Abort()
			return
		}
		c.Next()
	}
}

// extractToken 从请求头提取管理令牌
func extractToken(c *gin.Context) string {
	if h := c.GetHeader("Authorization"); len(h) > 7 && h[:7] == "Bearer " {
		return h[7:]
	}
	return c.GetHeader("X-Admin-Token")
}
