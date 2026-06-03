package eino

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/fishimei/NovelOS/internal/application/model"
)

type actionDecisionSharedPrompt struct {
	StoryTime      string                       `json:"story_time,omitempty"`
	WorldFacts     []promptFact                 `json:"world_facts,omitempty"`
	Locations      []promptLocation             `json:"locations,omitempty"`
	Factions       []promptFaction              `json:"factions,omitempty"`
	OutputContract actionDecisionOutputContract `json:"output_contract"`
}

type actionDecisionCharacterPrompt struct {
	Character       promptActionCharacter `json:"character"`
	State           promptActionState     `json:"state"`
	CurrentLocation *promptLocation       `json:"current_location,omitempty"`
	NearbyLocations []promptLocation      `json:"nearby_locations,omitempty"`
	Relationships   []promptRelationship  `json:"relationships,omitempty"`
	PrivateFacts    []string              `json:"private_facts,omitempty"`
}

type actionDecisionOutputContract struct {
	Format          string   `json:"format"`
	Fields          []string `json:"fields"`
	ActionTypes     []string `json:"action_types"`
	LocationRule    string   `json:"location_rule"`
	ParticipantRule string   `json:"participant_rule"`
	TimingRule      string   `json:"timing_rule"`
}

type promptActionCharacter struct {
	ID                  string   `json:"id"`
	Name                string   `json:"name,omitempty"`
	Role                string   `json:"role,omitempty"`
	Profile             string   `json:"profile,omitempty"`
	Personality         string   `json:"personality,omitempty"`
	Goals               []string `json:"goals,omitempty"`
	Fears               []string `json:"fears,omitempty"`
	Constraints         []string `json:"constraints,omitempty"`
	RecentMemorySummary string   `json:"recent_memory_summary,omitempty"`
}

type promptActionState struct {
	LocationKey   string              `json:"location_key,omitempty"`
	Position      *promptPoint        `json:"position,omitempty"`
	OngoingAction *promptOngoingBrief `json:"ongoing_action,omitempty"`
}

type promptPoint struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type promptOngoingBrief struct {
	ActionType        string   `json:"action_type,omitempty"`
	Description       string   `json:"description,omitempty"`
	TargetLocationKey string   `json:"target_location_key,omitempty"`
	ParticipantIDs    []string `json:"participant_ids,omitempty"`
	ResourceKeys      []string `json:"resource_keys,omitempty"`
	EndsAt            string   `json:"ends_at,omitempty"`
	Rationale         string   `json:"rationale,omitempty"`
}

type promptFact struct {
	Key        string `json:"key"`
	Value      any    `json:"value,omitempty"`
	Note       string `json:"note,omitempty"`
	Importance int    `json:"importance,omitempty"`
	Volatility int    `json:"volatility,omitempty"`
}

type promptRelationship struct {
	TargetCharacterID      string   `json:"target_character_id"`
	Summary                string   `json:"summary,omitempty"`
	PublicAttitude         string   `json:"public_attitude,omitempty"`
	PrivateAttitude        string   `json:"private_attitude,omitempty"`
	BelievedTargetAttitude string   `json:"believed_target_attitude,omitempty"`
	MaskingStrategy        string   `json:"masking_strategy,omitempty"`
	TensionPoints          []string `json:"tension_points,omitempty"`
	RecentEventSummaries   []string `json:"recent_event_summaries,omitempty"`
}

type promptLocation struct {
	ID          string   `json:"id"`
	Name        string   `json:"name,omitempty"`
	Type        string   `json:"type,omitempty"`
	Description string   `json:"description,omitempty"`
	Distance    float64  `json:"distance,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

type promptFaction struct {
	LocationID  string `json:"location_id,omitempty"`
	FactionName string `json:"faction_name"`
	Influence   int    `json:"influence,omitempty"`
	Attitude    string `json:"attitude,omitempty"`
	Description string `json:"description,omitempty"`
}

type scenePromptInput struct {
	Session         scenePromptSession     `json:"session"`
	AuthorBible     map[string]any         `json:"author_bible,omitempty"`
	PlotVariable    StoryVariablePlan      `json:"plot_variable_seed"`
	PlannedActions  []scenePromptAction    `json:"planned_actions,omitempty"`
	ConflictContext scenePromptConflict    `json:"conflict_context,omitempty"`
	Shared          scenePromptShared      `json:"shared"`
	Characters      []scenePromptCharacter `json:"characters"`
	Constraints     SceneConstraints       `json:"constraints"`
	OutputContract  sceneOutputContract    `json:"output_contract"`
}

type scenePromptSession struct {
	Title         string `json:"title,omitempty"`
	Situation     string `json:"situation,omitempty"`
	AuthorIntent  string `json:"author_intent,omitempty"`
	AuthorMessage string `json:"author_message,omitempty"`
	CurrentThread string `json:"current_thread,omitempty"`
}

type scenePromptConflict struct {
	InFlightActions   []scenePromptAction `json:"in_flight_actions,omitempty"`
	CompletedActions  []scenePromptAction `json:"completed_actions,omitempty"`
	SupersededActions []scenePromptAction `json:"superseded_actions,omitempty"`
	CollisionAt       string              `json:"collision_at,omitempty"`
}

type scenePromptShared struct {
	LocationHints  []promptFact     `json:"location_hints,omitempty"`
	WorldFacts     []promptFact     `json:"world_facts,omitempty"`
	RecentChapters []map[string]any `json:"recent_chapters,omitempty"`
}

type scenePromptCharacter struct {
	ID          string       `json:"id"`
	Name        string       `json:"name,omitempty"`
	Role        string       `json:"role,omitempty"`
	Profile     string       `json:"profile,omitempty"`
	Voice       string       `json:"voice,omitempty"`
	Drives      []string     `json:"drives,omitempty"`
	Known       []string     `json:"known,omitempty"`
	Private     []string     `json:"private,omitempty"`
	Misreadings []string     `json:"misreadings,omitempty"`
	Memories    []string     `json:"memories,omitempty"`
	Visible     []promptFact `json:"visible,omitempty"`
	Pressure    string       `json:"pressure,omitempty"`
	Bias        string       `json:"bias,omitempty"`
}

type scenePromptAction struct {
	CharacterID   string   `json:"character_id"`
	Type          string   `json:"type,omitempty"`
	Goal          string   `json:"goal,omitempty"`
	Location      string   `json:"location,omitempty"`
	DurationHours int      `json:"duration_hours,omitempty"`
	Timing        string   `json:"timing,omitempty"`
	Participants  []string `json:"participants,omitempty"`
	Resources     []string `json:"resources,omitempty"`
	Rationale     string   `json:"rationale,omitempty"`
}

type sceneOutputContract struct {
	Format                  string   `json:"format"`
	AllowedRecordTypes      []string `json:"allowed_record_types"`
	RecordOrder             []string `json:"record_order"`
	WhenPlannedActionsExist []string `json:"when_planned_actions_exist,omitempty"`
	WhenNoPlannedActions    []string `json:"when_no_planned_actions,omitempty"`
	MaxTurns                int      `json:"max_turns"`
	MaxInteractions         int      `json:"max_interactions"`
}

type reflectionPromptInput struct {
	Scene              reflectionPromptScene    `json:"scene"`
	Characters         []ReflectionCharacter    `json:"characters"`
	PerceptionIndex    []PerceptionIndexEntry   `json:"perception_index"`
	PriorMemories      map[string][]string      `json:"prior_memories,omitempty"`
	RelationshipBriefs []string                 `json:"relationships,omitempty"`
	WorldFacts         []promptFact             `json:"world_facts,omitempty"`
	OutputContract     reflectionOutputContract `json:"output_contract"`
}

type reflectionPromptScene struct {
	PlotVariable           StoryNarrativePlotVariable         `json:"plot_variable"`
	Events                 []model.StoryEventPlan             `json:"events"`
	Turns                  []model.StoryTurn                  `json:"turns"`
	InteractionTranscripts []model.StoryInteractionTranscript `json:"interaction_transcripts,omitempty"`
	StopReason             string                             `json:"stop_reason,omitempty"`
}

type reflectionOutputContract struct {
	Format string   `json:"format"`
	Rules  []string `json:"rules"`
}

func buildActionDecisionSharedPrompt(world model.WorldSnapshot) actionDecisionSharedPrompt {
	return actionDecisionSharedPrompt{
		StoryTime:      formatPromptTime(world.StoryTime),
		WorldFacts:     promptFactsFromWorldStateMap(world.WorldState, 12),
		Locations:      promptLocationsFromStates(world.Locations, 20),
		Factions:       promptFactions(world.Factions, ""),
		OutputContract: defaultActionDecisionOutputContract(),
	}
}

func buildActionDecisionCharacterPrompt(input model.CharacterActionDecisionInput) actionDecisionCharacterPrompt {
	currentLocation := promptLocationFromState(input.Location)
	var current *promptLocation
	if currentLocation.ID != "" {
		current = &currentLocation
	}
	return actionDecisionCharacterPrompt{
		Character:       promptActionCharacterFromModel(input.Character),
		State:           promptActionStateFromRuntime(input.CharacterState),
		CurrentLocation: current,
		NearbyLocations: promptNearbyLocations(input.NearbyLocations, 5),
		Relationships:   promptRelationshipsForCharacter(input.World.Relationships, input.Character.ID, 8),
		PrivateFacts:    firstStrings(input.Character.Secrets, 3),
	}
}

func defaultActionDecisionOutputContract() actionDecisionOutputContract {
	return actionDecisionOutputContract{
		Format:          "JSON object only",
		Fields:          []string{"action_type", "description", "duration_hours", "target_location_key", "participant_ids", "affected_resource_keys", "rationale"},
		ActionTypes:     []string{"observe", "action", "speak", "silence"},
		LocationRule:    "target_location_key must be current_location.id or one of nearby_locations[].id when supplied",
		ParticipantRule: "participant_ids only names characters this action actively contacts, targets, protects, follows, attacks, or negotiates with",
		TimingRule:      "duration_hours must be a positive integer; do not output absolute ArriveAt/EffectAt/EndsAt/StartAt fields",
	}
}

func buildScenePromptInput(ctx SceneContext) scenePromptInput {
	contract := sceneOutputContract{
		Format:             "strict NDJSON: one JSON object per line; no array; no markdown fence",
		AllowedRecordTypes: []string{"plot_variable", "event", "interaction", "turn", "stop"},
		RecordOrder:        []string{"plot_variable", "event when planned_actions empty", "interaction", "turn", "stop"},
		MaxTurns:           ctx.Constraints.MaxTurns,
		MaxInteractions:    ctx.Constraints.MaxInteractions,
	}
	if len(ctx.PlannedActions) > 0 {
		contract.WhenPlannedActionsExist = []string{"do not emit event records", "treat planned_actions as already committed event candidates", "emit plot_variable, interaction, turn, stop records only"}
	} else {
		contract.WhenNoPlannedActions = []string{"emit event records before interaction records to establish location/action candidates"}
	}
	return scenePromptInput{
		Session: scenePromptSession{
			Title:         ctx.Session.Title,
			Situation:     ctx.Session.OpeningSituation,
			AuthorIntent:  ctx.Session.AuthorIntent,
			AuthorMessage: ctx.Session.LastAuthorMessage,
			CurrentThread: ctx.Session.CurrentPlotVariable,
		},
		AuthorBible:    ctx.AuthorBible,
		PlotVariable:   ctx.PlotVariableSeed,
		PlannedActions: promptActionsFromScenePlanned(ctx.PlannedActions),
		ConflictContext: scenePromptConflict{
			InFlightActions:   promptActionsFromOngoing(ctx.InFlightActions),
			CompletedActions:  promptActionsFromOngoing(ctx.CompletedActions),
			SupersededActions: promptActionsFromOngoing(ctx.SupersededActions),
			CollisionAt:       ctx.CollisionAt,
		},
		Shared: scenePromptShared{
			LocationHints:  promptFactsFromMaps(ctx.SharedObservable.LocationHints, 5),
			WorldFacts:     promptFactsFromMaps(ctx.SharedObservable.PublicWorldState, 8),
			RecentChapters: ctx.SharedObservable.RecentChapters,
		},
		Characters:     promptSceneCharacters(ctx.CharacterViews),
		Constraints:    ctx.Constraints,
		OutputContract: contract,
	}
}

func buildReflectionPromptInput(ctx ReflectionContext) reflectionPromptInput {
	return reflectionPromptInput{
		Scene: reflectionPromptScene{
			PlotVariable:           ctx.Scene.PlotVariable,
			Events:                 ctx.Scene.Events,
			Turns:                  ctx.Scene.Turns,
			InteractionTranscripts: ctx.Scene.InteractionTranscripts,
			StopReason:             ctx.Scene.StopReason,
		},
		Characters:         ctx.Characters,
		PerceptionIndex:    ctx.PerceptionIndex,
		PriorMemories:      limitStringMap(ctx.PriorMemories, 8),
		RelationshipBriefs: promptRelationshipBriefs(ctx.Relationships, 20),
		WorldFacts:         promptFactsFromMaps(ctx.WorldState, 20),
		OutputContract: reflectionOutputContract{Format: "JSON object only", Rules: []string{
			"write only new or changed memories",
			"Deduplicate prior_memories and do not restate them",
			"character_memory_updates must be grounded in visible event/turn ids from perception_index",
			"relationship_updates must be based on actual interaction or observable consequence",
			"world_state_updates must be concrete state changes, not mood or prose",
		}},
	}
}

func promptActionCharacterFromModel(character model.Character) promptActionCharacter {
	return promptActionCharacter{
		ID:                  character.ID,
		Name:                character.Name,
		Role:                character.Role,
		Profile:             character.Profile,
		Personality:         character.Personality,
		Goals:               firstStrings(character.Goals, 3),
		Fears:               firstStrings(character.Fears, 3),
		Constraints:         firstStrings(character.Constraints, 3),
		RecentMemorySummary: character.RecentMemorySummary,
	}
}

func promptActionStateFromRuntime(state model.CharacterRuntimeState) promptActionState {
	out := promptActionState{LocationKey: state.LocationKey}
	if state.X != 0 || state.Y != 0 {
		out.Position = &promptPoint{X: state.X, Y: state.Y}
	}
	if state.OngoingAction != nil {
		out.OngoingAction = promptOngoingAction(state.OngoingAction)
	}
	return out
}

func promptOngoingAction(action *model.OngoingAction) *promptOngoingBrief {
	if action == nil {
		return nil
	}
	return &promptOngoingBrief{
		ActionType:        action.ActionType,
		Description:       action.Description,
		TargetLocationKey: action.TargetLocationKey,
		ParticipantIDs:    firstStrings(action.ParticipantIDs, 5),
		ResourceKeys:      firstStrings(action.ResourceKeys, 5),
		EndsAt:            formatPromptTime(action.EndsAt),
		Rationale:         action.Rationale,
	}
}

func promptFactsFromWorldStateMap(entries map[string]model.WorldStateEntry, limit int) []promptFact {
	facts := make([]promptFact, 0, len(entries))
	for key, entry := range entries {
		if entry.Key == "" {
			entry.Key = key
		}
		facts = append(facts, promptFactFromWorldState(entry))
	}
	sort.SliceStable(facts, func(i, j int) bool {
		if facts[i].Importance == facts[j].Importance {
			if facts[i].Volatility == facts[j].Volatility {
				return facts[i].Key < facts[j].Key
			}
			return facts[i].Volatility > facts[j].Volatility
		}
		return facts[i].Importance > facts[j].Importance
	})
	return firstEntries(facts, limit)
}

func promptFactFromWorldState(entry model.WorldStateEntry) promptFact {
	return promptFact{Key: entry.Key, Value: entry.Value, Note: entry.Note, Importance: entry.Importance, Volatility: entry.Volatility}
}

func promptFactsFromMaps(entries []map[string]any, limit int) []promptFact {
	facts := make([]promptFact, 0, len(entries))
	for _, entry := range entries {
		key, _ := entry["key"].(string)
		if strings.TrimSpace(key) == "" {
			continue
		}
		facts = append(facts, promptFact{Key: key, Value: entry["value"], Note: promptStringValue(entry["note"]), Importance: promptIntValue(entry["importance"]), Volatility: promptIntValue(entry["volatility"])})
	}
	return firstEntries(facts, limit)
}

func promptLocationFromState(location model.LocationState) promptLocation {
	return promptLocation{ID: location.ID, Name: location.Name, Type: location.Type, Description: location.Description}
}

func promptLocationsFromStates(locations []model.LocationState, limit int) []promptLocation {
	out := make([]promptLocation, 0, len(locations))
	for _, location := range locations {
		if location.ID != "" {
			out = append(out, promptLocationFromState(location))
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return firstEntries(out, limit)
}

func promptNearbyLocations(locations []model.NearbyLocationContext, limit int) []promptLocation {
	out := make([]promptLocation, 0, len(locations))
	for _, nearby := range firstEntries(locations, limit) {
		location := promptLocationFromState(nearby.Location)
		location.Distance = nearby.Distance
		if location.ID != "" {
			out = append(out, location)
		}
	}
	return out
}

func promptFactions(influences []model.FactionInfluence, locationID string) []promptFaction {
	out := make([]promptFaction, 0, len(influences))
	for _, influence := range influences {
		if locationID != "" && influence.LocationID != locationID {
			continue
		}
		if influence.FactionName == "" {
			continue
		}
		out = append(out, promptFaction{LocationID: influence.LocationID, FactionName: influence.FactionName, Influence: influence.Influence, Attitude: influence.Attitude, Description: influence.Description})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].LocationID == out[j].LocationID {
			return out[i].FactionName < out[j].FactionName
		}
		return out[i].LocationID < out[j].LocationID
	})
	return out
}

func promptRelationshipsForCharacter(relationships map[string]model.Relationship, characterID string, limit int) []promptRelationship {
	out := make([]promptRelationship, 0, len(relationships))
	keys := make([]string, 0, len(relationships))
	for key := range relationships {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		relationship := relationships[key]
		for _, view := range relationship.Views {
			if view.SourceCharacterID != characterID || view.TargetCharacterID == "" {
				continue
			}
			out = append(out, promptRelationship{
				TargetCharacterID:      view.TargetCharacterID,
				Summary:                relationship.Pair.Summary,
				PublicAttitude:         view.PublicAttitude,
				PrivateAttitude:        view.PrivateAttitude,
				BelievedTargetAttitude: view.BelievedTargetAttitude,
				MaskingStrategy:        view.MaskingStrategy,
				TensionPoints:          firstStrings(relationship.Pair.TensionPoints, 3),
				RecentEventSummaries:   relationshipEventSummaries(relationship.RecentEvents, 2),
			})
		}
	}
	return firstEntries(out, limit)
}

func relationshipEventSummaries(events []model.RelationshipEvent, limit int) []string {
	out := make([]string, 0, minInt(len(events), limit))
	for _, event := range firstEntries(events, limit) {
		out = append(out, firstText(event.Summary, event.EventType))
	}
	return cleanStrings(out)
}

func promptActionsFromScenePlanned(actions []ScenePlannedAction) []scenePromptAction {
	out := make([]scenePromptAction, 0, len(actions))
	for _, action := range actions {
		out = append(out, scenePromptAction{
			CharacterID:   action.CharacterID,
			Type:          action.ActionType,
			Goal:          action.Description,
			Location:      action.TargetLocationKey,
			DurationHours: action.DurationHours,
			Timing:        promptActionTiming(action.StartAt, action.ArriveAt, action.EffectAt, action.EndsAt),
			Participants:  firstStrings(action.ParticipantIDs, 5),
			Resources:     firstStrings(action.ResourceKeys, 5),
			Rationale:     action.Rationale,
		})
	}
	return out
}

func promptActionsFromOngoing(actions []model.OngoingAction) []scenePromptAction {
	out := make([]scenePromptAction, 0, len(actions))
	for _, action := range actions {
		out = append(out, scenePromptAction{
			CharacterID:  action.CharacterID,
			Type:         action.ActionType,
			Goal:         action.Description,
			Location:     action.TargetLocationKey,
			Timing:       promptActionTimingValues(action.StartAt, action.ArriveAt, action.EffectAt, action.EndsAt),
			Participants: firstStrings(action.ParticipantIDs, 5),
			Resources:    firstStrings(action.ResourceKeys, 5),
			Rationale:    action.Rationale,
		})
	}
	return out
}

func promptActionTimingValues(startAt time.Time, arriveAt time.Time, effectAt time.Time, endsAt time.Time) string {
	return promptActionTiming(optionalTime(startAt), optionalTime(arriveAt), optionalTime(effectAt), optionalTime(endsAt))
}

func promptActionTiming(startAt *time.Time, arriveAt *time.Time, effectAt *time.Time, endsAt *time.Time) string {
	parts := []string{}
	if value := formatPromptTimePtr(startAt); value != "" {
		parts = append(parts, "starts "+value)
	}
	if value := formatPromptTimePtr(arriveAt); value != "" {
		parts = append(parts, "arrives "+value)
	}
	if value := formatPromptTimePtr(effectAt); value != "" {
		parts = append(parts, "effect "+value)
	}
	if value := formatPromptTimePtr(endsAt); value != "" {
		parts = append(parts, "ends "+value)
	}
	return strings.Join(parts, "; ")
}

func promptSceneCharacters(views []SceneCharacterView) []scenePromptCharacter {
	out := make([]scenePromptCharacter, 0, len(views))
	for _, view := range views {
		character := scenePromptCharacter{
			ID:          view.CharacterID,
			Name:        promptStringValue(view.Identity["name"]),
			Role:        promptStringValue(view.Identity["role"]),
			Profile:     promptStringValue(view.Identity["profile"]),
			Voice:       promptStringValue(view.Identity["voice_style"]),
			Drives:      promptCharacterDrives(view.Identity),
			Known:       firstStrings(view.KnownFacts, 5),
			Private:     promptPrivateFacts(view.Secrets, view.PrivateAttitude),
			Misreadings: firstStrings(view.Misreadings, 3),
			Memories:    firstStrings(view.RecentMemories, 4),
			Visible:     promptFactsFromWorldStates(view.VisibleWorld, 5),
			Pressure:    view.EmotionalPressure,
			Bias:        view.ActionBias,
		}
		if memoryBrief := promptStringValue(view.Identity["memory_brief"]); memoryBrief != "" {
			character.Memories = append([]string{memoryBrief}, character.Memories...)
		}
		out = append(out, character)
	}
	return out
}

func promptCharacterDrives(identity map[string]any) []string {
	out := []string{}
	for _, key := range []string{"goals", "fears", "constraints"} {
		out = append(out, promptStringSliceValue(identity[key])...)
	}
	return firstStrings(cleanStrings(out), 9)
}

func promptPrivateFacts(secrets []string, attitudes []map[string]string) []string {
	out := append([]string(nil), firstStrings(secrets, 3)...)
	for _, attitude := range firstEntries(attitudes, 6) {
		target := attitude["target_character_id"]
		parts := []string{}
		if attitude["public_attitude"] != "" {
			parts = append(parts, "public="+attitude["public_attitude"])
		}
		if attitude["private_attitude"] != "" {
			parts = append(parts, "private="+attitude["private_attitude"])
		}
		if attitude["believed_target_attitude"] != "" {
			parts = append(parts, "believes_target="+attitude["believed_target_attitude"])
		}
		if attitude["masking_strategy"] != "" {
			parts = append(parts, "mask="+attitude["masking_strategy"])
		}
		if len(parts) > 0 {
			out = append(out, strings.TrimSpace(target+": "+strings.Join(parts, "; ")))
		}
	}
	return cleanStrings(out)
}

func promptFactsFromWorldStates(entries []model.WorldStateEntry, limit int) []promptFact {
	out := make([]promptFact, 0, minInt(len(entries), limit))
	for _, entry := range firstEntries(entries, limit) {
		if entry.Key != "" {
			out = append(out, promptFactFromWorldState(entry))
		}
	}
	return out
}

func promptRelationshipBriefs(relationships []map[string]any, limit int) []string {
	out := make([]string, 0, minInt(len(relationships), limit))
	for _, relationship := range firstEntries(relationships, limit) {
		left := promptStringValue(relationship["left_character_id"])
		right := promptStringValue(relationship["right_character_id"])
		summary := promptStringValue(relationship["summary"])
		tensions := promptStringSliceValue(relationship["tension_points"])
		brief := strings.TrimSpace(left + " -> " + right + ": " + summary)
		if len(tensions) > 0 {
			brief += "；tension: " + strings.Join(tensions, ", ")
		}
		out = append(out, brief)
	}
	return cleanStrings(out)
}

func limitStringMap(values map[string][]string, limit int) map[string][]string {
	if len(values) == 0 {
		return nil
	}
	out := make(map[string][]string, len(values))
	for key, items := range values {
		out[key] = firstStrings(items, limit)
	}
	return out
}

func promptStringValue(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case fmt.Stringer:
		return typed.String()
	default:
		return ""
	}
}

func promptStringSliceValue(value any) []string {
	switch typed := value.(type) {
	case []string:
		return typed
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			if text := promptStringValue(item); text != "" {
				out = append(out, text)
			}
		}
		return out
	default:
		return nil
	}
}

func promptIntValue(value any) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	default:
		return 0
	}
}

func formatPromptTimePtr(value *time.Time) string {
	if value == nil {
		return ""
	}
	return formatPromptTime(*value)
}

func formatPromptTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}
