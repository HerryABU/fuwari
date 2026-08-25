package handlers

import (
	"fuwari-server/version"

	"github.com/gin-gonic/gin"
)

// CollectHealth 汇总服务健康状态
func CollectHealth() gin.H {
	return gin.H{
		"service": "fuwari-server",
		"version": version.AppVersion,
		"status":  "ok",
	}
}
