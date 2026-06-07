package eino

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	openaimodel "github.com/cloudwego/eino-ext/components/model/openai"
	llmmodel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"

	"github.com/fishimei/NovelOS/internal/config"
	"github.com/fishimei/NovelOS/internal/pkgerr"
)

func newOpenAIChatModel(ctx context.Context, cfg config.AIConfig) (llmmodel.ToolCallingChatModel, error) {
	if cfg.Provider != "openai_compatible" {
		return nil, pkgerr.Validation("unsupported ai provider")
	}
	if cfg.BaseURL == "" {
		return nil, pkgerr.Validation("ai base_url is required")
	}
	if cfg.Model == "" {
		return nil, pkgerr.Validation("ai model is required")
	}
	return openaimodel.NewChatModel(ctx, &openaimodel.ChatModelConfig{
		APIKey:  cfg.APIKey,
		Model:   cfg.Model,
		BaseURL: cfg.BaseURL,
		Timeout: 5 * time.Minute,
	})
}

func toolCallOptions(modelName string, maxTokens int, tools []*schema.ToolInfo) []llmmodel.Option {
	options := []llmmodel.Option{maxTokensOption(modelName, maxTokens), llmmodel.WithTools(tools)}
	if usesMaxCompletionTokens(modelName) {
		options = append(options, openaimodel.WithRequestPayloadModifier(stripAllowedToolsChoicePayload))
	}
	return options
}

func stripAllowedToolsChoicePayload(_ context.Context, _ []*schema.Message, rawBody []byte) ([]byte, error) {
	var payload map[string]any
	if err := json.Unmarshal(rawBody, &payload); err != nil {
		return rawBody, nil
	}
	choice, ok := payload["tool_choice"].(map[string]any)
	if !ok {
		return rawBody, nil
	}
	if choiceType, _ := choice["type"].(string); choiceType != "allowed_tools" {
		return rawBody, nil
	}
	payload["tool_choice"] = "auto"
	modified, err := json.Marshal(payload)
	if err != nil {
		return rawBody, nil
	}
	return modified, nil
}

func maxTokensOption(modelName string, maxTokens int) llmmodel.Option {
	if usesMaxCompletionTokens(modelName) {
		return openaimodel.WithMaxCompletionTokens(maxTokens)
	}
	return llmmodel.WithMaxTokens(maxTokens)
}

func usesMaxCompletionTokens(modelName string) bool {
	name := strings.ToLower(strings.TrimSpace(modelName))
	return strings.HasPrefix(name, "gpt-5") ||
		strings.HasPrefix(name, "o1") ||
		strings.HasPrefix(name, "o3") ||
		strings.HasPrefix(name, "o4")
}
