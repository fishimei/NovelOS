package eino

import (
	"context"
	"encoding/json"
	"time"

	llmmodel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/schema"

	"github.com/fishimei/NovelOS/internal/application/model"
	"github.com/fishimei/NovelOS/internal/config"
	"github.com/fishimei/NovelOS/internal/pkgerr"
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
	tool, err := utils.InferTool("submit_character_action", "提交当前角色的下一步自主行动。", func(ctx context.Context, input model.CharacterActionDecision) (model.CharacterActionDecision, error) {
		return input, nil
	})
	if err != nil {
		return model.CharacterActionDecision{}, err
	}
	info, err := tool.Info(ctx)
	if err != nil {
		return model.CharacterActionDecision{}, err
	}
	msg, err := d.model.Generate(ctx, []*schema.Message{
		schema.SystemMessage(d.prompt),
		schema.UserMessage("shared_context:\n" + string(sharedPayload)),
		schema.UserMessage("character_context:\n" + string(characterPayload)),
	}, llmmodel.WithTools([]*schema.ToolInfo{info}), llmmodel.WithToolChoice(schema.ToolChoiceForced, "submit_character_action"), maxTokensOption(d.modelName, 800))
	if err != nil {
		return model.CharacterActionDecision{}, err
	}
	call, err := forcedToolCall(msg, "submit_character_action")
	if err != nil {
		return model.CharacterActionDecision{}, err
	}
	var decision model.CharacterActionDecision
	if _, err := decodeModelJSONWithMeta(call.Function.Arguments, &decision); err != nil {
		return model.CharacterActionDecision{}, err
	}
	return decision, nil
}

func forcedToolCall(msg *schema.Message, name string) (schema.ToolCall, error) {
	if msg == nil || len(msg.ToolCalls) == 0 {
		return schema.ToolCall{}, pkgerr.Validation("character action agent did not call " + name)
	}
	for _, call := range msg.ToolCalls {
		if call.Function.Name == name {
			return call, nil
		}
	}
	return schema.ToolCall{}, pkgerr.Validation("character action agent called unexpected tool")
}
