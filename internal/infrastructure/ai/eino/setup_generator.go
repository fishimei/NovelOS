package eino

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	llmmodel "github.com/cloudwego/eino/components/model"
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
	msg, err := g.model.Generate(ctx, []*schema.Message{
		schema.SystemMessage(g.systemPrompt()),
		schema.UserMessage(g.userPrompt(input)),
	}, maxTokensOption(g.modelName, 6000))
	if err != nil {
		return model.SetupRunResult{}, err
	}
	var out setupAgentOutput
	if err := decodeModelJSON(msg.Content, &out); err != nil {
		return model.SetupRunResult{}, err
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
  "assistant_summary": ""
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
	return model.SetupDraft{
		AuthorBible:      bible,
		Characters:       characters,
		Relationships:    relationships,
		WorldState:       worldState,
		OpenQuestions:    questions,
		AssistantSummary: out.AssistantSummary,
	}, nil
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
