package security

import (
	"regexp"
	"strings"
)

// 评论安全策略：评论内容以 Markdown 存储、由 Cherry Markdown 客户端渲染，
// Cherry 默认转义 HTML，因此服务端在入库前剥除全部 HTML 标签，双保险防 XSS。

var (
	tagRe      = regexp.MustCompile(`<[^>]*>`)
	schemeRe   = regexp.MustCompile(`(?i)javascript\s*:|vbscript\s*:|data\s*:\s*text/html`)
	ctrlCharRe = regexp.MustCompile(`[\x00-\x08\x0b\x0c\x0e-\x1f]`)
)

// SanitizeMarkdown 净化用户提交的 Markdown 内容：
//  1. 剥除所有 HTML 标签（含 script/iframe 等，成对标签在去掉尖括号后
//     残留的标签体文本会被 Cherry Markdown 按纯文本渲染，不构成执行载体）；
//  2. 中和 javascript:/data:text/html 等危险 scheme；
//  3. 移除控制字符。
func SanitizeMarkdown(content string) string {
	s := ctrlCharRe.ReplaceAllString(content, "")
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
