// Package domain 包含应用程序的领域模型常量。
// 定义了会话状态、运行状态和 SSE 事件名称的规范值。
// 这些常量确保整个系统中状态和事件命名的一致性。
package domain

// 故事会话状态常量
// 会话状态反映了作者与 AI 交互的当前阶段
const (
	SessionStatusIdle                 = "idle"                  // 空闲状态：会话初始状态，无进行中的操作
	SessionStatusAdvancing            = "advancing"             // 推进中：正在生成故事内容
	SessionStatusReviewing            = "reviewing"             // 审阅中：正在等待用户审阅生成的内容
	SessionStatusAwaitingConfirmation = "awaiting_confirmation" // 等待确认：对话 Agent 已提出可执行选项
	SessionStatusCommitted            = "committed"             // 已提交：内容已被确认并提交
	SessionStatusFailed               = "failed"                // 失败：操作未能成功完成
)

// 故事运行状态常量
// 运行状态反映了 AI 生成过程的当前步骤
const (
	RunStatusQueued                 = "queued"                   // 排队中：等待执行
	RunStatusLoadingState           = "loading_state"            // 加载状态：正在加载项目状态
	RunStatusPlanningActions        = "planning_actions"         // 规划动作：正在生成可确认的对话选项
	RunStatusExecutingAction        = "executing_action"         // 执行动作：正在执行已确认的对话选项
	RunStatusSelectingConflictAxis  = "selecting_conflict_axis"  // 选择冲突轴：正在确定戏剧冲突的核心
	RunStatusGeneratingPlotVariable = "generating_plot_variable" // 生成剧情变量：正在创建核心戏剧性选择
	RunStatusDrivingCharacterTurns  = "driving_character_turns"  // 推进角色回合：正在生成角色互动
	RunStatusWritingNarrative       = "writing_narrative"        // 书写叙事：正在撰写正文
	RunStatusCheckingContinuity     = "checking_continuity"      // 检查连续性：正在验证内容一致性
	RunStatusGeneratingMemoryPatch  = "generating_memory_patch"  // 生成记忆补丁：正在生成状态更新
	RunStatusReviewRequired         = "review_required"          // 需要审阅：等待用户确认
	RunStatusCompleted              = "completed"                // 已完成：对话回复已完成且无需确认
	RunStatusCommitted              = "committed"                // 已提交：内容已确认
	RunStatusFailed                 = "failed"                   // 失败：运行出错
	RunStatusCancelled              = "cancelled"                // 已取消：运行被终止
)

// 异步运行流式事件名称常量
// 这些事件通过 SSE (Server-Sent Events) 实时推送给客户端
const (
	EventGenerationStep            = "generation_step"             // 生成步骤事件：报告当前运行步骤
	EventStoryOrchestrationStarted = "story_orchestration_started" // 故事编排启动事件：传递用户 idea
	EventPlotVariable              = "plot_variable"               // 剧情变量事件：推送核心戏剧选择
	EventCharacterTurn             = "character_turn"              // 角色回合事件：推送角色互动内容
	EventDraftDelta                = "draft_delta"                 // 草稿增量事件：兼容旧正文片段事件
	EventReviewRequired            = "review_required"             // 需要审阅事件：通知用户需要确认
)

const (
	DialogueActionStatusPending   = "pending"
	DialogueActionStatusConfirmed = "confirmed"
	DialogueActionStatusExecuting = "executing"
	DialogueActionStatusExecuted  = "executed"
	DialogueActionStatusRejected  = "rejected"
	DialogueActionStatusFailed    = "failed"
)

const (
	DialogueActionSetupStartAndAdvance  = "setup.start_and_advance"
	DialogueActionSetupAdvance          = "setup.advance"
	DialogueActionSetupApply            = "setup.apply"
	DialogueActionStoryCreateAndAdvance = "story.create_and_advance"
	DialogueActionStoryAdvance          = "story.advance"
	DialogueActionStoryCommit           = "story.commit"
)
