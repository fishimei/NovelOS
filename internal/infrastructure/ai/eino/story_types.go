package eino

import "github.com/fishimei/NovelOS/internal/application/model"

type StoryContextSnapshot struct {
	Session        model.StorySession        `json:"session"`
	AuthorBible    *model.AuthorBible        `json:"author_bible,omitempty"`
	WorldState     []model.WorldStateEntry   `json:"world_state"`
	Characters     []model.Character         `json:"characters"`
	Relationships  []model.Relationship      `json:"relationships"`
	RecentChapters []model.Chapter           `json:"recent_chapters"`
	RecentMemories map[string][]model.Memory `json:"recent_memories"`
}

type StoryTurnPlan struct {
	TurnIndex  int    `json:"turn_index"`
	ActorID    string `json:"actor_id,omitempty"`
	ActorName  string `json:"actor_name,omitempty"`
	ActionType string `json:"action_type"`
	Intent     string `json:"intent"`
	Rationale  string `json:"rationale,omitempty"`
}

type StoryStopDecision struct {
	Stop   bool   `json:"stop"`
	Reason string `json:"reason"`
}

type StoryPlanResult struct {
	Summary    string          `json:"summary"`
	StopReason string          `json:"stop_reason"`
	Turns      []StoryTurnPlan `json:"turns"`
}

type LoadStoryContextInput struct {
	ProjectID string `json:"project_id" jsonschema:"required"`
	SessionID string `json:"session_id" jsonschema:"required"`
}

type ChooseNextStoryActorInput struct {
	TurnIndex  int    `json:"turn_index" jsonschema:"required"`
	ActorID    string `json:"actor_id"`
	ActorName  string `json:"actor_name"`
	ActionType string `json:"action_type" jsonschema:"required"`
	Intent     string `json:"intent" jsonschema:"required"`
	Rationale  string `json:"rationale"`
}

type DecideStoryStopInput struct {
	TurnIndex int    `json:"turn_index" jsonschema:"required"`
	Stop      bool   `json:"stop" jsonschema:"required"`
	Reason    string `json:"reason" jsonschema:"required"`
}

type FinalizeStoryPlanInput struct {
	Summary    string `json:"summary" jsonschema:"required"`
	StopReason string `json:"stop_reason" jsonschema:"required"`
}
