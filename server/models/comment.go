package models

import (
	"time"

	"gorm.io/gorm"
)

// Comment 博客评论。TargetType + TargetSlug 定位文章
// （与文件系统博客的 slug 关联，不使用数据库外键）。
type Comment struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	TargetType   string         `gorm:"size:32;default:post;index:idx_comments_target,priority:1" json:"target_type"`
	TargetSlug   string         `gorm:"size:255;index:idx_comments_target,priority:2" json:"target_slug"`
	Nickname     string         `gorm:"size:64;not null" json:"nickname"`
	Email        string         `gorm:"size:128;default:''" json:"email,omitempty"`
	Site         string         `gorm:"size:255;default:''" json:"site,omitempty"`
	Content      string         `gorm:"type:text;not null" json:"content"`
	ParentID     *uint          `gorm:"index" json:"parent_id"`
	IsMarkdown   bool           `gorm:"default:true" json:"is_markdown"`
	IP           string         `gorm:"size:64;default:''" json:"-"`
	CreatedAt    time.Time      `gorm:"index:idx_comments_target,priority:3" json:"created_at"`
	UpdatedAt    time.Time      `json:"-"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

func (Comment) TableName() string {
	return "comments"
}

// CreateComment 创建评论
func CreateComment(comment *Comment) error {
	return DB.Create(comment).Error
}

// GetCommentsByTarget 按目标分页查询评论（时间倒序）
func GetCommentsByTarget(targetType, targetSlug string, page, pageSize int) ([]Comment, int64, error) {
	var comments []Comment
	var total int64

	query := DB.Model(&Comment{}).
		Where("target_type = ? AND target_slug = ?", targetType, targetSlug)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").
		Offset(offset).Limit(pageSize).Find(&comments).Error; err != nil {
		return nil, 0, err
	}
	return comments, total, nil
}

// GetCommentByID 按 ID 查询评论
func GetCommentByID(id uint) (*Comment, error) {
	var comment Comment
	if err := DB.First(&comment, id).Error; err != nil {
		return nil, err
	}
	return &comment, nil
}

// DeleteComment 删除评论（软删除）
func DeleteComment(id uint) error {
	return DB.Delete(&Comment{}, id).Error
}

// CountCommentsByTarget 统计目标评论数（用于文章列表展示）
func CountCommentsByTarget(targetType, targetSlug string) int64 {
	var count int64
	DB.Model(&Comment{}).
		Where("target_type = ? AND target_slug = ?", targetType, targetSlug).
		Count(&count)
	return count
}
