package eino

import (
	"encoding/json"
	"strings"
	"time"

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
	ContinuityIssues       []string                           `json:"continuity_issues,omitempty"`
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

type SceneContext struct {
	StoryRunID         string                            `json:"story_run_id"`
	ProjectID          string                            `json:"project_id"`
	SessionID          string                            `json:"session_id"`
	Session            SceneSessionContext               `json:"session"`
	AuthorBible        map[string]any                    `json:"author_bible,omitempty"`
	PlotVariableSeed   StoryVariablePlan                 `json:"plot_variable_seed"`
	PlannedActions     []ScenePlannedAction              `json:"planned_actions,omitempty"`
	SecondarySummaries []model.SecondaryActionSummary    `json:"secondary_summaries,omitempty"`
	Arrangement        model.SceneArrangement            `json:"arrangement,omitempty"`
	AttentionSelection model.CharacterAttentionSelection `json:"attention_selection,omitempty"`
	InFlightActions    []model.OngoingAction             `json:"in_flight_actions,omitempty"`
	CompletedActions   []model.OngoingAction             `json:"completed_actions,omitempty"`
	SupersededActions  []model.OngoingAction             `json:"superseded_actions,omitempty"`
	CollisionAt        string                            `json:"collision_at,omitempty"`
	SharedObservable   SharedObservableContext           `json:"shared_observable"`
	CharacterViews     []SceneCharacterView              `json:"character_views"`
	Constraints        SceneConstraints                  `json:"constraints"`
}

type ScenePlannedAction struct {
	CharacterID       string     `json:"character_id"`
	CharacterName     string     `json:"character_name,omitempty"`
	ActionType        string     `json:"action_type"`
	Description       string     `json:"description"`
	TargetLocationKey string     `json:"target_location_key,omitempty"`
	DurationHours     int        `json:"duration_hours"`
	StartAt           *time.Time `json:"start_at,omitempty"`
	ArriveAt          *time.Time `json:"arrive_at,omitempty"`
	EffectAt          *time.Time `json:"effect_at,omitempty"`
	EndsAt            *time.Time `json:"ends_at,omitempty"`
	ParticipantIDs    []string   `json:"participant_ids,omitempty"`
	ResourceKeys      []string   `json:"resource_keys,omitempty"`
	Rationale         string     `json:"rationale,omitempty"`
}

type SceneSessionContext struct {
	Title               string `json:"title"`
	OpeningSituation    string `json:"opening_situation"`
	AuthorIntent        string `json:"author_intent"`
	CurrentPlotVariable string `json:"current_plot_variable"`
}

type SharedObservableContext struct {
	LocationHints    []map[string]any      `json:"location_hints"`
	ActionLocations  []model.LocationState `json:"action_locations,omitempty"`
	PublicWorldState []map[string]any      `json:"public_world_state"`
	RecentChapters   []map[string]any      `json:"recent_chapters"`
}

type SceneCharacterView struct {
	CharacterID       string                  `json:"character_id"`
	Identity          map[string]any          `json:"identity"`
	KnownFacts        []string                `json:"known_facts"`
	Secrets           []string                `json:"secrets"`
	Misreadings       []string                `json:"misreadings"`
	PrivateAttitude   []map[string]string     `json:"private_attitude"`
	RecentMemories    []string                `json:"recent_memories"`
	VisibleWorld      []model.WorldStateEntry `json:"visible_world"`
	EmotionalPressure string                  `json:"emotional_pressure"`
	ActionBias        string                  `json:"action_bias"`
}

type SceneConstraints struct {
	MaxTurns        int `json:"max_turns"`
	MaxInteractions int `json:"max_interactions"`
}

type sceneRecord struct {
	Type             string                      `json:"type"`
	PlotVariable     StoryNarrativePlotVariable  `json:"plot_variable"`
	Event            model.StoryEventPlan        `json:"event"`
	InteractionGroup model.StoryInteractionGroup `json:"interaction_group"`
	Turn             StoryTurnPlan               `json:"turn"`
	StopReason       string                      `json:"stop_reason"`
}

type sceneBatchResult struct {
	PlotVariable      StoryNarrativePlotVariable    `json:"plot_variable"`
	Events            []model.StoryEventPlan        `json:"events"`
	EventPlan         []model.StoryEventPlan        `json:"event_plan"`
	InteractionGroups []model.StoryInteractionGroup `json:"interaction_groups"`
	Turns             []StoryTurnPlan               `json:"turns"`
	StopReason        string                        `json:"stop_reason"`
}

type ReflectionContext struct {
	Scene              ReflectionScene                `json:"scene"`
	Characters         []ReflectionCharacter          `json:"characters"`
	PerceptionIndex    []PerceptionIndexEntry         `json:"perception_index"`
	PriorMemories      map[string][]string            `json:"prior_memories"`
	Relationships      []map[string]any               `json:"relationships"`
	WorldState         []map[string]any               `json:"world_state"`
	Arrangement        model.SceneArrangement         `json:"arrangement,omitempty"`
	SecondarySummaries []model.SecondaryActionSummary `json:"secondary_summaries,omitempty"`
}

type ReflectionScene struct {
	PlotVariable           StoryNarrativePlotVariable         `json:"plot_variable"`
	Events                 []model.StoryEventPlan             `json:"events"`
	Turns                  []model.StoryTurn                  `json:"turns"`
	InteractionTranscripts []model.StoryInteractionTranscript `json:"interaction_transcripts"`
	StopReason             string                             `json:"stop_reason"`
}

type ReflectionCharacter struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Role string `json:"role,omitempty"`
}

type PerceptionIndexEntry struct {
	CharacterID          string   `json:"character_id"`
	WitnessedTurnIndexes []int    `json:"witnessed_turn_indexes"`
	WitnessedEventIDs    []string `json:"witnessed_event_ids"`
}

type SceneReflectionResult struct {
	Summary            string                          `json:"summary"`
	CharacterTakeaways []CharacterReflectionTakeaway   `json:"character_takeaways"`
	MemoryPatch        StoryNarrativeMemoryPatch       `json:"memory_patch"`
	TierTransitions    []model.CharacterTierTransition `json:"tier_transitions,omitempty"`
	AmbientPromotions  []model.AmbientPromotionRequest `json:"ambient_promotions,omitempty"`
}

type CharacterReflectionTakeaway struct {
	CharacterID string `json:"character_id"`
	Summary     string `json:"summary"`
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
	ResourceKeys   []string `json:"resource_keys"`
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
