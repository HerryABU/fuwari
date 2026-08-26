// 扩展分组管理（后台可视化编辑，alist 风格）。
//
// 分组 = extensions/<name>/ 目录（与运行时扩展注入系统一致）：
//   - 组内可含多个文件（index.js / index.css / 任意资源），多文件或单文件
//   - index.js / index.css 自动注入所有页面（见 BuildExtensionInjection）
//   - 支持克隆分组（复制目录）、编辑文件内容、新建/删除文件
//
// API（均需 AdminAuth）：
//   GET    /api/admin/extensions                    分组列表（名称 + 文件清单）
//   POST   /api/admin/extensions                    新建分组 {name}
//   POST   /api/admin/extensions/clone              克隆分组 {source, target}
//   DELETE /api/admin/extensions/:name              删除分组
//   GET    /api/admin/extensions/:name/:file        读取文件内容（文本）
//   PUT    /api/admin/extensions/:name/:file        写入文件内容 {content}
//   DELETE /api/admin/extensions/:name/:file        删除文件
package handlers

import (
	"os"
	"path/filepath"
	"strings"

	"fuwari-server/config"
	"fuwari-server/utils"

	"github.com/gin-gonic/gin"
)

// extFileInfo 文件清单条目
type extFileInfo struct {
	Name    string `json:"name"`
	Size    int64  `json:"size"`
	IsIndex bool   `json:"is_index"` // index.js / index.css 自动注入
}

// extGroupInfo 分组信息
type extGroupInfo struct {
	Name  string        `json:"name"`
	Files []extFileInfo `json:"files"`
}

// listExtGroupFiles 列出分组目录内文件
func listExtGroupFiles(name string) ([]extFileInfo, bool) {
	if !validExtName(name) {
		return nil, false
	}
	dir := filepath.Join(config.ExtensionsDir, name)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, false
	}
	var files []extFileInfo
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil {
			info = nil
		}
		var sz int64
		if info != nil {
			sz = info.Size()
		}
		files = append(files, extFileInfo{
			Name:    e.Name(),
			Size:    sz,
			IsIndex: e.Name() == "index.js" || e.Name() == "index.css",
		})
	}
	return files, true
}

// resolveExtFilePath 解析分组内文件绝对路径（防路径穿越）
func resolveExtFilePath(name, file string) (string, bool) {
	if !validExtName(name) || file == "" || file == "." || file == ".." || strings.Contains(file, "..") {
		return "", false
	}
	file = filepath.FromSlash(file)
	if strings.ContainsAny(file, `/\`) || strings.Contains(file, ":") {
		return "", false
	}
	root, err := filepath.Abs(config.ExtensionsDir)
	if err != nil {
		return "", false
	}
	target, err := filepath.Abs(filepath.Join(root, name, file))
	if err != nil || !strings.HasPrefix(target, root+string(os.PathSeparator)) {
		return "", false
	}
	return target, true
}

// ListAdminExtensions GET /api/admin/extensions
func ListAdminExtensions(c *gin.Context) {
	names := ListExtensions()
	groups := make([]extGroupInfo, 0, len(names))
	for _, n := range names {
		files, _ := listExtGroupFiles(n)
		groups = append(groups, extGroupInfo{Name: n, Files: files})
	}
	utils.Success(c, gin.H{"groups": groups})
}

// CreateExtension POST /api/admin/extensions 新建分组（含 index.js 模板）
func CreateExtension(c *gin.Context) {
	var req struct {
		Name string `json:"name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || !validExtName(req.Name) {
		utils.BadRequest(c, "分组名不合法")
		return
	}
	dir := filepath.Join(config.ExtensionsDir, req.Name)
	if _, err := os.Stat(dir); err == nil {
		utils.BadRequest(c, "分组已存在: "+req.Name)
		return
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		utils.InternalError(c, "创建分组失败: "+err.Error())
		return
	}
	tpl := "/* " + req.Name + " — fuwari 扩展分组（自动注入所有页面） */\n"
	_ = os.WriteFile(filepath.Join(dir, "index.js"), []byte(tpl), 0644)
	_ = os.WriteFile(filepath.Join(dir, "index.css"), []byte(tpl), 0644)
	utils.Success(c, gin.H{"message": "分组已创建", "name": req.Name})
}

// CloneExtension POST /api/admin/extensions/clone 克隆分组
func CloneExtension(c *gin.Context) {
	var req struct {
		Source string `json:"source"`
		Target string `json:"target"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || !validExtName(req.Target) {
		utils.BadRequest(c, "参数不合法")
		return
	}
	srcDir, ok := ResolveExtensionDir(req.Source)
	if !ok {
		utils.BadRequest(c, "源分组不存在: "+req.Source)
		return
	}
	dstDir := filepath.Join(config.ExtensionsDir, req.Target)
	if _, err := os.Stat(dstDir); err == nil {
		utils.BadRequest(c, "目标分组已存在: "+req.Target)
		return
	}
	if err := copyDir(srcDir, dstDir); err != nil {
		utils.InternalError(c, "克隆失败: "+err.Error())
		return
	}
	utils.Success(c, gin.H{"message": "分组已克隆", "name": req.Target})
}

// DeleteExtension DELETE /api/admin/extensions/:name 删除分组
func DeleteExtension(c *gin.Context) {
	name := c.Param("name")
	if !validExtName(name) {
		utils.BadRequest(c, "分组名不合法")
		return
	}
	dir, ok := ResolveExtensionDir(name)
	if !ok {
		utils.BadRequest(c, "分组不存在: "+name)
		return
	}
	if err := os.RemoveAll(dir); err != nil {
		utils.InternalError(c, "删除分组失败: "+err.Error())
		return
	}
	utils.Success(c, gin.H{"message": "分组已删除"})
}

// GetExtensionFile GET /api/admin/extensions/:name/:file 读取文件内容
func GetExtensionFile(c *gin.Context) {
	target, ok := resolveExtFilePath(c.Param("name"), c.Param("file"))
	if !ok {
		utils.BadRequest(c, "路径不合法")
		return
	}
	data, err := os.ReadFile(target)
	if err != nil {
		utils.BadRequest(c, "文件不存在")
		return
	}
	utils.Success(c, gin.H{"content": string(data)})
}

// UpdateExtensionFile PUT /api/admin/extensions/:name/:file 写入文件内容
func UpdateExtensionFile(c *gin.Context) {
	target, ok := resolveExtFilePath(c.Param("name"), c.Param("file"))
	if !ok {
		utils.BadRequest(c, "路径不合法")
		return
	}
	var req struct {
		Content string `json:"content"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数格式错误")
		return
	}
	if err := os.WriteFile(target, []byte(req.Content), 0644); err != nil {
		utils.InternalError(c, "写入失败: "+err.Error())
		return
	}
	utils.Success(c, gin.H{"message": "已保存"})
}

// DeleteExtensionFile DELETE /api/admin/extensions/:name/:file 删除文件
func DeleteExtensionFile(c *gin.Context) {
	target, ok := resolveExtFilePath(c.Param("name"), c.Param("file"))
	if !ok {
		utils.BadRequest(c, "路径不合法")
		return
	}
	if err := os.Remove(target); err != nil {
		utils.InternalError(c, "删除文件失败: "+err.Error())
		return
	}
	utils.Success(c, gin.H{"message": "文件已删除"})
}

// ResolveExtensionDir 解析扩展目录绝对路径（存在返回 abs,true）
func ResolveExtensionDir(name string) (string, bool) {
	if !validExtName(name) {
		return "", false
	}
	abs, err := filepath.Abs(filepath.Join(config.ExtensionsDir, name))
	if err != nil {
		return "", false
	}
	if info, err := os.Stat(abs); err == nil && info.IsDir() {
		return abs, true
	}
	return "", false
}

// copyDir 递归复制目录
func copyDir(src, dst string) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dst, 0755); err != nil {
		return err
	}
	for _, e := range entries {
		s := filepath.Join(src, e.Name())
		d := filepath.Join(dst, e.Name())
		if e.IsDir() {
			if err := copyDir(s, d); err != nil {
				return err
			}
			continue
		}
		data, err := os.ReadFile(s)
		if err != nil {
			return err
		}
		if err := os.WriteFile(d, data, 0644); err != nil {
			return err
		}
	}
	return nil
}
