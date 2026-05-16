package presenter

import (
	"github.com/gin-gonic/gin"

	"github.com/fishimei/NovelOS/internal/transport/http/middleware"
)

// Package presenter 负责封装 HTTP 响应的统一格式。
// 所有 API 响应都使用标准信封结构：
// - 成功响应：{ data: ..., meta: { request_id: ... } }
// - 错误响应：{ error: { code: ..., message: ... }, meta: { request_id: ... } }
// 这种统一格式确保客户端可以一致地处理所有响应。

// Meta 是响应的元数据结构，包含请求追踪信息。
type Meta struct {
	RequestID string `json:"request_id"`
}

// Pagination 包含分页相关元数据。
type Pagination struct {
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
	Total    int `json:"total"`
}

// PaginatedMeta 是带分页的元数据结构。
type PaginatedMeta struct {
	RequestID  string     `json:"request_id"`
	Pagination Pagination `json:"pagination"`
}

// requestID 从 Gin 上下文中提取请求 ID。
// 尝试从中间件设置的值中获取，如果不存在则返回空字符串。
func requestID(c *gin.Context) string {
	if value, ok := c.Get(middleware.RequestIDKey); ok {
		if requestID, ok := value.(string); ok {
			return requestID
		}
	}

	return ""
}

// Data 发送标准成功响应（JSON 格式）。
// 响应结构：{ data: <实际数据>, meta: { request_id: <请求ID> } }
// 参数说明：
// - c: Gin 上下文
// - status: HTTP 状态码（如 200、201、202）
// - data: 要返回的实际数据（任意类型）
func Data(c *gin.Context, status int, data any) {
	c.JSON(status, gin.H{
		"data": data,
		"meta": Meta{RequestID: requestID(c)},
	})
}

// PaginatedData 发送带分页信息的成功响应（JSON 格式）。
// 响应结构：{ data: <数据列表>, meta: { request_id: ..., pagination: { page, pageSize, total } } }
// 参数说明：
// - c: Gin 上下文
// - status: HTTP 状态码（通常为 200）
// - data: 要返回的数据列表
// - page: 当前页码
// - pageSize: 每页条数
// - total: 总条数
func PaginatedData(c *gin.Context, status int, data any, page, pageSize, total int) {
	c.JSON(status, gin.H{
		"data": data,
		"meta": PaginatedMeta{
			RequestID: requestID(c),
			Pagination: Pagination{
				Page:     page,
				PageSize: pageSize,
				Total:    total,
			},
		},
	})
}
