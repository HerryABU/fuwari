// Package models 定义数据模型与数据库操作。
// 评论数据存数据库；博客文章存文件系统（见 handlers/post.go）。
// 管理员密码以 bcrypt 哈希存于 settings 表（可修改：/editor 页面 或 ./fuwari-server -re pwd 重置）。
package models

import (
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// Setting 键值配置表（管理员密码哈希等）
type Setting struct {
	Key   string `gorm:"primaryKey;size:64"`
	Value string `gorm:"size:512"`
}

// SettingAdminPasswordHash 管理员密码哈希键
const SettingAdminPasswordHash = "admin_password_hash"

// GetSetting 读取配置项（不存在返回空串，不报错）
func GetSetting(key string) (string, error) {
	var s Setting
	err := DB.Where("key = ?", key).First(&s).Error
	if err == gorm.ErrRecordNotFound {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return s.Value, nil
}

// SetSetting 写入（或覆盖）配置项
func SetSetting(key, value string) error {
	return DB.Save(&Setting{Key: key, Value: value}).Error
}

// HasAdminPassword 是否已设置管理员密码
func HasAdminPassword() bool {
	v, err := GetSetting(SettingAdminPasswordHash)
	return err == nil && v != ""
}

// SetAdminPasswordHash 写入管理员密码的 bcrypt 哈希
func SetAdminPasswordHash(hash string) error {
	return SetSetting(SettingAdminPasswordHash, hash)
}

// VerifyAdminPassword 校验明文密码是否匹配数据库中存储的 bcrypt 哈希
// （bcrypt.CompareHashAndPassword 本身为常数时间比较，防时序攻击）
func VerifyAdminPassword(plain string) bool {
	if plain == "" {
		return false
	}
	hash, err := GetSetting(SettingAdminPasswordHash)
	if err != nil || hash == "" {
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}

// HashPassword 对明文密码做 bcrypt 哈希
func HashPassword(plain string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
