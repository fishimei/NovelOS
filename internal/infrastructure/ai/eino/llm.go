package eino

import (
	"context"
	"time"

	"github.com/cloudwego/eino-ext/components/model/openai"
	llmmodel "github.com/cloudwego/eino/components/model"

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
	return openai.NewChatModel(ctx, &openai.ChatModelConfig{
		APIKey:  cfg.APIKey,
		Model:   cfg.Model,
		BaseURL: cfg.BaseURL,
		Timeout: 90 * time.Second,
	})
}
