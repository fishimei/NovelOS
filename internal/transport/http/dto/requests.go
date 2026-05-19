// Package dto 定义了 HTTP 请求的 DTO（Data Transfer Object）。
// DTO 用于接收客户端发送的 JSON 请求数据，并进行基本的验证。
// 遵循 Gin 的 binding 标签进行请求验证。
package dto

// PageQuery 是分页查询参数。
type PageQuery struct {
	Page     int `form:"page"`
	PageSize int `form:"page_size"`
}

// LimitQuery 是限制查询参数。
type LimitQuery struct {
	Limit int `form:"limit"`
}

// CreateProjectRequest 是创建项目的请求参数。
type CreateProjectRequest struct {
	Title       string `json:"title" binding:"required"`
	Genre       string `json:"genre"`
	Description string `json:"description"`
}

// UpdateProjectRequest 是更新项目的请求参数。
type UpdateProjectRequest struct {
	Title       string `json:"title"`
	Genre       string `json:"genre"`
	Description string `json:"description"`
}

// WorldStateEntryRequest 是世界状态条目的请求参数。
type WorldStateEntryRequest struct {
	Key   string `json:"key" binding:"required"`
	Value any    `json:"value"`
	Note  string `json:"note"`
}

// UpdateAuthorBibleRequest 是更新作者圣经的请求参数。
type UpdateAuthorBibleRequest struct {
	Theme               string                   `json:"theme"`
	StyleGuide          string                   `json:"style_guide"`
	WorldRules          []string                 `json:"world_rules"`
	AestheticPrinciples []string                 `json:"aesthetic_principles"`
	HardConstraints     []string                 `json:"hard_constraints"`
	SoftPreferences     []string                 `json:"soft_preferences"`
	ForbiddenMoves      []string                 `json:"forbidden_moves"`
	InitialWorldState   []WorldStateEntryRequest `json:"initial_world_state"`
}

// CreateCharacterRequest 是创建角色的请求参数。
type CreateCharacterRequest struct {
	Name        string   `json:"name" binding:"required"`
	Role        string   `json:"role" binding:"required"`
	Profile     string   `json:"profile"`
	Personality string   `json:"personality"`
	VoiceStyle  string   `json:"voice_style"`
	Goals       []string `json:"goals"`
	Fears       []string `json:"fears"`
	Secrets     []string `json:"secrets"`
	Constraints []string `json:"constraints"`
}

// UpdateCharacterRequest 是更新角色的请求参数。
type UpdateCharacterRequest struct {
	Name        string   `json:"name"`
	Role        string   `json:"role"`
	Profile     string   `json:"profile"`
	Personality string   `json:"personality"`
	VoiceStyle  string   `json:"voice_style"`
	Goals       []string `json:"goals"`
	Fears       []string `json:"fears"`
	Secrets     []string `json:"secrets"`
	Constraints []string `json:"constraints"`
}

// CreateRelationshipRequest 是创建关系的请求参数。
type CreateRelationshipRequest struct {
	CharacterAID  string   `json:"character_a_id" binding:"required"`
	CharacterBID  string   `json:"character_b_id" binding:"required"`
	Summary       string   `json:"summary" binding:"required"`
	Anchors       []string `json:"anchors"`
	TensionPoints []string `json:"tension_points"`
	Volatility    int      `json:"volatility"`
}

// UpdateRelationshipRequest 是更新关系的请求参数。
type UpdateRelationshipRequest struct {
	Summary       string   `json:"summary"`
	Anchors       []string `json:"anchors"`
	TensionPoints []string `json:"tension_points"`
	Volatility    int      `json:"volatility"`
}

// CreateSetupSessionRequest 是创建设置会话的请求参数。
type CreateSetupSessionRequest struct {
	SeedIdea string `json:"seed_idea" binding:"required"`
}

// AdvanceSetupSessionRequest 是推进设置会话的请求参数。
type AdvanceSetupSessionRequest struct {
	UserMessage string `json:"user_message" binding:"required"`
}

// UpdateSetupSessionRequest 是更新设置会话补充信息的请求参数。
type UpdateSetupSessionRequest struct {
	LastUserMessage string `json:"last_user_message"`
}

// ApplySetupRunRequest 是应用设置运行的请求参数。
type ApplySetupRunRequest struct {
	RunID               string `json:"run_id" binding:"required"`
	AcceptAuthorBible   bool   `json:"accept_author_bible"`
	AcceptCharacters    bool   `json:"accept_characters"`
	AcceptRelationships bool   `json:"accept_relationships"`
	AcceptWorldState    bool   `json:"accept_world_state"`
	AuthorNote          string `json:"author_note"`
}

// CreateStorySessionRequest 是创建故事会话的请求参数。
type CreateStorySessionRequest struct {
	Title            string `json:"title"`
	OpeningSituation string `json:"opening_situation"`
	AuthorIntent     string `json:"author_intent"`
}

type UpdateStorySessionRequest struct {
	Title string `json:"title"`
}

// AdvanceStorySessionRequest 是推进故事会话的请求参数。
type AdvanceStorySessionRequest struct {
	AuthorMessage string `json:"author_message" binding:"required"`
}

// CommitStoryRunRequest 是提交故事运行的请求参数。
type CommitStoryRunRequest struct {
	DraftID       string `json:"draft_id" binding:"required"`
	MemoryPatchID string `json:"memory_patch_id" binding:"required"`
	AuthorNote    string `json:"author_note"`
}

type CreateDialogueSessionRequest struct {
	Title string `json:"title"`
}

type AdvanceDialogueSessionRequest struct {
	UserMessage string `json:"user_message" binding:"required"`
}

type ConfirmDialogueActionOptionRequest struct {
	Confirm    bool   `json:"confirm" binding:"required"`
	AuthorNote string `json:"author_note"`
}

type RejectDialogueActionOptionRequest struct {
	Reason string `json:"reason"`
}

// CreateMemoryRequest 是创建记忆的请求参数。
type CreateMemoryRequest struct {
	Content    string `json:"content" binding:"required"`
	Importance int    `json:"importance"`
	Note       string `json:"note"`
}
