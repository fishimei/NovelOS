// Package middleware 提供 HTTP 中间件实现。
// 中间件在请求处理链中执行横切关注点，如请求 ID 追踪。
package middleware

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
)

// RequestIDKey 是上下文中的键，用于存储请求 ID。
const RequestIDKey = "request_id"

// RequestID 是 Gin 中间件，为每个请求生成或继承请求 ID。
// 功能说明：
// 1. 检查请求头是否包含 X-Request-Id
// 2. 如果有，保留该值（用于分布式追踪）
// 3. 如果没有，生成新的请求 ID（格式：req_<纳秒时间戳>）
// 4. 将请求 ID 存入 Gin 上下文（供后续处理使用）
// 5. 将请求 ID 写入响应头 X-Request-Id（让客户端能追踪）
// 6. 调用 c.Next() 执行后续处理链
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-Id")
		if requestID == "" {
			requestID = fmt.Sprintf("req_%d", time.Now().UnixNano())
		}

		c.Set(RequestIDKey, requestID)
		c.Writer.Header().Set("X-Request-Id", requestID)
		c.Next()
	}
}
