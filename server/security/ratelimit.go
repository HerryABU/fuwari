package security

import (
	"net"
	"sync"
	"time"

	"fuwari-server/utils"

	"github.com/gin-gonic/gin"
)

// 内存滑动窗口限流器：按客户端 IP 记录访问时间点。
// （单二进制博客服务，无需引入 Redis；参照 nvs security.RateLimit 的接口语义）

var (
	rateMu      sync.Mutex
	rateBuckets = make(map[string][]time.Time)
)

// rateGC 清理过期窗口记录（在每次访问检查时顺带执行）
func rateGC(now time.Time, window time.Duration) {
	for ip, hits := range rateBuckets {
		first := 0
		for first < len(hits) && now.Sub(hits[first]) > window {
			first++
		}
		if first == len(hits) {
			delete(rateBuckets, ip)
		} else if first > 0 {
			rateBuckets[ip] = append(hits[:0], hits[first:]...)
		}
	}
}

// RateLimit 限流中间件：窗口内每个客户端 IP 最多允许 limit 次请求。
// 超限返回 429（统一响应结构，业务码 1）。
func RateLimit(limit int, windowSec int) gin.HandlerFunc {
	window := time.Duration(windowSec) * time.Second
	return func(c *gin.Context) {
		ip := c.ClientIP()
		if host, _, err := net.SplitHostPort(c.Request.RemoteAddr); err == nil && host != "" {
			ip = host
		}
		now := time.Now()

		rateMu.Lock()
		rateGC(now, window)
		hits := rateBuckets[ip]
		// 统计窗口内命中数
		count := 0
		for _, t := range hits {
			if now.Sub(t) <= window {
				count++
			}
		}
		if count >= limit {
			rateMu.Unlock()
			c.JSON(429, utils.Response{
				Code:    utils.CodeBadRequest,
				Message: "请求过于频繁，请稍后再试",
				Data:    nil,
			})
			c.Abort()
			return
		}
		rateBuckets[ip] = append(hits, now)
		rateMu.Unlock()

		c.Next()
	}
}
