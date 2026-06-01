package eino

import (
	"context"
	"fmt"
	"log"
	"slices"
	"strings"
	"sync"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"

	"github.com/fishimei/NovelOS/internal/application/model"
	"github.com/fishimei/NovelOS/internal/application/port"
	"github.com/fishimei/NovelOS/internal/domain"
	"github.com/fishimei/NovelOS/internal/pkgerr"
)

type storyGeneratorDeps struct {
	sessions      port.StorySessionRepository
	authorBibles  port.AuthorBibleRepository
	worldState    port.WorldStateRepository
	characters    port.CharacterRepository
	relationships port.RelationshipRepository
	chapters      port.ChapterRepository
	memories      port.MemoryRepository
	memoryService port.CharacterMemoryService
	events        port.GenerationEventStream
	audit         port.AuditRepository
}

type storyRunState struct {
	mu                     sync.Mutex
	run                    model.StoryRun
	session                model.StorySession
	maxTurns               int
	characters             []model.Character
	turns                  []StoryTurnPlan
	events                 []model.StoryEventPlan
	locationGroups         []model.StoryLocationGroup
	interactionGroups      []model.StoryInteractionGroup
	interactionTranscripts []model.StoryInteractionTranscript
	stopReason             string
	summary                string
	variable               StoryVariablePlan
}

func newStoryTools(deps storyGeneratorDeps, state *storyRunState) ([]tool.BaseTool, error) {
	loadContext, err := utils.InferTool("load_story_context", "读取当前 story run 的项目状态、角色、关系、世界状态、章节和近期记忆。", func(ctx context.Context, input LoadStoryContextInput) (StoryContextSnapshot, error) {
		return loadStoryContext(ctx, deps, state, input)
	})
	if err != nil {
		return nil, err
	}
	recordEvent, err := utils.InferTool("record_story_event", "记录事件模拟阶段中某个角色在某个地点的行动。必须提供 location_key、action_type 和 summary。", func(ctx context.Context, input RecordStoryEventInput) (StoryEventRecordResult, error) {
		return recordStoryEvent(ctx, deps, state, input)
	})
	if err != nil {
		return nil, err
	}
	selectInteraction, err := utils.InferTool("select_story_interaction", "从同地点候选组中选择会实际发生交涉的角色组。角色必须来自同一个候选地点。", func(ctx context.Context, input SelectStoryInteractionInput) (model.StoryInteractionGroup, error) {
		return selectStoryInteraction(ctx, deps, state, input)
	})
	if err != nil {
		return nil, err
	}
	chooseActor, err := utils.InferTool("choose_next_story_actor", "记录后续剧情阶段产生的角色回合，并实时推送给前端。只提交哪个角色说了什么、做了什么，不要提交完整章节正文或关系分析。", func(ctx context.Context, input ChooseNextStoryActorInput) (StoryTurnPlan, error) {
		return chooseNextStoryActor(ctx, deps, state, input)
	})
	if err != nil {
		return nil, err
	}
	decideStop, err := utils.InferTool("decide_story_stop", "判断当前演绎是否应该停止。达到最大回合数时工具会强制停止。", func(ctx context.Context, input DecideStoryStopInput) (StoryStopDecision, error) {
		return decideStoryStop(ctx, deps, state, input)
	})
	if err != nil {
		return nil, err
	}
	finalizePlan, err := utils.InferTool("finalize_story_plan", "提交本次回合裁决的结构化摘要。停止时必须调用。", func(ctx context.Context, input FinalizeStoryPlanInput) (StoryPlanResult, error) {
		return finalizeStoryPlan(ctx, state, input)
	})
	if err != nil {
		return nil, err
	}
	return []tool.BaseTool{loadContext, recordEvent, selectInteraction, chooseActor, decideStop, finalizePlan}, nil
}

func loadStoryContext(ctx context.Context, deps storyGeneratorDeps, state *storyRunState, input LoadStoryContextInput) (StoryContextSnapshot, error) {
	projectID := input.ProjectID
	if projectID == "" {
		projectID = state.run.ProjectID
	}
	sessionID := input.SessionID
	if sessionID == "" {
		sessionID = state.session.ID
	}
	session := state.session
	if session.ID == "" || session.ID != sessionID {
		loaded, err := deps.sessions.GetSessionByID(ctx, sessionID)
		if err != nil {
			return StoryContextSnapshot{}, err
		}
		session = loaded
	}
	snapshot := StoryContextSnapshot{Session: session, RecentMemories: map[string][]model.Memory{}}
	bible, err := deps.authorBibles.GetByProjectID(ctx, projectID)
	if err == nil {
		snapshot.AuthorBible = &bible
	} else if !isNotFound(err) {
		return StoryContextSnapshot{}, err
	}
	worldState, err := deps.worldState.ListByProjectID(ctx, projectID)
	if err != nil {
		return StoryContextSnapshot{}, err
	}
	snapshot.WorldState = worldState
	characters, err := deps.characters.ListByProjectID(ctx, projectID, model.PageInput{Page: 1, PageSize: 50})
	if err != nil {
		return StoryContextSnapshot{}, err
	}
	snapshot.Characters = characters.Items
	state.mu.Lock()
	state.characters = characters.Items
	state.mu.Unlock()
	relationships, err := deps.relationships.ListByProjectID(ctx, projectID, model.PageInput{Page: 1, PageSize: 50})
	if err != nil {
		return StoryContextSnapshot{}, err
	}
	snapshot.Relationships = relationships.Items
	chapters, err := deps.chapters.ListByProjectID(ctx, projectID, model.PageInput{Page: 1, PageSize: 5})
	if err == nil {
		snapshot.RecentChapters = chapters.Items
	} else if !isNotFound(err) {
		return StoryContextSnapshot{}, err
	}
	query := storyMemoryRecallQuery(session, snapshot.WorldState, snapshot.RecentChapters)
	for _, character := range snapshot.Characters {
		memories, err := recallCharacterMemories(ctx, deps, state.run.RunID, projectID, character.ID, query)
		if err != nil && !isNotFound(err) {
			return StoryContextSnapshot{}, err
		}
		if len(memories) > 0 {
			snapshot.RecentMemories[character.ID] = memories
		}
	}
	return snapshot, nil
}

func recallCharacterMemories(ctx context.Context, deps storyGeneratorDeps, runID string, projectID string, characterID string, query string) ([]model.Memory, error) {
	if deps.memoryService != nil {
		memories, err := deps.memoryService.Recall(ctx, port.CharacterMemoryRecallInput{
			ProjectID:   projectID,
			CharacterID: characterID,
			Query:       query,
			Limit:       12,
		})
		if err == nil && len(memories) > 0 {
			return memories, nil
		}
		if err != nil {
			log.Printf("story run %s external memory recall failed for character %s: %v", runID, characterID, err)
			publishStoryEvent(ctx, deps, runID, domain.EventGenerationStep, map[string]any{"step": "external_memory_recall_failed", "character_id": characterID, "error": err.Error()})
		}
	}
	return deps.memories.ListByCharacterID(ctx, characterID, 5)
}

func storyMemoryRecallQuery(session model.StorySession, worldState []model.WorldStateEntry, chapters []model.Chapter) string {
	parts := []string{
		session.LastAuthorMessage,
		session.AuthorIntent,
		session.OpeningSituation,
		session.CurrentPlotVariableSummary,
	}
	for _, entry := range worldState {
		parts = append(parts, entry.Key, entry.Note)
	}
	for _, chapter := range chapters {
		parts = append(parts, chapter.Title, chapter.Summary)
	}
	out := strings.TrimSpace(strings.Join(parts, "\n"))
	if out == "" {
		return "当前章节中会影响角色选择、信念、情绪和行动模式的长期记忆"
	}
	return out
}

func recordStoryEvent(ctx context.Context, deps storyGeneratorDeps, state *storyRunState, input RecordStoryEventInput) (StoryEventRecordResult, error) {
	locationKey := strings.TrimSpace(input.LocationKey)
	if locationKey == "" {
		return StoryEventRecordResult{}, pkgerr.Validation("story event location_key is required")
	}
	if strings.TrimSpace(input.Summary) == "" {
		return StoryEventRecordResult{}, pkgerr.Validation("story event summary is required")
	}
	input.ActionType = normalizeStoryActionType(input.ActionType, "", input.Summary)
	allowedActions := []string{"speak", "action", "silence", "observe", "narration"}
	if !slices.Contains(allowedActions, input.ActionType) {
		return StoryEventRecordResult{}, pkgerr.Validation("invalid story event action type")
	}

	state.mu.Lock()
	actorID, actorName, err := resolveStoryActor(state.characters, input.CharacterID, input.CharacterName, input.ActionType)
	if err != nil {
		state.mu.Unlock()
		return StoryEventRecordResult{}, err
	}
	targetActorIDs, err := validStoryTargetActorIDs(state.characters, input.TargetActorIDs)
	if err != nil {
		state.mu.Unlock()
		return StoryEventRecordResult{}, err
	}
	timeIndex := input.TimeIndex
	if timeIndex <= 0 {
		timeIndex = len(state.events) + 1
	}
	event := model.StoryEventPlan{
		ID:             fmt.Sprintf("story_event_%d", len(state.events)+1),
		TimeIndex:      timeIndex,
		CharacterID:    actorID,
		CharacterName:  actorName,
		LocationKey:    locationKey,
		LocationName:   strings.TrimSpace(input.LocationName),
		ActionType:     input.ActionType,
		Summary:        strings.TrimSpace(input.Summary),
		Intent:         strings.TrimSpace(input.Intent),
		Visibility:     strings.TrimSpace(input.Visibility),
		TargetActorIDs: targetActorIDs,
	}
	state.events = append(state.events, event)
	state.locationGroups = buildStoryLocationGroups(state.events)
	analysis := model.StoryInteractionAnalysis{LocationGroups: copyStoryLocationGroups(state.locationGroups), InteractionGroups: copyStoryInteractionGroups(state.interactionGroups)}
	result := StoryEventRecordResult{Event: event, SameLocationCandidates: analysis.LocationGroups, InteractionAnalysis: analysis}
	state.mu.Unlock()

	publishStoryEvent(ctx, deps, state.run.RunID, domain.EventStoryEventPlanned, map[string]any{"event": event})
	if len(result.SameLocationCandidates) > 0 {
		updateStoryRunStep(ctx, deps, state.run.RunID, domain.RunStatusSelectingInteractions, 45)
		publishStoryEvent(ctx, deps, state.run.RunID, domain.EventSameLocationCandidates, map[string]any{"location_groups": result.SameLocationCandidates})
	}
	return result, nil
}

func selectStoryInteraction(ctx context.Context, deps storyGeneratorDeps, state *storyRunState, input SelectStoryInteractionInput) (model.StoryInteractionGroup, error) {
	locationKey := strings.TrimSpace(input.LocationKey)
	if locationKey == "" {
		return model.StoryInteractionGroup{}, pkgerr.Validation("interaction location_key is required")
	}
	state.mu.Lock()
	candidate, ok := locationGroupByKey(state.locationGroups, locationKey)
	if !ok {
		state.mu.Unlock()
		return model.StoryInteractionGroup{}, pkgerr.Validation("interaction must use a same-location candidate")
	}
	characterIDs, err := validStoryTargetActorIDs(state.characters, input.CharacterIDs)
	if err != nil {
		state.mu.Unlock()
		return model.StoryInteractionGroup{}, err
	}
	characterIDs = uniqueStoryIDs(characterIDs)
	if len(characterIDs) < 2 {
		state.mu.Unlock()
		return model.StoryInteractionGroup{}, pkgerr.Validation("interaction requires at least two characters")
	}
	for _, characterID := range characterIDs {
		if !containsString(candidate.CharacterIDs, characterID) {
			state.mu.Unlock()
			return model.StoryInteractionGroup{}, pkgerr.Validation("interaction characters must share the same location")
		}
	}
	if input.ShouldInteract && selectedInteractionCount(state.interactionGroups) >= 3 {
		state.mu.Unlock()
		return model.StoryInteractionGroup{}, pkgerr.Validation("max interaction groups reached")
	}
	eventIDs := input.EventIDs
	if len(eventIDs) == 0 {
		eventIDs = candidate.EventIDs
	}
	group := model.StoryInteractionGroup{
		ID:              fmt.Sprintf("interaction_%d", len(state.interactionGroups)+1),
		LocationKey:     candidate.LocationKey,
		LocationName:    candidate.LocationName,
		CharacterIDs:    characterIDs,
		EventIDs:        eventIDs,
		ShouldInteract:  input.ShouldInteract,
		InteractionType: strings.TrimSpace(input.InteractionType),
		Stakes:          strings.TrimSpace(input.Stakes),
		Rationale:       strings.TrimSpace(input.Rationale),
		Priority:        input.Priority,
	}
	state.interactionGroups = append(state.interactionGroups, group)
	analysis := model.StoryInteractionAnalysis{LocationGroups: copyStoryLocationGroups(state.locationGroups), InteractionGroups: copyStoryInteractionGroups(state.interactionGroups)}
	state.mu.Unlock()

	if group.ShouldInteract {
		updateStoryRunStep(ctx, deps, state.run.RunID, domain.RunStatusNegotiatingInteractions, 60)
	} else {
		updateStoryRunStep(ctx, deps, state.run.RunID, domain.RunStatusSelectingInteractions, 45)
	}
	publishStoryEvent(ctx, deps, state.run.RunID, domain.EventInteractionAnalysis, map[string]any{"analysis": analysis})
	if group.ShouldInteract {
		publishStoryEvent(ctx, deps, state.run.RunID, domain.EventInteractionSelected, map[string]any{"interaction_group": group})
	}
	return group, nil
}

func chooseNextStoryActor(ctx context.Context, deps storyGeneratorDeps, state *storyRunState, input ChooseNextStoryActorInput) (StoryTurnPlan, error) {
	input.ActionType = normalizeStoryActionType(input.ActionType, input.Speech, input.ActionSummary)
	allowedActions := []string{"speak", "action", "silence", "observe", "narration"}
	if !slices.Contains(allowedActions, input.ActionType) {
		return StoryTurnPlan{}, pkgerr.Validation("invalid action type")
	}
	state.mu.Lock()
	defer state.mu.Unlock()
	actorID, actorName, err := resolveStoryActor(state.characters, input.ActorID, input.ActorName, input.ActionType)
	if err != nil {
		return StoryTurnPlan{}, err
	}
	targetActorIDs, err := validStoryTargetActorIDs(state.characters, input.TargetActorIDs)
	if err != nil {
		return StoryTurnPlan{}, err
	}
	if len(state.turns) >= state.maxTurns {
		state.stopReason = "达到最大回合数"
		return StoryTurnPlan{}, pkgerr.Validation("max story turns reached")
	}
	turnIndex := input.TurnIndex
	if turnIndex <= 0 {
		turnIndex = len(state.turns) + 1
	}
	turn := StoryTurnPlan{
		TurnIndex:          turnIndex,
		ActorID:            actorID,
		ActorName:          actorName,
		ActionType:         input.ActionType,
		Speech:             input.Speech,
		ActionSummary:      input.ActionSummary,
		TargetActorIDs:     targetActorIDs,
		Intent:             input.Intent,
		Rationale:          input.Rationale,
		InteractionGroupID: strings.TrimSpace(input.InteractionGroupID),
		LocationKey:        strings.TrimSpace(input.LocationKey),
		LocationName:       strings.TrimSpace(input.LocationName),
		Phase:              strings.TrimSpace(input.Phase),
	}
	if turn.InteractionGroupID != "" {
		if err := validateStoryTurnInteraction(state.interactionGroups, turn); err != nil {
			return StoryTurnPlan{}, err
		}
		if turn.Phase == "" {
			turn.Phase = "negotiation"
		}
		if turn.LocationKey == "" || turn.LocationName == "" {
			turn.LocationKey, turn.LocationName = interactionLocation(state.interactionGroups, turn.InteractionGroupID)
		}
	}
	state.turns = append(state.turns, turn)
	if turn.InteractionGroupID != "" {
		state.interactionTranscripts = upsertStoryInteractionTurn(state.interactionTranscripts, state.interactionGroups, turn)
	}
	display := storyTurnDisplayPayload(turn)
	if turn.InteractionGroupID != "" {
		publishStoryEvent(ctx, deps, state.run.RunID, domain.EventNegotiationTurn, display)
	}
	publishStoryEvent(ctx, deps, state.run.RunID, domain.EventCharacterTurn, display)
	return turn, nil
}

func normalizeStoryActionType(actionType string, speech string, actionSummary string) string {
	action := strings.ToLower(strings.TrimSpace(actionType))
	action = strings.ReplaceAll(action, "-", "_")
	action = strings.ReplaceAll(action, " ", "_")
	switch action {
	case "speak", "speech", "dialogue", "dialog", "say", "talk", "line", "台词", "说话", "发言", "对话":
		return "speak"
	case "action", "act", "move", "do", "movement", "动作", "行动", "行为":
		return "action"
	case "silence", "silent", "pause", "沉默", "停顿":
		return "silence"
	case "observe", "observation", "watch", "look", "观察", "注视":
		return "observe"
	case "narration", "narrate", "narrative", "旁白", "叙述":
		return "narration"
	}
	if strings.TrimSpace(speech) != "" {
		return "speak"
	}
	if strings.TrimSpace(actionSummary) != "" {
		return "action"
	}
	return action
}

func resolveStoryActor(characters []model.Character, actorID string, actorName string, actionType string) (string, string, error) {
	if len(characters) == 0 {
		if strings.TrimSpace(actorID) == "" && strings.TrimSpace(actorName) == "" {
			return "", "旁白", nil
		}
		return actorID, actorName, nil
	}
	if strings.TrimSpace(actorID) == "" && strings.TrimSpace(actorName) == "" {
		if actionType == "narration" {
			return "", "旁白", nil
		}
		return "", "", pkgerr.Validation("story actor is required")
	}
	for _, character := range characters {
		if actorID != "" && character.ID == actorID {
			return character.ID, firstText(actorName, character.Name), nil
		}
	}
	for _, character := range characters {
		if actorName != "" && character.Name == actorName {
			return character.ID, character.Name, nil
		}
	}
	return "", "", pkgerr.Validation("story actor must match an existing character")
}

func validStoryTargetActorIDs(characters []model.Character, targetActorIDs []string) ([]string, error) {
	if len(characters) == 0 || len(targetActorIDs) == 0 {
		return targetActorIDs, nil
	}
	valid := map[string]struct{}{}
	for _, character := range characters {
		valid[character.ID] = struct{}{}
		valid[character.Name] = struct{}{}
	}
	out := make([]string, 0, len(targetActorIDs))
	for _, targetID := range targetActorIDs {
		if _, ok := valid[targetID]; !ok {
			return nil, pkgerr.Validation("story target actor must match an existing character")
		}
		for _, character := range characters {
			if character.ID == targetID || character.Name == targetID {
				out = append(out, character.ID)
				break
			}
		}
	}
	return out, nil
}

func storyTurnDisplayPayload(turn StoryTurnPlan) StoryTurnDisplayEvent {
	return StoryTurnDisplayEvent{
		TurnIndex:          turn.TurnIndex,
		ActorID:            turn.ActorID,
		ActorName:          turn.ActorName,
		ActionType:         turn.ActionType,
		Speech:             turn.Speech,
		ActionSummary:      turn.ActionSummary,
		TargetActorIDs:     turn.TargetActorIDs,
		InteractionGroupID: turn.InteractionGroupID,
		LocationKey:        turn.LocationKey,
		LocationName:       turn.LocationName,
		Phase:              turn.Phase,
	}
}

func decideStoryStop(ctx context.Context, deps storyGeneratorDeps, state *storyRunState, input DecideStoryStopInput) (StoryStopDecision, error) {
	state.mu.Lock()
	defer state.mu.Unlock()
	stop := input.Stop
	reason := input.Reason
	if len(state.turns) >= state.maxTurns {
		stop = true
		if reason == "" {
			reason = "达到最大回合数"
		}
	}
	if stop {
		state.stopReason = reason
	}
	decision := StoryStopDecision{Stop: stop, Reason: reason}
	publishStoryEvent(ctx, deps, state.run.RunID, domain.EventGenerationStep, map[string]any{"step": "stop_decision", "stop": stop, "reason": reason})
	return decision, nil
}

func finalizeStoryPlan(ctx context.Context, state *storyRunState, input FinalizeStoryPlanInput) (StoryPlanResult, error) {
	state.mu.Lock()
	defer state.mu.Unlock()
	state.summary = input.Summary
	if input.StopReason != "" {
		state.stopReason = input.StopReason
	}
	turns := make([]StoryTurnPlan, len(state.turns))
	copy(turns, state.turns)
	return StoryPlanResult{Summary: state.summary, StopReason: state.stopReason, Turns: turns}, nil
}

func isNotFound(err error) bool {
	appErr, ok := err.(*pkgerr.Error)
	return ok && appErr.Code == pkgerr.CodeNotFound
}

func buildStoryLocationGroups(events []model.StoryEventPlan) []model.StoryLocationGroup {
	byLocation := map[string]*model.StoryLocationGroup{}
	for _, event := range events {
		if event.LocationKey == "" || event.CharacterID == "" {
			continue
		}
		group := byLocation[event.LocationKey]
		if group == nil {
			group = &model.StoryLocationGroup{ID: fmt.Sprintf("location_group_%d", len(byLocation)+1), LocationKey: event.LocationKey, LocationName: event.LocationName}
			byLocation[event.LocationKey] = group
		}
		group.CharacterIDs = appendUniqueString(group.CharacterIDs, event.CharacterID)
		group.EventIDs = appendUniqueString(group.EventIDs, event.ID)
		if group.LocationName == "" {
			group.LocationName = event.LocationName
		}
	}
	groups := make([]model.StoryLocationGroup, 0, len(byLocation))
	for _, group := range byLocation {
		if len(group.CharacterIDs) >= 2 {
			groups = append(groups, *group)
		}
	}
	return groups
}

func locationGroupByKey(groups []model.StoryLocationGroup, locationKey string) (model.StoryLocationGroup, bool) {
	for _, group := range groups {
		if group.LocationKey == locationKey {
			return group, true
		}
	}
	return model.StoryLocationGroup{}, false
}

func selectedInteractionCount(groups []model.StoryInteractionGroup) int {
	count := 0
	for _, group := range groups {
		if group.ShouldInteract {
			count++
		}
	}
	return count
}

func uniqueStoryIDs(ids []string) []string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		out = appendUniqueString(out, id)
	}
	return out
}

func validateStoryTurnInteraction(groups []model.StoryInteractionGroup, turn StoryTurnPlan) error {
	for _, group := range groups {
		if group.ID != turn.InteractionGroupID || !group.ShouldInteract {
			continue
		}
		if turn.ActorID != "" && !containsString(group.CharacterIDs, turn.ActorID) {
			return pkgerr.Validation("story turn actor must belong to interaction group")
		}
		for _, targetID := range turn.TargetActorIDs {
			if !containsString(group.CharacterIDs, targetID) {
				return pkgerr.Validation("story turn targets must belong to interaction group")
			}
		}
		if turn.LocationKey != "" && turn.LocationKey != group.LocationKey {
			return pkgerr.Validation("story turn location must match interaction group")
		}
		return nil
	}
	return pkgerr.Validation("story turn interaction group must be selected")
}

func interactionLocation(groups []model.StoryInteractionGroup, groupID string) (string, string) {
	for _, group := range groups {
		if group.ID == groupID {
			return group.LocationKey, group.LocationName
		}
	}
	return "", ""
}

func upsertStoryInteractionTurn(transcripts []model.StoryInteractionTranscript, groups []model.StoryInteractionGroup, turn StoryTurnPlan) []model.StoryInteractionTranscript {
	interactionTurn := model.StoryInteractionTurn{
		TurnIndex:          turn.TurnIndex,
		InteractionGroupID: turn.InteractionGroupID,
		ActorID:            turn.ActorID,
		ActorName:          turn.ActorName,
		ActionType:         turn.ActionType,
		Speech:             turn.Speech,
		ActionSummary:      turn.ActionSummary,
		TargetActorIDs:     turn.TargetActorIDs,
		Intent:             turn.Intent,
		LocationKey:        turn.LocationKey,
		LocationName:       turn.LocationName,
	}
	for i := range transcripts {
		if transcripts[i].GroupID == turn.InteractionGroupID {
			transcripts[i].Turns = append(transcripts[i].Turns, interactionTurn)
			transcripts[i].OutcomeSummary = firstText(turn.Intent, turn.ActionSummary, turn.Speech, transcripts[i].OutcomeSummary)
			return transcripts
		}
	}
	transcript := model.StoryInteractionTranscript{GroupID: turn.InteractionGroupID, LocationKey: turn.LocationKey, LocationName: turn.LocationName, Turns: []model.StoryInteractionTurn{interactionTurn}, OutcomeSummary: firstText(turn.Intent, turn.ActionSummary, turn.Speech)}
	for _, group := range groups {
		if group.ID == turn.InteractionGroupID {
			transcript.LocationKey = firstText(transcript.LocationKey, group.LocationKey)
			transcript.LocationName = firstText(transcript.LocationName, group.LocationName)
			transcript.CharacterIDs = append([]string(nil), group.CharacterIDs...)
			break
		}
	}
	return append(transcripts, transcript)
}

func copyStoryLocationGroups(groups []model.StoryLocationGroup) []model.StoryLocationGroup {
	out := make([]model.StoryLocationGroup, len(groups))
	copy(out, groups)
	return out
}

func copyStoryInteractionGroups(groups []model.StoryInteractionGroup) []model.StoryInteractionGroup {
	out := make([]model.StoryInteractionGroup, len(groups))
	copy(out, groups)
	return out
}

func copyStoryInteractionTranscripts(transcripts []model.StoryInteractionTranscript) []model.StoryInteractionTranscript {
	out := make([]model.StoryInteractionTranscript, len(transcripts))
	copy(out, transcripts)
	return out
}

func updateStoryRunStep(ctx context.Context, deps storyGeneratorDeps, runID string, status string, progress int) {
	if deps.sessions != nil {
		_ = deps.sessions.UpdateRunStatus(ctx, runID, status, status, progress)
	}
	publishStoryEvent(ctx, deps, runID, domain.EventGenerationStep, map[string]any{"step": status, "progress": progress})
}

func publishStoryEvent(ctx context.Context, deps storyGeneratorDeps, runID string, name string, data any) {
	if deps.events != nil {
		_ = deps.events.Publish(ctx, runID, port.GenerationEvent{Name: name, Data: data})
	}
	if deps.audit == nil {
		return
	}
	payload, ok := data.(map[string]any)
	if !ok {
		payload = map[string]any{"data": data}
	}
	if _, err := deps.audit.AppendRunEvent(ctx, model.RunEvent{RunKind: "story", RunID: runID, EventName: name, Payload: payload}); err != nil {
		log.Printf("append story run event %s failed: %v", runID, err)
	}
}

func (s *storyRunState) planResult() StoryPlanResult {
	s.mu.Lock()
	defer s.mu.Unlock()
	turns := make([]StoryTurnPlan, len(s.turns))
	copy(turns, s.turns)
	events := make([]model.StoryEventPlan, len(s.events))
	copy(events, s.events)
	locationGroups := copyStoryLocationGroups(s.locationGroups)
	interactionGroups := copyStoryInteractionGroups(s.interactionGroups)
	transcripts := copyStoryInteractionTranscripts(s.interactionTranscripts)
	summary := s.summary
	if summary == "" {
		summary = fmt.Sprintf("完成 %d 轮回合裁决", len(turns))
	}
	stopReason := s.stopReason
	if stopReason == "" {
		stopReason = "回合裁决结束"
	}
	return StoryPlanResult{Summary: summary, StopReason: stopReason, Turns: turns, EventPlan: events, InteractionAnalysis: model.StoryInteractionAnalysis{LocationGroups: locationGroups, InteractionGroups: interactionGroups}, InteractionTranscripts: transcripts}
}
