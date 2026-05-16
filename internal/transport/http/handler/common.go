package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/fishimei/NovelOS/internal/application/model"
	"github.com/fishimei/NovelOS/internal/pkgerr"
)

// 辅助函数组：负责 HTTP 请求数据的绑定与规范化。
// 这些函数仅属于 HTTP 适配器层，不包含任何业务逻辑：
// - bindJSON：将 JSON 请求体绑定到目标结构体，绑定失败时记录验证错误
// - bindQuery：将 URL 查询参数绑定到目标结构体，绑定失败时记录验证错误
// - normalizePageInput：将分页参数规范化为有效值（page >= 1, pageSize >= 1）
// - normalizeLimit：将列表限制值规范化为有效值（limit >= 1）
// 注意：更深层次的业务验证应放在仓储或具体应用用例中，而不是放在 HTTP 适配器里。

// bindJSON 从请求体解析 JSON 数据到目标结构体。
// 解析失败时返回 false，并将验证错误附加到 Gin 上下文。
func bindJSON(c *gin.Context, target any) bool {
	if err := c.ShouldBindJSON(target); err != nil {
		_ = c.Error(pkgerr.Validation(err.Error()))
		return false
	}
	return true
}

// bindQuery 从 URL 查询参数解析数据到目标结构体。
// 解析失败时返回 false，并将验证错误附加到 Gin 上下文。
func bindQuery(c *gin.Context, target any) bool {
	if err := c.ShouldBindQuery(target); err != nil {
		_ = c.Error(pkgerr.Validation(err.Error()))
		return false
	}
	return true
}

// normalizePageInput 将分页参数规范化为有效值。
// 具体规则如下：
// - 如果 page <= 0，设置为 1（首页）
// - 如果 pageSize <= 0，设置为 20（默认每页条数）
// 返回规范化的 PageInput 结构体。
func normalizePageInput(page, pageSize int) model.PageInput {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	return model.PageInput{Page: page, PageSize: pageSize}
}

// normalizeLimit 将列表限制值规范化为有效值。
// 具体规则如下：
// - 如果 limit <= 0，设置为 20（默认限制）
// - 否则返回原始值
// 用于确保列表查询不会返回过多或过少的结果。
func normalizeLimit(limit int) int {
	if limit <= 0 {
		return 20
	}
	return limit
}
