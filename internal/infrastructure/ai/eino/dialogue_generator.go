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
)

type DialogueRunGeneratorDeps struct {
	Config           config.AIConfig
	Projects         port.ProjectRepository
	AuthorBibles     port.AuthorBibleRepository
	WorldState       port.WorldStateRepository
	Characters       port.CharacterRepository
	Relationships    port.RelationshipRepository
	SetupSessions    port.SetupSessionRepository
	StorySessions    port.StorySessionRepository
	Chapters         port.ChapterRepository
	DialogueSessions port.DialogueSessionRepository
	ActionExecutor   port.DialogueConfirmedActionExecutor
	OptionValidator  port.DialogueActionOptionValidator
	Events           port.GenerationEventStream
	Clock            port.Clock
	IDs              port.IDGenerator
}

type DialogueRunGenerator struct {
	model    llmmodel.ToolCallingChatModel
	deps     dialogueGeneratorDeps
	prompt   string
	maxSteps int
}

func NewDialogueRunGenerator(ctx context.Context, deps DialogueRunGeneratorDeps) (*DialogueRunGenerator, error) {
	chatModel, err := newOpenAIChatModel(ctx, deps.Config)
	if err != nil {
		return nil, err
	}
	maxSteps := deps.Config.DialogueAgent.MaxSteps
	if maxSteps <= 0 {
		maxSteps = 16
	}
	if maxSteps > 24 {
		maxSteps = 24
	}
	return &DialogueRunGenerator{
		model: chatModel,
		deps: dialogueGeneratorDeps{
			projects:         deps.Projects,
			authorBibles:     deps.AuthorBibles,
			worldState:       deps.WorldState,
			characters:       deps.Characters,
			relationships:    deps.Relationships,
			setupSessions:    deps.SetupSessions,
			storySessions:    deps.StorySessions,
			chapters:         deps.Chapters,
			dialogueSessions: deps.DialogueSessions,
			actionExecutor:   deps.ActionExecutor,
			optionValidator:  deps.OptionValidator,
			events:           deps.Events,
			clock:            deps.Clock,
			ids:              deps.IDs,
		},
		prompt:   deps.Config.DialogueAgent.Prompt,
		maxSteps: maxSteps,
	}, nil
}

func (g *DialogueRunGenerator) Generate(ctx context.Context, input port.DialogueRunGenerationInput) (model.DialogueRunResult, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	state := &dialogueRunState{run: input.Run, session: input.Session}
	tools, err := newDialogueTools(g.deps, state)
	if err != nil {
		return model.DialogueRunResult{}, err
	}
	agent, err := react.NewAgent(ctx, &react.AgentConfig{
		ToolCallingModel: g.model,
		ToolsConfig:      compose.ToolsNodeConfig{Tools: tools},
		MaxStep:          g.maxSteps,
		ToolReturnDirectly: map[string]struct{}{
			"finalize_dialogue_response": {},
		},
	})
	if err != nil {
		return model.DialogueRunResult{}, err
	}
	_, err = agent.Generate(ctx, []*schema.Message{
		schema.SystemMessage(g.systemPrompt()),
		schema.UserMessage(g.userPrompt(input)),
	})
	if err != nil {
		return model.DialogueRunResult{}, fmt.Errorf("run dialogue agent: %w", err)
	}
	result := state.result()
	if strings.TrimSpace(result.AssistantMessage) == "" {
		result.AssistantMessage = "我已经理解了你的想法，但还需要你补充一点目标，才能继续提出下一步。"
	}
	return result, nil
}

func (g *DialogueRunGenerator) systemPrompt() string {
	return firstText(g.prompt, `你是 NovelOS 的统一对话 Agent。每轮必须先调用 load_dialogue_context。你通过 propose_* 工具创建待确认选项，通过 finalize_dialogue_response 回复用户。用户未明确确认前，不能调用 execute_confirmed_action。`)
}

func (g *DialogueRunGenerator) userPrompt(input port.DialogueRunGenerationInput) string {
	messages, _ := json.Marshal(input.Session.Messages)
	payload, _ := json.Marshal(map[string]any{
		"project_id":          input.Run.ProjectID,
		"dialogue_session_id": input.Session.ID,
		"dialogue_run_id":     input.Run.RunID,
		"last_user_message":   input.Session.LastUserMessage,
		"messages":            json.RawMessage(messages),
	})
	return fmt.Sprintf(`%s

请根据用户最后一条消息工作：先调用 load_dialogue_context；需要变更状态时提出待确认 option；如果只是询问状态或闲聊，则直接 finalize_dialogue_response。`, string(payload))
}
