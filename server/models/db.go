// Package models 定义数据模型与数据库操作。
// 评论数据存数据库；博客文章存文件系统（见 handlers/post.go）。
package models

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"fuwari-server/config"
)

// DB 全局数据库连接
var DB *gorm.DB

// AllModels 所有需要建表与迁移的模型
func AllModels() []interface{} {
	return []interface{}{
		&Comment{},
		&Setting{},
	}
}

// InitDatabase 连接数据库并执行迁移
func InitDatabase() error {
	if err := os.MkdirAll(filepath.Dir(config.DBPath), 0755); err != nil {
		return fmt.Errorf("创建数据目录失败: %w", err)
	}

	// _pragma=foreign_keys(1)：每个新连接自动启用外键约束
	// _pragma=busy_timeout(5000)：写锁等待 5s，避免并发写直接报 database is locked
	dsn := config.DBPath
	if strings.Contains(dsn, "?") {
		dsn += "&_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)"
	} else {
		dsn += "?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)"
	}

	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("打开数据库失败: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("获取底层连接失败: %w", err)
	}
	sqlDB.SetMaxOpenConns(1) // SQLite 单写：限制连接数避免锁竞争
	sqlDB.SetConnMaxLifetime(time.Hour)

	if err := db.AutoMigrate(AllModels()...); err != nil {
		return fmt.Errorf("数据库迁移失败: %w", err)
	}

	DB = db
	return nil
}
