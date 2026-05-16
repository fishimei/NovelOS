package eino

import (
	"context"
	"fmt"
	"slices"
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
	events        port.GenerationEventStream
}

type storyRunState struct {
	mu         sync.Mutex
	run        model.StoryRun
	session    model.StorySession
	maxTurns   int
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
	chooseActor, err := utils.InferTool("choose_next_story_actor", "记录下一轮应该由谁发言或产生动作。不要生成完整对白，只给行动者、动作类型、意图和理由。", func(ctx context.Context, input ChooseNextStoryActorInput) (StoryTurnPlan, error) {
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
	for _, character := range snapshot.Characters {
		memories, err := deps.memories.ListByCharacterID(ctx, character.ID, 5)
		if err != nil && !isNotFound(err) {
			return StoryContextSnapshot{}, err
		}
		if len(memories) > 0 {
			snapshot.RecentMemories[character.ID] = memories
		}
	}
	return snapshot, nil
}

func chooseNextStoryActor(ctx context.Context, deps storyGeneratorDeps, state *storyRunState, input ChooseNextStoryActorInput) (StoryTurnPlan, error) {
	allowedActions := []string{"speak", "action", "silence", "observe", "narration"}
	if !slices.Contains(allowedActions, input.ActionType) {
		return StoryTurnPlan{}, pkgerr.Validation("invalid action type")
	}
	state.mu.Lock()
	defer state.mu.Unlock()
	if len(state.turns) >= state.maxTurns {
		state.stopReason = "达到最大回合数"
		return StoryTurnPlan{}, pkgerr.Validation("max story turns reached")
	}
	turnIndex := input.TurnIndex
	if turnIndex <= 0 {
		turnIndex = len(state.turns) + 1
	}
	turn := StoryTurnPlan{
		TurnIndex:  turnIndex,
		ActorID:    input.ActorID,
		ActorName:  input.ActorName,
		ActionType: input.ActionType,
		Intent:     input.Intent,
		Rationale:  input.Rationale,
	}
	state.turns = append(state.turns, turn)
	if deps.events != nil {
		_ = deps.events.Publish(ctx, state.run.RunID, port.GenerationEvent{Name: domain.EventCharacterTurn, Data: turn})
	}
	return turn, nil
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
