package security

import (
	"regexp"
	"strings"
)

// 评论安全策略：评论内容以 Markdown 存储、由 Cherry Markdown 客户端渲染，
// Cherry 默认转义 HTML，因此服务端在入库前剥除全部 HTML 标签，双保险防 XSS。

var (
	tagRe      = regexp.MustCompile(`<[^>]*>`)
	scriptRe   = regexp.MustCompile(`(?is)<(script|style|iframe|object|embed|form|link|meta)[^>]*>.*?</\1>`)
	schemeRe   = regexp.MustCompile(`(?i)javascript\s*:|vbscript\s*:|data\s*:\s*text/html`)
	ctrlCharRe = regexp.MustCompile(`[\x00-\x08\x0b\x0c\x0e-\x1f]`)
)

// SanitizeMarkdown 净化用户提交的 Markdown 内容：
//  1. 剥除 script/style 等危险标签对；
//  2. 剥除所有残留 HTML 标签；
//  3. 中和 javascript:/data:text/html 等危险 scheme；
//  4. 移除控制字符。
func SanitizeMarkdown(content string) string {
	s := ctrlCharRe.ReplaceAllString(content, "")
	s = scriptRe.ReplaceAllString(s, "")
	s = tagRe.ReplaceAllString(s, "")
	s = schemeRe.ReplaceAllStringFunc(s, func(m string) string {
		return strings.ReplaceAll(m, ":", "%3a")
	})
	return strings.TrimSpace(s)
}

// SanitizePlain 净化纯文本字段（昵称等）：剥除全部 HTML 标签与首尾空白
func SanitizePlain(s string) string {
	s = ctrlCharRe.ReplaceAllString(s, "")
	s = tagRe.ReplaceAllString(s, "")
	return strings.TrimSpace(s)
}
