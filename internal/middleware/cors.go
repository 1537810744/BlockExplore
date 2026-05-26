// Package middleware 提供 Gin 中间件
// 中间件在请求到达 Handler 之前/之后执行，用于通用处理
package middleware

import (
	"github.com/gin-gonic/gin"
)

// CORS 跨域资源共享中间件
// 允许前端跨域访问 API
// 浏览器出于安全考虑，会限制跨域请求，此中间件添加必要的响应头
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 允许所有来源访问（生产环境应该限制为具体域名）
		c.Header("Access-Control-Allow-Origin", "*")
		// 允许的 HTTP 方法
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		// 允许的请求头
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization, X-Request-ID")
		// 允许暴露的响应头
		c.Header("Access-Control-Expose-Headers", "X-Request-ID")
		// 预检请求缓存时间（秒）
		c.Header("Access-Control-Max-Age", "86400")

		// 处理预检请求（OPTIONS 方法）
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204) // 返回 204 No Content
			return
		}

		c.Next() // 继续处理下一个中间件或 Handler
	}
}
