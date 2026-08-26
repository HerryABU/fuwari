// 站点设置（alist 风格自定义注入）。
//
// 支持三类全局注入，注入到所有 HTML 页面（前台 + 后台 + 编辑器）：
//   - HeadHTML  自定义头部 HTML/JS，注入 <head> 尾部
//   - BodyHTML  自定义尾部 HTML/JS，注入 </body> 前
//   - GlobalCSS 全局样式，注入 <head> 的 <style>
//   - FrontTheme 前台显示主题切换器（右下角浮动按钮）
//
// 存储：data/site-settings.json（运行时热加载，无需重新编译）。
package handlers

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"fuwari-server/config"
	"fuwari-server/utils"

	"github.com/gin-gonic/gin"
)

// SiteSettings 站点设置（alist 风格注入配置）
type SiteSettings struct {
	HeadHTML   string `json:"head_html"`   // 注入 <head> 尾部
	BodyHTML   string `json:"body_html"`   // 注入 </body> 前
	GlobalCSS  string `json:"global_css"`  // 注入 <head> 的 <style>
	FrontTheme bool   `json:"front_theme"` // 前台显示主题切换器
}

// siteSettingsPath 设置文件路径
func siteSettingsPath() string {
	return filepath.Join(config.DataDir, "site-settings.json")
}

// LoadSiteSettings 读取站点设置（文件不存在返回默认值）
func LoadSiteSettings() SiteSettings {
	var s SiteSettings
	data, err := os.ReadFile(siteSettingsPath())
	if err != nil {
		return s
	}
	_ = json.Unmarshal(data, &s)
	return s
}

// saveSiteSettings 写入站点设置
func saveSiteSettings(s SiteSettings) error {
	if err := os.MkdirAll(config.DataDir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(siteSettingsPath(), data, 0644)
}

// GetSiteSettings GET /api/admin/settings
// 返回站点设置 + 扩展列表（管理后台「设置」页数据源；主题列表走公开 /api/themes）
func GetSiteSettings(c *gin.Context) {
	utils.Success(c, gin.H{
		"settings":   LoadSiteSettings(),
		"extensions": ListExtensions(),
	})
}

// UpdateSiteSettings POST /api/admin/settings
// 保存站点设置（alist 风格自定义注入）
func UpdateSiteSettings(c *gin.Context) {
	var req struct {
		HeadHTML   string `json:"head_html"`
		BodyHTML   string `json:"body_html"`
		GlobalCSS  string `json:"global_css"`
		FrontTheme bool   `json:"front_theme"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数格式错误")
		return
	}
	s := SiteSettings{
		HeadHTML:   req.HeadHTML,
		BodyHTML:   req.BodyHTML,
		GlobalCSS:  req.GlobalCSS,
		FrontTheme: req.FrontTheme,
	}
	if err := saveSiteSettings(s); err != nil {
		utils.InternalError(c, "保存设置失败: "+err.Error())
		return
	}
	utils.Success(c, gin.H{"message": "站点设置已保存"})
}

// BuildSiteInjection 构建站点级注入 HTML：
//   - headPart：注入 <head> 尾部（自定义头 + 全局 CSS）
//   - bodyPart：注入 </body> 前（自定义尾 + 前台主题切换器）
//
// isAdmin：当前页面是否为管理后台（后台不注入主题切换器）
func BuildSiteInjection(isAdmin bool) (headPart, bodyPart string) {
	s := LoadSiteSettings()

	var head strings.Builder
	if strings.TrimSpace(s.GlobalCSS) != "" {
		head.WriteString("\n<style data-fuwari-site-css>\n")
		head.WriteString(s.GlobalCSS)
		head.WriteString("\n</style>")
	}
	if strings.TrimSpace(s.HeadHTML) != "" {
		head.WriteString("\n")
		head.WriteString(s.HeadHTML)
	}

	var body strings.Builder
	if strings.TrimSpace(s.BodyHTML) != "" {
		body.WriteString("\n")
		body.WriteString(s.BodyHTML)
	}
	// 前台主题切换器（管理后台不注入）
	if s.FrontTheme && !isAdmin {
		body.WriteString("\n")
		body.WriteString(themePickerSnippet)
	}
	return head.String(), body.String()
}

// themePickerSnippet 前台主题切换器（右下角浮动按钮，读取 /api/themes + POST /api/theme）
const themePickerSnippet = `<script data-fuwari-theme-picker>
(function () {
  'use strict';
  var base = window.FUWARI_BASE || '/';
  function pick() {
    if (document.getElementById('fw-theme-picker')) return;
    fetch(base + 'api/themes').then(function (r) { return r.json(); }).then(function (j) {
      if (!j || j.code !== 0 || !j.data || !j.data.list || !j.data.list.length) return;
      var list = j.data.list;
      var btn = document.createElement('button');
      btn.id = 'fw-theme-picker';
      btn.textContent = '\uD83C\uDFA8';
      btn.setAttribute('aria-label', 'Switch theme');
      btn.title = 'Theme';
      btn.style.cssText = 'position:fixed;right:1.25rem;bottom:1.25rem;z-index:9999;width:3rem;height:3rem;border-radius:1rem;border:none;cursor:pointer;background:var(--btn-regular-bg, rgba(0,0,0,.05));color:var(--btn-content, #333);font-size:1.35rem;box-shadow:0 8px 24px rgba(0,0,0,.16);transition:transform .2s, background .2s;';
      btn.onmouseenter = function () { btn.style.transform = 'scale(1.07)'; };
      btn.onmouseleave = function () { btn.style.transform = ''; };
      var panel = document.createElement('div');
      panel.id = 'fw-theme-picker-panel';
      panel.style.cssText = 'position:fixed;right:1.25rem;bottom:4.75rem;z-index:9999;display:none;min-width:13rem;max-height:60vh;overflow-y:auto;padding:.4rem;background:var(--float-panel-bg, #fff);border-radius:1rem;box-shadow:0 16px 40px rgba(0,0,0,.18);';
      list.forEach(function (t) {
        var item = document.createElement('button');
        item.textContent = (t.active ? '\u2713 ' : '') + t.name + (t.description ? ' \u00B7 ' + t.description : '');
        item.style.cssText = 'display:block;width:100%;padding:.5rem .75rem;border:none;background:none;cursor:pointer;text-align:left;font-size:.85rem;font-weight:600;border-radius:.6rem;color:' + (t.active ? 'var(--primary, #3b82f6)' : 'var(--deep-text, #333)') + ';transition:background .15s;';
        item.onmouseenter = function () { item.style.background = 'var(--btn-plain-bg-hover, rgba(0,0,0,.05))'; };
        item.onmouseleave = function () { item.style.background = ''; };
        item.onclick = function () {
          if (t.active) { panel.style.display = 'none'; return; }
          fetch(base + 'api/theme', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ theme: t.name }) })
            .then(function () { location.reload(); }).catch(function () { location.reload(); });
        };
        panel.appendChild(item);
      });
      btn.onclick = function () { panel.style.display = panel.style.display === 'none' ? 'block' : 'none'; };
      document.body.appendChild(btn);
      document.body.appendChild(panel);
    }).catch(function () {});
  }
  if (document.readyState === 'loading') { document.addEventListener('DOMContentLoaded', pick); } else { pick(); }
})();
</script>`
