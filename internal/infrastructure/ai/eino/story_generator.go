package eino

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	llmmodel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"
	"github.com/cloudwego/eino-ext/components/model/openai"

	"github.com/fishimei/NovelOS/internal/application/model"
	"github.com/fishimei/NovelOS/internal/application/port"
	"github.com/fishimei/NovelOS/internal/config"
	"github.com/fishimei/NovelOS/internal/domain"
	"github.com/fishimei/NovelOS/internal/pkgerr"
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
	cfg          config.AIConfig
	model        llmmodel.ToolCallingChatModel
	deps         storyGeneratorDeps
	clock        port.Clock
	ids          port.IDGenerator
	maxTurns     int
	controller   string
	toolPrompt   string
	resultPrompt string
}

func NewStoryRunGenerator(ctx context.Context, deps StoryRunGeneratorDeps) (*StoryRunGenerator, error) {
	if deps.Config.Provider != "openai_compatible" {
		return nil, pkgerr.Validation("unsupported ai provider")
	}
	if deps.Config.BaseURL == "" {
		return nil, pkgerr.Validation("ai base_url is required")
	}
	if deps.Config.Model == "" {
		return nil, pkgerr.Validation("ai model is required")
	}
	maxTurns := deps.Config.StoryAgent.MaxTurns
	if maxTurns <= 0 {
		maxTurns = 25
	}
	if maxTurns > 25 {
		maxTurns = 25
	}
	chatModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		APIKey:  deps.Config.APIKey,
		Model:   deps.Config.Model,
		BaseURL: deps.Config.BaseURL,
		Timeout: 90 * time.Second,
	})
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
		clock:        deps.Clock,
		ids:          deps.IDs,
		maxTurns:     maxTurns,
		controller:   deps.Config.StoryAgent.ControllerPrompt,
		toolPrompt:   deps.Config.StoryAgent.ToolPrompt,
		resultPrompt: deps.Config.StoryAgent.ResultPrompt,
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
		_ = g.deps.events.Publish(ctx, input.Run.RunID, port.GenerationEvent{Name: domain.EventGenerationStep, Data: map[string]any{"step": domain.RunStatusGeneratingMemoryPatch, "progress": 90}})
	}
	result, err := g.buildResult(ctx, input, plan)
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

func (g *StoryRunGenerator) buildResult(ctx context.Context, input port.StoryRunGenerationInput, plan StoryPlanResult) (model.StoryRunResult, error) {
	chapterNumber, err := g.nextChapterNumber(ctx, input.Run.ProjectID)
	if err != nil {
		return model.StoryRunResult{}, err
	}
	content := formatDraftContent(plan)
	focalID := firstActorID(plan.Turns)
	relatedIDs := relatedCharacterIDs(plan.Turns)
	coreChoice := firstText(plan.Summary, input.Session.LastAuthorMessage, input.Session.AuthorIntent, "推进当前故事变量")
	return model.StoryRunResult{
		RunID:     input.Run.RunID,
		SessionID: input.Run.SessionID,
		Status:    domain.RunStatusReviewRequired,
		PlotVariable: model.PlotVariable{
			PressureSource:      firstText(input.Session.OpeningSituation, input.Session.LastAuthorMessage, input.Session.AuthorIntent, "当前故事压力"),
			FocalCharacterID:    focalID,
			CoreChoice:          coreChoice,
			OptionA:             "暂时维持当前局面",
			OptionB:             "主动打破当前局面",
			CostA:               "压力继续累积",
			CostB:               "暴露意图或承担代价",
			IrreversibleEffect:  firstText(plan.StopReason, "本轮回合裁决结束"),
			RelatedCharacterIDs: relatedIDs,
		},
		Draft: model.Draft{
			ID:            g.newID("draft"),
			Title:         firstText(input.Session.Title, "未命名章节"),
			ChapterNumber: chapterNumber,
			Content:       content,
			Summary:       coreChoice,
			WordCount:     utf8.RuneCountInString(content),
		},
		Review: model.ReviewReport{
			Pass:           true,
			SuggestedFixes: []string{"人物对白生成尚未启用，本草稿仅用于验证回合控制链路。"},
		},
		MemoryPatch: model.MemoryPatch{
			ID:     g.newID("patch"),
			Status: "draft",
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

func formatDraftContent(plan StoryPlanResult) string {
	var builder strings.Builder
	if plan.Summary != "" {
		builder.WriteString(plan.Summary)
		builder.WriteString("\n\n")
	}
	for _, turn := range plan.Turns {
		actor := firstText(turn.ActorName, turn.ActorID, "旁白")
		builder.WriteString(fmt.Sprintf("【回合 %d】%s（%s）：%s", turn.TurnIndex, actor, turn.ActionType, turn.Intent))
		if turn.Rationale != "" {
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
