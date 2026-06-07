package eino

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"sort"
	"strings"
	"sync"
	"time"

	llmmodel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"

	"github.com/fishimei/NovelOS/internal/application/model"
	"github.com/fishimei/NovelOS/internal/application/port"
	"github.com/fishimei/NovelOS/internal/config"
	"github.com/fishimei/NovelOS/internal/domain"
)

type StoryRunGeneratorDeps struct {
	Config            config.AIConfig
	Sessions          port.StorySessionRepository
	AuthorBibles      port.AuthorBibleRepository
	WorldState        port.WorldStateRepository
	Characters        port.CharacterRepository
	Relationships     port.RelationshipRepository
	Chapters          port.ChapterRepository
	Memories          port.MemoryRepository
	MemoryService     port.CharacterMemoryService
	StoryEvents       port.StoryEventStore
	ActionDecider     port.CharacterActionDecider
	LocationInspector port.LocationInspectionService
	Events            port.GenerationEventStream
	Audit             port.AuditRepository
	Clock             port.Clock
	IDs               port.IDGenerator
}

type StoryRunGenerator struct {
	cfg               config.AIConfig
	model             llmmodel.ToolCallingChatModel
	actionDecider     port.CharacterActionDecider
	locationInspector port.LocationInspectionService
	deps              storyGeneratorDeps
	clock             port.Clock
	ids               port.IDGenerator
	maxTurns          int
	scenePrompt       string
	reflectPrompt     string
	resultPrompt      string
	maxSceneTokens    int
	maxReflectTokens  int
}

func NewStoryRunGenerator(ctx context.Context, deps StoryRunGeneratorDeps) (*StoryRunGenerator, error) {
	maxTurns := positiveIntOrDefault(deps.Config.StoryAgent.MaxTurns, config.DefaultStoryAgentMaxTurns)
	chatModel, err := newOpenAIChatModel(ctx, deps.Config)
	if err != nil {
		return nil, err
	}
	generator := &StoryRunGenerator{
		cfg:   deps.Config,
		model: chatModel,
		deps: storyGeneratorDeps{
			sessions:      deps.Sessions,
			authorBibles:  deps.AuthorBibles,
			worldState:    deps.WorldState,
			characters:    deps.Characters,
			relationships: deps.Relationships,
			chapters:      deps.Chapters,
			memories:      deps.Memories,
			memoryService: deps.MemoryService,
			storyEvents:   deps.StoryEvents,
			events:        deps.Events,
			audit:         deps.Audit,
		},
		actionDecider:     deps.ActionDecider,
		locationInspector: deps.LocationInspector,
		clock:             deps.Clock,
		ids:               deps.IDs,
		maxTurns:          maxTurns,
		scenePrompt:       deps.Config.StoryAgent.ScenePrompt,
		reflectPrompt:     deps.Config.StoryAgent.ReflectPrompt,
		resultPrompt:      deps.Config.StoryAgent.ResultPrompt,
		maxSceneTokens:    positiveIntOrDefault(deps.Config.StoryAgent.MaxSceneTokens, config.DefaultStoryAgentMaxSceneTokens),
		maxReflectTokens:  positiveIntOrDefault(deps.Config.StoryAgent.MaxReflectTokens, config.DefaultStoryAgentMaxReflectTokens),
	}
	return generator, nil
}

func (g *StoryRunGenerator) Generate(ctx context.Context, input port.StoryRunGenerationInput) (model.StoryRunResult, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	state := &storyRunState{run: input.Run, session: input.Session, maxTurns: g.maxTurns}
	g.publishStoryOrchestrationStarted(ctx, input)
	if err := g.returnIfStopped(ctx, input, "orchestration_start"); err != nil {
		return model.StoryRunResult{}, err
	}
	snapshot, err := loadStoryContext(ctx, g.deps, state, LoadStoryContextInput{ProjectID: input.Run.ProjectID, SessionID: input.Session.ID})
	if err != nil {
		return model.StoryRunResult{}, err
	}
	updateStoryRunStep(ctx, g.deps, input.Run.RunID, domain.RunStatusGeneratingPlotVariable, 20)
	variable, err := g.seedPlotVariable(ctx, input, snapshot)
	if err != nil {
		return model.StoryRunResult{}, fmt.Errorf("seed plot variable: %w", err)
	}
	state.variable = variable
	plannedActions, err := g.planCharacterActions(ctx, input, snapshot)
	if err != nil {
		return model.StoryRunResult{}, fmt.Errorf("plan character actions: %w", err)
	}
	if err := g.returnIfStopped(ctx, input, "planning_actions"); err != nil {
		return model.StoryRunResult{}, err
	}
	seedPlannedActionEvents(state, plannedActions)
	if len(plannedActions) > 0 && len(input.SupersededActions) == 0 && !plannedActionsHaveRenderableEncounter(plannedActions) {
		plan := actionCompletionTickPlan(state, variable)
		reflection := fallbackReflectionResult(plan, variable)
		result := g.assembleStoryRunResult(input, plan, reflection, variable)
		result.CompletedActions = append([]model.OngoingAction(nil), input.CompletedActions...)
		return result, nil
	}
	sceneCharacterIDs := sceneCharacterIDsForContext(variable, snapshot.Characters, plannedActions)
	sceneCharacterIDs = append(sceneCharacterIDs, actionCharacterIDs(input.SupersededActions)...)
	if err := loadRecentMemoriesForCharacters(ctx, g.deps, state, &snapshot, input.Run.ProjectID, sceneCharacterIDs); err != nil {
		return model.StoryRunResult{}, fmt.Errorf("load scene memories: %w", err)
	}
	sceneContext := g.buildSceneContext(input, snapshot, variable, plannedActions)
	actionLocations, err := g.actionLocationsForScene(ctx, input, plannedActions)
	if err != nil {
		return model.StoryRunResult{}, fmt.Errorf("load action locations: %w", err)
	}
	sceneContext.SharedObservable.ActionLocations = actionLocations
	plan, finalVariable, err := g.simulateScene(ctx, input, snapshot, state, sceneContext, variable)
	if err != nil {
		return model.StoryRunResult{}, err
	}
	if stopped, err := g.stopRequested(ctx, input, "reflection"); err != nil {
		return model.StoryRunResult{}, err
	} else if stopped {
		reflection := fallbackReflectionResult(plan, finalVariable)
		result := g.assembleStoryRunResult(input, plan, reflection, finalVariable)
		result.CompletedActions = append([]model.OngoingAction(nil), input.CompletedActions...)
		result.SupersededActions = append([]model.OngoingAction(nil), input.SupersededActions...)
		if !input.CollisionAt.IsZero() {
			collisionAt := input.CollisionAt
			result.CollisionAt = &collisionAt
		}
		return result, nil
	}
	updateStoryRunStep(ctx, g.deps, input.Run.RunID, domain.RunStatusGeneratingMemoryPatch, 90)
	reflection, err := g.reflectScene(ctx, input, snapshot, plan, finalVariable)
	if err != nil {
		if ctx.Err() != nil {
			return model.StoryRunResult{}, ctx.Err()
		}
		publishStoryEvent(ctx, g.deps, input.Run.RunID, domain.EventGenerationStep, map[string]any{
			"step":  "reflection_failed",
			"error": err.Error(),
		})
		reflection = fallbackReflectionResult(plan, finalVariable)
	}
	result := g.assembleStoryRunResult(input, plan, reflection, finalVariable)
	result.CompletedActions = append([]model.OngoingAction(nil), input.CompletedActions...)
	result.SupersededActions = append([]model.OngoingAction(nil), input.SupersededActions...)
	if !input.CollisionAt.IsZero() {
		collisionAt := input.CollisionAt
		result.CollisionAt = &collisionAt
	}
	return result, nil
}

func (g *StoryRunGenerator) returnIfStopped(ctx context.Context, input port.StoryRunGenerationInput, boundary string) error {
	stopped, err := g.stopRequested(ctx, input, boundary)
	if err != nil {
		return err
	}
	if stopped {
		return port.ErrRunStopRequested
	}
	return nil
}

func (g *StoryRunGenerator) stopRequested(ctx context.Context, input port.StoryRunGenerationInput, boundary string) (bool, error) {
	if input.Run.StopRequested {
		publishStoryEvent(ctx, g.deps, input.Run.RunID, domain.EventGenerationStep, map[string]any{"step": "stop_requested", "boundary": boundary})
		return true, nil
	}
	if g.deps.sessions == nil {
		return false, nil
	}
	run, err := g.deps.sessions.GetRunByID(ctx, input.Run.RunID)
	if err != nil {
		return false, err
	}
	if run.StopRequested || run.Status == domain.RunStatusCancelled {
		publishStoryEvent(ctx, g.deps, input.Run.RunID, domain.EventGenerationStep, map[string]any{"step": "stop_requested", "boundary": boundary})
		return true, nil
	}
	return false, nil
}

func (g *StoryRunGenerator) publishStoryOrchestrationStarted(ctx context.Context, input port.StoryRunGenerationInput) {
	publishStoryEvent(ctx, g.deps, input.Run.RunID, domain.EventStoryOrchestrationStarted, map[string]any{
		"story_run_id":      input.Run.RunID,
		"session_id":        input.Session.ID,
		"author_intent":     input.Session.AuthorIntent,
		"opening_situation": input.Session.OpeningSituation,
	})
}

func (g *StoryRunGenerator) sceneSystemPrompt() string {
	return firstText(g.scenePrompt, defaultScenePrompt())
}

func (g *StoryRunGenerator) reflectSystemPrompt() string {
	return firstText(g.reflectPrompt, defaultReflectPrompt())
}

func defaultScenePrompt() string {
	return config.DefaultStoryAgentScenePrompt()
}

func defaultSceneFallbackPrompt() string {
	return config.DefaultStoryAgentResultPrompt()
}

func defaultReflectPrompt() string {
	return config.DefaultStoryAgentReflectPrompt()
}
func (g *StoryRunGenerator) sceneUserPrompt(sceneContext SceneContext) string {
	payload, _ := json.Marshal(buildScenePromptInput(sceneContext))
	return string(payload)
}

func (g *StoryRunGenerator) buildSceneContext(input port.StoryRunGenerationInput, snapshot StoryContextSnapshot, seed StoryVariablePlan, plannedActions []ScenePlannedAction) SceneContext {
	publicWorld := make([]model.WorldStateEntry, 0, len(snapshot.WorldState))
	for _, entry := range snapshot.WorldState {
		if entry.Importance >= 4 || entry.Volatility >= 4 {
			publicWorld = append(publicWorld, entry)
		}
	}
	if len(publicWorld) == 0 {
		publicWorld = firstEntries(snapshot.WorldState, 5)
	}
	characterIDs := sceneCharacterIDsForContext(seed, snapshot.Characters, plannedActions)
	characterViews := make([]SceneCharacterView, 0, len(characterIDs))
	for _, characterID := range characterIDs {
		character := characterByID(snapshot.Characters, characterID)
		if character.ID == "" {
			continue
		}
		variableView := variableViewForCharacter(seed.CharacterViews, character.ID)
		view := SceneCharacterView{
			CharacterID: character.ID,
			Identity: map[string]any{
				"id":           character.ID,
				"name":         character.Name,
				"role":         character.Role,
				"profile":      character.Profile,
				"personality":  character.Personality,
				"voice_style":  character.VoiceStyle,
				"goals":        firstStrings(character.Goals, 3),
				"fears":        firstStrings(character.Fears, 3),
				"constraints":  firstStrings(character.Constraints, 3),
				"memory_brief": character.RecentMemorySummary,
			},
			Secrets:         firstStrings(character.Secrets, 3),
			PrivateAttitude: privateAttitudesForCharacter(snapshot.Relationships, character.ID),
			RecentMemories:  memoryContents(snapshot.RecentMemories[character.ID], 4),
			VisibleWorld:    visibleWorldForCharacter(snapshot.WorldState, character),
		}
		if variableView != nil {
			view.KnownFacts = variableView.KnownFacts
			view.Misreadings = variableView.Misreadings
			view.EmotionalPressure = variableView.EmotionalPressure
			view.ActionBias = variableView.ActionBias
		}
		characterViews = append(characterViews, view)
	}
	return SceneContext{
		StoryRunID: input.Run.RunID,
		ProjectID:  input.Run.ProjectID,
		SessionID:  input.Session.ID,
		Session: SceneSessionContext{
			Title:               input.Session.Title,
			OpeningSituation:    input.Session.OpeningSituation,
			AuthorIntent:        input.Session.AuthorIntent,
			CurrentPlotVariable: input.Session.CurrentPlotVariableSummary,
		},
		AuthorBible:       compactAuthorBible(snapshot.AuthorBible),
		PlotVariableSeed:  seed,
		PlannedActions:    plannedActions,
		InFlightActions:   append([]model.OngoingAction(nil), input.InFlightActions...),
		CompletedActions:  append([]model.OngoingAction(nil), input.CompletedActions...),
		SupersededActions: append([]model.OngoingAction(nil), input.SupersededActions...),
		CollisionAt:       formatOptionalTime(input.CollisionAt),
		SharedObservable: SharedObservableContext{
			LocationHints:    compactWorldState(snapshot.WorldState, 5),
			PublicWorldState: compactWorldState(publicWorld, 8),
			RecentChapters:   compactChapters(snapshot.RecentChapters, 3),
		},
		CharacterViews: characterViews,
		Constraints: SceneConstraints{
			MaxTurns:        g.maxTurns,
			MaxInteractions: 3,
		},
	}
}

type actionLocationTarget struct {
	locationID     string
	characterID    string
	requiresDetail bool
}

func (g *StoryRunGenerator) actionLocationsForScene(ctx context.Context, input port.StoryRunGenerationInput, plannedActions []ScenePlannedAction) ([]model.LocationState, error) {
	targets := actionLocationTargets(plannedActions, input.InFlightActions, input.CompletedActions, input.SupersededActions)
	if len(targets) == 0 {
		return nil, nil
	}
	locations, err := g.sceneLocationDirectory(ctx, input)
	if err != nil {
		return nil, err
	}
	if len(locations) == 0 {
		return nil, nil
	}
	byID := locationStateMap(locations)
	projectID := firstText(input.Run.ProjectID, input.Session.ProjectID)
	out := make([]model.LocationState, 0, len(targets))
	seen := map[string]struct{}{}
	for _, target := range targets {
		if _, ok := seen[target.locationID]; ok {
			continue
		}
		location, ok := byID[target.locationID]
		if !ok {
			continue
		}
		if location.DetailState != model.LocationDetailInitialized {
			initialized, initErr := g.inspectActionLocation(ctx, input, projectID, target, location)
			if initErr != nil {
				if target.requiresDetail {
					return nil, initErr
				}
			} else if initialized.ID != "" {
				location = initialized
			}
		}
		if location.DetailState != model.LocationDetailInitialized {
			continue
		}
		seen[target.locationID] = struct{}{}
		out = append(out, location)
	}
	return out, nil
}

func (g *StoryRunGenerator) sceneLocationDirectory(ctx context.Context, input port.StoryRunGenerationInput) ([]model.LocationState, error) {
	projectID := firstText(input.Run.ProjectID, input.Session.ProjectID)
	if g.deps.storyEvents != nil && projectID != "" {
		locations, err := g.deps.storyEvents.ListLocationsByProjectID(ctx, projectID)
		if err != nil {
			return nil, err
		}
		return locations, nil
	}
	return append([]model.LocationState(nil), input.World.Locations...), nil
}

func (g *StoryRunGenerator) inspectActionLocation(ctx context.Context, input port.StoryRunGenerationInput, projectID string, target actionLocationTarget, location model.LocationState) (model.LocationState, error) {
	if g.locationInspector == nil {
		return model.LocationState{}, fmt.Errorf("action location %q is not initialized", target.locationID)
	}
	currentLocationID := ""
	if state, ok := input.World.Characters[target.characterID]; ok {
		currentLocationID = state.LocationKey
	}
	if currentLocationID == "" {
		currentLocationID = target.locationID
	}
	result, err := g.locationInspector.InspectLocation(ctx, model.LocationInspectionInput{
		ProjectID:         firstText(projectID, location.ProjectID),
		CharacterID:       target.characterID,
		CurrentLocationID: currentLocationID,
		LocationID:        target.locationID,
		Reason:            "scene context for planned action",
		World:             input.World,
	})
	if err != nil {
		return model.LocationState{}, fmt.Errorf("initialize action location %q: %w", target.locationID, err)
	}
	return result.InspectedLocation, nil
}

func actionLocationTargets(planned []ScenePlannedAction, inFlight []model.OngoingAction, completed []model.OngoingAction, superseded []model.OngoingAction) []actionLocationTarget {
	targets := make([]actionLocationTarget, 0, len(planned)+len(inFlight)+len(completed)+len(superseded))
	for _, action := range planned {
		targets = appendActionLocationTarget(targets, action.CharacterID, action.TargetLocationKey, action.ActionType, action.Description)
	}
	for _, action := range inFlight {
		targets = appendActionLocationTarget(targets, action.CharacterID, action.TargetLocationKey, action.ActionType, action.Description)
	}
	for _, action := range completed {
		targets = appendActionLocationTarget(targets, action.CharacterID, action.TargetLocationKey, action.ActionType, action.Description)
	}
	for _, action := range superseded {
		targets = appendActionLocationTarget(targets, action.CharacterID, action.TargetLocationKey, action.ActionType, action.Description)
	}
	return targets
}

func appendActionLocationTarget(targets []actionLocationTarget, characterID string, locationID string, actionType string, description string) []actionLocationTarget {
	locationID = strings.TrimSpace(locationID)
	if locationID == "" {
		return targets
	}
	requiresDetail := actionNeedsInspectedTarget(model.CharacterActionDecision{ActionType: actionType, Description: description})
	return append(targets, actionLocationTarget{locationID: locationID, characterID: strings.TrimSpace(characterID), requiresDetail: requiresDetail})
}

func locationStateMap(locations []model.LocationState) map[string]model.LocationState {
	byID := make(map[string]model.LocationState, len(locations))
	for _, location := range locations {
		if location.ID != "" {
			byID[location.ID] = location
		}
	}
	return byID
}

func (g *StoryRunGenerator) planCharacterActions(ctx context.Context, input port.StoryRunGenerationInput, snapshot StoryContextSnapshot) ([]ScenePlannedAction, error) {
	if g.actionDecider == nil {
		return nil, nil
	}
	world, err := g.decisionWorldSnapshot(ctx, input, snapshot)
	if err != nil {
		return nil, err
	}
	characterIDs := input.WakeCharacterIDs
	if len(characterIDs) == 0 {
		characterIDs = idleCharacterIDs(world, snapshot.Characters)
	}
	if len(characterIDs) == 0 {
		return nil, nil
	}
	results := make([]plannedActionResult, len(characterIDs))
	var wg sync.WaitGroup
	for idx, characterID := range characterIDs {
		wg.Add(1)
		go func(idx int, characterID string) {
			defer wg.Done()
			action, err := g.planCharacterAction(ctx, input, snapshot, world, characterID)
			results[idx] = plannedActionResult{action: action, err: err}
		}(idx, characterID)
	}
	wg.Wait()
	planned := make([]ScenePlannedAction, 0, len(results))
	for _, result := range results {
		if result.err != nil {
			return nil, result.err
		}
		if result.action.CharacterID != "" {
			planned = append(planned, result.action)
		}
	}
	return planned, nil
}

type plannedActionResult struct {
	action ScenePlannedAction
	err    error
}

func (g *StoryRunGenerator) planCharacterAction(ctx context.Context, input port.StoryRunGenerationInput, snapshot StoryContextSnapshot, world model.WorldSnapshot, characterID string) (ScenePlannedAction, error) {
	character := characterByID(snapshot.Characters, characterID)
	if character.ID == "" {
		return ScenePlannedAction{}, nil
	}
	decisionInput := characterDecisionInput(world, character)
	decision, err := g.actionDecider.Decide(ctx, decisionInput)
	if err != nil {
		g.publishCharacterActionDecisionFailure(ctx, input.Run.RunID, character, err)
		return ScenePlannedAction{}, fmt.Errorf("%s: %w", character.ID, err)
	}
	action, err := scenePlannedActionFromDecision(snapshot.Characters, character, decision)
	if err != nil {
		return ScenePlannedAction{}, fmt.Errorf("%s: %w", character.ID, err)
	}
	return action, nil
}

func (g *StoryRunGenerator) publishCharacterActionDecisionFailure(ctx context.Context, runID string, character model.Character, err error) {
	payload := modelJSONErrorSummary(err)
	payload["step"] = "character_action_decision_failed"
	payload["character_id"] = character.ID
	payload["character_name"] = character.Name
	publishStoryEvent(ctx, g.deps, runID, domain.EventCharacterActionDecisionFailed, payload)
}

func (g *StoryRunGenerator) decisionWorldSnapshot(ctx context.Context, input port.StoryRunGenerationInput, snapshot StoryContextSnapshot) (model.WorldSnapshot, error) {
	if input.World.AtEventID != "" || !input.World.StoryTime.IsZero() || len(input.World.Characters) > 0 {
		return input.World, nil
	}
	if g.deps.storyEvents != nil && input.Run.BaseEventID != "" {
		return g.deps.storyEvents.ResolveStateAt(ctx, input.Run.BaseEventID)
	}
	return worldSnapshotFromStoryContext(input, snapshot, g.now()), nil
}

func worldSnapshotFromStoryContext(input port.StoryRunGenerationInput, snapshot StoryContextSnapshot, now time.Time) model.WorldSnapshot {
	storyTime := input.Run.CreatedAt
	if storyTime.IsZero() {
		storyTime = now
	}
	worldState := make(map[string]model.WorldStateEntry, len(snapshot.WorldState))
	for _, entry := range snapshot.WorldState {
		if entry.Key != "" {
			worldState[entry.Key] = entry
		}
	}
	characters := make(map[string]model.CharacterRuntimeState, len(snapshot.Characters))
	for _, character := range snapshot.Characters {
		if character.ID == "" {
			continue
		}
		characters[character.ID] = model.CharacterRuntimeState{
			CharacterID: character.ID,
			Status:      firstText(character.Status, "active"),
		}
	}
	relationships := make(map[string]model.Relationship, len(snapshot.Relationships))
	for idx, relationship := range snapshot.Relationships {
		relationships[relationshipMapKey(relationship, idx)] = relationship
	}
	return model.WorldSnapshot{
		AtEventID:     input.Run.BaseEventID,
		StoryTime:     storyTime,
		WorldState:    worldState,
		Characters:    characters,
		Relationships: relationships,
	}
}

func relationshipMapKey(relationship model.Relationship, idx int) string {
	if relationship.Pair.ID != "" {
		return relationship.Pair.ID
	}
	if relationship.Pair.LeftCharacterID != "" || relationship.Pair.RightCharacterID != "" {
		return relationship.Pair.LeftCharacterID + ":" + relationship.Pair.RightCharacterID
	}
	return fmt.Sprintf("relationship_%d", idx+1)
}

func characterDecisionInput(world model.WorldSnapshot, character model.Character) model.CharacterActionDecisionInput {
	state := world.Characters[character.ID]
	if state.CharacterID == "" {
		state.CharacterID = character.ID
		state.Status = firstText(character.Status, "active")
	}
	location := locationByKey(world.Locations, state.LocationKey)
	return model.CharacterActionDecisionInput{
		World:             visibleWorldSnapshotForCharacter(world, character.ID),
		Character:         character,
		CharacterState:    state,
		Location:          location,
		FactionInfluences: factionInfluencesForLocation(world.Factions, location.ID),
		NearbyLocations:   nearbyLocationContexts(world.Locations, world.Factions, location),
	}
}

func idleCharacterIDs(world model.WorldSnapshot, characters []model.Character) []string {
	ids := make([]string, 0, len(characters))
	for _, character := range characters {
		if character.ID == "" {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(character.Status), "inactive") {
			continue
		}
		state, ok := world.Characters[character.ID]
		if ok && !characterStateIsIdle(state, world.StoryTime) {
			continue
		}
		if ok && strings.EqualFold(strings.TrimSpace(state.Status), "inactive") {
			continue
		}
		ids = appendUniqueString(ids, character.ID)
	}
	return ids
}

func characterStateIsIdle(state model.CharacterRuntimeState, clock time.Time) bool {
	if state.OngoingAction == nil {
		return true
	}
	return !state.OngoingAction.EndsAt.After(clock)
}

func visibleWorldSnapshotForCharacter(world model.WorldSnapshot, characterID string) model.WorldSnapshot {
	visible := world
	visible.WorldState = make(map[string]model.WorldStateEntry, len(world.WorldState))
	for key, entry := range world.WorldState {
		visible.WorldState[key] = entry
	}
	visible.Characters = make(map[string]model.CharacterRuntimeState, len(world.Characters))
	for id, state := range world.Characters {
		if id != characterID {
			state.OngoingAction = nil
		}
		visible.Characters[id] = state
	}
	visible.Relationships = make(map[string]model.Relationship, len(world.Relationships))
	for key, relationship := range world.Relationships {
		if filtered, ok := relationshipVisibleToCharacter(relationship, characterID); ok {
			visible.Relationships[key] = filtered
		}
	}
	return visible
}

func relationshipVisibleToCharacter(relationship model.Relationship, characterID string) (model.Relationship, bool) {
	involved := relationshipPairInvolves(relationship.Pair, characterID)
	visible := relationship
	visible.Views = nil
	visible.CharacterAView = nil
	visible.CharacterBView = nil
	for _, view := range relationship.Views {
		if view.SourceCharacterID != characterID {
			continue
		}
		involved = true
		visible.Views = append(visible.Views, view)
	}
	if !involved {
		return model.Relationship{}, false
	}
	for i := range visible.Views {
		view := &visible.Views[i]
		if view.SourceCharacterID == visible.Pair.LeftCharacterID {
			visible.CharacterAView = view
		}
		if view.SourceCharacterID == visible.Pair.RightCharacterID {
			visible.CharacterBView = view
		}
	}
	return visible, true
}

func relationshipPairInvolves(pair model.RelationshipPair, characterID string) bool {
	return pair.LeftCharacterID == characterID || pair.RightCharacterID == characterID
}

func locationByKey(locations []model.LocationState, locationKey string) model.LocationState {
	for _, location := range locations {
		if location.ID == locationKey {
			return location
		}
	}
	return model.LocationState{}
}

func factionInfluencesForLocation(influences []model.FactionInfluence, locationID string) []model.FactionInfluence {
	if locationID == "" {
		return nil
	}
	out := make([]model.FactionInfluence, 0, len(influences))
	for _, influence := range influences {
		if influence.LocationID == locationID {
			out = append(out, influence)
		}
	}
	return out
}

func nearbyLocationContexts(locations []model.LocationState, influences []model.FactionInfluence, current model.LocationState) []model.NearbyLocationContext {
	if current.ID == "" {
		return nil
	}
	out := make([]model.NearbyLocationContext, 0, len(locations))
	for _, location := range locations {
		if location.ID == "" || location.ID == current.ID {
			continue
		}
		out = append(out, model.NearbyLocationContext{
			Location:          location,
			Distance:          locationDistance(current, location),
			FactionInfluences: factionInfluencesForLocation(influences, location.ID),
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].Distance < out[j].Distance
	})
	out = firstEntries(out, 5)
	return out
}

func locationDistance(a, b model.LocationState) float64 {
	dx := float64(a.X - b.X)
	dy := float64(a.Y - b.Y)
	return math.Sqrt(dx*dx + dy*dy)
}

func scenePlannedActionFromDecision(characters []model.Character, character model.Character, decision model.CharacterActionDecision) (ScenePlannedAction, error) {
	actionType := normalizeStoryActionType(decision.ActionType, "", decision.Description)
	if actionType == "" {
		actionType = "observe"
	}
	description := strings.TrimSpace(decision.Description)
	if description == "" {
		description = "观察当前局势"
	}
	durationHours := decision.DurationHours
	if durationHours <= 0 {
		durationHours = 1
	}
	participantIDs, err := validStoryTargetActorIDs(characters, decision.ParticipantIDs)
	if err != nil {
		return ScenePlannedAction{}, err
	}
	return ScenePlannedAction{
		CharacterID:       character.ID,
		CharacterName:     character.Name,
		ActionType:        actionType,
		Description:       description,
		TargetLocationKey: strings.TrimSpace(decision.TargetLocationKey),
		DurationHours:     durationHours,
		ArriveAt:          optionalTime(decision.ArriveAt),
		EffectAt:          optionalTime(decision.EffectAt),
		EndsAt:            optionalTime(decision.EndsAt),
		ParticipantIDs:    uniqueStoryIDs(participantIDs),
		ResourceKeys:      uniqueStoryIDs(decision.AffectedResourceKeys),
		Rationale:         strings.TrimSpace(decision.Rationale),
	}, nil
}

func sceneCharacterIDs(relatedIDs []string, characters []model.Character) []string {
	ids := make([]string, 0, len(relatedIDs))
	for _, id := range relatedIDs {
		ids = appendUniqueString(ids, id)
	}
	if len(ids) > 0 {
		return ids
	}
	for _, character := range firstEntries(characters, 4) {
		ids = appendUniqueString(ids, character.ID)
	}
	return ids
}

func sceneCharacterIDsForContext(seed StoryVariablePlan, characters []model.Character, plannedActions []ScenePlannedAction) []string {
	if len(plannedActions) == 0 {
		return sceneCharacterIDs(seed.PlotVariable.RelatedCharacterIDs, characters)
	}
	ids := plannedActionCharacterIDs(plannedActions)
	if len(ids) == 0 {
		return sceneCharacterIDs(seed.PlotVariable.RelatedCharacterIDs, characters)
	}
	return ids
}

func actionCharacterIDs(actions []model.OngoingAction) []string {
	ids := make([]string, 0, len(actions)*2)
	for _, action := range actions {
		ids = appendUniqueString(ids, action.CharacterID)
		for _, participantID := range action.ParticipantIDs {
			ids = appendUniqueString(ids, participantID)
		}
	}
	return ids
}

func formatOptionalTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.Format(time.RFC3339)
}

func plannedActionCharacterIDs(plannedActions []ScenePlannedAction) []string {
	ids := make([]string, 0, len(plannedActions)*2)
	for _, action := range plannedActions {
		ids = appendUniqueString(ids, action.CharacterID)
		for _, participantID := range action.ParticipantIDs {
			ids = appendUniqueString(ids, participantID)
		}
	}
	return ids
}

func plannedActionsAllowTurn(plannedActions []ScenePlannedAction, actorID string, targetActorIDs []string) bool {
	allowed := plannedActionCharacterIDs(plannedActions)
	if actorID != "" && !containsString(allowed, actorID) {
		return false
	}
	for _, targetID := range targetActorIDs {
		if !containsString(allowed, targetID) {
			return false
		}
	}
	return true
}

func plannedActionsHaveRenderableEncounter(plannedActions []ScenePlannedAction) bool {
	for i, left := range plannedActions {
		for j := i + 1; j < len(plannedActions); j++ {
			right := plannedActions[j]
			if plannedActionsShareLocation(left, right) || plannedActionTargetsCharacter(left, right.CharacterID) || plannedActionTargetsCharacter(right, left.CharacterID) || plannedActionsShareResource(left, right) {
				return true
			}
		}
	}
	return false
}

func plannedActionsShareLocation(left, right ScenePlannedAction) bool {
	leftLocation := strings.TrimSpace(left.TargetLocationKey)
	rightLocation := strings.TrimSpace(right.TargetLocationKey)
	return leftLocation != "" && rightLocation != "" && leftLocation == rightLocation
}

func plannedActionTargetsCharacter(action ScenePlannedAction, characterID string) bool {
	if characterID == "" {
		return false
	}
	for _, participantID := range action.ParticipantIDs {
		if participantID == characterID {
			return true
		}
	}
	return false
}

func plannedActionsShareResource(left, right ScenePlannedAction) bool {
	if len(left.ResourceKeys) == 0 || len(right.ResourceKeys) == 0 {
		return false
	}
	seen := make(map[string]struct{}, len(left.ResourceKeys))
	for _, key := range left.ResourceKeys {
		key = strings.TrimSpace(key)
		if key != "" {
			seen[key] = struct{}{}
		}
	}
	for _, key := range right.ResourceKeys {
		if _, ok := seen[strings.TrimSpace(key)]; ok {
			return true
		}
	}
	return false
}

func privateAttitudesForCharacter(relationships []model.Relationship, characterID string) []map[string]string {
	views := relationshipViewsForCharacter(relationships, characterID)
	out := make([]map[string]string, 0, len(views))
	for _, view := range views {
		out = append(out, map[string]string{
			"target_character_id":      view.TargetCharacterID,
			"public_attitude":          view.PublicAttitude,
			"private_attitude":         view.PrivateAttitude,
			"believed_target_attitude": view.BelievedTargetAttitude,
			"masking_strategy":         view.MaskingStrategy,
		})
	}
	return out
}

func memoryContents(memories []model.Memory, limit int) []string {
	out := make([]string, 0, minInt(len(memories), limit))
	for _, memory := range firstEntries(memories, limit) {
		if memory.Content != "" {
			out = append(out, memory.Content)
		}
	}
	return out
}

func (g *StoryRunGenerator) seedPlotVariable(ctx context.Context, input port.StoryRunGenerationInput, snapshot StoryContextSnapshot) (StoryVariablePlan, error) {
	plot := StoryNarrativePlotVariable{
		PressureSource:     firstText(input.Session.OpeningSituation, input.Session.AuthorIntent, input.Session.CurrentPlotVariableSummary, "当前故事压力"),
		FocalCharacterID:   storyVariableFocalCharacterID(input, snapshot),
		CoreChoice:         firstText(input.Session.AuthorIntent, input.Session.OpeningSituation, input.Session.CurrentPlotVariableSummary, "推进当前故事变量"),
		OptionA:            "暂时维持当前局面",
		OptionB:            "主动打破当前局面",
		CostA:              "压力继续累积",
		CostB:              "暴露意图或承担代价",
		IrreversibleEffect: firstText(input.Session.CurrentPlotVariableSummary, "本章状态将发生不可逆变化"),
		WorldStatePressure: storyVariableWorldStatePressure(snapshot.WorldState),
	}
	plot.RelatedCharacterIDs = storyVariableRelatedCharacterIDs(plot.FocalCharacterID, input, snapshot)
	variable := StoryVariablePlan{
		PlotVariable:   plot,
		CharacterViews: storyVariableCharacterViews(plot, snapshot),
	}
	return normalizeStoryVariable(variable, input, snapshot), nil
}

func storyVariableFocalCharacterID(input port.StoryRunGenerationInput, snapshot StoryContextSnapshot) string {
	text := strings.Join([]string{input.Session.AuthorIntent, input.Session.OpeningSituation, input.Session.CurrentPlotVariableSummary}, "\n")
	for _, character := range snapshot.Characters {
		if character.Name != "" && strings.Contains(text, character.Name) {
			return character.ID
		}
	}
	if len(snapshot.Characters) > 0 {
		return snapshot.Characters[0].ID
	}
	return ""
}

func storyVariableWorldStatePressure(worldState []model.WorldStateEntry) []string {
	keys := make([]string, 0, 3)
	for _, entry := range worldState {
		if entry.Key == "" || (entry.Importance < 4 && entry.Volatility < 4) {
			continue
		}
		keys = append(keys, entry.Key)
		if len(keys) == 3 {
			return keys
		}
	}
	for _, entry := range worldState {
		if entry.Key == "" || containsString(keys, entry.Key) {
			continue
		}
		keys = append(keys, entry.Key)
		if len(keys) == 3 {
			break
		}
	}
	return keys
}

func storyVariableRelatedCharacterIDs(focalCharacterID string, input port.StoryRunGenerationInput, snapshot StoryContextSnapshot) []string {
	ids := make([]string, 0, 4)
	if focalCharacterID != "" {
		ids = append(ids, focalCharacterID)
	}
	text := strings.Join([]string{input.Session.AuthorIntent, input.Session.OpeningSituation, input.Session.CurrentPlotVariableSummary}, "\n")
	for _, character := range snapshot.Characters {
		if character.ID == focalCharacterID || character.Name == "" || !strings.Contains(text, character.Name) {
			continue
		}
		ids = append(ids, character.ID)
	}
	for _, relationship := range snapshot.Relationships {
		if relationship.Pair.LeftCharacterID == focalCharacterID {
			ids = appendUniqueString(ids, relationship.Pair.RightCharacterID)
		}
		if relationship.Pair.RightCharacterID == focalCharacterID {
			ids = appendUniqueString(ids, relationship.Pair.LeftCharacterID)
		}
		if len(ids) >= 4 {
			break
		}
	}
	return ids
}

func storyVariableCharacterViews(plot StoryNarrativePlotVariable, snapshot StoryContextSnapshot) []CharacterVariableView {
	views := make([]CharacterVariableView, 0, len(plot.RelatedCharacterIDs))
	for _, characterID := range plot.RelatedCharacterIDs {
		views = append(views, CharacterVariableView{
			CharacterID:       characterID,
			KnownFacts:        storyVariableKnownFacts(plot, snapshot, characterID),
			Misreadings:       storyVariableMisreadings(snapshot, characterID),
			EmotionalPressure: plot.CoreChoice,
			ActionBias:        firstText(plot.OptionB, plot.OptionA),
		})
	}
	return views
}

func storyVariableKnownFacts(plot StoryNarrativePlotVariable, snapshot StoryContextSnapshot, characterID string) []string {
	facts := []string{plot.PressureSource}
	for _, entry := range visibleWorldForCharacter(snapshot.WorldState, characterByID(snapshot.Characters, characterID)) {
		facts = append(facts, firstText(entry.Note, entry.Key))
		if len(facts) == 3 {
			break
		}
	}
	return cleanStrings(facts)
}

func storyVariableMisreadings(snapshot StoryContextSnapshot, characterID string) []string {
	for _, relationship := range snapshot.Relationships {
		for _, view := range relationship.Views {
			if view.SourceCharacterID == characterID && view.BelievedTargetAttitude != "" {
				return []string{view.BelievedTargetAttitude}
			}
		}
	}
	return nil
}

func characterByID(characters []model.Character, characterID string) model.Character {
	for _, character := range characters {
		if character.ID == characterID {
			return character
		}
	}
	return model.Character{}
}

func appendUniqueString(values []string, value string) []string {
	if value == "" || containsString(values, value) {
		return values
	}
	return append(values, value)
}

func containsString(values []string, value string) bool {
	for _, existing := range values {
		if existing == value {
			return true
		}
	}
	return false
}

func compactAuthorBible(bible *model.AuthorBible) map[string]any {
	if bible == nil {
		return nil
	}
	return map[string]any{
		"theme":                bible.Theme,
		"style_guide":          bible.StyleGuide,
		"world_rules":          firstStrings(bible.WorldRules, 4),
		"hard_constraints":     firstStrings(bible.HardConstraints, 4),
		"forbidden_moves":      firstStrings(bible.ForbiddenMoves, 4),
		"aesthetic_principles": firstStrings(bible.AestheticPrinciples, 4),
	}
}

func compactWorldState(entries []model.WorldStateEntry, limit int) []map[string]any {
	out := make([]map[string]any, 0, minInt(len(entries), limit))
	for _, entry := range firstEntries(entries, limit) {
		out = append(out, map[string]any{
			"key":        entry.Key,
			"value":      entry.Value,
			"note":       entry.Note,
			"importance": entry.Importance,
			"volatility": entry.Volatility,
		})
	}
	return out
}

func compactCharacters(characters []model.Character, limit int) []map[string]any {
	out := make([]map[string]any, 0, minInt(len(characters), limit))
	for _, character := range firstEntries(characters, limit) {
		out = append(out, map[string]any{
			"id":          character.ID,
			"name":        character.Name,
			"role":        character.Role,
			"profile":     character.Profile,
			"goals":       firstStrings(character.Goals, 3),
			"fears":       firstStrings(character.Fears, 3),
			"constraints": firstStrings(character.Constraints, 3),
		})
	}
	return out
}

func compactRelationships(relationships []model.Relationship, limit int) []map[string]any {
	out := make([]map[string]any, 0, minInt(len(relationships), limit))
	for _, relationship := range firstEntries(relationships, limit) {
		out = append(out, map[string]any{
			"pair_id":            relationship.Pair.ID,
			"left_character_id":  relationship.Pair.LeftCharacterID,
			"right_character_id": relationship.Pair.RightCharacterID,
			"summary":            relationship.Pair.Summary,
			"tension_points":     firstStrings(relationship.Pair.TensionPoints, 3),
		})
	}
	return out
}

func compactChapters(chapters []model.Chapter, limit int) []map[string]any {
	out := make([]map[string]any, 0, minInt(len(chapters), limit))
	for _, chapter := range firstEntries(chapters, limit) {
		out = append(out, map[string]any{
			"title":   chapter.Title,
			"summary": chapter.Summary,
		})
	}
	return out
}

func compactMemories(memories map[string][]model.Memory, limitPerCharacter int) map[string][]string {
	out := make(map[string][]string, len(memories))
	for characterID, items := range memories {
		contents := make([]string, 0, minInt(len(items), limitPerCharacter))
		for _, memory := range firstEntries(items, limitPerCharacter) {
			contents = append(contents, memory.Content)
		}
		if len(contents) > 0 {
			out[characterID] = contents
		}
	}
	return out
}

func firstStrings(values []string, limit int) []string {
	return firstEntries(values, limit)
}

func firstEntries[T any](values []T, limit int) []T {
	if limit < 0 || len(values) <= limit {
		return values
	}
	return values[:limit]
}

func minInt(a int, b int) int {
	if a < b {
		return a
	}
	return b
}

func positiveIntOrDefault(value int, defaultValue int) int {
	if value > 0 {
		return value
	}
	return defaultValue
}

func (g *StoryRunGenerator) sceneTokenLimit() int {
	return positiveIntOrDefault(g.maxSceneTokens, config.DefaultStoryAgentMaxSceneTokens)
}

func (g *StoryRunGenerator) reflectTokenLimit() int {
	return positiveIntOrDefault(g.maxReflectTokens, config.DefaultStoryAgentMaxReflectTokens)
}

func (g *StoryRunGenerator) simulateScene(ctx context.Context, input port.StoryRunGenerationInput, snapshot StoryContextSnapshot, state *storyRunState, sceneContext SceneContext, seed StoryVariablePlan) (StoryPlanResult, StoryVariablePlan, error) {
	if g.model == nil {
		return StoryPlanResult{}, StoryVariablePlan{}, errors.New("story model is not configured")
	}
	updateStoryRunStep(ctx, g.deps, input.Run.RunID, domain.RunStatusPlanningEvents, 30)
	stream, streamErr := g.model.Stream(ctx, []*schema.Message{
		schema.SystemMessage(g.sceneSystemPrompt()),
		schema.UserMessage(g.sceneUserPrompt(sceneContext)),
	}, maxTokensOption(g.cfg.Model, g.sceneTokenLimit()))
	if streamErr == nil {
		if stream == nil {
			streamErr = errors.New("story model returned nil stream")
		}
	}
	if streamErr == nil {
		plan, variable, consumeErr := g.consumeSceneStream(ctx, input, snapshot, state, stream, seed)
		if consumeErr == nil {
			return plan, variable, nil
		}
		if ctx.Err() != nil {
			return StoryPlanResult{}, StoryVariablePlan{}, ctx.Err()
		}
		streamErr = consumeErr
	}
	publishStoryEvent(ctx, g.deps, input.Run.RunID, domain.EventGenerationStep, map[string]any{
		"step":  "scene_stream_fallback",
		"error": streamErr.Error(),
	})
	batch, fallbackErr := g.generateSceneFallback(ctx, sceneContext)
	if fallbackErr != nil {
		return StoryPlanResult{}, StoryVariablePlan{}, fmt.Errorf("generate story scene stream: %w; fallback: %v", streamErr, fallbackErr)
	}
	state = &storyRunState{run: input.Run, session: input.Session, maxTurns: g.maxTurns, characters: snapshot.Characters, variable: seed}
	seedPlannedActionEvents(state, sceneContext.PlannedActions)
	return g.consumeSceneBatch(ctx, input, snapshot, state, batch, seed)
}

func (g *StoryRunGenerator) generateSceneFallback(ctx context.Context, sceneContext SceneContext) (sceneBatchResult, error) {
	prompt := firstText(g.resultPrompt, defaultSceneFallbackPrompt())
	msg, err := g.model.Generate(ctx, []*schema.Message{
		schema.SystemMessage(prompt),
		schema.UserMessage(g.sceneUserPrompt(sceneContext)),
	}, maxTokensOption(g.cfg.Model, g.sceneTokenLimit()))
	if err != nil {
		return sceneBatchResult{}, err
	}
	var batch sceneBatchResult
	if err := decodeModelJSON(msg.Content, &batch); err != nil {
		return sceneBatchResult{}, err
	}
	return batch, nil
}

func (g *StoryRunGenerator) consumeSceneStream(ctx context.Context, input port.StoryRunGenerationInput, snapshot StoryContextSnapshot, state *storyRunState, stream *schema.StreamReader[*schema.Message], seed StoryVariablePlan) (StoryPlanResult, StoryVariablePlan, error) {
	defer stream.Close()
	consumer := newSceneConsumer(g, input, snapshot, state, seed)
	var chunkBuffer string
	for {
		select {
		case <-ctx.Done():
			return StoryPlanResult{}, StoryVariablePlan{}, ctx.Err()
		default:
		}
		if stopped, err := g.stopRequested(ctx, input, "scene_stream"); err != nil {
			return StoryPlanResult{}, StoryVariablePlan{}, err
		} else if stopped {
			if consumer.hasCommittableScene() {
				return consumer.finish(ctx)
			}
			return StoryPlanResult{}, StoryVariablePlan{}, port.ErrRunStopRequested
		}
		msg, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return StoryPlanResult{}, StoryVariablePlan{}, err
		}
		if msg == nil || msg.Content == "" {
			continue
		}
		chunkBuffer += msg.Content
		for {
			lineEnd := strings.IndexByte(chunkBuffer, '\n')
			if lineEnd < 0 {
				break
			}
			line := chunkBuffer[:lineEnd]
			chunkBuffer = chunkBuffer[lineEnd+1:]
			if err := consumer.consumeRawLine(ctx, line); err != nil {
				return StoryPlanResult{}, StoryVariablePlan{}, err
			}
		}
	}
	if strings.TrimSpace(chunkBuffer) != "" {
		if err := consumer.consumeRawLine(ctx, chunkBuffer); err != nil {
			return StoryPlanResult{}, StoryVariablePlan{}, err
		}
	}
	return consumer.finish(ctx)
}

func (g *StoryRunGenerator) consumeSceneBatch(ctx context.Context, input port.StoryRunGenerationInput, snapshot StoryContextSnapshot, state *storyRunState, batch sceneBatchResult, seed StoryVariablePlan) (StoryPlanResult, StoryVariablePlan, error) {
	consumer := newSceneConsumer(g, input, snapshot, state, seed)
	if storyPlotVariableUsable(batch.PlotVariable) {
		if err := consumer.consumeRecord(ctx, sceneRecord{Type: "plot_variable", PlotVariable: batch.PlotVariable}); err != nil {
			return StoryPlanResult{}, StoryVariablePlan{}, err
		}
	}
	events := batch.EventPlan
	if len(events) == 0 {
		events = batch.Events
	}
	for _, event := range events {
		if stopped, err := g.stopRequested(ctx, input, "scene_batch"); err != nil {
			return StoryPlanResult{}, StoryVariablePlan{}, err
		} else if stopped && !consumer.hasCommittableScene() {
			return StoryPlanResult{}, StoryVariablePlan{}, port.ErrRunStopRequested
		}
		if err := consumer.consumeRecord(ctx, sceneRecord{Type: "event", Event: event}); err != nil {
			return StoryPlanResult{}, StoryVariablePlan{}, err
		}
	}
	for _, group := range batch.InteractionGroups {
		if stopped, err := g.stopRequested(ctx, input, "scene_batch"); err != nil {
			return StoryPlanResult{}, StoryVariablePlan{}, err
		} else if stopped && !consumer.hasCommittableScene() {
			return StoryPlanResult{}, StoryVariablePlan{}, port.ErrRunStopRequested
		}
		if err := consumer.consumeRecord(ctx, sceneRecord{Type: "interaction", InteractionGroup: group}); err != nil {
			return StoryPlanResult{}, StoryVariablePlan{}, err
		}
	}
	for _, turn := range batch.Turns {
		if stopped, err := g.stopRequested(ctx, input, "scene_batch"); err != nil {
			return StoryPlanResult{}, StoryVariablePlan{}, err
		} else if stopped && consumer.hasCommittableScene() {
			break
		} else if stopped {
			return StoryPlanResult{}, StoryVariablePlan{}, port.ErrRunStopRequested
		}
		if err := consumer.consumeRecord(ctx, sceneRecord{Type: "turn", Turn: turn}); err != nil {
			return StoryPlanResult{}, StoryVariablePlan{}, err
		}
	}
	if batch.StopReason != "" {
		if err := consumer.consumeRecord(ctx, sceneRecord{Type: "stop", StopReason: batch.StopReason}); err != nil {
			return StoryPlanResult{}, StoryVariablePlan{}, err
		}
	}
	return consumer.finish(ctx)
}

type sceneConsumer struct {
	g                       *StoryRunGenerator
	input                   port.StoryRunGenerationInput
	snapshot                StoryContextSnapshot
	state                   *storyRunState
	seed                    StoryVariablePlan
	variable                StoryVariablePlan
	pendingLine             string
	locationGroupsPublished bool
	firstTurnPublished      bool
	reviewIssues            []string
}

func newSceneConsumer(g *StoryRunGenerator, input port.StoryRunGenerationInput, snapshot StoryContextSnapshot, state *storyRunState, seed StoryVariablePlan) *sceneConsumer {
	return &sceneConsumer{g: g, input: input, snapshot: snapshot, state: state, seed: seed, variable: seed}
}

func (c *sceneConsumer) consumeRawLine(ctx context.Context, line string) error {
	text := strings.TrimSpace(line)
	if text == "" || strings.HasPrefix(text, "```") {
		return nil
	}
	if c.pendingLine != "" {
		text = c.pendingLine + text
		c.pendingLine = ""
	}
	record, ok, err := parseSceneRecordLine(text)
	if err != nil {
		c.pendingLine = text
		return nil
	}
	if !ok {
		return nil
	}
	return c.consumeRecord(ctx, record)
}

func (c *sceneConsumer) consumeRecord(ctx context.Context, record sceneRecord) error {
	switch strings.TrimSpace(record.Type) {
	case "plot_variable":
		return c.consumePlotVariable(ctx, record.PlotVariable)
	case "event":
		return c.consumeEvent(ctx, record.Event)
	case "interaction":
		return c.consumeInteraction(ctx, record.InteractionGroup)
	case "turn":
		return c.consumeTurn(ctx, record.Turn)
	case "stop":
		return c.consumeStop(ctx, record.StopReason)
	case "draft_delta", "result":
		c.addIssue(fmt.Sprintf("ignored v1-only NDJSON record type: %s", record.Type))
		return nil
	default:
		c.addIssue(fmt.Sprintf("跳过未知 NDJSON 记录类型：%s", record.Type))
		return nil
	}
}

func (c *sceneConsumer) consumePlotVariable(ctx context.Context, plot StoryNarrativePlotVariable) error {
	variable := StoryVariablePlan{PlotVariable: plot, CharacterViews: c.seed.CharacterViews}
	c.variable = normalizeStoryVariable(variable, c.input, c.snapshot)
	c.state.mu.Lock()
	c.state.variable = c.variable
	c.state.mu.Unlock()
	publishStoryEvent(ctx, c.g.deps, c.input.Run.RunID, domain.EventPlotVariable, map[string]any{"plot_variable": c.variable.PlotVariable})
	return nil
}

func (c *sceneConsumer) consumeEvent(ctx context.Context, event model.StoryEventPlan) error {
	if len(c.state.plannedActions) > 0 {
		c.addIssue("ignored scene event because planned_actions are authoritative")
		return nil
	}
	locationKey := strings.TrimSpace(event.LocationKey)
	if locationKey == "" || strings.TrimSpace(event.Summary) == "" {
		c.addIssue("跳过缺少 location_key 或 summary 的事件记录")
		return nil
	}
	actionType := normalizeStoryActionType(event.ActionType, "", event.Summary)
	if !isAllowedStoryActionType(actionType) {
		c.addIssue(fmt.Sprintf("跳过非法 action_type 的事件记录：%s", event.ActionType))
		return nil
	}
	c.state.mu.Lock()
	actorID, actorName, err := resolveStoryActor(c.state.characters, event.CharacterID, event.CharacterName, actionType)
	if err != nil {
		c.state.mu.Unlock()
		c.addIssue(err.Error())
		return nil
	}
	targetActorIDs, err := validStoryTargetActorIDs(c.state.characters, event.TargetActorIDs)
	if err != nil {
		c.state.mu.Unlock()
		c.addIssue(err.Error())
		return nil
	}
	if event.TimeIndex <= 0 {
		event.TimeIndex = len(c.state.events) + 1
	}
	if event.ID == "" {
		event.ID = fmt.Sprintf("story_event_%d", len(c.state.events)+1)
	}
	event.CharacterID = actorID
	event.CharacterName = actorName
	event.LocationKey = locationKey
	event.LocationName = strings.TrimSpace(event.LocationName)
	event.ActionType = actionType
	event.Summary = strings.TrimSpace(event.Summary)
	event.Intent = strings.TrimSpace(event.Intent)
	event.Visibility = strings.TrimSpace(event.Visibility)
	event.TargetActorIDs = targetActorIDs
	c.state.events = append(c.state.events, event)
	c.state.locationGroups = buildStoryLocationGroups(c.state.events)
	c.state.mu.Unlock()
	publishStoryEvent(ctx, c.g.deps, c.input.Run.RunID, domain.EventStoryEventPlanned, map[string]any{"event": event})
	return nil
}

func (c *sceneConsumer) consumeInteraction(ctx context.Context, group model.StoryInteractionGroup) error {
	c.publishLocationGroups(ctx)
	locationKey := strings.TrimSpace(group.LocationKey)
	if locationKey == "" {
		c.addIssue("跳过缺少 location_key 的交涉组")
		return nil
	}
	c.state.mu.Lock()
	candidate, ok := locationGroupByKey(c.state.locationGroups, locationKey)
	if !ok {
		c.state.mu.Unlock()
		c.addIssue("跳过未命中同地点候选的交涉组")
		return nil
	}
	characterIDs, err := validStoryTargetActorIDs(c.state.characters, group.CharacterIDs)
	if err != nil {
		c.state.mu.Unlock()
		c.addIssue(err.Error())
		return nil
	}
	characterIDs = uniqueStoryIDs(characterIDs)
	if len(characterIDs) < 2 {
		c.state.mu.Unlock()
		c.addIssue("跳过少于两个角色的交涉组")
		return nil
	}
	for _, characterID := range characterIDs {
		if !containsString(candidate.CharacterIDs, characterID) {
			c.state.mu.Unlock()
			c.addIssue("跳过跨地点交涉组")
			return nil
		}
	}
	if group.ShouldInteract && selectedInteractionCount(c.state.interactionGroups) >= 3 {
		c.state.mu.Unlock()
		c.addIssue("跳过超过 max_interactions 的交涉组")
		return nil
	}
	eventIDs := group.EventIDs
	if len(eventIDs) == 0 {
		eventIDs = candidate.EventIDs
	}
	if group.ID == "" {
		group.ID = fmt.Sprintf("interaction_%d", len(c.state.interactionGroups)+1)
	}
	group.LocationKey = candidate.LocationKey
	group.LocationName = firstText(group.LocationName, candidate.LocationName)
	group.CharacterIDs = characterIDs
	group.EventIDs = eventIDs
	group.InteractionType = strings.TrimSpace(group.InteractionType)
	group.Stakes = strings.TrimSpace(group.Stakes)
	group.Rationale = strings.TrimSpace(group.Rationale)
	c.state.interactionGroups = append(c.state.interactionGroups, group)
	analysis := model.StoryInteractionAnalysis{LocationGroups: copyStoryLocationGroups(c.state.locationGroups), InteractionGroups: copyStoryInteractionGroups(c.state.interactionGroups)}
	c.state.mu.Unlock()
	if group.ShouldInteract {
		updateStoryRunStep(ctx, c.g.deps, c.input.Run.RunID, domain.RunStatusNegotiatingInteractions, 60)
	} else {
		updateStoryRunStep(ctx, c.g.deps, c.input.Run.RunID, domain.RunStatusSelectingInteractions, 45)
	}
	publishStoryEvent(ctx, c.g.deps, c.input.Run.RunID, domain.EventInteractionAnalysis, map[string]any{"analysis": analysis})
	if group.ShouldInteract {
		publishStoryEvent(ctx, c.g.deps, c.input.Run.RunID, domain.EventInteractionSelected, map[string]any{"interaction_group": group})
	}
	return nil
}

func (c *sceneConsumer) consumeTurn(ctx context.Context, turn StoryTurnPlan) error {
	c.publishLocationGroups(ctx)
	turn.ActionType = normalizeStoryActionType(turn.ActionType, turn.Speech, turn.ActionSummary)
	if !isAllowedStoryActionType(turn.ActionType) {
		c.addIssue(fmt.Sprintf("跳过非法 action_type 的回合：%s", turn.ActionType))
		return nil
	}
	c.state.mu.Lock()
	if len(c.state.turns) >= c.state.maxTurns {
		c.state.stopReason = "达到最大回合数"
		c.state.mu.Unlock()
		c.addIssue("超过 max_turns 的回合已截断")
		return nil
	}
	actorID, actorName, err := resolveStoryActor(c.state.characters, turn.ActorID, turn.ActorName, turn.ActionType)
	if err != nil {
		c.state.mu.Unlock()
		c.addIssue(err.Error())
		return nil
	}
	targetActorIDs, err := validStoryTargetActorIDs(c.state.characters, turn.TargetActorIDs)
	if err != nil {
		c.state.mu.Unlock()
		c.addIssue(err.Error())
		return nil
	}
	if turn.TurnIndex <= 0 {
		turn.TurnIndex = len(c.state.turns) + 1
	}
	turn.ActorID = actorID
	turn.ActorName = actorName
	turn.TargetActorIDs = targetActorIDs
	if len(c.state.plannedActions) > 0 && !plannedActionsAllowTurn(c.state.plannedActions, turn.ActorID, turn.TargetActorIDs) {
		c.state.mu.Unlock()
		c.addIssue("story turn actor must belong to planned action participants")
		return nil
	}
	turn.Intent = strings.TrimSpace(turn.Intent)
	turn.Rationale = strings.TrimSpace(turn.Rationale)
	turn.InteractionGroupID = strings.TrimSpace(turn.InteractionGroupID)
	turn.LocationKey = strings.TrimSpace(turn.LocationKey)
	turn.LocationName = strings.TrimSpace(turn.LocationName)
	turn.Phase = strings.TrimSpace(turn.Phase)
	turn.Content = firstText(turn.Content, turn.Speech, turn.ActionSummary, turn.Intent, turn.Rationale)
	if turn.InteractionGroupID != "" {
		if err := validateStoryTurnInteraction(c.state.interactionGroups, turn); err != nil {
			c.state.mu.Unlock()
			c.addIssue(err.Error())
			return nil
		}
		if turn.Phase == "" {
			turn.Phase = "negotiation"
		}
		if turn.LocationKey == "" || turn.LocationName == "" {
			turn.LocationKey, turn.LocationName = interactionLocation(c.state.interactionGroups, turn.InteractionGroupID)
		}
	}
	c.state.turns = append(c.state.turns, turn)
	if turn.InteractionGroupID != "" {
		c.state.interactionTranscripts = upsertStoryInteractionTurn(c.state.interactionTranscripts, c.state.interactionGroups, turn)
	}
	c.state.mu.Unlock()
	if !c.firstTurnPublished {
		updateStoryRunStep(ctx, c.g.deps, c.input.Run.RunID, domain.RunStatusDrivingCharacterTurns, 70)
		c.firstTurnPublished = true
	}
	display := storyTurnDisplayPayload(turn)
	if turn.InteractionGroupID != "" {
		publishStoryEvent(ctx, c.g.deps, c.input.Run.RunID, domain.EventNegotiationTurn, display)
	}
	publishStoryEvent(ctx, c.g.deps, c.input.Run.RunID, domain.EventCharacterTurn, display)
	return nil
}

func (c *sceneConsumer) consumeStop(ctx context.Context, stopReason string) error {
	c.publishLocationGroups(ctx)
	c.state.mu.Lock()
	if stopReason != "" {
		c.state.stopReason = stopReason
	}
	c.state.mu.Unlock()
	publishStoryEvent(ctx, c.g.deps, c.input.Run.RunID, domain.EventGenerationStep, map[string]any{"step": "scene_stop", "stop_reason": stopReason})
	return nil
}

func (c *sceneConsumer) finish(ctx context.Context) (StoryPlanResult, StoryVariablePlan, error) {
	if strings.TrimSpace(c.pendingLine) != "" {
		record, ok, err := parseSceneRecordLine(c.pendingLine)
		if err != nil {
			return StoryPlanResult{}, StoryVariablePlan{}, fmt.Errorf("parse pending NDJSON record: %w", err)
		}
		if ok {
			if err := c.consumeRecord(ctx, record); err != nil {
				return StoryPlanResult{}, StoryVariablePlan{}, err
			}
		}
	}
	c.publishLocationGroups(ctx)
	plan := c.state.planResult()
	if len(plan.Turns) == 0 {
		plan.Turns = []StoryTurnPlan{{
			TurnIndex:     1,
			ActorName:     "旁白",
			ActionType:    "narration",
			Intent:        firstText(c.variable.PlotVariable.CoreChoice, c.input.Session.AuthorIntent, c.input.Session.OpeningSituation, "推进当前故事变量"),
			ActionSummary: "模型未产生回合，使用旁白占位",
		}}
	}
	if len(plan.EventPlan) == 0 {
		plan.EventPlan = fallbackEventPlanFromTurns(plan.Turns)
		plan.InteractionAnalysis.LocationGroups = buildStoryLocationGroups(plan.EventPlan)
	}
	plan.ContinuityIssues = append(plan.ContinuityIssues, c.reviewIssues...)
	return plan, c.variable, nil
}

func (c *sceneConsumer) hasCommittableScene() bool {
	c.state.mu.Lock()
	defer c.state.mu.Unlock()
	return len(c.state.turns) > 0 || len(c.state.interactionTranscripts) > 0
}

func (c *sceneConsumer) publishLocationGroups(ctx context.Context) {
	if c.locationGroupsPublished {
		return
	}
	c.state.mu.Lock()
	groups := copyStoryLocationGroups(c.state.locationGroups)
	c.state.mu.Unlock()
	c.locationGroupsPublished = true
	if len(groups) == 0 {
		return
	}
	updateStoryRunStep(ctx, c.g.deps, c.input.Run.RunID, domain.RunStatusSelectingInteractions, 45)
	publishStoryEvent(ctx, c.g.deps, c.input.Run.RunID, domain.EventSameLocationCandidates, map[string]any{"location_groups": groups})
}

func (c *sceneConsumer) addIssue(issue string) {
	issue = strings.TrimSpace(issue)
	if issue != "" {
		c.reviewIssues = append(c.reviewIssues, issue)
	}
}

func parseSceneRecordLine(line string) (sceneRecord, bool, error) {
	text := strings.TrimSpace(line)
	if text == "" || strings.HasPrefix(text, "```") {
		return sceneRecord{}, false, nil
	}
	start := strings.IndexByte(text, '{')
	end := strings.LastIndexByte(text, '}')
	if start < 0 || end < start {
		return sceneRecord{}, false, fmt.Errorf("line is not a JSON object")
	}
	text = text[start : end+1]
	var record sceneRecord
	if err := json.Unmarshal([]byte(text), &record); err != nil {
		return sceneRecord{}, false, err
	}
	if strings.TrimSpace(record.Type) == "" {
		return sceneRecord{}, false, fmt.Errorf("scene record missing type")
	}
	return record, true, nil
}

func fallbackEventPlanFromTurns(turns []StoryTurnPlan) []model.StoryEventPlan {
	events := make([]model.StoryEventPlan, 0, len(turns))
	for _, turn := range turns {
		summary := firstText(turn.ActionSummary, turn.Speech, turn.Content, turn.Intent)
		if summary == "" {
			continue
		}
		events = append(events, model.StoryEventPlan{
			ID:             fmt.Sprintf("story_event_%d", len(events)+1),
			TimeIndex:      len(events) + 1,
			CharacterID:    turn.ActorID,
			CharacterName:  turn.ActorName,
			LocationKey:    firstText(turn.LocationKey, "scene"),
			LocationName:   turn.LocationName,
			ActionType:     normalizeStoryActionType(turn.ActionType, turn.Speech, turn.ActionSummary),
			Summary:        summary,
			Intent:         turn.Intent,
			TargetActorIDs: turn.TargetActorIDs,
		})
	}
	return events
}

func seedPlannedActionEvents(state *storyRunState, plannedActions []ScenePlannedAction) {
	if len(plannedActions) == 0 {
		return
	}
	events := eventPlanFromPlannedActions(plannedActions)
	if len(events) == 0 {
		return
	}
	state.mu.Lock()
	state.plannedActions = append([]ScenePlannedAction(nil), plannedActions...)
	state.events = events
	state.locationGroups = buildStoryLocationGroups(events)
	state.mu.Unlock()
}

func actionCompletionTickPlan(state *storyRunState, variable StoryVariablePlan) StoryPlanResult {
	state.mu.Lock()
	events := append([]model.StoryEventPlan(nil), state.events...)
	locationGroups := copyStoryLocationGroups(state.locationGroups)
	state.mu.Unlock()
	summary := firstText(variable.PlotVariable.CoreChoice, "advanced to next action completion")
	return StoryPlanResult{
		Summary:    summary,
		StopReason: "advanced to next action completion",
		EventPlan:  events,
		InteractionAnalysis: model.StoryInteractionAnalysis{
			LocationGroups: locationGroups,
		},
	}
}

func eventPlanFromPlannedActions(plannedActions []ScenePlannedAction) []model.StoryEventPlan {
	events := make([]model.StoryEventPlan, 0, len(plannedActions))
	for _, action := range plannedActions {
		characterID := strings.TrimSpace(action.CharacterID)
		description := strings.TrimSpace(action.Description)
		if characterID == "" || description == "" {
			continue
		}
		durationHours := action.DurationHours
		if durationHours <= 0 {
			durationHours = 1
		}
		events = append(events, model.StoryEventPlan{
			ID:             fmt.Sprintf("planned_action_%d", len(events)+1),
			TimeIndex:      0,
			DurationHours:  durationHours,
			StartAt:        copyTimePtr(action.StartAt),
			ArriveAt:       copyTimePtr(action.ArriveAt),
			EffectAt:       copyTimePtr(action.EffectAt),
			EndsAt:         copyTimePtr(action.EndsAt),
			CharacterID:    characterID,
			CharacterName:  strings.TrimSpace(action.CharacterName),
			LocationKey:    firstText(strings.TrimSpace(action.TargetLocationKey), "scene"),
			ActionType:     normalizeStoryActionType(action.ActionType, "", description),
			Summary:        description,
			Intent:         strings.TrimSpace(action.Rationale),
			TargetActorIDs: uniqueStoryIDs(action.ParticipantIDs),
			ResourceKeys:   uniqueStoryIDs(action.ResourceKeys),
		})
	}
	return events
}

func optionalTime(value time.Time) *time.Time {
	if value.IsZero() {
		return nil
	}
	copyValue := value
	return &copyValue
}

func copyTimePtr(value *time.Time) *time.Time {
	if value == nil || value.IsZero() {
		return nil
	}
	copyValue := *value
	return &copyValue
}

func isAllowedStoryActionType(actionType string) bool {
	switch actionType {
	case "speak", "action", "silence", "observe", "narration":
		return true
	default:
		return false
	}
}

func storyPlotVariableUsable(variable StoryNarrativePlotVariable) bool {
	return strings.TrimSpace(variable.PressureSource) != "" || strings.TrimSpace(variable.CoreChoice) != "" || strings.TrimSpace(variable.FocalCharacterID) != "" || len(variable.RelatedCharacterIDs) > 0 || len(variable.WorldStatePressure) > 0
}

func (g *StoryRunGenerator) reflectScene(ctx context.Context, input port.StoryRunGenerationInput, snapshot StoryContextSnapshot, plan StoryPlanResult, variable StoryVariablePlan) (SceneReflectionResult, error) {
	if g.model == nil {
		return SceneReflectionResult{}, errors.New("story model is not configured")
	}
	reflectionContext := g.buildReflectionContext(input, snapshot, plan, variable)
	payload, _ := json.Marshal(buildReflectionPromptInput(reflectionContext))
	msg, err := g.model.Generate(ctx, []*schema.Message{
		schema.SystemMessage(g.reflectSystemPrompt()),
		schema.UserMessage(string(payload)),
	}, maxTokensOption(g.cfg.Model, g.reflectTokenLimit()))
	if err != nil {
		return SceneReflectionResult{}, err
	}
	var reflection SceneReflectionResult
	if err := decodeModelJSON(msg.Content, &reflection); err != nil {
		return SceneReflectionResult{}, err
	}
	if strings.TrimSpace(reflection.Summary) == "" {
		reflection.Summary = fallbackSceneSummary(plan, variable)
	}
	return reflection, nil
}

func (g *StoryRunGenerator) buildReflectionContext(input port.StoryRunGenerationInput, snapshot StoryContextSnapshot, plan StoryPlanResult, variable StoryVariablePlan) ReflectionContext {
	characters := make([]ReflectionCharacter, 0, len(snapshot.Characters))
	for _, character := range snapshot.Characters {
		characters = append(characters, ReflectionCharacter{ID: character.ID, Name: character.Name, Role: character.Role})
	}
	return ReflectionContext{
		Scene: ReflectionScene{
			PlotVariable:           variable.PlotVariable,
			Events:                 plan.EventPlan,
			Turns:                  toStoryTurns(plan.Turns),
			InteractionTranscripts: plan.InteractionTranscripts,
			StopReason:             plan.StopReason,
		},
		Characters:      characters,
		PerceptionIndex: buildPerceptionIndex(snapshot.Characters, plan),
		PriorMemories:   compactMemories(snapshot.RecentMemories, 8),
		Relationships:   compactRelationships(snapshot.Relationships, 20),
		WorldState:      compactWorldState(snapshot.WorldState, 20),
	}
}

func fallbackReflectionResult(plan StoryPlanResult, variable StoryVariablePlan) SceneReflectionResult {
	return SceneReflectionResult{Summary: fallbackSceneSummary(plan, variable)}
}

func fallbackSceneSummary(plan StoryPlanResult, variable StoryVariablePlan) string {
	return firstText(plan.Summary, variable.PlotVariable.CoreChoice, plan.StopReason, "scene simulation completed")
}

func (g *StoryRunGenerator) assembleStoryRunResult(input port.StoryRunGenerationInput, plan StoryPlanResult, reflection SceneReflectionResult, variable StoryVariablePlan) model.StoryRunResult {
	sceneSummary := firstText(reflection.Summary, fallbackSceneSummary(plan, variable))
	plotVariable := variable.PlotVariable
	focalID := firstText(plotVariable.FocalCharacterID, firstActorID(plan.Turns))
	relatedIDs := plotVariable.RelatedCharacterIDs
	if len(relatedIDs) == 0 {
		relatedIDs = relatedCharacterIDs(plan.Turns)
	}
	coreChoice := firstText(plotVariable.CoreChoice, sceneSummary, input.Session.AuthorIntent, "advance current story variable")
	return model.StoryRunResult{
		RunID:     input.Run.RunID,
		SessionID: input.Run.SessionID,
		Status:    domain.RunStatusCompleted,
		PlotVariable: model.PlotVariable{
			PressureSource:      firstText(plotVariable.PressureSource, input.Session.OpeningSituation, input.Session.AuthorIntent, "current story pressure"),
			FocalCharacterID:    focalID,
			CoreChoice:          coreChoice,
			OptionA:             firstText(plotVariable.OptionA, "hold the current line"),
			OptionB:             firstText(plotVariable.OptionB, "break the current line"),
			CostA:               firstText(plotVariable.CostA, "pressure continues accumulating"),
			CostB:               firstText(plotVariable.CostB, "intent is exposed or a cost is paid"),
			IrreversibleEffect:  firstText(plotVariable.IrreversibleEffect, plan.StopReason, "the scene closes with irreversible pressure"),
			RelatedCharacterIDs: relatedIDs,
			WorldStatePressure:  plotVariable.WorldStatePressure,
		},
		EventPlan:              plan.EventPlan,
		Turns:                  toStoryTurns(plan.Turns),
		SceneSummary:           sceneSummary,
		InteractionAnalysis:    plan.InteractionAnalysis,
		InteractionTranscripts: plan.InteractionTranscripts,
		Draft: model.Draft{
			ID:        g.newID("draft"),
			Title:     firstText(input.Session.Title, "Untitled scene"),
			Content:   "",
			Summary:   sceneSummary,
			WordCount: 0,
		},
		Review: model.ReviewReport{
			Pass:             true,
			ContinuityIssues: plan.ContinuityIssues,
		},
		MemoryPatch: model.MemoryPatch{
			ID:                     g.newID("patch"),
			Status:                 domain.MemoryPatchStatusRunLocal,
			CharacterMemoryUpdates: toCharacterMemoryUpdates(reflection.MemoryPatch.CharacterMemoryUpdates),
			RelationshipUpdates:    toRelationshipUpdates(reflection.MemoryPatch.RelationshipUpdates),
			WorldStateUpdates:      toWorldStateUpdates(reflection.MemoryPatch.WorldStateUpdates),
		},
	}
}

func toStoryTurns(turns []StoryTurnPlan) []model.StoryTurn {
	out := make([]model.StoryTurn, 0, len(turns))
	for _, turn := range turns {
		out = append(out, model.StoryTurn{
			TurnIndex:          turn.TurnIndex,
			ActorID:            turn.ActorID,
			ActorName:          turn.ActorName,
			ActionType:         turn.ActionType,
			Speech:             turn.Speech,
			ActionSummary:      turn.ActionSummary,
			TargetActorIDs:     turn.TargetActorIDs,
			Intent:             turn.Intent,
			InteractionGroupID: turn.InteractionGroupID,
			LocationKey:        turn.LocationKey,
			LocationName:       turn.LocationName,
			Phase:              turn.Phase,
		})
	}
	return out
}

func buildPerceptionIndex(characters []model.Character, plan StoryPlanResult) []PerceptionIndexEntry {
	out := make([]PerceptionIndexEntry, 0, len(characters))
	for _, character := range characters {
		entry := PerceptionIndexEntry{CharacterID: character.ID}
		for _, event := range plan.EventPlan {
			if characterWitnessesEvent(character.ID, event, plan.EventPlan) {
				entry.WitnessedEventIDs = appendUniqueString(entry.WitnessedEventIDs, event.ID)
			}
		}
		for _, turn := range plan.Turns {
			if characterWitnessesTurn(character.ID, turn, plan.EventPlan) {
				entry.WitnessedTurnIndexes = appendUniqueInt(entry.WitnessedTurnIndexes, turn.TurnIndex)
			}
		}
		out = append(out, entry)
	}
	return out
}

func characterWitnessesEvent(characterID string, event model.StoryEventPlan, events []model.StoryEventPlan) bool {
	if characterID == "" {
		return false
	}
	if event.CharacterID == characterID || containsString(event.TargetActorIDs, characterID) {
		return true
	}
	if strings.EqualFold(strings.TrimSpace(event.Visibility), "private") {
		return false
	}
	return characterAppearsAtLocation(events, characterID, event.LocationKey)
}

func characterWitnessesTurn(characterID string, turn StoryTurnPlan, events []model.StoryEventPlan) bool {
	if characterID == "" {
		return false
	}
	if turn.ActorID == characterID || containsString(turn.TargetActorIDs, characterID) {
		return true
	}
	if turn.ActorID == "" && turn.ActionType == "narration" {
		return true
	}
	return characterAppearsAtLocation(events, characterID, turn.LocationKey)
}

func characterAppearsAtLocation(events []model.StoryEventPlan, characterID string, locationKey string) bool {
	if characterID == "" || locationKey == "" {
		return false
	}
	for _, event := range events {
		if event.LocationKey != locationKey {
			continue
		}
		if event.CharacterID == characterID || containsString(event.TargetActorIDs, characterID) {
			return true
		}
	}
	return false
}

func appendUniqueInt(values []int, value int) []int {
	if value <= 0 {
		return values
	}
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}
func (g *StoryRunGenerator) newID(prefix string) string {
	if g.ids != nil {
		return g.ids.New(prefix)
	}
	return fmt.Sprintf("%s_%d", prefix, g.now().UnixNano())
}

func (g *StoryRunGenerator) now() time.Time {
	if g.clock != nil {
		return g.clock.Now()
	}
	return time.Now().UTC()
}

func visibleWorldForCharacter(worldState []model.WorldStateEntry, character model.Character) []model.WorldStateEntry {
	visible := make([]model.WorldStateEntry, 0, len(worldState))
	for _, entry := range worldState {
		if entry.Importance >= 4 || entry.Volatility >= 4 || strings.Contains(entry.Note, character.Name) || strings.Contains(entry.Key, character.ID) {
			visible = append(visible, entry)
		}
	}
	if len(visible) == 0 && len(worldState) > 0 {
		limit := 3
		if len(worldState) < limit {
			limit = len(worldState)
		}
		visible = append(visible, worldState[:limit]...)
	}
	return visible
}

func relationshipViewsForCharacter(relationships []model.Relationship, characterID string) []model.RelationshipView {
	views := make([]model.RelationshipView, 0)
	for _, relationship := range relationships {
		for _, view := range relationship.Views {
			if view.SourceCharacterID == characterID {
				views = append(views, view)
			}
		}
	}
	return views
}

func worldStateKeys(worldState []model.WorldStateEntry) []string {
	keys := make([]string, 0, len(worldState))
	for _, entry := range worldState {
		keys = append(keys, entry.Key)
	}
	return keys
}

func normalizeStoryVariable(variable StoryVariablePlan, input port.StoryRunGenerationInput, snapshot StoryContextSnapshot) StoryVariablePlan {
	validCharacters := map[string]struct{}{}
	for _, character := range snapshot.Characters {
		validCharacters[character.ID] = struct{}{}
	}
	plot := variable.PlotVariable
	if _, ok := validCharacters[plot.FocalCharacterID]; !ok {
		plot.FocalCharacterID = ""
	}
	if plot.FocalCharacterID == "" && len(snapshot.Characters) > 0 {
		plot.FocalCharacterID = snapshot.Characters[0].ID
	}
	plot.PressureSource = firstText(plot.PressureSource, input.Session.OpeningSituation, input.Session.AuthorIntent, "当前故事压力")
	plot.CoreChoice = firstText(plot.CoreChoice, input.Session.AuthorIntent, input.Session.OpeningSituation, "推进当前故事变量")
	plot.OptionA = firstText(plot.OptionA, "暂时维持当前局面")
	plot.OptionB = firstText(plot.OptionB, "主动打破当前局面")
	plot.CostA = firstText(plot.CostA, "压力继续累积")
	plot.CostB = firstText(plot.CostB, "暴露意图或承担代价")
	plot.IrreversibleEffect = firstText(plot.IrreversibleEffect, "本章状态将发生不可逆变化")
	plot.RelatedCharacterIDs = validCharacterIDs(plot.RelatedCharacterIDs, validCharacters)
	if len(plot.RelatedCharacterIDs) == 0 && plot.FocalCharacterID != "" {
		plot.RelatedCharacterIDs = []string{plot.FocalCharacterID}
	}
	if len(plot.WorldStatePressure) == 0 {
		plot.WorldStatePressure = worldStateKeys(snapshot.WorldState)
		if len(plot.WorldStatePressure) > 3 {
			plot.WorldStatePressure = plot.WorldStatePressure[:3]
		}
	}
	views := make([]CharacterVariableView, 0, len(variable.CharacterViews))
	seenViews := map[string]struct{}{}
	for _, view := range variable.CharacterViews {
		if _, ok := validCharacters[view.CharacterID]; !ok {
			continue
		}
		if _, ok := seenViews[view.CharacterID]; ok {
			continue
		}
		seenViews[view.CharacterID] = struct{}{}
		views = append(views, view)
	}
	for _, characterID := range plot.RelatedCharacterIDs {
		if _, ok := seenViews[characterID]; ok {
			continue
		}
		views = append(views, CharacterVariableView{
			CharacterID:       characterID,
			KnownFacts:        []string{plot.PressureSource},
			EmotionalPressure: plot.CoreChoice,
			ActionBias:        firstText(plot.OptionA, plot.OptionB),
		})
	}
	return StoryVariablePlan{PlotVariable: plot, CharacterViews: views}
}

func validCharacterIDs(ids []string, validCharacters map[string]struct{}) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		if _, ok := validCharacters[id]; !ok {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

func variableViewForCharacter(views []CharacterVariableView, characterID string) *CharacterVariableView {
	for _, view := range views {
		if view.CharacterID != characterID {
			continue
		}
		copyView := view
		return &copyView
	}
	return nil
}

func toCharacterMemoryUpdates(updates []StoryNarrativeCharacterMemoryUpdate) []model.CharacterMemoryUpdate {
	out := make([]model.CharacterMemoryUpdate, 0, len(updates))
	for _, update := range updates {
		if update.CharacterID == "" || update.Content == "" {
			continue
		}
		out = append(out, model.CharacterMemoryUpdate{CharacterID: update.CharacterID, Type: update.Type, Content: update.Content, Importance: update.Importance})
	}
	return out
}

func toRelationshipUpdates(updates []StoryNarrativeRelationshipUpdate) []model.RelationshipUpdate {
	out := make([]model.RelationshipUpdate, 0, len(updates))
	for _, update := range updates {
		if update.PairID == "" && update.Summary == "" && len(update.Events) == 0 {
			continue
		}
		out = append(out, model.RelationshipUpdate{PairID: update.PairID, Summary: update.Summary, TensionDelta: update.TensionDelta, Events: update.Events})
	}
	return out
}

func toWorldStateUpdates(updates []StoryNarrativeWorldStateUpdate) []model.WorldStateUpdate {
	out := make([]model.WorldStateUpdate, 0, len(updates))
	for _, update := range updates {
		if update.Key == "" {
			continue
		}
		out = append(out, model.WorldStateUpdate{Key: update.Key, Operation: update.Operation, Value: update.Value, Note: update.Note})
	}
	return out
}

func firstActorID(turns []StoryTurnPlan) string {
	for _, turn := range turns {
		if turn.ActorID != "" {
			return turn.ActorID
		}
	}
	return ""
}

func relatedCharacterIDs(turns []StoryTurnPlan) []string {
	seen := map[string]struct{}{}
	ids := make([]string, 0)
	for _, turn := range turns {
		if turn.ActorID == "" {
			continue
		}
		if _, ok := seen[turn.ActorID]; ok {
			continue
		}
		seen[turn.ActorID] = struct{}{}
		ids = append(ids, turn.ActorID)
	}
	return ids
}

func firstText(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
