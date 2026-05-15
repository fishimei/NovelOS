// Package service 包含应用程序的具体业务逻辑实现。
// 这些服务类封装了跨多个仓库的复杂业务操作，如设置运行应用和故事运行提交。
// 服务类遵循单一职责原则，每个服务负责一个特定的业务流程。
package service

import (
	"fmt"
	"time"

	"github.com/fishimei/NovelOS/internal/application/port"
)

// currentTime 获取当前时间的辅助函数。
// 如果提供了时钟接口则使用，否则使用系统时钟。
func currentTime(clock port.Clock) time.Time {
	if clock != nil {
		return clock.Now().UTC()
	}
	return systemClockImpl{}.Now()
}

// generatedID 生成唯一ID的辅助函数。
// 如果提供了ID生成器则使用，否则使用时间戳生成。
func generatedID(ids port.IDGenerator, clock port.Clock, prefix string) string {
	if ids != nil {
		return ids.New(prefix)
	}
	return fmt.Sprintf("%s_%d", prefix, currentTime(clock).UnixNano())
}

// firstNonEmpty 返回第一个非空字符串，如果都为空则返回默认值。
func firstNonEmpty(value string, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}
