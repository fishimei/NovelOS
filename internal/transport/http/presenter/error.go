package presenter

import (
	"github.com/gin-gonic/gin"

	"github.com/fishimei/NovelOS/internal/pkgerr"
)

// ErrorBody 是错误响应体的结构。
type ErrorBody struct {
	Code    pkgerr.Code `json:"code"`
	Message string      `json:"message"`
}

// Error 发送统一格式的错误响应。
// 响应结构：{ error: { code: <错误码>, message: <错误信息> }, meta: { request_id: <请求ID> } }
// 参数说明：
// - c: Gin 上下文
// - err: 任意错误，会被 pkgerr.AsError 规范化为应用错误类型
// 错误码映射规则：
// - 验证错误 -> VALIDATION_ERROR (400)
// - 未找到 -> NOT_FOUND (404)
// - 冲突 -> CONFLICT (409)
// - 生成失败 -> GENERATION_FAILED (500)
// - 约束违反 -> CONSTRAINT_VIOLATION (422)
// - 内部错误 -> INTERNAL_ERROR (500)
// - 未实现 -> INTERNAL_ERROR (501)
func Error(c *gin.Context, err error) {
	appErr := pkgerr.AsError(err)
	c.JSON(appErr.Status, gin.H{
		"error": ErrorBody{
			Code:    appErr.Code,
			Message: appErr.Message,
		},
		"meta": Meta{RequestID: requestID(c)},
	})
}
