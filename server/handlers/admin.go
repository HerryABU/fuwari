// 管理员密码管理 handler。
// 修改密码：POST /api/admin/password（AdminAuth 保护，body 校验旧密码）。
package handlers

import (
	"fuwari-server/models"
	"fuwari-server/utils"

	"github.com/gin-gonic/gin"
)

// ChangeAdminPassword POST /api/admin/password
// 修改管理员密码（知道当前密码的场景）。
// AdminAuth 中间件已校验请求头携带的当前密码，body 中的 old_password
// 再与数据库比对作双重校验，防止误操作与 CSRF 场景下的意外改密。
func ChangeAdminPassword(c *gin.Context) {
	var req struct {
		OldPassword string `json:"old_password"`
		NewPassword string `json:"new_password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.OldPassword == "" || req.NewPassword == "" {
		utils.BadRequest(c, "old_password 与 new_password 不能为空")
		return
	}
	if len(req.NewPassword) < 6 {
		utils.BadRequest(c, "新密码至少 6 个字符")
		return
	}
	if !models.VerifyAdminPassword(req.OldPassword) {
		utils.Forbidden(c, "当前密码不正确")
		return
	}
	hash, err := models.HashPassword(req.NewPassword)
	if err != nil {
		utils.InternalError(c, "密码加密失败")
		return
	}
	if err := models.SetAdminPasswordHash(hash); err != nil {
		utils.InternalError(c, "保存密码失败")
		return
	}
	utils.Success(c, gin.H{"message": "管理员密码已更新"})
}
