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
	TurnIndex      int      `json:"turn_index"`
	ActorID        string   `json:"actor_id,omitempty"`
	ActorName      string   `json:"actor_name,omitempty"`
	ActionType     string   `json:"action_type"`
	Speech         string   `json:"speech,omitempty"`
	ActionSummary  string   `json:"action_summary,omitempty"`
	TargetActorIDs []string `json:"target_actor_ids,omitempty"`
	Intent         string   `json:"intent"`
	Rationale      string   `json:"rationale,omitempty"`
	Content        string   `json:"content,omitempty"`
}

type StoryTurnDisplayEvent struct {
	TurnIndex      int      `json:"turn_index"`
	ActorID        string   `json:"actor_id,omitempty"`
	ActorName      string   `json:"actor_name,omitempty"`
	ActionType     string   `json:"action_type"`
	Speech         string   `json:"speech,omitempty"`
	ActionSummary  string   `json:"action_summary,omitempty"`
	TargetActorIDs []string `json:"target_actor_ids,omitempty"`
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

type StoryVariablePlan struct {
	PlotVariable   StoryNarrativePlotVariable `json:"plot_variable"`
	CharacterViews []CharacterVariableView    `json:"character_views"`
}

type CharacterVariableView struct {
	CharacterID       string   `json:"character_id"`
	KnownFacts        []string `json:"known_facts"`
	Misreadings       []string `json:"misreadings"`
	EmotionalPressure string   `json:"emotional_pressure"`
	ActionBias        string   `json:"action_bias"`
}

type LoadStoryContextInput struct {
	ProjectID string `json:"project_id" jsonschema:"required"`
	SessionID string `json:"session_id" jsonschema:"required"`
}

type ChooseNextStoryActorInput struct {
	TurnIndex      int      `json:"turn_index" jsonschema:"required"`
	ActorID        string   `json:"actor_id"`
	ActorName      string   `json:"actor_name"`
	ActionType     string   `json:"action_type" jsonschema:"required"`
	Speech         string   `json:"speech"`
	ActionSummary  string   `json:"action_summary"`
	TargetActorIDs []string `json:"target_actor_ids"`
	Intent         string   `json:"intent" jsonschema:"required"`
	Rationale      string   `json:"rationale"`
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

type StoryNarrativeResult struct {
	Title        string                     `json:"title"`
	Summary      string                     `json:"summary"`
	Content      string                     `json:"content"`
	PlotVariable StoryNarrativePlotVariable `json:"plot_variable"`
	MemoryPatch  StoryNarrativeMemoryPatch  `json:"memory_patch"`
	Review       StoryNarrativeReview       `json:"review"`
	Turns        []StoryTurnPlan            `json:"turns"`
}

type StoryNarrativePlotVariable struct {
	PressureSource      string   `json:"pressure_source"`
	FocalCharacterID    string   `json:"focal_character_id"`
	CoreChoice          string   `json:"core_choice"`
	OptionA             string   `json:"option_a"`
	OptionB             string   `json:"option_b"`
	CostA               string   `json:"cost_a"`
	CostB               string   `json:"cost_b"`
	IrreversibleEffect  string   `json:"irreversible_effect"`
	RelatedCharacterIDs []string `json:"related_character_ids"`
	WorldStatePressure  []string `json:"world_state_pressure"`
}

type StoryNarrativeMemoryPatch struct {
	CharacterMemoryUpdates []StoryNarrativeCharacterMemoryUpdate `json:"character_memory_updates"`
	RelationshipUpdates    []StoryNarrativeRelationshipUpdate    `json:"relationship_updates"`
	WorldStateUpdates      []StoryNarrativeWorldStateUpdate      `json:"world_state_updates"`
}

type StoryNarrativeCharacterMemoryUpdate struct {
	CharacterID string `json:"character_id"`
	Type        string `json:"type"`
	Content     string `json:"content"`
	Importance  int    `json:"importance"`
}

type StoryNarrativeRelationshipUpdate struct {
	PairID       string `json:"pair_id"`
	Summary      string `json:"summary"`
	TensionDelta string `json:"tension_delta"`
}

type StoryNarrativeWorldStateUpdate struct {
	Key       string `json:"key"`
	Operation string `json:"operation"`
	Value     any    `json:"value"`
	Note      string `json:"note"`
}

type StoryNarrativeReview struct {
	Pass             bool     `json:"pass"`
	HardViolations   []string `json:"hard_violations"`
	ContinuityIssues []string `json:"continuity_issues"`
	StyleIssues      []string `json:"style_issues"`
	SuggestedFixes   []string `json:"suggested_fixes"`
}

type CharacterPerspective struct {
	Character         model.Character          `json:"character"`
	VisibleWorld      []model.WorldStateEntry  `json:"visible_world"`
	RecentMemories    []model.Memory           `json:"recent_memories"`
	RelationshipViews []model.RelationshipView `json:"relationship_views"`
	VariableView      *CharacterVariableView   `json:"variable_view,omitempty"`
}

type StoryNarrativeInput struct {
	Session        model.StorySession     `json:"session"`
	AuthorBible    *model.AuthorBible     `json:"author_bible,omitempty"`
	RecentChapters []model.Chapter        `json:"recent_chapters"`
	Plan           StoryPlanResult        `json:"plan"`
	Perspectives   []CharacterPerspective `json:"perspectives"`
}
