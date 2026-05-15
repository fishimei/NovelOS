// Package port 定义了应用程序的端口接口（Ports and Adapters 架构）。
// 端口接口是应用层与基础设施层之间的契约，实现了依赖倒置原则。
// 主要包含：
// 1. 仓库接口：定义数据持久化的抽象
// 2. 运行时接口：定义时钟、ID生成器、事务管理器、事件流等基础设施抽象
package port

import (
	"context"
	"time"

	"github.com/fishimei/NovelOS/internal/application/model"
)

// GenerationEvent 是 AI 生成过程中的事件。
type GenerationEvent struct {
	Name string // 事件名称
	Data any    // 事件数据
}

// GenerationEventStream 是生成事件流的接口，支持发布和订阅。
// 用于 SSE 实时推送 AI 生成进度。
type GenerationEventStream interface {
	// Publish 发布事件到指定运行
	Publish(ctx context.Context, runID string, event GenerationEvent) error
	// Subscribe 订阅指定运行的事件流，返回事件通道和取消函数
	Subscribe(ctx context.Context, runID string) (<-chan GenerationEvent, func(), error)
}

type SetupRunGenerator interface {
	Generate(ctx context.Context, input SetupRunGenerationInput) (model.SetupRunResult, error)
}

type SetupRunGenerationInput struct {
	Run     model.SetupRun
	Session model.SetupSession
}

type StoryRunGenerator interface {
	Generate(ctx context.Context, input StoryRunGenerationInput) (model.StoryRunResult, error)
}

type StoryRunGenerationInput struct {
	Run     model.StoryRun
	Session model.StorySession
}

// TxManager 是事务管理器的接口。
// 封装数据库事务逻辑，确保跨多个仓库操作的一致性。
type TxManager interface {
	// WithinTransaction 在事务中执行指定函数
	WithinTransaction(ctx context.Context, fn func(context.Context) error) error
}

// Clock 是时钟接口，用于获取当前时间。
// 主要用于测试时注入可控的时间源。
type Clock interface {
	// Now 返回当前时间
	Now() time.Time
}

// IDGenerator 是 ID 生成器接口。
// 负责生成全局唯一的业务 ID。
type IDGenerator interface {
	// New 生成指定前缀的新 ID
	New(prefix string) string
}
