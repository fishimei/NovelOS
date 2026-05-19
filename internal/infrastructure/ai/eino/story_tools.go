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
}

type storyRunState struct {
	mu         sync.Mutex
	run        model.StoryRun
	session    model.StorySession
	maxTurns   int
	characters []model.Character
	turns      []StoryTurnPlan
	stopReason string
	summary    string
	variable   StoryVariablePlan
}

func newStoryTools(deps storyGeneratorDeps, state *storyRunState) ([]tool.BaseTool, error) {
	loadContext, err := utils.InferTool("load_story_context", "读取当前 story run 的项目状态、角色、关系、世界状态、章节和近期记忆。", func(ctx context.Context, input LoadStoryContextInput) (StoryContextSnapshot, error) {
		return loadStoryContext(ctx, deps, state, input)
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
	return []tool.BaseTool{loadContext, chooseActor, decideStop, finalizePlan}, nil
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
			if deps.events != nil {
				_ = deps.events.Publish(ctx, runID, port.GenerationEvent{Name: domain.EventGenerationStep, Data: map[string]any{"step": "external_memory_recall_failed", "character_id": characterID, "error": err.Error()}})
			}
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
		TurnIndex:      turnIndex,
		ActorID:        actorID,
		ActorName:      actorName,
		ActionType:     input.ActionType,
		Speech:         input.Speech,
		ActionSummary:  input.ActionSummary,
		TargetActorIDs: targetActorIDs,
		Intent:         input.Intent,
		Rationale:      input.Rationale,
	}
	state.turns = append(state.turns, turn)
	if deps.events != nil {
		_ = deps.events.Publish(ctx, state.run.RunID, port.GenerationEvent{Name: domain.EventCharacterTurn, Data: storyTurnDisplayPayload(turn)})
	}
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
		TurnIndex:      turn.TurnIndex,
		ActorID:        turn.ActorID,
		ActorName:      turn.ActorName,
		ActionType:     turn.ActionType,
		Speech:         turn.Speech,
		ActionSummary:  turn.ActionSummary,
		TargetActorIDs: turn.TargetActorIDs,
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
	if deps.events != nil {
		_ = deps.events.Publish(ctx, state.run.RunID, port.GenerationEvent{Name: domain.EventGenerationStep, Data: map[string]any{"step": "stop_decision", "stop": stop, "reason": reason}})
	}
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

func (s *storyRunState) planResult() StoryPlanResult {
	s.mu.Lock()
	defer s.mu.Unlock()
	turns := make([]StoryTurnPlan, len(s.turns))
	copy(turns, s.turns)
	summary := s.summary
	if summary == "" {
		summary = fmt.Sprintf("完成 %d 轮回合裁决", len(turns))
	}
	stopReason := s.stopReason
	if stopReason == "" {
		stopReason = "回合裁决结束"
	}
	return StoryPlanResult{Summary: summary, StopReason: stopReason, Turns: turns}
}
