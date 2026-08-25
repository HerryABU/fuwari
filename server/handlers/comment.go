// Package handlers 实现 HTTP 接口处理。
package handlers

import (
	"strconv"
	"strings"

	"fuwari-server/config"
	"fuwari-server/models"
	"fuwari-server/security"
	"fuwari-server/utils"

	"github.com/gin-gonic/gin"
)

// GetComments 获取文章评论列表（公开）
// GET /api/comments?slug=xxx&page=1&page_size=20
func GetComments(c *gin.Context) {
	slug := strings.TrimSpace(c.Query("slug"))
	if slug == "" {
		utils.BadRequest(c, "缺少文章标识")
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", strconv.Itoa(config.CommentPageSize)))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = config.CommentPageSize
	}

	comments, total, err := models.GetCommentsByTarget("post", slug, page, pageSize)
	if err != nil {
		utils.InternalError(c, "获取评论失败")
		return
	}

	utils.Success(c, gin.H{
		"list":  comments,
		"total": total,
	})
}

// CreateComment 发表评论（管理令牌保护，防止匿名垃圾评论）
// POST /api/comments
func CreateComment(c *gin.Context) {
	var req struct {
		Slug     string `json:"slug" binding:"required"`
		Nickname string `json:"nickname" binding:"required"`
		Email    string `json:"email"`
		Site     string `json:"site"`
		Content  string `json:"content" binding:"required"`
		ParentID *uint  `json:"parent_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "请填写昵称与评论内容")
		return
	}

	slug := strings.TrimSpace(req.Slug)
	nickname := security.SanitizePlain(req.Nickname)
	content := security.SanitizeMarkdown(req.Content)

	if nickname == "" {
		utils.BadRequest(c, "请填写昵称")
		return
	}
	if len([]rune(nickname)) > 64 {
		utils.BadRequest(c, "昵称过长")
		return
	}
	if content == "" {
		utils.BadRequest(c, "请填写评论内容")
		return
	}
	if len([]rune(content)) > config.CommentMaxLength {
		utils.BadRequest(c, "评论内容过长")
		return
	}

	comment := &models.Comment{
		TargetType: "post",
		TargetSlug: slug,
		Nickname:   nickname,
		Email:      security.SanitizePlain(req.Email),
		Site:       security.SanitizePlain(req.Site),
		Content:    content,
		ParentID:   req.ParentID,
		IsMarkdown: true,
		IP:         c.ClientIP(),
	}

	if err := models.CreateComment(comment); err != nil {
		utils.InternalError(c, "发表评论失败")
		return
	}

	utils.Success(c, comment)
}

// DeleteComment 删除评论（管理令牌保护）
// DELETE /api/comments/:id
func DeleteComment(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		utils.BadRequest(c, "无效的评论 ID")
		return
	}

	if _, err := models.GetCommentByID(uint(id)); err != nil {
		utils.NotFound(c, "评论不存在")
		return
	}

	if err := models.DeleteComment(uint(id)); err != nil {
		utils.InternalError(c, "删除评论失败")
		return
	}

	utils.Success(c, gin.H{"message": "评论已删除"})
}
