// Package model 定义了应用程序的核心领域模型。
// 这些模型是业务逻辑的核心表示，独立于任何特定的持久化或传输机制。
// 模型分为以下几个主要类别：
// 1. 项目管理：Project, ProjectDetail
// 2. 世界观设定：AuthorBible, WorldStateEntry
// 3. 角色系统：Character, CreateCharacterInput, UpdateCharacterInput
// 4. 关系系统：Relationship, RelationshipPair, RelationshipView, RelationshipEvent
// 5. 设置流程：SetupSession, SetupRun, SetupDraft, SetupQuestion
// 6. 故事流程：StorySession, StoryRun, Draft, PlotVariable, ReviewReport, MemoryPatch
// 7. 章节与记忆：Chapter, Memory
package model

import "time"

// PageInput 定义分页查询的输入参数。
type PageInput struct {
	Page     int // 当前页码，从 1 开始
	PageSize int // 每页条数
}

// ListResult 是通用分页结果包装器。
// 用于返回带有总数统计的分页数据。
type ListResult[T any] struct {
	Items []T // 当前页的数据项
	Total int // 符合查询条件的总条数
}

// CreateProjectInput 是创建新项目的输入参数。
type CreateProjectInput struct {
	Title       string // 项目标题
	Genre       string // 作品类型/体裁
	Description string // 项目描述
}

// UpdateProjectInput 是更新项目信息的输入参数。
type UpdateProjectInput struct {
	Title       string // 项目标题
	Genre       string // 作品类型/体裁
	Description string // 项目描述
}

// Project 是小说的基本项目单元。
type Project struct {
	ID          string    // 项目唯一标识符
	Title       string    // 项目标题
	Genre       string    // 作品类型/体裁
	Description string    // 项目描述
	CreatedAt   time.Time // 创建时间
	UpdatedAt   time.Time // 最后更新时间
}

// ProjectDetail 是项目的详细信息，包含关联实体的统计。
type ProjectDetail struct {
	Project                        // 嵌入 Project 的所有字段
	CharacterCount             int // 项目中的角色数量
	RelationshipCount          int // 项目中的关系数量
	StorySessionCount          int // 项目中的故事会话数量
	LastCommittedChapterNumber int // 最后一个已提交章节的编号
}

// WorldStateEntry 代表世界状态中的一个条目，用于跟踪故事世界中的关键信息。
// 世界状态是 AI 在生成内容时需要参考的背景知识库。
type WorldStateEntry struct {
	ID         string    // 条目唯一标识符
	ProjectID  string    // 所属项目 ID
	Key        string    // 状态键（如 "king_name", "current_year"）
	Value      any       // 状态值（可以是任意类型）
	Note       string    // 备注说明
	Status     string    // 状态标识
	Importance int       // 重要性等级（影响 AI 生成时的权重）
	Volatility int       // 变化频率（高 volatility 表示该状态可能频繁变化）
	UpdatedAt  time.Time // 最后更新时间
}

// UpdateAuthorBibleInput 是更新作者圣经的输入参数。
// 作者圣经是指导 AI 生成风格和规则的元数据集合。
type UpdateAuthorBibleInput struct {
	Theme               string            // 作品主题
	StyleGuide          string            // 风格指南
	WorldRules          []string          // 世界规则列表
	AestheticPrinciples []string          // 审美原则列表
	HardConstraints     []string          // 硬性约束（必须遵守的规则）
	SoftPreferences     []string          // 软性偏好（尽量遵守的规则）
	ForbiddenMoves      []string          // 禁止行为
	InitialWorldState   []WorldStateEntry // 初始世界状态
}

// AuthorBible 是作者圣经的完整模型，包含指导 AI 创作的元数据。
type AuthorBible struct {
	ID                  string            // 圣经唯一标识符
	ProjectID           string            // 所属项目 ID
	Theme               string            // 作品主题
	StyleGuide          string            // 风格指南
	WorldRules          []string          // 世界规则列表
	AestheticPrinciples []string          // 审美原则列表
	HardConstraints     []string          // 硬性约束
	SoftPreferences     []string          // 软性偏好
	ForbiddenMoves      []string          // 禁止行为
	InitialWorldState   []WorldStateEntry // 初始世界状态
	Status              string            // 状态标识
	UpdatedAt           time.Time         // 最后更新时间
}

// CreateCharacterInput 是创建新角色的输入参数。
type CreateCharacterInput struct {
	Name        string   // 角色名称
	Role        string   // 角色定位（如 protagonist, antagonist, supporting）
	Profile     string   // 角色简介
	Personality string   // 性格特征描述
	VoiceStyle  string   // 说话风格
	Goals       []string // 角色目标列表
	Fears       []string // 角色恐惧列表
	Secrets     []string // 角色秘密列表
	Constraints []string // 角色行为约束列表
}

// UpdateCharacterInput 是更新角色信息的输入参数。
type UpdateCharacterInput struct {
	Name        string
	Role        string
	Profile     string
	Personality string
	VoiceStyle  string
	Goals       []string
	Fears       []string
	Secrets     []string
	Constraints []string
}

// Character 是故事中的角色实体。
// 角色是故事生成的核心要素，AI 会根据角色的目标、恐惧和约束来生成符合角色特点的对话和行为。
type Character struct {
	ID                  string    // 角色唯一标识符
	ProjectID           string    // 所属项目 ID
	Name                string    // 角色名称
	Role                string    // 角色定位
	Profile             string    // 角色简介
	Personality         string    // 性格特征描述
	VoiceStyle          string    // 说话风格
	Goals               []string  // 角色目标列表
	Fears               []string  // 角色恐惧列表
	Secrets             []string  // 角色秘密列表
	Constraints         []string  // 角色行为约束列表
	RecentMemorySummary string    // 近期记忆摘要（用于 AI 生成时的上下文）
	Status              string    // 状态标识
	CreatedAt           time.Time // 创建时间
	UpdatedAt           time.Time // 最后更新时间
}

// CreateRelationshipInput 是创建角色关系的输入参数。
// 关系描述了两个角色之间的互动模式和历史。
type CreateRelationshipInput struct {
	CharacterAID   string                // 角色 A 的 ID
	CharacterBID   string                // 角色 B 的 ID
	Summary        string                // 关系概要描述
	Anchors        []string              // 关系锚点（关系中的关键事件）
	TensionPoints  []string              // 紧张点（可能导致冲突的因素）
	SharedHistory  []string              // 共同经历
	Volatility     int                   // 关系变化程度
	CharacterAView RelationshipViewInput // 角色 A 对关系的看法
	CharacterBView RelationshipViewInput // 角色 B 对关系的看法
}

// UpdateRelationshipInput 是更新关系信息的输入参数。
type UpdateRelationshipInput struct {
	Summary       string   // 关系概要描述
	Anchors       []string // 关系锚点
	TensionPoints []string // 紧张点
	SharedHistory []string // 共同经历
	Volatility    int      // 关系变化程度
}

// RelationshipViewInput 是关系视角的输入参数，描述单方对关系的认知。
type RelationshipViewInput struct {
	PublicAttitude         string // 公开态度（他人可见的态度）
	PrivateAttitude        string // 私下态度（真实态度）
	BelievedTargetAttitude string // 以为对方的态度
	MaskingStrategy        string // 掩饰策略
}

// RelationshipPair 代表两个角色之间的关系对。
// 关系对是关系的基础实体，存储双方共同的信息。
type RelationshipPair struct {
	ID               string    // 关系对唯一标识符
	ProjectID        string    // 所属项目 ID
	LeftCharacterID  string    // 左侧角色 ID（角色 A）
	RightCharacterID string    // 右侧角色 ID（角色 B）
	Summary          string    // 关系概要描述
	Anchors          []string  // 关系锚点
	TensionPoints    []string  // 紧张点
	SharedHistory    []string  // 共同经历
	Volatility       int       // 关系变化程度
	Status           string    // 状态标识
	CreatedAt        time.Time // 创建时间
	UpdatedAt        time.Time // 最后更新时间
}

// RelationshipView 代表角色对关系的单方视角。
// 每个 RelationshipPair 会有两个 RelationshipView，分别描述两个角色对这段关系的认知。
type RelationshipView struct {
	ID                     string    // 视角唯一标识符
	ProjectID              string    // 所属项目 ID
	PairID                 string    // 所属关系对 ID
	SourceCharacterID      string    // 视角来源角色 ID
	TargetCharacterID      string    // 视角目标角色 ID
	PublicAttitude         string    // 公开态度
	PrivateAttitude        string    // 私下态度
	BelievedTargetAttitude string    // 以为对方的态度
	MaskingStrategy        string    // 掩饰策略
	Status                 string    // 状态标识
	CreatedAt              time.Time // 创建时间
	UpdatedAt              time.Time // 最后更新时间
}

// RelationshipEvent 代表关系中发生的事件，用于跟踪关系的变化历史。
type RelationshipEvent struct {
	ID        string         // 事件唯一标识符
	ProjectID string         // 所属项目 ID
	PairID    string         // 所属关系对 ID
	EventType string         // 事件类型
	Summary   string         // 事件摘要
	Payload   map[string]any // 事件附加数据
	CreatedAt time.Time      // 事件发生时间
}

// Relationship 是关系的完整聚合，包含关系对、双方视角和最近事件。
type Relationship struct {
	Pair           RelationshipPair    // 关系对基础信息
	Views          []RelationshipView  // 所有相关视角
	RecentEvents   []RelationshipEvent // 最近发生的事件
	CharacterAView *RelationshipView   // 角色 A 的视角
	CharacterBView *RelationshipView   // 角色 B 的视角
}

// CreateSetupSessionInput 是创建设置会话的输入参数。
// 设置会话用于通过 AI 辅助将粗略想法转化为结构化项目状态。
type CreateSetupSessionInput struct {
	SeedIdea string // 种子想法/初始概念
}

// AdvanceSetupSessionInput 是推进设置会话的输入参数。
type AdvanceSetupSessionInput struct {
	UserMessage string // 用户输入的消息
}

// ApplySetupRunInput 是应用设置运行结果的输入参数。
// 用户可以选择性地接受生成的各项内容。
type ApplySetupRunInput struct {
	RunID               string // 要应用的运行 ID
	AcceptAuthorBible   bool   // 是否接受作者圣经
	AcceptCharacters    bool   // 是否接受角色
	AcceptRelationships bool   // 是否接受关系
	AcceptWorldState    bool   // 是否接受世界状态
	AuthorNote          string // 作者备注
}

// SetupQuestion 是设置过程中 AI 提出的问题。
type SetupQuestion struct {
	Key          string // 问题标识键
	Question     string // 问题内容
	WhyItMatters string // 为什么这个问题重要
}

// ConversationMessage 是会话消息的模型。
type ConversationMessage struct {
	ID        string    // 消息唯一标识符
	SessionID string    // 所属会话 ID
	Role      string    // 消息角色（user/assistant）
	Content   string    // 消息内容
	CreatedAt time.Time // 创建时间
}

// SetupSession 是设置会话的模型。
// 设置会话是作者与 AI 之间的多轮对话，用于逐步构建项目的基础设定。
type SetupSession struct {
	ID              string                // 会话唯一标识符
	ProjectID       string                // 所属项目 ID
	SeedIdea        string                // 种子想法
	LastUserMessage string                // 最后一条用户消息
	Status          string                // 会话状态
	Messages        []ConversationMessage // 会话消息历史
	CreatedAt       time.Time             // 创建时间
	UpdatedAt       time.Time             // 最后更新时间
}

// SetupRun 是设置运行的模型。
// 每次用户推进设置会话时创建一个运行，跟踪 AI 处理进度。
type SetupRun struct {
	RunID       string    // 运行唯一标识符
	SessionID   string    // 所属会话 ID
	ProjectID   string    // 所属项目 ID
	Status      string    // 运行状态
	CurrentStep string    // 当前步骤
	Progress    int       // 进度百分比
	CreatedAt   time.Time // 创建时间
	UpdatedAt   time.Time // 最后更新时间
}

// SetupDraft 是设置生成的草稿。
// 包含 AI 生成的完整项目设定。
type SetupDraft struct {
	AuthorBible      AuthorBible       // 作者圣经
	Characters       []Character       // 角色列表
	Relationships    []Relationship    // 关系列表
	WorldState       []WorldStateEntry // 世界状态
	OpenQuestions    []SetupQuestion   // 待解答问题
	AssistantSummary string            // AI 总结
}

// SetupRunResult 是设置运行结果的模型。
type SetupRunResult struct {
	RunID      string     // 运行 ID
	SessionID  string     // 会话 ID
	Status     string     // 结果状态
	SetupDraft SetupDraft // 设置草稿
}

// ApplySetupRunResult 是应用设置运行后的结果。
type ApplySetupRunResult struct {
	ProjectID string // 项目 ID
	RunID     string // 运行 ID
	Status    string // 结果状态
}

// CreateStorySessionInput 是创建故事会话的输入参数。
// 故事会话用于 AI 辅助生成故事内容。
type CreateStorySessionInput struct {
	Title            string // 章节/故事标题
	OpeningSituation string // 开局情境
	AuthorIntent     string // 作者意图
}

// AdvanceStorySessionInput 是推进故事会话的输入参数。
type AdvanceStorySessionInput struct {
	AuthorMessage string // 作者输入的消息
}

// CommitStoryRunInput 是提交故事运行的输入参数。
type CommitStoryRunInput struct {
	DraftID       string // 草稿 ID
	MemoryPatchID string // 记忆补丁 ID
	AuthorNote    string // 作者备注
}

// StorySession 是故事会话的模型。
// 故事会话是单个故事生成周期的上下文容器。
type StorySession struct {
	ID                         string                // 会话唯一标识符
	ProjectID                  string                // 所属项目 ID
	Title                      string                // 会话标题
	OpeningSituation           string                // 开局情境
	AuthorIntent               string                // 作者意图
	LastAuthorMessage          string                // 最后一条作者消息
	Status                     string                // 会话状态
	CurrentPlotVariableSummary string                // 当前剧情变量摘要
	Messages                   []ConversationMessage // 会话消息历史
	CreatedAt                  time.Time             // 创建时间
	UpdatedAt                  time.Time             // 最后更新时间
}

// StoryRun 是故事运行的模型。
// 每次用户推进故事会话时创建一个运行，跟踪 AI 内容生成进度。
type StoryRun struct {
	RunID       string     // 运行唯一标识符
	SessionID   string     // 所属会话 ID
	ProjectID   string     // 所属项目 ID
	Status      string     // 运行状态
	CurrentStep string     // 当前步骤
	Progress    int        // 进度百分比
	CommittedAt *time.Time // 提交时间（如果有）
	CreatedAt   time.Time  // 创建时间
	UpdatedAt   time.Time  // 最后更新时间
}

// Draft 是生成的章节草稿。
type Draft struct {
	ID            string // 草稿唯一标识符
	Title         string // 章节标题
	ChapterNumber int    // 章节编号
	Content       string // 章节正文
	Summary       string // 章节摘要
	WordCount     int    // 字数统计
}

// PlotVariable 是剧情变量，定义故事中的核心戏剧性选择。
// AI 在生成内容前会确定剧情变量，为角色提供有意义的选择。
type PlotVariable struct {
	PressureSource      string   // 压力来源
	FocalCharacterID    string   // 核心角色 ID
	CoreChoice          string   // 核心选择描述
	OptionA             string   // 选项 A
	OptionB             string   // 选项 B
	CostA               string   // 选择 A 的代价
	CostB               string   // 选择 B 的代价
	IrreversibleEffect  string   // 不可逆影响
	RelatedCharacterIDs []string // 相关角色 ID 列表
	WorldStatePressure  []string // 世界状态压力
}

// ReviewReport 是审阅报告，包含 AI 对生成内容的质量评估。
type ReviewReport struct {
	Pass             bool     // 是否通过
	HardViolations   []string // 硬性违规（必须修复的问题）
	ContinuityIssues []string // 连续性问题
	StyleIssues      []string // 风格问题
	SuggestedFixes   []string // 建议修复
}

// CharacterMemoryUpdate 是角色记忆更新。
type CharacterMemoryUpdate struct {
	CharacterID string // 角色 ID
	Type        string // 更新类型
	Content     string // 更新内容
	Importance  int    // 重要性
}

// RelationshipViewUpdate 是关系视角更新。
type RelationshipViewUpdate struct {
	ViewID                 string // 视角 ID
	PairID                 string // 关系对 ID
	SourceCharacterID      string // 源角色 ID
	TargetCharacterID      string // 目标角色 ID
	PublicAttitude         string // 公开态度
	PrivateAttitude        string // 私下态度
	BelievedTargetAttitude string // 以为对方的态度
	MaskingStrategy        string // 掩饰策略
}

// RelationshipUpdate 是关系更新。
type RelationshipUpdate struct {
	PairID       string                   // 关系对 ID
	Summary      string                   // 更新后的摘要
	TensionDelta string                   // 紧张度变化
	Pair         *RelationshipPair        // 更新的关系对（可选）
	Views        []RelationshipViewUpdate // 更新的视角列表
	Events       []RelationshipEvent      // 新增的事件列表
}

// WorldStateUpdate 是世界状态更新。
type WorldStateUpdate struct {
	Key       string // 状态键
	Operation string // 操作类型（set, update, delete）
	Value     any    // 新值
	Note      string // 备注说明
}

// MemoryPatch 是记忆补丁，封装了所有状态更新。
// 当故事运行被提交时，相关的状态变化会通过记忆补丁统一应用。
type MemoryPatch struct {
	ID                     string                  // 补丁唯一标识符
	Status                 string                  // 补丁状态
	CharacterMemoryUpdates []CharacterMemoryUpdate // 角色记忆更新列表
	RelationshipUpdates    []RelationshipUpdate    // 关系更新列表
	WorldStateUpdates      []WorldStateUpdate      // 世界状态更新列表
}

// StoryRunResult 是故事运行结果的模型。
type StoryRunResult struct {
	RunID        string       // 运行 ID
	SessionID    string       // 会话 ID
	Status       string       // 结果状态
	PlotVariable PlotVariable // 剧情变量
	Draft        Draft        // 章节草稿
	Review       ReviewReport // 审阅报告
	MemoryPatch  MemoryPatch  // 记忆补丁
}

// CommitStoryRunResult 是提交故事运行后的结果。
type CommitStoryRunResult struct {
	Chapter  Chapter     // 提交的章节
	Patch    MemoryPatch // 应用的记忆补丁
	StoryRun StoryRun    // 关联的故事运行
}

// Chapter 是已提交的故事章节。
// 章节是故事的基本单位，每个提交的故事运行会产生一个新的章节。
type Chapter struct {
	ID            string    // 章节唯一标识符
	ProjectID     string    // 所属项目 ID
	ChapterNumber int       // 章节编号（递增）
	Title         string    // 章节标题
	Summary       string    // 章节摘要
	Content       string    // 章节正文
	AuthorNote    string    // 作者备注
	Status        string    // 章节状态
	WordCount     int       // 字数统计
	CommittedAt   time.Time // 提交时间
}

// CreateMemoryInput 是创建角色记忆的输入参数。
type CreateMemoryInput struct {
	Content    string // 记忆内容
	Importance int    // 重要性等级
	Note       string // 备注说明
}

// Memory 是角色的记忆条目。
// 记忆是角色在故事中经历事件的记录，用于 AI 生成时的上下文参考。
type Memory struct {
	ID              string    // 记忆唯一标识符
	CharacterID     string    // 所属角色 ID
	Content         string    // 记忆内容
	SourceChapterID string    // 来源章节 ID
	Importance      int       // 重要性等级
	Status          string    // 状态标识
	CreatedAt       time.Time // 创建时间
}

// RunEvent 是运行事件，用于记录 AI 生成过程中的关键事件。
// 事件通过 SSE 实时推送给客户端，并持久化用于审计。
type RunEvent struct {
	ID        string         // 事件唯一标识符
	RunKind   string         // 运行类型（setup/story）
	RunID     string         // 所属运行 ID
	EventName string         // 事件名称
	Sequence  int            // 事件序号
	Payload   map[string]any // 事件附加数据
	CreatedAt time.Time      // 事件发生时间
}

// StateRevision 是状态修订快照。
// 在重要状态变更前保存快照，用于回滚或审计。
type StateRevision struct {
	ID          string         // 修订唯一标识符
	ProjectID   string         // 所属项目 ID
	EntityType  string         // 实体类型（如 character, relationship）
	EntityID    string         // 实体 ID
	SourceRunID string         // 来源运行 ID
	Snapshot    map[string]any // 状态快照数据
	CreatedAt   time.Time      // 修订时间
}
