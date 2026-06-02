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
	ActionExecutor   port.DialogueActionExecutor
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
		maxSteps = config.DefaultDialogueAgentMaxSteps
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
			autoPilot:        deps.Config.DialogueAgent.AutoPilot,
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
	basePrompt := g.prompt
	if strings.TrimSpace(basePrompt) == "" {
		basePrompt = "You are the NovelOS dialogue agent. Always call load_dialogue_context first, create action options through propose tools, and finish with finalize_dialogue_response."
	}
	return fmt.Sprintf(`%s

Execution mode: %s.
manual_confirm: create pending options for every state-changing action and wait for user confirmation.
auto_pilot: only auto-execute policy-approved low-risk actions: story_advance and cut latest completed story span. setup_apply, fork, rollback, delete, overwrite, and publish-frontier movement still require manual confirmation.
When cutting the latest completed span, use propose_story_cut_latest_completed_span or auto_cut_latest_completed_span; never guess branch_id/from_event_id/to_event_id.`, basePrompt, dialogueExecutionMode(g.deps.autoPilot))
}

func (g *DialogueRunGenerator) userPrompt(input port.DialogueRunGenerationInput) string {
	messages, _ := json.Marshal(input.Session.Messages)
	payload, _ := json.Marshal(map[string]any{
		"project_id":           input.Run.ProjectID,
		"dialogue_session_id":  input.Session.ID,
		"dialogue_run_id":      input.Run.RunID,
		"execution_mode":       dialogueExecutionMode(g.deps.autoPilot),
		"auto_allowed_actions": autoAllowedDialogueActions(g.deps.autoPilot),
		"last_user_message":    input.Session.LastUserMessage,
		"messages":             json.RawMessage(messages),
	})
	return fmt.Sprintf(`%s

请根据用户最后一条消息工作：先调用 load_dialogue_context；需要变更状态时提出待确认 option；如果只是询问状态或闲聊，则直接 finalize_dialogue_response。`, string(payload))
}
