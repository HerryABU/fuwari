package utils

import (
	"strings"
	"unicode"
)

// Slugify 将标题转为 URL slug：
//   - ASCII 字母/数字保留；
//   - 空白与标点转为单个连字符；
//   - 其余字符（如中日韩文字）丢弃；
//   - 结果为空时返回 fallback。
func Slugify(title, fallback string) string {
	var sb strings.Builder
	lastHyphen := true // 开头视为已有连字符，避免前导 -
	for _, r := range title {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			sb.WriteRune(unicode.ToLower(r))
			lastHyphen = false
		case r == '-' || r == '_' || unicode.IsSpace(r) || unicode.IsPunct(r):
			if !lastHyphen {
				sb.WriteByte('-')
				lastHyphen = true
			}
		}
	}
	slug := strings.Trim(sb.String(), "-")
	if slug == "" {
		return fallback
	}
	return slug
}

// TrimSpaceRunes 截断字符串到最多 n 个 rune（用于评论摘要）
func TruncateRunes(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n])
}
