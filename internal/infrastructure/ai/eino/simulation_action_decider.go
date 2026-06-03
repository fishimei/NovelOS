package eino

import (
	"context"
	"encoding/json"
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
	sharedPayload, _ := json.Marshal(buildActionDecisionSharedPrompt(input.World))
	characterPayload, _ := json.Marshal(buildActionDecisionCharacterPrompt(input))
	msg, err := d.model.Generate(ctx, []*schema.Message{
		schema.SystemMessage(d.prompt),
		schema.UserMessage("shared_context:\n" + string(sharedPayload)),
		schema.UserMessage("character_context:\n" + string(characterPayload)),
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
