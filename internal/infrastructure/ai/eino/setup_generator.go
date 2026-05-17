package eino

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	llmmodel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent/react"
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
	model     llmmodel.ToolCallingChatModel
	modelName string
	prompt    string
	events    port.GenerationEventStream
	clock     port.Clock
	ids       port.IDGenerator
}

func NewSetupRunGenerator(ctx context.Context, deps SetupRunGeneratorDeps) (*SetupRunGenerator, error) {
	chatModel, err := newOpenAIChatModel(ctx, deps.Config)
	if err != nil {
		return nil, err
	}
	return &SetupRunGenerator{
		model:     chatModel,
		modelName: deps.Config.Model,
		prompt:    deps.Config.SetupAgent.Prompt,
		events:    deps.Events,
		clock:     deps.Clock,
		ids:       deps.IDs,
	}, nil
}

func (g *SetupRunGenerator) Generate(ctx context.Context, input port.SetupRunGenerationInput) (model.SetupRunResult, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	if g.events != nil {
		_ = g.events.Publish(ctx, input.Run.RunID, port.GenerationEvent{Name: domain.EventGenerationStep, Data: map[string]any{"step": "drafting_setup", "progress": 50}})
	}
	out, err := g.generateWithTools(ctx, input)
	if err != nil {
		out, err = g.generateLegacyJSON(ctx, input)
		if err != nil {
			return model.SetupRunResult{}, err
		}
	}
	draft, err := g.buildDraft(input, out)
	if err != nil {
		return model.SetupRunResult{}, err
	}
	return model.SetupRunResult{
		RunID:      input.Run.RunID,
		SessionID:  input.Run.SessionID,
		Status:     domain.RunStatusReviewRequired,
		SetupDraft: draft,
	}, nil
}

func (g *SetupRunGenerator) generateWithTools(ctx context.Context, input port.SetupRunGenerationInput) (setupAgentOutput, error) {
	state := &setupRunState{}
	tools, err := newSetupTools(state)
	if err != nil {
		return setupAgentOutput{}, err
	}
	agent, err := react.NewAgent(ctx, &react.AgentConfig{
		ToolCallingModel: g.model,
		ToolsConfig:      compose.ToolsNodeConfig{Tools: tools},
		MaxStep:          24,
		ToolReturnDirectly: map[string]struct{}{
			"show_setup_draft":   {},
			"revise_setup_draft": {},
		},
	})
	if err != nil {
		return setupAgentOutput{}, err
	}
	_, err = agent.Generate(ctx, []*schema.Message{
		schema.SystemMessage(g.toolSystemPrompt()),
		schema.UserMessage(g.toolUserPrompt(input)),
	})
	if err != nil {
		return setupAgentOutput{}, err
	}
	out := state.agentOutput()
	if len(out.Characters) == 0 {
		return setupAgentOutput{}, pkgerr.Validation("setup agent returned no characters")
	}
	return out, nil
}

func (g *SetupRunGenerator) generateLegacyJSON(ctx context.Context, input port.SetupRunGenerationInput) (setupAgentOutput, error) {
	msg, err := g.model.Generate(ctx, []*schema.Message{
		schema.SystemMessage(g.systemPrompt()),
		schema.UserMessage(g.userPrompt(input)),
	}, maxTokensOption(g.modelName, 6000))
	if err != nil {
		return setupAgentOutput{}, err
	}
	var out setupAgentOutput
	if err := decodeModelJSON(msg.Content, &out); err != nil {
		return setupAgentOutput{}, err
	}
	return out, nil
}

func (g *SetupRunGenerator) toolSystemPrompt() string {
	base := firstText(g.prompt, `你是 NovelOS 的 Setup 主控 agent。用户只需要说想创作哪类小说或给出粗略灵感，你要主动推理类型约定、世界压力、人物功能位、关系张力和初始状态。`)
	return base + `

你要先在内部完成理解、起草和深化，不要把“拆解意图”“设计世界”“设计人物”“编排关系”当作工具调用。工具只用于把已经深化完成的完整候选草案交给后端展示，或记录下一步 agent 建议。

当用户是首次给出种子、要求继续生成或没有明确要求沿用旧版时，调用 show_setup_draft。若用户要求微调、重起草或替换上一版，调用 revise_setup_draft。两者都必须一次性提交完整详细草案，包括：
- author_bible：主题、文风、世界规则、审美原则、硬约束、软偏好、禁用套路。
- world_state：关键世界变量，每项要有 key、value、note、importance、volatility。
- characters：主要角色，每人要有 key、name、role、profile、personality、voice_style、goals、fears、secrets、constraints。
- relationships：使用 characters 里的 key，写出摘要、锚点、张力、共同历史、波动值和双方视角。
- visual_draft：给用户看的 logline、风格标签、气质、大胆程度、世界压力卡、人物卡、关系边、待确认问题、agent 摘要和下一步建议。
- open_questions：只放真正无法合理推断且会改变主方向的问题。
- assistant_summary：解释这版草案的核心取向和用户应该重点审阅的地方。
- next_agent_suggestions：确认后可进入的 agent，例如角色深化、关系深化或第一章编排。

如果需要单独补充下一步建议，可以调用 handoff_next_agent；它不替代 show_setup_draft 或 revise_setup_draft。所有工具都只生成候选草案，不写正式数据库。`
}

func (g *SetupRunGenerator) toolUserPrompt(input port.SetupRunGenerationInput) string {
	messages, _ := json.Marshal(input.Session.Messages)
	return fmt.Sprintf(`project_id: %s
setup_session_id: %s
seed_idea: %s
last_user_message: %s
conversation_messages_json: %s

请根据以上信息调用工具起草 NovelOS 初始项目状态和可视化草案。`,
		input.Run.ProjectID,
		input.Session.ID,
		input.Session.SeedIdea,
		input.Session.LastUserMessage,
		string(messages),
	)
}

func (g *SetupRunGenerator) systemPrompt() string {
	if strings.TrimSpace(g.prompt) != "" {
		return g.prompt
	}
	return `你是 NovelOS 的 Setup 编剧 agent。用户会用自然语言说明想创作的小说类型或粗略灵感，你要主动推理并拆解，而不是处处追问。

你的目标：生成可直接进入项目状态的作者圣经、世界状态、主要角色、角色关系与少量必要澄清问题。

原则：
- 从类型文学约定、用户语气、题材关键词中合理推断世界压力、核心矛盾、人物功能位和初始关系。
- 不要等待用户提供所有细节；只有影响主设定方向且无法合理推断的信息才放进 open_questions。
- 角色要有目标、恐惧、秘密、约束和明显 voice_style，方便后续受限视角演绎。
- relationship 必须使用 characters 中的 key 引用角色，不要编数据库 ID。
- 输出必须是 JSON 对象，不要 markdown，不要代码块，不要额外解释。

JSON 结构：
{
  "author_bible": {
    "theme": "",
    "style_guide": "",
    "world_rules": [],
    "aesthetic_principles": [],
    "hard_constraints": [],
    "soft_preferences": [],
    "forbidden_moves": []
  },
  "world_state": [{"key":"", "value":{}, "note":"", "importance": 1, "volatility": 1}],
  "characters": [{"key":"", "name":"", "role":"", "profile":"", "personality":"", "voice_style":"", "goals":[], "fears":[], "secrets":[], "constraints":[]}],
  "relationships": [{"character_a_key":"", "character_b_key":"", "summary":"", "anchors":[], "tension_points":[], "shared_history":[], "volatility": 1, "character_a_view":{"public_attitude":"", "private_attitude":"", "believed_target_attitude":"", "masking_strategy":""}, "character_b_view":{"public_attitude":"", "private_attitude":"", "believed_target_attitude":"", "masking_strategy":""}}],
  "open_questions": [{"key":"", "question":"", "why_it_matters":""}],
  "assistant_summary": "",
  "visual_draft": {"logline":"", "style_tags":[], "tone":"", "boldness_level": 1, "world_pressure_cards":[], "character_cards":[], "relationship_edges":[], "open_questions":[], "agent_summary":"", "next_agent_suggestions":[]}
}`
}

func (g *SetupRunGenerator) userPrompt(input port.SetupRunGenerationInput) string {
	messages, _ := json.Marshal(input.Session.Messages)
	return fmt.Sprintf(`project_id: %s
setup_session_id: %s
seed_idea: %s
last_user_message: %s
conversation_messages_json: %s

请根据以上信息主动拆解成 NovelOS 初始项目状态。`,
		input.Run.ProjectID,
		input.Session.ID,
		input.Session.SeedIdea,
		input.Session.LastUserMessage,
		string(messages),
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
