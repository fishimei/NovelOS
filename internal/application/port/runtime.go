// Package port 定义了应用程序的端口接口（Ports and Adapters 架构）。
// 端口接口是应用层与基础设施层之间的契约，实现了依赖倒置原则。
// 主要包含：
// 1. 仓库接口：定义数据持久化的抽象
// 2. 运行时接口：定义时钟、ID生成器、事务管理器、事件流等基础设施抽象
package port

import (
	"context"
	"errors"
	"time"

	"github.com/fishimei/NovelOS/internal/application/model"
)

var ErrRunStopRequested = errors.New("run stop requested")

// GenerationEvent 是 AI 生成过程中的事件。
type GenerationEvent struct {
	ID       string // 可选的事件 ID；持久化 run_events 驱动 SSE 时可作为 SSE id
	Name     string // 事件名称
	Sequence int    // 可选的事件序号；对应持久化 RunEvent.Sequence
	Data     any    // 事件数据
}

// GenerationEventCursor 描述订阅生成事件时的恢复游标。
type GenerationEventCursor struct {
	AfterSequence int    // 只读取该序号之后的事件；0 表示从实时事件开始
	LastEventID   string // 原始 SSE Last-Event-ID，供适配层转换或透传
}

func (c GenerationEventCursor) IsZero() bool {
	return c.AfterSequence == 0 && c.LastEventID == ""
}

// GenerationEventStream 是生成事件流的接口，支持发布和订阅。
// 用于 SSE 实时推送 AI 生成进度。
type GenerationEventStream interface {
	// Publish 发布事件到指定运行
	Publish(ctx context.Context, runID string, event GenerationEvent) error
	// Subscribe 订阅指定运行的事件流，返回事件通道和取消函数
	Subscribe(ctx context.Context, runID string) (<-chan GenerationEvent, func(), error)
}

// GenerationEventStreamWithCursor 是可感知恢复游标的事件流扩展。
type GenerationEventStreamWithCursor interface {
	SubscribeAfter(ctx context.Context, runID string, cursor GenerationEventCursor) (<-chan GenerationEvent, func(), error)
}

func SubscribeGenerationEvents(ctx context.Context, stream GenerationEventStream, runID string, cursor GenerationEventCursor) (<-chan GenerationEvent, func(), error) {
	if !cursor.IsZero() {
		if cursorStream, ok := stream.(GenerationEventStreamWithCursor); ok {
			return cursorStream.SubscribeAfter(ctx, runID, cursor)
		}
	}
	return stream.Subscribe(ctx, runID)
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
	Run               model.StoryRun
	Session           model.StorySession
	World             model.WorldSnapshot
	WakeCharacterIDs  []string
	InFlightActions   []model.OngoingAction
	CompletedActions  []model.OngoingAction
	SupersededActions []model.OngoingAction
	CollisionAt       time.Time
}

type CharacterActionDecider interface {
	Decide(ctx context.Context, input model.CharacterActionDecisionInput) (model.CharacterActionDecision, error)
}

type LocationInspectionService interface {
	EnsureReachableLocations(ctx context.Context, input model.LocationReachabilityInput) (model.LocationInspectionContext, error)
	InspectLocation(ctx context.Context, input model.LocationInspectionInput) (model.LocationInspectionResult, error)
}

type LocationSubdivisionGenerator interface {
	GenerateLocationSubdivision(ctx context.Context, input model.LocationSubdivisionInput) (model.LocationSubdivisionPlan, error)
}

type WorldInitializationInput struct {
	ProjectID     string
	Seed          string
	LocationCount int
	MapWidth      int
	MapHeight     int
	SetupRun      model.SetupRun
	SetupDraft    model.SetupDraft
	Characters    []model.Character
	CurrentTime   time.Time
}

type WorldInitializationResult struct {
	Map             model.WorldMap
	Areas           []model.MapArea
	Tiles           []model.MapTile
	Locations       []model.LocationState
	Factions        []model.FactionInfluence
	CharacterStates []model.CharacterRuntimeState
	Snapshot        model.WorldSnapshot
}

type WorldInitializer interface {
	Initialize(ctx context.Context, input WorldInitializationInput) (WorldInitializationResult, error)
}

type DialogueRunGenerator interface {
	Generate(ctx context.Context, input DialogueRunGenerationInput) (model.DialogueRunResult, error)
}

type DialogueRunGenerationInput struct {
	Run     model.DialogueRun
	Session model.DialogueSession
}

type DialogueActionExecutor interface {
	ExecuteConfirmed(ctx context.Context, optionID string, input model.ExecuteDialogueActionInput) (model.DialogueActionOption, error)
	ExecuteAutoApproved(ctx context.Context, optionID string, input model.AutoExecuteDialogueActionInput) (model.DialogueActionOption, error)
}

type DialogueActionOptionValidator interface {
	ValidateOption(ctx context.Context, option model.DialogueActionOption) error
}

type CharacterMemoryRecallInput struct {
	ProjectID   string
	CharacterID string
	Query       string
	Limit       int
}

type CharacterMemoryCommitInput struct {
	ProjectID string
	RunID     string
	Chapter   model.Chapter
	Memories  []model.Memory
}

type CharacterMemoryService interface {
	Recall(ctx context.Context, input CharacterMemoryRecallInput) ([]model.Memory, error)
	Commit(ctx context.Context, input CharacterMemoryCommitInput) error
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
