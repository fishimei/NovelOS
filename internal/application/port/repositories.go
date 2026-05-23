// Package port 定义了应用程序的仓库接口（Ports and Adapters 架构）。
// 仓库接口抽象了数据持久化逻辑，使应用层不依赖于具体的数据存储实现。
package port

import (
	"context"

	"github.com/fishimei/NovelOS/internal/application/model"
)

// ProjectRepository 是项目仓库的接口。
type ProjectRepository interface {
	Create(ctx context.Context, input model.CreateProjectInput) (model.Project, error)
	GetByID(ctx context.Context, id string) (model.Project, error)
	Update(ctx context.Context, id string, input model.UpdateProjectInput) (model.Project, error)
	GetDetail(ctx context.Context, id string) (model.ProjectDetail, error)
}

// AuthorBibleRepository 是作者圣经仓库的接口。
type AuthorBibleRepository interface {
	GetByProjectID(ctx context.Context, projectID string) (model.AuthorBible, error)
	UpdateByProjectID(ctx context.Context, projectID string, input model.UpdateAuthorBibleInput) (model.AuthorBible, error)
	Upsert(ctx context.Context, bible model.AuthorBible) (model.AuthorBible, error)
}

// WorldStateRepository 是世界状态仓库的接口。
type WorldStateRepository interface {
	ListByProjectID(ctx context.Context, projectID string) ([]model.WorldStateEntry, error)
	UpsertEntries(ctx context.Context, projectID string, entries []model.WorldStateEntry) error
}

// CharacterRepository 是角色仓库的接口。
type CharacterRepository interface {
	Create(ctx context.Context, projectID string, input model.CreateCharacterInput) (model.Character, error)
	ListByProjectID(ctx context.Context, projectID string, pageInput model.PageInput) (model.ListResult[model.Character], error)
	GetByID(ctx context.Context, id string) (model.Character, error)
	Update(ctx context.Context, id string, input model.UpdateCharacterInput) (model.Character, error)
	Upsert(ctx context.Context, character model.Character) (model.Character, error)
}

// RelationshipRepository 是关系仓库的接口。
type RelationshipRepository interface {
	Create(ctx context.Context, projectID string, input model.CreateRelationshipInput) (model.Relationship, error)
	ListByProjectID(ctx context.Context, projectID string, pageInput model.PageInput) (model.ListResult[model.Relationship], error)
	GetByID(ctx context.Context, id string) (model.Relationship, error)
	Update(ctx context.Context, id string, input model.UpdateRelationshipInput) (model.Relationship, error)
	UpsertPair(ctx context.Context, pair model.RelationshipPair) (model.RelationshipPair, error)
	UpsertViews(ctx context.Context, pairID string, views []model.RelationshipView) error
	AddEvent(ctx context.Context, event model.RelationshipEvent) (model.RelationshipEvent, error)
}

// SetupSessionRepository 是设置会话仓库的接口。
// 管理设置流程相关的会话、消息和运行状态。
type SetupSessionRepository interface {
	CreateSession(ctx context.Context, projectID string, input model.CreateSetupSessionInput) (model.SetupSession, error)
	ListSessionsByProjectID(ctx context.Context, projectID string, pageInput model.PageInput) (model.ListResult[model.SetupSession], error)
	GetSessionByID(ctx context.Context, sessionID string) (model.SetupSession, error)
	UpdateSession(ctx context.Context, session model.SetupSession) (model.SetupSession, error)
	DeleteSession(ctx context.Context, sessionID string) error
	AppendMessage(ctx context.Context, sessionID string, role string, content string) (model.ConversationMessage, error)
	CreateRun(ctx context.Context, sessionID string, input model.AdvanceSetupSessionInput) (model.SetupRun, error)
	GetRunByID(ctx context.Context, runID string) (model.SetupRun, error)
	GetRunResultByID(ctx context.Context, runID string) (model.SetupRunResult, error)
	SaveRunResult(ctx context.Context, runID string, result model.SetupRunResult) error
	UpdateRunStatus(ctx context.Context, runID string, status string, currentStep string, progress int, errorMessage ...string) error
	MarkApplied(ctx context.Context, sessionID string, runID string) error
}

type DialogueSessionRepository interface {
	CreateSession(ctx context.Context, projectID string, input model.CreateDialogueSessionInput) (model.DialogueSession, error)
	ListSessionsByProjectID(ctx context.Context, projectID string, pageInput model.PageInput) (model.ListResult[model.DialogueSession], error)
	GetSessionByID(ctx context.Context, sessionID string) (model.DialogueSession, error)
	UpdateSession(ctx context.Context, session model.DialogueSession) (model.DialogueSession, error)
	AppendMessage(ctx context.Context, sessionID string, role string, content string, metadata map[string]any) (model.DialogueMessage, error)
	ListMessagesBySessionID(ctx context.Context, sessionID string) ([]model.DialogueMessage, error)
	CreateRun(ctx context.Context, sessionID string, input model.AdvanceDialogueSessionInput) (model.DialogueRun, error)
	GetRunByID(ctx context.Context, runID string) (model.DialogueRun, error)
	UpdateRunStatus(ctx context.Context, runID string, status string, currentStep string, progress int, errorMessage ...string) error
	SaveRunResult(ctx context.Context, runID string, result model.DialogueRunResult) error
	GetRunResultByID(ctx context.Context, runID string) (model.DialogueRunResult, error)
	SaveActionOptions(ctx context.Context, options []model.DialogueActionOption) error
	ListActionOptionsByRunID(ctx context.Context, runID string) ([]model.DialogueActionOption, error)
	ListPendingActionOptionsBySessionID(ctx context.Context, sessionID string) ([]model.DialogueActionOption, error)
	GetActionOptionByID(ctx context.Context, optionID string) (model.DialogueActionOption, error)
	UpdateActionOption(ctx context.Context, option model.DialogueActionOption) (model.DialogueActionOption, error)
	TryStartActionExecution(ctx context.Context, optionID string) (model.DialogueActionOption, error)
}

// StorySessionRepository 是故事会话仓库的接口。
// 管理故事流程相关的会话、消息和运行状态。
type StorySessionRepository interface {
	CreateSession(ctx context.Context, projectID string, input model.CreateStorySessionInput) (model.StorySession, error)
	ListSessionsByProjectID(ctx context.Context, projectID string, pageInput model.PageInput) (model.ListResult[model.StorySession], error)
	GetSessionByID(ctx context.Context, sessionID string) (model.StorySession, error)
	UpdateSession(ctx context.Context, session model.StorySession) (model.StorySession, error)
	DeleteSession(ctx context.Context, sessionID string) error
	AppendMessage(ctx context.Context, sessionID string, role string, content string) (model.ConversationMessage, error)
	CreateRun(ctx context.Context, sessionID string, input model.AdvanceStorySessionInput) (model.StoryRun, error)
	GetRunByID(ctx context.Context, runID string) (model.StoryRun, error)
	GetRunResultByID(ctx context.Context, runID string) (model.StoryRunResult, error)
	SaveRunResult(ctx context.Context, runID string, result model.StoryRunResult) error
	UpdateRunStatus(ctx context.Context, runID string, status string, currentStep string, progress int, errorMessage ...string) error
	UpdateRunTimeline(ctx context.Context, runID string, headTickID string) error
	MarkCommitted(ctx context.Context, runID string) error
}

type StoryTimelineRepository interface {
	CreateBranch(ctx context.Context, branch model.StoryBranch) (model.StoryBranch, error)
	GetBranchByID(ctx context.Context, branchID string) (model.StoryBranch, error)
	ListBranchesBySessionID(ctx context.Context, sessionID string) ([]model.StoryBranch, error)
	UpdateBranchHead(ctx context.Context, branchID string, headTickID string) error
	AppendTick(ctx context.Context, tick model.StoryTick, refs []model.StoryTickStateRef, versions []model.StoryStateVersion) (model.StoryTick, error)
	GetTickByID(ctx context.Context, tickID string) (model.StoryTick, error)
	ListTicksByBranchID(ctx context.Context, branchID string) ([]model.StoryTick, error)
	ListTickStateRefs(ctx context.Context, tickID string) ([]model.StoryTickStateRef, error)
	ResolveTickState(ctx context.Context, tickID string) (model.StoryTickState, error)
}

type SimulationRepository interface {
	GetTimelineByProjectID(ctx context.Context, projectID string) (model.StoryTimeline, error)
	UpsertTimeline(ctx context.Context, timeline model.StoryTimeline) (model.StoryTimeline, error)
	GetWorldMapByProjectID(ctx context.Context, projectID string) (model.WorldMap, error)
	UpsertWorldMap(ctx context.Context, worldMap model.WorldMap) (model.WorldMap, error)
	ListMapTilesByProjectID(ctx context.Context, projectID string) ([]model.MapTile, error)
	UpsertMapTiles(ctx context.Context, projectID string, tiles []model.MapTile) error
	CreateTickRun(ctx context.Context, run model.StoryTickRun) (model.StoryTickRun, error)
	UpdateTickRun(ctx context.Context, run model.StoryTickRun) (model.StoryTickRun, error)
	GetTickRunByID(ctx context.Context, tickRunID string) (model.StoryTickRun, error)
	ListLocationsByProjectID(ctx context.Context, projectID string) ([]model.LocationState, error)
	UpsertLocations(ctx context.Context, projectID string, locations []model.LocationState) error
	ListFactionInfluencesByProjectID(ctx context.Context, projectID string) ([]model.FactionInfluence, error)
	UpsertFactionInfluences(ctx context.Context, projectID string, influences []model.FactionInfluence) error
	ListCharacterStatesByProjectID(ctx context.Context, projectID string) ([]model.CharacterSimulationState, error)
	UpsertCharacterStates(ctx context.Context, projectID string, states []model.CharacterSimulationState) error
	AppendEvent(ctx context.Context, event model.SimulationEvent) (model.SimulationEvent, error)
	ListEventsByTickRunID(ctx context.Context, tickRunID string) ([]model.SimulationEvent, error)
	CreateSnapshot(ctx context.Context, snapshot model.SimulationSnapshot) (model.SimulationSnapshot, error)
	GetSnapshotByTickRunID(ctx context.Context, tickRunID string) (model.SimulationSnapshot, error)
}

// ChapterRepository 是章节仓库的接口。
type ChapterRepository interface {
	ListByProjectID(ctx context.Context, projectID string, pageInput model.PageInput) (model.ListResult[model.Chapter], error)
	GetByID(ctx context.Context, id string) (model.Chapter, error)
	Create(ctx context.Context, chapter model.Chapter) (model.Chapter, error)
}

// MemoryRepository 是记忆仓库的接口。
type MemoryRepository interface {
	ListByCharacterID(ctx context.Context, characterID string, limit int) ([]model.Memory, error)
	Create(ctx context.Context, characterID string, input model.CreateMemoryInput) (model.Memory, error)
	CreateBatch(ctx context.Context, memories []model.Memory) error
}

// AuditRepository 是审计仓库的接口。
// 负责存储运行事件和状态修订快照。
type AuditRepository interface {
	AppendRunEvent(ctx context.Context, event model.RunEvent) (model.RunEvent, error)
	ListRunEvents(ctx context.Context, runKind string, runID string) ([]model.RunEvent, error)
	CreateRevision(ctx context.Context, revision model.StateRevision) (model.StateRevision, error)
}

// Repositories 是所有仓库接口的聚合。
// 方便在需要多个仓库时进行传递。
type Repositories struct {
	Projects         ProjectRepository
	AuthorBibles     AuthorBibleRepository
	WorldState       WorldStateRepository
	Characters       CharacterRepository
	Relationships    RelationshipRepository
	SetupSessions    SetupSessionRepository
	DialogueSessions DialogueSessionRepository
	StorySessions    StorySessionRepository
	StoryTimeline    StoryTimelineRepository
	Simulation       SimulationRepository
	Chapters         ChapterRepository
	Memories         MemoryRepository
	Audit            AuditRepository
}
