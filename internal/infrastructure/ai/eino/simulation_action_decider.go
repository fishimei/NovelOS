package eino

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	llmmodel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"

	"github.com/fishimei/NovelOS/internal/application/model"
	"github.com/fishimei/NovelOS/internal/config"
)

type CharacterActionDecider struct {
	model     llmmodel.ToolCallingChatModel
	modelName string
	prompt    string
}

func NewCharacterActionDecider(ctx context.Context, cfg config.AIConfig) (*CharacterActionDecider, error) {
	chatModel, err := newOpenAIChatModel(ctx, cfg)
	if err != nil {
		return nil, err
	}
	return &CharacterActionDecider{model: chatModel, modelName: cfg.Model, prompt: cfg.StoryAgent.SimulationPrompt}, nil
}

func (d *CharacterActionDecider) Decide(ctx context.Context, input model.CharacterActionDecisionInput) (model.CharacterActionDecision, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	payload, _ := json.Marshal(input)
	msg, err := d.model.Generate(ctx, []*schema.Message{
		schema.SystemMessage(d.prompt),
		schema.UserMessage(fmt.Sprintf(`请基于以下 JSON 决定该角色的下一步行动：
%s

只返回 JSON 对象，字段为 action_type、description、duration_hours、rationale。duration_hours 必须是正整数。`, string(payload))),
	}, maxTokensOption(d.modelName, 800))
	if err != nil {
		return model.CharacterActionDecision{}, err
	}
	var decision model.CharacterActionDecision
	if err := decodeModelJSON(msg.Content, &decision); err != nil {
		return model.CharacterActionDecision{}, err
	}
	return decision, nil
}
