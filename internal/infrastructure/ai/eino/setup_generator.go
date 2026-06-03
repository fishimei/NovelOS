package eino

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	llmmodel "github.com/cloudwego/eino/components/model"
	einotool "github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"

	"github.com/fishimei/NovelOS/internal/application/model"
	"github.com/fishimei/NovelOS/internal/application/port"
	"github.com/fishimei/NovelOS/internal/config"
	"github.com/fishimei/NovelOS/internal/domain"
	"github.com/fishimei/NovelOS/internal/pkgerr"
)

type SetupRunGeneratorDeps struct {
	Config config.AIConfig
	Events port.GenerationEventStream
	Clock  port.Clock
	IDs    port.IDGenerator
}

type SetupRunGenerator struct {
	model  llmmodel.ToolCallingChatModel
	prompt string
	events port.GenerationEventStream
	clock  port.Clock
	ids    port.IDGenerator
}

func NewSetupRunGenerator(ctx context.Context, deps SetupRunGeneratorDeps) (*SetupRunGenerator, error) {
	chatModel, err := newOpenAIChatModel(ctx, deps.Config)
	if err != nil {
		return nil, err
	}
	return &SetupRunGenerator{
		model:  chatModel,
		prompt: deps.Config.SetupAgent.Prompt,
		events: deps.Events,
		clock:  deps.Clock,
		ids:    deps.IDs,
	}, nil
}

func (g *SetupRunGenerator) Generate(ctx context.Context, input port.SetupRunGenerationInput) (model.SetupRunResult, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	if g.events != nil {
		_ = g.events.Publish(ctx, input.Run.RunID, port.GenerationEvent{Name: domain.EventGenerationStep, Data: map[string]any{"step": "drafting_setup", "progress": 50}})
	}
	out, err := g.generateWithTools(ctx, input)
	if err != nil {
		return model.SetupRunResult{}, err
	}
	draft, err := g.buildDraft(input, out)
	if err != nil {
		return model.SetupRunResult{}, err
	}
	return model.SetupRunResult{
		RunID:      input.Run.RunID,
		SessionID:  input.Run.SessionID,
		Status:     domain.RunStatusCompleted,
		SetupDraft: draft,
	}, nil
}

func (g *SetupRunGenerator) generateWithTools(ctx context.Context, input port.SetupRunGenerationInput) (setupAgentOutput, error) {
	state := &setupRunState{}
	tools, err := newSetupTools(state)
	if err != nil {
		return setupAgentOutput{}, err
	}
	for _, step := range setupGenerationSteps() {
		if err := g.runSetupToolStep(ctx, input, state, tools, step); err != nil {
			return setupAgentOutput{}, err
		}
	}
	out, err := finalizeSetupDraft(ctx, state, FinalizeSetupDraftInput{})
	if err != nil {
		return setupAgentOutput{}, err
	}
	if len(out.Characters) == 0 {
		return setupAgentOutput{}, pkgerr.Validation("setup agent returned no characters")
	}
	return out, nil
}

type setupGenerationStep struct {
	toolName    string
	instruction string
}

func setupGenerationSteps() []setupGenerationStep {
	return []setupGenerationStep{
		{toolName: "set_setup_author_bible", instruction: "只生成 author_bible。必须包含 theme 和 style_guide，并给出世界规则、审美原则、硬约束、软偏好和禁用套路。"},
		{toolName: "set_setup_world_state", instruction: "只生成 world_state。至少 3 项，每项包含 key、value、note、importance、volatility。"},
		{toolName: "set_setup_characters", instruction: "只生成 characters。至少 3 个角色，每个角色包含 key、name、role、profile、personality、voice_style、goals、fears、secrets、constraints。"},
		{toolName: "set_setup_relationships", instruction: "只生成 relationships。至少 2 条关系，character_a_key 和 character_b_key 必须引用已生成 characters 的 key。"},
		{toolName: "set_setup_visual_draft", instruction: "只生成 visual_draft、open_questions、assistant_summary、next_agent_suggestions。给用户看的摘要要短而完整。"},
	}
}

func (g *SetupRunGenerator) runSetupToolStep(ctx context.Context, input port.SetupRunGenerationInput, state *setupRunState, tools []einotool.BaseTool, step setupGenerationStep) error {
	selected, info, err := setupToolByName(ctx, tools, step.toolName)
	if err != nil {
		return err
	}
	invokable, ok := selected.(einotool.InvokableTool)
	if !ok {
		return fmt.Errorf("setup tool %s is not invokable", step.toolName)
	}
	msg, err := g.model.Generate(ctx, []*schema.Message{
		schema.SystemMessage(g.toolStepSystemPrompt(step)),
		schema.UserMessage(g.toolStepUserPrompt(input, state.agentOutput(), step)),
	}, llmmodel.WithTools([]*schema.ToolInfo{info}), llmmodel.WithToolChoice(schema.ToolChoiceForced, step.toolName))
	if err != nil {
		return err
	}
	call, err := setupToolCall(msg, step.toolName)
	if err != nil {
		return err
	}
	_, err = invokable.InvokableRun(ctx, call.Function.Arguments)
	return err
}

func setupToolByName(ctx context.Context, tools []einotool.BaseTool, name string) (einotool.BaseTool, *schema.ToolInfo, error) {
	for _, candidate := range tools {
		info, err := candidate.Info(ctx)
		if err != nil {
			return nil, nil, err
		}
		if info.Name == name {
			return candidate, info, nil
		}
	}
	return nil, nil, fmt.Errorf("setup tool %s not found", name)
}

func setupToolCall(msg *schema.Message, name string) (schema.ToolCall, error) {
	if msg == nil || len(msg.ToolCalls) == 0 {
		return schema.ToolCall{}, pkgerr.Validation("setup agent did not call " + name)
	}
	for _, call := range msg.ToolCalls {
		if call.Function.Name == name {
			return call, nil
		}
	}
	return schema.ToolCall{}, pkgerr.Validation("setup agent called unexpected tool")
}

func (g *SetupRunGenerator) toolStepSystemPrompt(step setupGenerationStep) string {
	base := firstText(g.prompt, `你是 NovelOS 的 Setup 主控 agent。用户只需要说想创作哪类小说或给出粗略灵感，你要主动推理类型约定、世界压力、人物功能位、关系张力和初始状态。`)
	return base + `

当前请求只允许调用一个工具：` + step.toolName + `。
` + step.instruction + `
即使种子很短，也要合理推断，不要只返回澄清问题。输出要短而完整：每个字符串字段不超过 80 个汉字，每个数组优先 2-4 项。所有内容都是候选草案，不写正式数据库。`
}

func (g *SetupRunGenerator) toolStepUserPrompt(input port.SetupRunGenerationInput, current setupAgentOutput, step setupGenerationStep) string {
	messages, _ := json.Marshal(input.Session.Messages)
	currentDraft, _ := json.Marshal(current)
	return fmt.Sprintf(`seed_idea: %s
last_user_message: %s
conversation_messages_json: %s
current_draft_json: %s

请调用 %s，完成本阶段 Setup 草案。`,
		input.Session.SeedIdea,
		input.Session.LastUserMessage,
		string(messages),
		string(currentDraft),
		step.toolName,
	)
}

func (g *SetupRunGenerator) buildDraft(input port.SetupRunGenerationInput, out setupAgentOutput) (model.SetupDraft, error) {
	if len(out.Characters) == 0 {
		return model.SetupDraft{}, pkgerr.Validation("setup agent returned no characters")
	}
	characterIDs := map[string]string{}
	characters := make([]model.Character, 0, len(out.Characters))
	for i, character := range out.Characters {
		key := firstText(character.Key, fmt.Sprintf("character_%d", i+1))
		id := g.newID("character")
		characterIDs[key] = id
		characters = append(characters, model.Character{
			ID:          id,
			ProjectID:   input.Run.ProjectID,
			Name:        firstText(character.Name, key),
			Role:        character.Role,
			Profile:     character.Profile,
			Personality: character.Personality,
			VoiceStyle:  character.VoiceStyle,
			Goals:       character.Goals,
			Fears:       character.Fears,
			Secrets:     character.Secrets,
			Constraints: character.Constraints,
			Status:      "draft",
		})
	}
	worldState := make([]model.WorldStateEntry, 0, len(out.WorldState))
	for _, entry := range out.WorldState {
		if strings.TrimSpace(entry.Key) == "" {
			continue
		}
		worldState = append(worldState, model.WorldStateEntry{
			ID:         g.newID("world"),
			ProjectID:  input.Run.ProjectID,
			Key:        entry.Key,
			Value:      entry.Value,
			Note:       entry.Note,
			Status:     "draft",
			Importance: entry.Importance,
			Volatility: entry.Volatility,
		})
	}
	relationships := make([]model.Relationship, 0, len(out.Relationships))
	for _, relationship := range out.Relationships {
		leftID := characterIDs[relationship.CharacterAKey]
		rightID := characterIDs[relationship.CharacterBKey]
		if leftID == "" || rightID == "" || leftID == rightID {
			continue
		}
		pairID := g.newID("pair")
		relationships = append(relationships, model.Relationship{
			Pair: model.RelationshipPair{
				ID:               pairID,
				ProjectID:        input.Run.ProjectID,
				LeftCharacterID:  leftID,
				RightCharacterID: rightID,
				Summary:          relationship.Summary,
				Anchors:          relationship.Anchors,
				TensionPoints:    relationship.TensionPoints,
				SharedHistory:    relationship.SharedHistory,
				Volatility:       relationship.Volatility,
				Status:           "draft",
			},
			Views: []model.RelationshipView{
				g.relationshipView(input.Run.ProjectID, pairID, leftID, rightID, relationship.CharacterAView),
				g.relationshipView(input.Run.ProjectID, pairID, rightID, leftID, relationship.CharacterBView),
			},
		})
	}
	questions := make([]model.SetupQuestion, 0, len(out.OpenQuestions))
	for _, question := range out.OpenQuestions {
		if strings.TrimSpace(question.Question) == "" {
			continue
		}
		questions = append(questions, model.SetupQuestion{Key: question.Key, Question: question.Question, WhyItMatters: question.WhyItMatters})
	}
	bible := model.AuthorBible{
		ID:                  g.newID("bible"),
		ProjectID:           input.Run.ProjectID,
		Theme:               out.AuthorBible.Theme,
		StyleGuide:          out.AuthorBible.StyleGuide,
		WorldRules:          out.AuthorBible.WorldRules,
		AestheticPrinciples: out.AuthorBible.AestheticPrinciples,
		HardConstraints:     out.AuthorBible.HardConstraints,
		SoftPreferences:     out.AuthorBible.SoftPreferences,
		ForbiddenMoves:      out.AuthorBible.ForbiddenMoves,
		InitialWorldState:   worldState,
		Status:              "draft",
	}
	visualDraft := setupVisualDraft(out, questions)
	return model.SetupDraft{
		AuthorBible:      bible,
		Characters:       characters,
		Relationships:    relationships,
		WorldState:       worldState,
		OpenQuestions:    questions,
		AssistantSummary: out.AssistantSummary,
		VisualDraft:      visualDraft,
	}, nil
}

func setupVisualDraft(out setupAgentOutput, questions []model.SetupQuestion) *model.SetupVisualDraft {
	visual := out.VisualDraft
	if visual.Logline == "" && visual.AgentSummary == "" && len(visual.StyleTags) == 0 && len(visual.WorldPressureCards) == 0 && len(visual.CharacterCards) == 0 && len(visual.RelationshipEdges) == 0 {
		return nil
	}
	openQuestions := questions
	if len(visual.OpenQuestions) > 0 {
		openQuestions = setupQuestions(visual.OpenQuestions)
	}
	return &model.SetupVisualDraft{
		Logline:              visual.Logline,
		StyleTags:            visual.StyleTags,
		Tone:                 visual.Tone,
		BoldnessLevel:        visual.BoldnessLevel,
		WorldPressureCards:   setupWorldPressureCards(visual.WorldPressureCards),
		CharacterCards:       setupCharacterCards(visual.CharacterCards),
		RelationshipEdges:    setupRelationshipEdges(visual.RelationshipEdges),
		OpenQuestions:        openQuestions,
		AgentSummary:         firstText(visual.AgentSummary, out.AssistantSummary),
		NextAgentSuggestions: setupNextAgentSuggestions(firstSuggestions(visual.NextAgentSuggestions, out.NextAgentSuggestions)),
	}
}

func setupQuestions(inputs []setupQuestionOutput) []model.SetupQuestion {
	questions := make([]model.SetupQuestion, 0, len(inputs))
	for _, question := range inputs {
		if strings.TrimSpace(question.Question) == "" {
			continue
		}
		questions = append(questions, model.SetupQuestion{Key: question.Key, Question: question.Question, WhyItMatters: question.WhyItMatters})
	}
	return questions
}

func setupWorldPressureCards(inputs []setupWorldPressureCardOutput) []model.SetupVisualWorldPressureCard {
	cards := make([]model.SetupVisualWorldPressureCard, 0, len(inputs))
	for _, input := range inputs {
		cards = append(cards, model.SetupVisualWorldPressureCard{
			Title:                 input.Title,
			Detail:                input.Detail,
			Stakes:                input.Stakes,
			RelatedWorldStateKeys: input.RelatedWorldStateKeys,
		})
	}
	return cards
}

func setupCharacterCards(inputs []setupCharacterCardOutput) []model.SetupVisualCharacterCard {
	cards := make([]model.SetupVisualCharacterCard, 0, len(inputs))
	for _, input := range inputs {
		cards = append(cards, model.SetupVisualCharacterCard{
			CharacterKey: input.CharacterKey,
			Name:         input.Name,
			Role:         input.Role,
			Hook:         input.Hook,
			Goal:         input.Goal,
			Fear:         input.Fear,
			Secret:       input.Secret,
		})
	}
	return cards
}

func setupRelationshipEdges(inputs []setupRelationshipEdgeOutput) []model.SetupVisualRelationshipEdge {
	edges := make([]model.SetupVisualRelationshipEdge, 0, len(inputs))
	for _, input := range inputs {
		edges = append(edges, model.SetupVisualRelationshipEdge{
			FromCharacterKey: input.FromCharacterKey,
			ToCharacterKey:   input.ToCharacterKey,
			Summary:          input.Summary,
			Tension:          input.Tension,
			Misreading:       input.Misreading,
		})
	}
	return edges
}

func setupNextAgentSuggestions(inputs []setupNextAgentSuggestion) []model.SetupNextAgentSuggestion {
	suggestions := make([]model.SetupNextAgentSuggestion, 0, len(inputs))
	for _, input := range inputs {
		if strings.TrimSpace(input.Key) == "" && strings.TrimSpace(input.Label) == "" {
			continue
		}
		suggestions = append(suggestions, model.SetupNextAgentSuggestion{Key: input.Key, Label: input.Label, Reason: input.Reason})
	}
	return suggestions
}

func firstSuggestions(primary []setupNextAgentSuggestion, fallback []setupNextAgentSuggestion) []setupNextAgentSuggestion {
	if len(primary) > 0 {
		return primary
	}
	return fallback
}

func (g *SetupRunGenerator) relationshipView(projectID string, pairID string, sourceID string, targetID string, input setupRelationshipViewOutput) model.RelationshipView {
	return model.RelationshipView{
		ID:                     g.newID("view"),
		ProjectID:              projectID,
		PairID:                 pairID,
		SourceCharacterID:      sourceID,
		TargetCharacterID:      targetID,
		PublicAttitude:         input.PublicAttitude,
		PrivateAttitude:        input.PrivateAttitude,
		BelievedTargetAttitude: input.BelievedTargetAttitude,
		MaskingStrategy:        input.MaskingStrategy,
		Status:                 "draft",
	}
}

func (g *SetupRunGenerator) newID(prefix string) string {
	if g.ids != nil {
		return g.ids.New(prefix)
	}
	return fmt.Sprintf("%s_%d", prefix, g.now().UnixNano())
}

func (g *SetupRunGenerator) now() time.Time {
	if g.clock != nil {
		return g.clock.Now()
	}
	return time.Now().UTC()
}
