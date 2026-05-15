package eino

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	llmmodel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"

	"github.com/fishimei/NovelOS/internal/application/model"
	"github.com/fishimei/NovelOS/internal/application/port"
	"github.com/fishimei/NovelOS/internal/config"
	"github.com/fishimei/NovelOS/internal/domain"
)

type StoryRunGeneratorDeps struct {
	Config        config.AIConfig
	Sessions      port.StorySessionRepository
	AuthorBibles  port.AuthorBibleRepository
	WorldState    port.WorldStateRepository
	Characters    port.CharacterRepository
	Relationships port.RelationshipRepository
	Chapters      port.ChapterRepository
	Memories      port.MemoryRepository
	Events        port.GenerationEventStream
	Clock         port.Clock
	IDs           port.IDGenerator
}

type StoryRunGenerator struct {
	cfg             config.AIConfig
	model           llmmodel.ToolCallingChatModel
	deps            storyGeneratorDeps
	clock           port.Clock
	ids             port.IDGenerator
	maxTurns        int
	controller      string
	toolPrompt      string
	resultPrompt    string
	narrativePrompt string
}

func NewStoryRunGenerator(ctx context.Context, deps StoryRunGeneratorDeps) (*StoryRunGenerator, error) {
	maxTurns := deps.Config.StoryAgent.MaxTurns
	if maxTurns <= 0 {
		maxTurns = 25
	}
	if maxTurns > 25 {
		maxTurns = 25
	}
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
			events:        deps.Events,
		},
		clock:           deps.Clock,
		ids:             deps.IDs,
		maxTurns:        maxTurns,
		controller:      deps.Config.StoryAgent.ControllerPrompt,
		toolPrompt:      deps.Config.StoryAgent.ToolPrompt,
		resultPrompt:    deps.Config.StoryAgent.ResultPrompt,
		narrativePrompt: deps.Config.StoryAgent.NarrativePrompt,
	}
	return generator, nil
}

func (g *StoryRunGenerator) Generate(ctx context.Context, input port.StoryRunGenerationInput) (model.StoryRunResult, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	state := &storyRunState{run: input.Run, session: input.Session, maxTurns: g.maxTurns}
	tools, err := newStoryTools(g.deps, state)
	if err != nil {
		return model.StoryRunResult{}, err
	}
	agent, err := react.NewAgent(ctx, &react.AgentConfig{
		ToolCallingModel: g.model,
		ToolsConfig:      compose.ToolsNodeConfig{Tools: tools},
		MaxStep:          g.maxTurns*3 + 5,
		ToolReturnDirectly: map[string]struct{}{
			"finalize_story_plan": {},
		},
	})
	if err != nil {
		return model.StoryRunResult{}, err
	}
	if g.deps.events != nil {
		_ = g.deps.events.Publish(ctx, input.Run.RunID, port.GenerationEvent{Name: domain.EventGenerationStep, Data: map[string]any{"step": domain.RunStatusDrivingCharacterTurns, "progress": 30}})
	}
	_, err = agent.Generate(ctx, []*schema.Message{
		schema.SystemMessage(g.systemPrompt()),
		schema.UserMessage(g.userPrompt(input)),
	})
	if err != nil {
		return model.StoryRunResult{}, err
	}
	plan := state.planResult()
	if len(plan.Turns) == 0 {
		plan.Turns = []StoryTurnPlan{{TurnIndex: 1, ActorName: "旁白", ActionType: "narration", Intent: firstText(input.Session.LastAuthorMessage, input.Session.AuthorIntent, input.Session.OpeningSituation, "推进当前故事变量"), Rationale: "模型未产生回合，使用旁白占位"}}
	}
	if g.deps.events != nil {
		_ = g.deps.events.Publish(ctx, input.Run.RunID, port.GenerationEvent{Name: domain.EventGenerationStep, Data: map[string]any{"step": domain.RunStatusWritingNarrative, "progress": 80}})
	}
	narrative, err := g.generateNarrative(ctx, input, plan)
	if err != nil {
		return model.StoryRunResult{}, err
	}
	if g.deps.events != nil {
		_ = g.deps.events.Publish(ctx, input.Run.RunID, port.GenerationEvent{Name: domain.EventGenerationStep, Data: map[string]any{"step": domain.RunStatusGeneratingMemoryPatch, "progress": 90}})
	}
	result, err := g.buildResult(ctx, input, plan, narrative)
	if err != nil {
		return model.StoryRunResult{}, err
	}
	return result, nil
}

func (g *StoryRunGenerator) systemPrompt() string {
	parts := []string{g.controller, g.toolPrompt, g.resultPrompt}
	return strings.Join(parts, "\n\n")
}

func (g *StoryRunGenerator) userPrompt(input port.StoryRunGenerationInput) string {
	return fmt.Sprintf(`story_run_id: %s
project_id: %s
session_id: %s
title: %s
opening_situation: %s
author_intent: %s
last_author_message: %s

请先调用 load_story_context，然后在最多 %d 个业务回合内循环判断：是否停止；如果不停，调用 choose_next_story_actor 记录下一行动者。停止时调用 decide_story_stop 与 finalize_story_plan。`,
		input.Run.RunID,
		input.Run.ProjectID,
		input.Session.ID,
		input.Session.Title,
		input.Session.OpeningSituation,
		input.Session.AuthorIntent,
		input.Session.LastAuthorMessage,
		g.maxTurns,
	)
}

func (g *StoryRunGenerator) generateNarrative(ctx context.Context, input port.StoryRunGenerationInput, plan StoryPlanResult) (StoryNarrativeResult, error) {
	snapshot, err := loadStoryContext(ctx, g.deps, &storyRunState{run: input.Run, session: input.Session, maxTurns: g.maxTurns}, LoadStoryContextInput{ProjectID: input.Run.ProjectID, SessionID: input.Session.ID})
	if err != nil {
		return StoryNarrativeResult{}, err
	}
	turns := make([]StoryTurnPlan, 0, len(plan.Turns))
	for _, turn := range plan.Turns {
		written, err := g.generateTurnContent(ctx, input, snapshot, plan, turns, turn)
		if err != nil {
			return StoryNarrativeResult{}, err
		}
		turns = append(turns, written)
		if g.deps.events != nil {
			_ = g.deps.events.Publish(ctx, input.Run.RunID, port.GenerationEvent{Name: domain.EventDraftDelta, Data: map[string]any{"turn_index": written.TurnIndex, "actor_id": written.ActorID, "content": written.Content}})
		}
	}
	return g.generateNarrativeSummary(ctx, input, snapshot, StoryPlanResult{Summary: plan.Summary, StopReason: plan.StopReason, Turns: turns}), nil
}

func (g *StoryRunGenerator) generateTurnContent(ctx context.Context, input port.StoryRunGenerationInput, snapshot StoryContextSnapshot, plan StoryPlanResult, previousTurns []StoryTurnPlan, turn StoryTurnPlan) (StoryTurnPlan, error) {
	promptInput := map[string]any{
		"session":         input.Session,
		"author_bible":    snapshot.AuthorBible,
		"recent_chapters": snapshot.RecentChapters,
		"plan_summary":    plan.Summary,
		"stop_reason":     plan.StopReason,
		"previous_turns":  previousTurns,
		"current_turn":    turn,
		"perspective":     g.perspectiveForTurn(snapshot, turn),
	}
	payload, _ := json.Marshal(promptInput)
	msg, err := g.model.Generate(ctx, []*schema.Message{
		schema.SystemMessage(g.turnNarrativePrompt()),
		schema.UserMessage(string(payload)),
	}, llmmodel.WithTemperature(0.8), llmmodel.WithMaxTokens(1200))
	if err != nil {
		return StoryTurnPlan{}, err
	}
	var out struct {
		Content string `json:"content"`
	}
	if err := decodeModelJSON(msg.Content, &out); err != nil {
		return StoryTurnPlan{}, err
	}
	turn.Content = firstText(out.Content, turn.Intent)
	return turn, nil
}

func (g *StoryRunGenerator) generateNarrativeSummary(ctx context.Context, input port.StoryRunGenerationInput, snapshot StoryContextSnapshot, plan StoryPlanResult) StoryNarrativeResult {
	promptInput := map[string]any{
		"session":          input.Session,
		"author_bible":     snapshot.AuthorBible,
		"recent_chapters":  snapshot.RecentChapters,
		"world_state_keys": worldStateKeys(snapshot.WorldState),
		"relationships":    relationshipPublicSummaries(snapshot.Relationships),
		"turns":            plan.Turns,
		"stop_reason":      plan.StopReason,
	}
	payload, _ := json.Marshal(promptInput)
	msg, err := g.model.Generate(ctx, []*schema.Message{
		schema.SystemMessage(g.summaryPrompt()),
		schema.UserMessage(string(payload)),
	}, llmmodel.WithTemperature(0.5), llmmodel.WithMaxTokens(4000))
	if err == nil {
		var out StoryNarrativeResult
		if decodeModelJSON(msg.Content, &out) == nil {
			if len(out.Turns) == 0 {
				out.Turns = plan.Turns
			}
			if out.Content == "" {
				out.Content = formatDraftContent(StoryPlanResult{Summary: out.Summary, StopReason: plan.StopReason, Turns: out.Turns})
			}
			return out
		}
	}
	return StoryNarrativeResult{
		Title:   firstText(input.Session.Title, "未命名章节"),
		Summary: plan.Summary,
		Content: formatDraftContent(plan),
		PlotVariable: StoryNarrativePlotVariable{
			PressureSource:      firstText(input.Session.OpeningSituation, input.Session.LastAuthorMessage, input.Session.AuthorIntent, "当前故事压力"),
			FocalCharacterID:    firstActorID(plan.Turns),
			CoreChoice:          firstText(plan.Summary, input.Session.LastAuthorMessage, input.Session.AuthorIntent, "推进当前故事变量"),
			RelatedCharacterIDs: relatedCharacterIDs(plan.Turns),
		},
		Review: StoryNarrativeReview{Pass: true},
		Turns:  plan.Turns,
	}
}

func (g *StoryRunGenerator) turnNarrativePrompt() string {
	return firstText(g.narrativePrompt, "你是 NovelOS 的受限视角多角色演绎 agent。只根据当前 actor 的 perspective 生成本回合内容，输出 JSON：{\"content\":\"...\"}。") + `

本次只生成 current_turn 的正文。禁止使用 perspective 之外的秘密、private_attitude 或全局真相。speak 写角色台词和必要动作；action 写可观察动作；silence/observe 写沉默或观察；narration 写短叙事。只输出 JSON。`
}

func (g *StoryRunGenerator) summaryPrompt() string {
	return firstText(g.resultPrompt, "整理故事演绎结果。") + `

根据已生成的 turns 输出 JSON 对象，字段包括 title、summary、content、plot_variable、memory_patch、review、turns。content 要是连贯章节正文，不要只列提纲。memory_patch 只记录本章实际发生且角色会记住/世界会改变的内容。relationship_updates 只填 pair_id、summary、tension_delta。只输出 JSON。`
}

func (g *StoryRunGenerator) perspectiveForTurn(snapshot StoryContextSnapshot, turn StoryTurnPlan) *CharacterPerspective {
	if turn.ActorID == "" {
		return nil
	}
	for _, character := range snapshot.Characters {
		if character.ID != turn.ActorID {
			continue
		}
		return &CharacterPerspective{
			Character:         character,
			VisibleWorld:      visibleWorldForCharacter(snapshot.WorldState, character),
			RecentMemories:    snapshot.RecentMemories[character.ID],
			RelationshipViews: relationshipViewsForCharacter(snapshot.Relationships, character.ID),
		}
	}
	return nil
}

func (g *StoryRunGenerator) buildResult(ctx context.Context, input port.StoryRunGenerationInput, plan StoryPlanResult, narrative StoryNarrativeResult) (model.StoryRunResult, error) {
	chapterNumber, err := g.nextChapterNumber(ctx, input.Run.ProjectID)
	if err != nil {
		return model.StoryRunResult{}, err
	}
	content := firstText(narrative.Content, formatDraftContent(plan))
	focalID := firstText(narrative.PlotVariable.FocalCharacterID, firstActorID(plan.Turns))
	relatedIDs := narrative.PlotVariable.RelatedCharacterIDs
	if len(relatedIDs) == 0 {
		relatedIDs = relatedCharacterIDs(plan.Turns)
	}
	coreChoice := firstText(narrative.PlotVariable.CoreChoice, narrative.Summary, plan.Summary, input.Session.LastAuthorMessage, input.Session.AuthorIntent, "推进当前故事变量")
	return model.StoryRunResult{
		RunID:     input.Run.RunID,
		SessionID: input.Run.SessionID,
		Status:    domain.RunStatusReviewRequired,
		PlotVariable: model.PlotVariable{
			PressureSource:      firstText(narrative.PlotVariable.PressureSource, input.Session.OpeningSituation, input.Session.LastAuthorMessage, input.Session.AuthorIntent, "当前故事压力"),
			FocalCharacterID:    focalID,
			CoreChoice:          coreChoice,
			OptionA:             firstText(narrative.PlotVariable.OptionA, "暂时维持当前局面"),
			OptionB:             firstText(narrative.PlotVariable.OptionB, "主动打破当前局面"),
			CostA:               firstText(narrative.PlotVariable.CostA, "压力继续累积"),
			CostB:               firstText(narrative.PlotVariable.CostB, "暴露意图或承担代价"),
			IrreversibleEffect:  firstText(narrative.PlotVariable.IrreversibleEffect, plan.StopReason, "本轮回合裁决结束"),
			RelatedCharacterIDs: relatedIDs,
			WorldStatePressure:  narrative.PlotVariable.WorldStatePressure,
		},
		Draft: model.Draft{
			ID:            g.newID("draft"),
			Title:         firstText(narrative.Title, input.Session.Title, "未命名章节"),
			ChapterNumber: chapterNumber,
			Content:       content,
			Summary:       coreChoice,
			WordCount:     utf8.RuneCountInString(content),
		},
		Review: model.ReviewReport{
			Pass:             narrative.Review.Pass,
			HardViolations:   narrative.Review.HardViolations,
			ContinuityIssues: narrative.Review.ContinuityIssues,
			StyleIssues:      narrative.Review.StyleIssues,
			SuggestedFixes:   narrative.Review.SuggestedFixes,
		},
		MemoryPatch: model.MemoryPatch{
			ID:                     g.newID("patch"),
			Status:                 "draft",
			CharacterMemoryUpdates: toCharacterMemoryUpdates(narrative.MemoryPatch.CharacterMemoryUpdates),
			RelationshipUpdates:    toRelationshipUpdates(narrative.MemoryPatch.RelationshipUpdates),
			WorldStateUpdates:      toWorldStateUpdates(narrative.MemoryPatch.WorldStateUpdates),
		},
	}, nil
}

func (g *StoryRunGenerator) nextChapterNumber(ctx context.Context, projectID string) (int, error) {
	chapters, err := g.deps.chapters.ListByProjectID(ctx, projectID, model.PageInput{Page: 1, PageSize: 1000})
	if err != nil {
		return 0, err
	}
	maxNumber := 0
	for _, chapter := range chapters.Items {
		if chapter.ChapterNumber > maxNumber {
			maxNumber = chapter.ChapterNumber
		}
	}
	return maxNumber + 1, nil
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

func relationshipPublicSummaries(relationships []model.Relationship) []map[string]any {
	items := make([]map[string]any, 0, len(relationships))
	for _, relationship := range relationships {
		items = append(items, map[string]any{
			"pair_id":            relationship.Pair.ID,
			"left_character_id":  relationship.Pair.LeftCharacterID,
			"right_character_id": relationship.Pair.RightCharacterID,
			"summary":            relationship.Pair.Summary,
			"tension_points":     relationship.Pair.TensionPoints,
		})
	}
	return items
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
		if update.PairID == "" && update.Summary == "" {
			continue
		}
		out = append(out, model.RelationshipUpdate{PairID: update.PairID, Summary: update.Summary, TensionDelta: update.TensionDelta})
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

func formatDraftContent(plan StoryPlanResult) string {
	var builder strings.Builder
	if plan.Summary != "" {
		builder.WriteString(plan.Summary)
		builder.WriteString("\n\n")
	}
	for _, turn := range plan.Turns {
		actor := firstText(turn.ActorName, turn.ActorID, "旁白")
		content := firstText(turn.Content, turn.Intent)
		builder.WriteString(fmt.Sprintf("【回合 %d】%s（%s）：%s", turn.TurnIndex, actor, turn.ActionType, content))
		if turn.Rationale != "" && turn.Content == "" {
			builder.WriteString("。理由：")
			builder.WriteString(turn.Rationale)
		}
		builder.WriteString("\n")
	}
	if plan.StopReason != "" {
		builder.WriteString("\n停止原因：")
		builder.WriteString(plan.StopReason)
	}
	return builder.String()
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
