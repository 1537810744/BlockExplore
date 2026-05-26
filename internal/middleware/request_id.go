package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// RequestID 请求 ID 中间件
// 为每个请求生成唯一的 UUID，用于链路追踪和日志关联
// 如果请求头中已包含 X-Request-ID，则复用该值
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从请求头获取请求 ID
		requestID := c.GetHeader("X-Request-ID")

		// 如果没有，生成新的 UUID
		if requestID == "" {
			requestID = uuid.New().String()
		}

		// 存储到上下文中，供后续 Handler 使用
		c.Set("request_id", requestID)

		// 添加到响应头
		c.Header("X-Request-ID", requestID)

		c.Next()
	}
}
