package eino

import (
	"encoding/json"
	"strings"

	"github.com/fishimei/NovelOS/internal/application/model"
)

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
	TurnIndex          int      `json:"turn_index"`
	ActorID            string   `json:"actor_id,omitempty"`
	ActorName          string   `json:"actor_name,omitempty"`
	ActionType         string   `json:"action_type"`
	Speech             string   `json:"speech,omitempty"`
	ActionSummary      string   `json:"action_summary,omitempty"`
	TargetActorIDs     []string `json:"target_actor_ids,omitempty"`
	Intent             string   `json:"intent"`
	Rationale          string   `json:"rationale,omitempty"`
	Content            string   `json:"content,omitempty"`
	InteractionGroupID string   `json:"interaction_group_id,omitempty"`
	LocationKey        string   `json:"location_key,omitempty"`
	LocationName       string   `json:"location_name,omitempty"`
	Phase              string   `json:"phase,omitempty"`
}

type StoryTurnDisplayEvent struct {
	TurnIndex          int      `json:"turn_index"`
	ActorID            string   `json:"actor_id,omitempty"`
	ActorName          string   `json:"actor_name,omitempty"`
	ActionType         string   `json:"action_type"`
	Speech             string   `json:"speech,omitempty"`
	ActionSummary      string   `json:"action_summary,omitempty"`
	TargetActorIDs     []string `json:"target_actor_ids,omitempty"`
	InteractionGroupID string   `json:"interaction_group_id,omitempty"`
	LocationKey        string   `json:"location_key,omitempty"`
	LocationName       string   `json:"location_name,omitempty"`
	Phase              string   `json:"phase,omitempty"`
}

type StoryStopDecision struct {
	Stop   bool   `json:"stop"`
	Reason string `json:"reason"`
}

type StoryPlanResult struct {
	Summary                string                             `json:"summary"`
	StopReason             string                             `json:"stop_reason"`
	Turns                  []StoryTurnPlan                    `json:"turns"`
	EventPlan              []model.StoryEventPlan             `json:"event_plan,omitempty"`
	InteractionAnalysis    model.StoryInteractionAnalysis     `json:"interaction_analysis,omitempty"`
	InteractionTranscripts []model.StoryInteractionTranscript `json:"interaction_transcripts,omitempty"`
}

type StoryEventRecordResult struct {
	Event                  model.StoryEventPlan           `json:"event"`
	SameLocationCandidates []model.StoryLocationGroup     `json:"same_location_candidates"`
	InteractionAnalysis    model.StoryInteractionAnalysis `json:"interaction_analysis"`
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
	TurnIndex          int      `json:"turn_index" jsonschema:"required"`
	ActorID            string   `json:"actor_id"`
	ActorName          string   `json:"actor_name"`
	ActionType         string   `json:"action_type" jsonschema:"required"`
	Speech             string   `json:"speech"`
	ActionSummary      string   `json:"action_summary"`
	TargetActorIDs     []string `json:"target_actor_ids"`
	Intent             string   `json:"intent" jsonschema:"required"`
	Rationale          string   `json:"rationale"`
	InteractionGroupID string   `json:"interaction_group_id"`
	LocationKey        string   `json:"location_key"`
	LocationName       string   `json:"location_name"`
	Phase              string   `json:"phase"`
}

type RecordStoryEventInput struct {
	TimeIndex      int      `json:"time_index"`
	CharacterID    string   `json:"character_id"`
	CharacterName  string   `json:"character_name"`
	LocationKey    string   `json:"location_key" jsonschema:"required"`
	LocationName   string   `json:"location_name"`
	ActionType     string   `json:"action_type" jsonschema:"required"`
	Summary        string   `json:"summary" jsonschema:"required"`
	Intent         string   `json:"intent"`
	Visibility     string   `json:"visibility"`
	TargetActorIDs []string `json:"target_actor_ids"`
}

type SelectStoryInteractionInput struct {
	LocationKey     string   `json:"location_key" jsonschema:"required"`
	CharacterIDs    []string `json:"character_ids" jsonschema:"required"`
	EventIDs        []string `json:"event_ids"`
	ShouldInteract  bool     `json:"should_interact" jsonschema:"required"`
	InteractionType string   `json:"interaction_type"`
	Stakes          string   `json:"stakes"`
	Rationale       string   `json:"rationale"`
	Priority        int      `json:"priority"`
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

type flexibleStrings []string

func (s *flexibleStrings) UnmarshalJSON(data []byte) error {
	var values []string
	if err := json.Unmarshal(data, &values); err == nil {
		*s = cleanStrings(values)
		return nil
	}
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	*s = cleanStrings([]string{value})
	return nil
}

func cleanStrings(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			out = append(out, value)
		}
	}
	return out
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

func (v *StoryNarrativePlotVariable) UnmarshalJSON(data []byte) error {
	var raw struct {
		PressureSource      string          `json:"pressure_source"`
		FocalCharacterID    string          `json:"focal_character_id"`
		CoreChoice          string          `json:"core_choice"`
		OptionA             string          `json:"option_a"`
		OptionB             string          `json:"option_b"`
		CostA               string          `json:"cost_a"`
		CostB               string          `json:"cost_b"`
		IrreversibleEffect  string          `json:"irreversible_effect"`
		RelatedCharacterIDs flexibleStrings `json:"related_character_ids"`
		WorldStatePressure  flexibleStrings `json:"world_state_pressure"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	*v = StoryNarrativePlotVariable{
		PressureSource:      raw.PressureSource,
		FocalCharacterID:    raw.FocalCharacterID,
		CoreChoice:          raw.CoreChoice,
		OptionA:             raw.OptionA,
		OptionB:             raw.OptionB,
		CostA:               raw.CostA,
		CostB:               raw.CostB,
		IrreversibleEffect:  raw.IrreversibleEffect,
		RelatedCharacterIDs: []string(raw.RelatedCharacterIDs),
		WorldStatePressure:  []string(raw.WorldStatePressure),
	}
	return nil
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
	PairID       string                    `json:"pair_id"`
	Summary      string                    `json:"summary"`
	TensionDelta string                    `json:"tension_delta"`
	Events       []model.RelationshipEvent `json:"events"`
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
