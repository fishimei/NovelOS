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
	ID          string    `json:"id"`          // 项目唯一标识符
	Title       string    `json:"title"`       // 项目标题
	Genre       string    `json:"genre"`       // 作品类型/体裁
	Description string    `json:"description"` // 项目描述
	CreatedAt   time.Time `json:"created_at"`  // 创建时间
	UpdatedAt   time.Time `json:"updated_at"`  // 最后更新时间
}

// ProjectDetail 是项目的详细信息，包含关联实体的统计。
type ProjectDetail struct {
	Project
	CharacterCount             int `json:"character_count"`               // 项目中的角色数量
	RelationshipCount          int `json:"relationship_count"`            // 项目中的关系数量
	StorySessionCount          int `json:"story_session_count"`           // 项目中的故事会话数量
	LastCommittedChapterNumber int `json:"last_committed_chapter_number"` // 最后一个已提交章节的编号
}

// WorldStateEntry 代表世界状态中的一个条目，用于跟踪故事世界中的关键信息。
// 世界状态是 AI 在生成内容时需要参考的背景知识库。
type WorldStateEntry struct {
	ID         string    `json:"id"`         // 条目唯一标识符
	ProjectID  string    `json:"project_id"` // 所属项目 ID
	Key        string    `json:"key"`        // 状态键（如 "king_name", "current_year"）
	Value      any       `json:"value"`      // 状态值（可以是任意类型）
	Note       string    `json:"note"`       // 备注说明
	Status     string    `json:"status"`     // 状态标识
	Importance int       `json:"importance"` // 重要性等级（影响 AI 生成时的权重）
	Volatility int       `json:"volatility"` // 变化频率（高 volatility 表示该状态可能频繁变化）
	UpdatedAt  time.Time `json:"updated_at"` // 最后更新时间
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
	ID                  string            `json:"id"`                   // 圣经唯一标识符
	ProjectID           string            `json:"project_id"`           // 所属项目 ID
	Theme               string            `json:"theme"`                // 作品主题
	StyleGuide          string            `json:"style_guide"`          // 风格指南
	WorldRules          []string          `json:"world_rules"`          // 世界规则列表
	AestheticPrinciples []string          `json:"aesthetic_principles"` // 审美原则列表
	HardConstraints     []string          `json:"hard_constraints"`     // 硬性约束
	SoftPreferences     []string          `json:"soft_preferences"`     // 软性偏好
	ForbiddenMoves      []string          `json:"forbidden_moves"`      // 禁止行为
	InitialWorldState   []WorldStateEntry `json:"initial_world_state"`  // 初始世界状态
	Status              string            `json:"status"`               // 状态标识
	UpdatedAt           time.Time         `json:"updated_at"`           // 最后更新时间
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
	ID                  string    `json:"id"`                    // 角色唯一标识符
	ProjectID           string    `json:"project_id"`            // 所属项目 ID
	Name                string    `json:"name"`                  // 角色名称
	Role                string    `json:"role"`                  // 角色定位
	Profile             string    `json:"profile"`               // 角色简介
	Personality         string    `json:"personality"`           // 性格特征描述
	VoiceStyle          string    `json:"voice_style"`           // 说话风格
	Goals               []string  `json:"goals"`                 // 角色目标列表
	Fears               []string  `json:"fears"`                 // 角色恐惧列表
	Secrets             []string  `json:"secrets"`               // 角色秘密列表
	Constraints         []string  `json:"constraints"`           // 角色行为约束列表
	RecentMemorySummary string    `json:"recent_memory_summary"` // 近期记忆摘要（用于 AI 生成时的上下文）
	Status              string    `json:"status"`                // 状态标识
	CreatedAt           time.Time `json:"created_at"`            // 创建时间
	UpdatedAt           time.Time `json:"updated_at"`            // 最后更新时间
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
	ID               string    `json:"id"`                 // 关系对唯一标识符
	ProjectID        string    `json:"project_id"`         // 所属项目 ID
	LeftCharacterID  string    `json:"left_character_id"`  // 左侧角色 ID（角色 A）
	RightCharacterID string    `json:"right_character_id"` // 右侧角色 ID（角色 B）
	Summary          string    `json:"summary"`            // 关系概要描述
	Anchors          []string  `json:"anchors"`            // 关系锚点
	TensionPoints    []string  `json:"tension_points"`     // 紧张点
	SharedHistory    []string  `json:"shared_history"`     // 共同经历
	Volatility       int       `json:"volatility"`         // 关系变化程度
	Status           string    `json:"status"`             // 状态标识
	CreatedAt        time.Time `json:"created_at"`         // 创建时间
	UpdatedAt        time.Time `json:"updated_at"`         // 最后更新时间
}

// RelationshipView 代表角色对关系的单方视角。
// 每个 RelationshipPair 会有两个 RelationshipView，分别描述两个角色对这段关系的认知。
type RelationshipView struct {
	ID                     string    `json:"id"`                       // 视角唯一标识符
	ProjectID              string    `json:"project_id"`               // 所属项目 ID
	PairID                 string    `json:"pair_id"`                  // 所属关系对 ID
	SourceCharacterID      string    `json:"source_character_id"`      // 视角来源角色 ID
	TargetCharacterID      string    `json:"target_character_id"`      // 视角目标角色 ID
	PublicAttitude         string    `json:"public_attitude"`          // 公开态度
	PrivateAttitude        string    `json:"private_attitude"`         // 私下态度
	BelievedTargetAttitude string    `json:"believed_target_attitude"` // 以为对方的态度
	MaskingStrategy        string    `json:"masking_strategy"`         // 掩饰策略
	Status                 string    `json:"status"`                   // 状态标识
	CreatedAt              time.Time `json:"created_at"`               // 创建时间
	UpdatedAt              time.Time `json:"updated_at"`               // 最后更新时间
}

// RelationshipEvent 代表关系中发生的事件，用于跟踪关系的变化历史。
type RelationshipEvent struct {
	ID        string         `json:"id"`         // 事件唯一标识符
	ProjectID string         `json:"project_id"` // 所属项目 ID
	PairID    string         `json:"pair_id"`    // 所属关系对 ID
	EventType string         `json:"event_type"` // 事件类型
	Summary   string         `json:"summary"`    // 事件摘要
	Payload   map[string]any `json:"payload"`    // 事件附加数据
	CreatedAt time.Time      `json:"created_at"` // 事件发生时间
}

// Relationship 是关系的完整聚合，包含关系对、双方视角和最近事件。
type Relationship struct {
	Pair           RelationshipPair    `json:"pair"`             // 关系对基础信息
	Views          []RelationshipView  `json:"views"`            // 所有相关视角
	RecentEvents   []RelationshipEvent `json:"recent_events"`    // 最近发生的事件
	CharacterAView *RelationshipView   `json:"character_a_view"` // 角色 A 的视角
	CharacterBView *RelationshipView   `json:"character_b_view"` // 角色 B 的视角
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
	RunID               string `json:"run_id"`               // 要应用的运行 ID
	AcceptAuthorBible   bool   `json:"accept_author_bible"`  // 是否接受作者圣经
	AcceptCharacters    bool   `json:"accept_characters"`    // 是否接受角色
	AcceptRelationships bool   `json:"accept_relationships"` // 是否接受关系
	AcceptWorldState    bool   `json:"accept_world_state"`   // 是否接受世界状态
	AuthorNote          string `json:"author_note"`          // 作者备注
}

// SetupQuestion 是设置过程中 AI 提出的问题。
type SetupQuestion struct {
	Key          string `json:"key"`            // 问题标识键
	Question     string `json:"question"`       // 问题内容
	WhyItMatters string `json:"why_it_matters"` // 为什么这个问题重要
}

// ConversationMessage 是会话消息的模型。
type ConversationMessage struct {
	ID        string    `json:"id"`         // 消息唯一标识符
	SessionID string    `json:"session_id"` // 所属会话 ID
	Role      string    `json:"role"`       // 消息角色（user/assistant）
	Content   string    `json:"content"`    // 消息内容
	CreatedAt time.Time `json:"created_at"` // 创建时间
}

// SetupSession 是设置会话的模型。
// 设置会话是作者与 AI 之间的多轮对话，用于逐步构建项目的基础设定。
type SetupSession struct {
	ID              string                `json:"id"`                          // 会话唯一标识符
	ProjectID       string                `json:"project_id"`                  // 所属项目 ID
	SeedIdea        string                `json:"seed_idea"`                   // 种子想法
	LastUserMessage string                `json:"last_user_message"`           // 最后一条用户消息
	Status          string                `json:"status"`                      // 会话状态
	LatestRunID     string                `json:"latest_run_id,omitempty"`     // 最近一次运行 ID
	LatestRunStatus string                `json:"latest_run_status,omitempty"` // 最近一次运行状态
	LatestRunError  string                `json:"latest_run_error,omitempty"`  // 最近一次运行错误
	Messages        []ConversationMessage `json:"messages"`                    // 会话消息历史
	CreatedAt       time.Time             `json:"created_at"`                  // 创建时间
	UpdatedAt       time.Time             `json:"updated_at"`                  // 最后更新时间
}

// SetupRun 是设置运行的模型。
// 每次用户推进设置会话时创建一个运行，跟踪 AI 处理进度。
type SetupRun struct {
	RunID       string    `json:"run_id"`       // 运行唯一标识符
	SessionID   string    `json:"session_id"`   // 所属会话 ID
	ProjectID   string    `json:"project_id"`   // 所属项目 ID
	Status      string    `json:"status"`       // 运行状态
	CurrentStep string    `json:"current_step"` // 当前步骤
	Progress    int       `json:"progress"`     // 进度百分比
	Error       string    `json:"error,omitempty"`
	CreatedAt   time.Time `json:"created_at"` // 创建时间
	UpdatedAt   time.Time `json:"updated_at"` // 最后更新时间
}

// SetupVisualWorldPressureCard 是 Setup 草案中用于展示世界压力的卡片。
type SetupVisualWorldPressureCard struct {
	Title                 string   `json:"title"`
	Detail                string   `json:"detail"`
	Stakes                string   `json:"stakes"`
	RelatedWorldStateKeys []string `json:"related_world_state_keys"`
}

// SetupVisualCharacterCard 是 Setup 草案中用于展示角色功能位的卡片。
type SetupVisualCharacterCard struct {
	CharacterKey string `json:"character_key"`
	Name         string `json:"name"`
	Role         string `json:"role"`
	Hook         string `json:"hook"`
	Goal         string `json:"goal"`
	Fear         string `json:"fear"`
	Secret       string `json:"secret"`
}

// SetupVisualRelationshipEdge 是 Setup 草案中用于展示关系图的边。
type SetupVisualRelationshipEdge struct {
	FromCharacterKey string `json:"from_character_key"`
	ToCharacterKey   string `json:"to_character_key"`
	Summary          string `json:"summary"`
	Tension          string `json:"tension"`
	Misreading       string `json:"misreading"`
}

// SetupNextAgentSuggestion 是 Setup 完成后建议进入的下一步 agent。
type SetupNextAgentSuggestion struct {
	Key    string `json:"key"`
	Label  string `json:"label"`
	Reason string `json:"reason"`
}

// SetupVisualDraft 是面向用户确认的可视化 Setup 草案视图。
type SetupVisualDraft struct {
	Logline              string                         `json:"logline"`
	StyleTags            []string                       `json:"style_tags"`
	Tone                 string                         `json:"tone"`
	BoldnessLevel        int                            `json:"boldness_level"`
	WorldPressureCards   []SetupVisualWorldPressureCard `json:"world_pressure_cards"`
	CharacterCards       []SetupVisualCharacterCard     `json:"character_cards"`
	RelationshipEdges    []SetupVisualRelationshipEdge  `json:"relationship_edges"`
	OpenQuestions        []SetupQuestion                `json:"open_questions"`
	AgentSummary         string                         `json:"agent_summary"`
	NextAgentSuggestions []SetupNextAgentSuggestion     `json:"next_agent_suggestions,omitempty"`
}

// SetupDraft 是设置生成的草稿。
// 包含 AI 生成的完整项目设定。
type SetupDraft struct {
	AuthorBible      AuthorBible       `json:"author_bible"`           // 作者圣经
	Characters       []Character       `json:"characters"`             // 角色列表
	Relationships    []Relationship    `json:"relationships"`          // 关系列表
	WorldState       []WorldStateEntry `json:"world_state"`            // 世界状态
	OpenQuestions    []SetupQuestion   `json:"open_questions"`         // 待解答问题
	AssistantSummary string            `json:"assistant_summary"`      // AI 总结
	VisualDraft      *SetupVisualDraft `json:"visual_draft,omitempty"` // 可视化草案
}

// SetupRunResult 是设置运行结果的模型。
type SetupRunResult struct {
	RunID      string     `json:"run_id"`      // 运行 ID
	SessionID  string     `json:"session_id"`  // 会话 ID
	Status     string     `json:"status"`      // 结果状态
	SetupDraft SetupDraft `json:"setup_draft"` // 设置草稿
}

// ApplySetupRunResult 是应用设置运行后的结果。
type ApplySetupRunResult struct {
	ProjectID string `json:"project_id"` // 项目 ID
	RunID     string `json:"run_id"`     // 运行 ID
	Status    string `json:"status"`     // 结果状态
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
	AuthorMessage    string // 作者输入的消息；自动推进时为空
	BranchID         string // 继续推进的事件分支 ID
	BaseEventID      string // 继续推进的基础事件 ID
	AdvanceMode      string // 推进模式：manual 或 auto
	TickDelaySeconds int    // 自动连续推进时每轮之间的等待秒数
}

type ForkStoryEventInput struct {
	Name          string `json:"name"`
	AuthorMessage string `json:"author_message,omitempty"`
}

// CutChapterInput 是从事件账本中裁出章节的输入参数。
type CutChapterInput struct {
	BranchID    string `json:"branch_id"`
	FromEventID string `json:"from_event_id"`
	ToEventID   string `json:"to_event_id"`
	Title       string `json:"title,omitempty"`
	AuthorNote  string `json:"author_note,omitempty"`
}

// StorySession 是故事会话的模型。
// 故事会话是单个故事生成周期的上下文容器。
type StorySession struct {
	ID                         string                `json:"id"`                            // 会话唯一标识符
	ProjectID                  string                `json:"project_id"`                    // 所属项目 ID
	Title                      string                `json:"title"`                         // 会话标题
	OpeningSituation           string                `json:"opening_situation"`             // 开局情境
	AuthorIntent               string                `json:"author_intent"`                 // 作者意图
	LastAuthorMessage          string                `json:"last_author_message"`           // 最后一条作者消息
	Status                     string                `json:"status"`                        // 会话状态
	CurrentPlotVariableSummary string                `json:"current_plot_variable_summary"` // 当前剧情变量摘要
	Messages                   []ConversationMessage `json:"messages"`                      // 会话消息历史
	CreatedAt                  time.Time             `json:"created_at"`                    // 创建时间
	UpdatedAt                  time.Time             `json:"updated_at"`                    // 最后更新时间
}

// StoryRun 是故事运行的模型。
// 每次用户推进故事会话时创建一个运行，跟踪 AI 内容生成进度。
type StoryRun struct {
	RunID         string     `json:"run_id"`     // 运行唯一标识符
	SessionID     string     `json:"session_id"` // 所属会话 ID
	ProjectID     string     `json:"project_id"` // 所属项目 ID
	BranchID      string     `json:"branch_id,omitempty"`
	BaseEventID   string     `json:"base_event_id,omitempty"`
	HeadEventID   string     `json:"head_event_id,omitempty"`
	Status        string     `json:"status"`       // 运行状态
	CurrentStep   string     `json:"current_step"` // 当前步骤
	Progress      int        `json:"progress"`     // 进度百分比
	Error         string     `json:"error,omitempty"`
	StopRequested bool       `json:"stop_requested"`
	CutAt         *time.Time `json:"cut_at"`     // 裁章时间（如果有）
	CreatedAt     time.Time  `json:"created_at"` // 创建时间
	UpdatedAt     time.Time  `json:"updated_at"` // 最后更新时间
}

// Draft 是生成的章节草稿。
// 草稿是事件段的编辑投影，裁章时会转成正式章节。
type Draft struct {
	ID            string `json:"id"`             // 草稿唯一标识符
	Title         string `json:"title"`          // 章节标题
	ChapterNumber int    `json:"chapter_number"` // 章节编号
	Content       string `json:"content"`        // 章节正文
	Summary       string `json:"summary"`        // 章节摘要
	WordCount     int    `json:"word_count"`     // 字数统计
}

// PlotVariable 是剧情变量，定义故事中的核心戏剧性选择。
// AI 在生成内容前会确定剧情变量，为角色提供有意义的选择。
type PlotVariable struct {
	PressureSource      string   `json:"pressure_source"`       // 压力来源
	FocalCharacterID    string   `json:"focal_character_id"`    // 核心角色 ID
	CoreChoice          string   `json:"core_choice"`           // 核心选择描述
	OptionA             string   `json:"option_a"`              // 选项 A
	OptionB             string   `json:"option_b"`              // 选项 B
	CostA               string   `json:"cost_a"`                // 选择 A 的代价
	CostB               string   `json:"cost_b"`                // 选择 B 的代价
	IrreversibleEffect  string   `json:"irreversible_effect"`   // 不可逆影响
	RelatedCharacterIDs []string `json:"related_character_ids"` // 相关角色 ID 列表
	WorldStatePressure  []string `json:"world_state_pressure"`  // 世界状态压力
}

type StoryEventPlan struct {
	ID             string     `json:"id"`
	TimeIndex      int        `json:"time_index"`
	DurationHours  int        `json:"duration_hours,omitempty"`
	StartAt        *time.Time `json:"start_at,omitempty"`
	ArriveAt       *time.Time `json:"arrive_at,omitempty"`
	EffectAt       *time.Time `json:"effect_at,omitempty"`
	EndsAt         *time.Time `json:"ends_at,omitempty"`
	CharacterID    string     `json:"character_id,omitempty"`
	CharacterName  string     `json:"character_name,omitempty"`
	LocationKey    string     `json:"location_key"`
	LocationName   string     `json:"location_name,omitempty"`
	ActionType     string     `json:"action_type"`
	Summary        string     `json:"summary"`
	Intent         string     `json:"intent,omitempty"`
	Visibility     string     `json:"visibility,omitempty"`
	TargetActorIDs []string   `json:"target_actor_ids,omitempty"`
	ResourceKeys   []string   `json:"resource_keys,omitempty"`
}

type StoryLocationGroup struct {
	ID           string   `json:"id"`
	LocationKey  string   `json:"location_key"`
	LocationName string   `json:"location_name,omitempty"`
	CharacterIDs []string `json:"character_ids"`
	EventIDs     []string `json:"event_ids"`
}

type StoryInteractionGroup struct {
	ID              string   `json:"id"`
	LocationKey     string   `json:"location_key"`
	LocationName    string   `json:"location_name,omitempty"`
	CharacterIDs    []string `json:"character_ids"`
	EventIDs        []string `json:"event_ids,omitempty"`
	ShouldInteract  bool     `json:"should_interact"`
	InteractionType string   `json:"interaction_type,omitempty"`
	Stakes          string   `json:"stakes,omitempty"`
	Rationale       string   `json:"rationale,omitempty"`
	Priority        int      `json:"priority,omitempty"`
}

type StoryInteractionAnalysis struct {
	LocationGroups    []StoryLocationGroup    `json:"location_groups"`
	InteractionGroups []StoryInteractionGroup `json:"interaction_groups"`
}

type StoryInteractionTurn struct {
	TurnIndex          int      `json:"turn_index"`
	InteractionGroupID string   `json:"interaction_group_id"`
	ActorID            string   `json:"actor_id,omitempty"`
	ActorName          string   `json:"actor_name,omitempty"`
	ActionType         string   `json:"action_type"`
	Speech             string   `json:"speech,omitempty"`
	ActionSummary      string   `json:"action_summary,omitempty"`
	TargetActorIDs     []string `json:"target_actor_ids,omitempty"`
	Intent             string   `json:"intent,omitempty"`
	LocationKey        string   `json:"location_key,omitempty"`
	LocationName       string   `json:"location_name,omitempty"`
}

type StoryInteractionTranscript struct {
	GroupID        string                 `json:"group_id"`
	LocationKey    string                 `json:"location_key"`
	LocationName   string                 `json:"location_name,omitempty"`
	CharacterIDs   []string               `json:"character_ids"`
	Turns          []StoryInteractionTurn `json:"turns"`
	OutcomeSummary string                 `json:"outcome_summary,omitempty"`
}

type StoryTurn struct {
	TurnIndex          int      `json:"turn_index"`
	ActorID            string   `json:"actor_id,omitempty"`
	ActorName          string   `json:"actor_name,omitempty"`
	ActionType         string   `json:"action_type"`
	Speech             string   `json:"speech,omitempty"`
	ActionSummary      string   `json:"action_summary,omitempty"`
	TargetActorIDs     []string `json:"target_actor_ids,omitempty"`
	Intent             string   `json:"intent,omitempty"`
	InteractionGroupID string   `json:"interaction_group_id,omitempty"`
	LocationKey        string   `json:"location_key,omitempty"`
	LocationName       string   `json:"location_name,omitempty"`
	Phase              string   `json:"phase,omitempty"`
}

const (
	EventKindGenesis          = "genesis"
	EventKindActionScheduled  = "action_scheduled"
	EventKindActionCompleted  = "action_completed"
	EventKindActionVoided     = "action_voided"
	EventKindActionSuperseded = "action_superseded"
	EventKindSceneResolved    = "scene_resolved"
	EventKindVariableInjected = "variable_injected"
	EventKindPromotion        = "promotion"
)

// StoryEvent 是事件账本的最小单位。正史由事件链重放得到。
type StoryEvent struct {
	ID            string          `json:"id"`
	ProjectID     string          `json:"project_id"`
	SessionID     string          `json:"session_id"`
	BranchID      string          `json:"branch_id"`
	ParentEventID string          `json:"parent_event_id,omitempty"`
	Sequence      int             `json:"sequence"`
	StoryTime     time.Time       `json:"story_time"`
	Kind          string          `json:"kind"`
	ActorIDs      []string        `json:"actor_ids,omitempty"`
	LocationKey   string          `json:"location_key,omitempty"`
	ResourceKeys  []string        `json:"resource_keys,omitempty"`
	Summary       string          `json:"summary"`
	Payload       map[string]any  `json:"payload,omitempty"`
	StateDelta    EventStateDelta `json:"state_delta"`
	Published     bool            `json:"published"`
	CreatedAt     time.Time       `json:"created_at"`
}

// EventStateDelta 是事件对正史状态的完整增量；world/relationship 变更必须留在这里，不能随外部记忆提交迁走。
type EventStateDelta struct {
	MemoryPatch
	CharacterMoves []CharacterMove `json:"character_moves,omitempty"`
	FactionDeltas  []FactionDelta  `json:"faction_deltas,omitempty"`
	LocationDeltas []LocationDelta `json:"location_deltas,omitempty"`
}

type CharacterMove struct {
	CharacterID string `json:"character_id"`
	LocationKey string `json:"location_key"`
	X           int    `json:"x"`
	Y           int    `json:"y"`
}

type FactionDelta struct {
	LocationID  string `json:"location_id"`
	FactionName string `json:"faction_name"`
	Influence   int    `json:"influence"`
	Attitude    string `json:"attitude"`
	Description string `json:"description"`
}

type LocationDelta struct {
	LocationKey string         `json:"location_key"`
	Status      string         `json:"status"`
	Properties  map[string]any `json:"properties,omitempty"`
}

// OngoingAction 是排程阶段登记在事件账本中的动作占用。
type OngoingAction struct {
	ID                string    `json:"id,omitempty"`
	CharacterID       string    `json:"character_id"`
	ActionType        string    `json:"action_type"`
	Description       string    `json:"description"`
	TargetLocationKey string    `json:"target_location_key,omitempty"`
	ParticipantIDs    []string  `json:"participant_ids,omitempty"`
	StartAt           time.Time `json:"start_at"`
	ArriveAt          time.Time `json:"arrive_at"`
	EffectAt          time.Time `json:"effect_at"`
	EndsAt            time.Time `json:"ends_at"`
	ResourceKeys      []string  `json:"resource_keys"`
	Status            string    `json:"status"`
	Rationale         string    `json:"rationale"`
}

type CharacterRuntimeState struct {
	CharacterID   string         `json:"character_id"`
	Tier          string         `json:"tier"`
	LocationKey   string         `json:"location_key"`
	X             int            `json:"x"`
	Y             int            `json:"y"`
	Status        string         `json:"status"`
	OngoingAction *OngoingAction `json:"ongoing_action,omitempty"`
}

type WorldSnapshot struct {
	AtEventID     string                           `json:"at_event_id"`
	StoryTime     time.Time                        `json:"story_time"`
	WorldState    map[string]WorldStateEntry       `json:"world_state"`
	Characters    map[string]CharacterRuntimeState `json:"characters"`
	Relationships map[string]Relationship          `json:"relationships"`
	Factions      []FactionInfluence               `json:"factions"`
	Locations     []LocationState                  `json:"locations"`
}

type Branch struct {
	ID                       string    `json:"id"`
	ProjectID                string    `json:"project_id"`
	SessionID                string    `json:"session_id"`
	Name                     string    `json:"name"`
	BaseEventID              string    `json:"base_event_id,omitempty"`
	HeadEventID              string    `json:"head_event_id,omitempty"`
	PublishedFrontierEventID string    `json:"published_frontier_event_id,omitempty"`
	Status                   string    `json:"status"`
	CreatedAt                time.Time `json:"created_at"`
	UpdatedAt                time.Time `json:"updated_at"`
}

type StoryEventLog struct {
	SessionID string       `json:"session_id"`
	Branches  []Branch     `json:"branches"`
	Events    []StoryEvent `json:"events"`
}

type ChapterEventSpan struct {
	ID          string    `json:"id"`
	ProjectID   string    `json:"project_id"`
	ChapterID   string    `json:"chapter_id"`
	BranchID    string    `json:"branch_id"`
	FromEventID string    `json:"from_event_id"`
	ToEventID   string    `json:"to_event_id"`
	CreatedAt   time.Time `json:"created_at"`
}

type WorldMap struct {
	ID         string         `json:"id"`
	ProjectID  string         `json:"project_id"`
	Name       string         `json:"name"`
	Seed       string         `json:"seed"`
	Width      int            `json:"width"`
	Height     int            `json:"height"`
	Status     string         `json:"status"`
	Properties map[string]any `json:"properties,omitempty"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
}

type MapTile struct {
	ID          string         `json:"id"`
	ProjectID   string         `json:"project_id"`
	MapID       string         `json:"map_id"`
	X           int            `json:"x"`
	Y           int            `json:"y"`
	Altitude    int            `json:"altitude"`
	Temperature int            `json:"temperature"`
	Humidity    int            `json:"humidity"`
	IsOcean     bool           `json:"is_ocean"`
	Terrain     string         `json:"terrain"`
	Properties  map[string]any `json:"properties,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

type LocationState struct {
	ID          string         `json:"id"`
	ProjectID   string         `json:"project_id"`
	MapID       string         `json:"map_id,omitempty"`
	RegionID    string         `json:"region_id,omitempty"`
	Name        string         `json:"name"`
	Type        string         `json:"type"`
	Description string         `json:"description"`
	X           int            `json:"x"`
	Y           int            `json:"y"`
	Radius      int            `json:"radius,omitempty"`
	Status      string         `json:"status"`
	Properties  map[string]any `json:"properties,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

type FactionInfluence struct {
	ID          string    `json:"id"`
	ProjectID   string    `json:"project_id"`
	LocationID  string    `json:"location_id"`
	FactionName string    `json:"faction_name"`
	Influence   int       `json:"influence"`
	Attitude    string    `json:"attitude"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type NearbyLocationContext struct {
	Location          LocationState      `json:"location"`
	Distance          float64            `json:"distance"`
	FactionInfluences []FactionInfluence `json:"faction_influences,omitempty"`
}

type CharacterActionDecisionInput struct {
	World             WorldSnapshot           `json:"world"`
	Character         Character               `json:"character"`
	CharacterState    CharacterRuntimeState   `json:"character_state"`
	Location          LocationState           `json:"location"`
	FactionInfluences []FactionInfluence      `json:"faction_influences"`
	NearbyLocations   []NearbyLocationContext `json:"nearby_locations,omitempty"`
}

type CharacterActionDecision struct {
	ActionType           string    `json:"action_type"`
	Description          string    `json:"description"`
	TargetLocationKey    string    `json:"target_location_key,omitempty"`
	ArriveAt             time.Time `json:"arrive_at"`
	EffectAt             time.Time `json:"effect_at"`
	EndsAt               time.Time `json:"ends_at"`
	DurationHours        int       `json:"duration_hours,omitempty"`
	AffectedResourceKeys []string  `json:"affected_resource_keys,omitempty"`
	ParticipantIDs       []string  `json:"participant_ids,omitempty"`
	Rationale            string    `json:"rationale"`
}

// ReviewReport 是审阅报告，包含 AI 对生成内容的质量评估。
type ReviewReport struct {
	Pass             bool     `json:"pass"`              // 是否通过
	HardViolations   []string `json:"hard_violations"`   // 硬性违规（必须修复的问题）
	ContinuityIssues []string `json:"continuity_issues"` // 连续性问题
	StyleIssues      []string `json:"style_issues"`      // 风格问题
	SuggestedFixes   []string `json:"suggested_fixes"`   // 建议修复
}

// CharacterMemoryUpdate 是角色记忆更新。
type CharacterMemoryUpdate struct {
	CharacterID string `json:"character_id"` // 角色 ID
	Type        string `json:"type"`         // 更新类型
	Content     string `json:"content"`      // 更新内容
	Importance  int    `json:"importance"`   // 重要性
}

// RelationshipViewUpdate 是关系视角更新。
type RelationshipViewUpdate struct {
	ViewID                 string `json:"view_id"`                  // 视角 ID
	PairID                 string `json:"pair_id"`                  // 关系对 ID
	SourceCharacterID      string `json:"source_character_id"`      // 源角色 ID
	TargetCharacterID      string `json:"target_character_id"`      // 目标角色 ID
	PublicAttitude         string `json:"public_attitude"`          // 公开态度
	PrivateAttitude        string `json:"private_attitude"`         // 私下态度
	BelievedTargetAttitude string `json:"believed_target_attitude"` // 以为对方的态度
	MaskingStrategy        string `json:"masking_strategy"`         // 掩饰策略
}

// RelationshipUpdate 是关系更新。
type RelationshipUpdate struct {
	PairID       string                   `json:"pair_id"`       // 关系对 ID
	Summary      string                   `json:"summary"`       // 更新后的摘要
	TensionDelta string                   `json:"tension_delta"` // 紧张度变化
	Pair         *RelationshipPair        `json:"pair"`          // 更新的关系对（可选）
	Views        []RelationshipViewUpdate `json:"views"`         // 更新的视角列表
	Events       []RelationshipEvent      `json:"events"`        // 新增的事件列表
}

// WorldStateUpdate 是世界状态更新。
type WorldStateUpdate struct {
	Key       string `json:"key"`       // 状态键
	Operation string `json:"operation"` // 操作类型（set, update, delete）
	Value     any    `json:"value"`     // 新值
	Note      string `json:"note"`      // 备注说明
}

// MemoryPatch 是事件状态增量中的记忆/关系/世界状态片段。
// 故事正史由 StoryEvent 重放得到；裁章时只允许把角色记忆同步到外部检索服务。
type MemoryPatch struct {
	ID                     string                  `json:"id"`                       // 补丁唯一标识符
	Status                 string                  `json:"status"`                   // 补丁状态
	CharacterMemoryUpdates []CharacterMemoryUpdate `json:"character_memory_updates"` // 角色记忆更新列表
	RelationshipUpdates    []RelationshipUpdate    `json:"relationship_updates"`     // 关系更新列表
	WorldStateUpdates      []WorldStateUpdate      `json:"world_state_updates"`      // 世界状态更新列表
}

// StoryRunResult 是故事运行结果的模型。
type StoryRunResult struct {
	RunID                  string                       `json:"run_id"`     // 运行 ID
	SessionID              string                       `json:"session_id"` // 会话 ID
	Status                 string                       `json:"status"`     // 结果状态
	BranchID               string                       `json:"branch_id,omitempty"`
	BaseEventID            string                       `json:"base_event_id,omitempty"`
	HeadEventID            string                       `json:"head_event_id,omitempty"`
	PlotVariable           PlotVariable                 `json:"plot_variable"`                // 剧情变量
	EventPlan              []StoryEventPlan             `json:"event_plan"`                   // 排程事件素材
	Turns                  []StoryTurn                  `json:"turns"`                        // 全部场景回合素材
	SceneSummary           string                       `json:"scene_summary"`                // 场景复盘摘要
	InteractionAnalysis    StoryInteractionAnalysis     `json:"interaction_analysis"`         // 交互分析
	InteractionTranscripts []StoryInteractionTranscript `json:"interaction_transcripts"`      // 交涉记录
	Draft                  Draft                        `json:"draft"`                        // 章节草稿
	Review                 ReviewReport                 `json:"review"`                       // 审阅报告
	MemoryPatch            MemoryPatch                  `json:"memory_patch"`                 // 记忆补丁
	Events                 []StoryEvent                 `json:"events,omitempty"`             // 已写入正史的事件
	CompletedActions       []OngoingAction              `json:"completed_actions,omitempty"`  // 本 tick 完成并唤醒的既有动作
	SupersededActions      []OngoingAction              `json:"superseded_actions,omitempty"` // 本 tick 被场景抢占/作废的既有动作
	CollisionAt            *time.Time                   `json:"collision_at,omitempty"`       // 抢占/碰撞发生的世界时间
}

type CutChapterResult struct {
	Chapter  Chapter          `json:"chapter"`
	Span     ChapterEventSpan `json:"span"`
	StoryRun StoryRun         `json:"story_run,omitempty"`
}

type CreateDialogueSessionInput struct {
	Title string `json:"title"`
}

type AdvanceDialogueSessionInput struct {
	UserMessage string `json:"user_message"`
}

type ExecuteDialogueActionInput struct {
	Confirm       bool   `json:"confirm"`
	AuthorNote    string `json:"author_note"`
	ExecutionMode string `json:"execution_mode,omitempty"`
	PolicyReason  string `json:"policy_reason,omitempty"`
}

type AutoExecuteDialogueActionInput struct {
	AuthorNote   string `json:"author_note"`
	PolicyReason string `json:"policy_reason"`
}

type RejectDialogueActionInput struct {
	Reason string `json:"reason"`
}

type DialogueMessage struct {
	ID        string         `json:"id"`
	SessionID string         `json:"session_id"`
	Role      string         `json:"role"`
	Content   string         `json:"content"`
	Metadata  map[string]any `json:"metadata,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
}

type DialogueSession struct {
	ID              string            `json:"id"`
	ProjectID       string            `json:"project_id"`
	Title           string            `json:"title"`
	LastUserMessage string            `json:"last_user_message"`
	Status          string            `json:"status"`
	LatestRunID     string            `json:"latest_run_id,omitempty"`
	LatestRunStatus string            `json:"latest_run_status,omitempty"`
	LatestRunError  string            `json:"latest_run_error,omitempty"`
	Messages        []DialogueMessage `json:"messages"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
}

type DialogueRun struct {
	RunID       string    `json:"run_id"`
	SessionID   string    `json:"session_id"`
	ProjectID   string    `json:"project_id"`
	Status      string    `json:"status"`
	CurrentStep string    `json:"current_step"`
	Progress    int       `json:"progress"`
	Error       string    `json:"error,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type DialogueQuestion struct {
	Key          string `json:"key"`
	Question     string `json:"question"`
	WhyItMatters string `json:"why_it_matters"`
}

type DialogueToolTrace struct {
	ToolName  string    `json:"tool_name"`
	Summary   string    `json:"summary"`
	OK        bool      `json:"ok"`
	CreatedAt time.Time `json:"created_at"`
}

type DialogueActionOption struct {
	ID                   string         `json:"id"`
	SessionID            string         `json:"session_id"`
	RunID                string         `json:"run_id"`
	ProjectID            string         `json:"project_id"`
	ActionType           string         `json:"action_type"`
	Label                string         `json:"label"`
	Description          string         `json:"description"`
	Rationale            string         `json:"rationale"`
	ConfirmationRequired bool           `json:"confirmation_required"`
	Payload              map[string]any `json:"payload"`
	Status               string         `json:"status"`
	Result               map[string]any `json:"result,omitempty"`
	Error                string         `json:"error,omitempty"`
	ExpiresAt            *time.Time     `json:"expires_at,omitempty"`
	CreatedAt            time.Time      `json:"created_at"`
	UpdatedAt            time.Time      `json:"updated_at"`
}

type DialogueRunResult struct {
	RunID               string                 `json:"run_id"`
	SessionID           string                 `json:"session_id"`
	Status              string                 `json:"status"`
	AssistantMessage    string                 `json:"assistant_message"`
	ActionOptions       []DialogueActionOption `json:"action_options"`
	ClarifyingQuestions []DialogueQuestion     `json:"clarifying_questions"`
	SuggestedReplies    []string               `json:"suggested_replies"`
	ContextSummary      string                 `json:"context_summary"`
	ToolTrace           []DialogueToolTrace    `json:"tool_trace"`
}

// Chapter 是从事件账本裁出的正式故事章节。
// 章节是故事的基本阅读单位，每次裁章会绑定一个 ChapterEventSpan。
type Chapter struct {
	ID            string    `json:"id"`             // 章节唯一标识符
	ProjectID     string    `json:"project_id"`     // 所属项目 ID
	ChapterNumber int       `json:"chapter_number"` // 章节编号（递增）
	Title         string    `json:"title"`          // 章节标题
	Summary       string    `json:"summary"`        // 章节摘要
	Content       string    `json:"content"`        // 章节正文
	AuthorNote    string    `json:"author_note"`    // 作者备注
	Status        string    `json:"status"`         // 章节状态
	WordCount     int       `json:"word_count"`     // 字数统计
	CommittedAt   time.Time `json:"committed_at"`   // 章节发布时间
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
	ID              string    `json:"id"`                // 记忆唯一标识符
	CharacterID     string    `json:"character_id"`      // 所属角色 ID
	Content         string    `json:"content"`           // 记忆内容
	SourceChapterID string    `json:"source_chapter_id"` // 来源章节 ID
	SourceRunID     string    `json:"source_run_id,omitempty"`
	BranchID        string    `json:"branch_id,omitempty"`
	SourceEventID   string    `json:"source_event_id,omitempty"`
	Importance      int       `json:"importance"` // 重要性等级
	Note            string    `json:"note"`       // 备注说明
	Status          string    `json:"status"`     // 状态标识
	CreatedAt       time.Time `json:"created_at"` // 创建时间
}

// RunEvent 是运行事件，用于记录 AI 生成过程中的关键事件。
// 事件通过 SSE 实时推送给客户端，并持久化用于审计。
// RunExecutionWork ?????????????????????????
// ????? run execution ???????? types.go???????????????????
type RunExecutionWork struct {
	RunKind   string    `json:"run_kind"`
	RunID     string    `json:"run_id"`
	SessionID string    `json:"session_id"`
	ProjectID string    `json:"project_id"`
	Status    string    `json:"status"`
	UpdatedAt time.Time `json:"updated_at"`
}

type RunEvent struct {
	ID        string         `json:"id"`         // 事件唯一标识符
	RunKind   string         `json:"run_kind"`   // 运行类型（setup/story）
	RunID     string         `json:"run_id"`     // 所属运行 ID
	EventName string         `json:"event_name"` // 事件名称
	Sequence  int            `json:"sequence"`   // 事件序号
	Payload   map[string]any `json:"payload"`    // 事件附加数据
	CreatedAt time.Time      `json:"created_at"` // 事件发生时间
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
