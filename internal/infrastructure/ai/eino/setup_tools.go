package eino

import (
	"context"
	"sync"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"

	"github.com/fishimei/NovelOS/internal/pkgerr"
)

type setupRunState struct {
	mu     sync.Mutex
	output setupAgentOutput
}

type ShowSetupDraftInput struct {
	AuthorBible          setupAuthorBibleOutput     `json:"author_bible"`
	WorldState           []setupWorldStateOutput    `json:"world_state"`
	Characters           []setupCharacterOutput     `json:"characters"`
	Relationships        []setupRelationshipOutput  `json:"relationships"`
	OpenQuestions        []setupQuestionOutput      `json:"open_questions"`
	AssistantSummary     string                     `json:"assistant_summary"`
	VisualDraft          setupVisualDraftOutput     `json:"visual_draft"`
	NextAgentSuggestions []setupNextAgentSuggestion `json:"next_agent_suggestions"`
}

type ReviseSetupDraftInput struct {
	RevisionIntent       string                     `json:"revision_intent"`
	RevisionSummary      string                     `json:"revision_summary"`
	AuthorBible          setupAuthorBibleOutput     `json:"author_bible"`
	WorldState           []setupWorldStateOutput    `json:"world_state"`
	Characters           []setupCharacterOutput     `json:"characters"`
	Relationships        []setupRelationshipOutput  `json:"relationships"`
	OpenQuestions        []setupQuestionOutput      `json:"open_questions"`
	AssistantSummary     string                     `json:"assistant_summary"`
	VisualDraft          setupVisualDraftOutput     `json:"visual_draft"`
	NextAgentSuggestions []setupNextAgentSuggestion `json:"next_agent_suggestions"`
}

type HandoffNextAgentInput struct {
	NextAgentSuggestions []setupNextAgentSuggestion `json:"next_agent_suggestions"`
}

func newSetupTools(state *setupRunState) ([]tool.BaseTool, error) {
	showDraft, err := utils.InferTool("show_setup_draft", "把主控 agent 已经在内部深化完成的完整设定草案交给后端展示给用户。必须包含作者圣经、世界状态、角色、关系和可视化草案。", func(ctx context.Context, input ShowSetupDraftInput) (setupAgentOutput, error) {
		return showSetupDraft(ctx, state, input)
	})
	if err != nil {
		return nil, err
	}
	reviseDraft, err := utils.InferTool("revise_setup_draft", "根据用户微调或重起草要求，提交一版已经重新深化过的完整设定草案给用户查看。", func(ctx context.Context, input ReviseSetupDraftInput) (setupAgentOutput, error) {
		return reviseSetupDraft(ctx, state, input)
	})
	if err != nil {
		return nil, err
	}
	handoffAgent, err := utils.InferTool("handoff_next_agent", "记录确认草案后建议进入的下一个 agent，例如角色深化、关系深化或第一章故事编排。它不写库，只作为展示建议。", func(ctx context.Context, input HandoffNextAgentInput) ([]setupNextAgentSuggestion, error) {
		return handoffNextAgent(ctx, state, input)
	})
	if err != nil {
		return nil, err
	}
	return []tool.BaseTool{showDraft, reviseDraft, handoffAgent}, nil
}

func showSetupDraft(ctx context.Context, state *setupRunState, input ShowSetupDraftInput) (setupAgentOutput, error) {
	out := setupAgentOutput{
		AuthorBible:          input.AuthorBible,
		WorldState:           input.WorldState,
		Characters:           input.Characters,
		Relationships:        input.Relationships,
		OpenQuestions:        input.OpenQuestions,
		AssistantSummary:     input.AssistantSummary,
		VisualDraft:          input.VisualDraft,
		NextAgentSuggestions: input.NextAgentSuggestions,
	}
	return saveSetupDraftOutput(state, out)
}

func reviseSetupDraft(ctx context.Context, state *setupRunState, input ReviseSetupDraftInput) (setupAgentOutput, error) {
	summary := firstText(input.AssistantSummary, input.RevisionSummary)
	out := setupAgentOutput{
		AuthorBible:          input.AuthorBible,
		WorldState:           input.WorldState,
		Characters:           input.Characters,
		Relationships:        input.Relationships,
		OpenQuestions:        input.OpenQuestions,
		AssistantSummary:     summary,
		VisualDraft:          input.VisualDraft,
		NextAgentSuggestions: input.NextAgentSuggestions,
	}
	return saveSetupDraftOutput(state, out)
}

func handoffNextAgent(ctx context.Context, state *setupRunState, input HandoffNextAgentInput) ([]setupNextAgentSuggestion, error) {
	state.mu.Lock()
	defer state.mu.Unlock()
	state.output.NextAgentSuggestions = input.NextAgentSuggestions
	state.output.VisualDraft.NextAgentSuggestions = input.NextAgentSuggestions
	return input.NextAgentSuggestions, nil
}

func saveSetupDraftOutput(state *setupRunState, out setupAgentOutput) (setupAgentOutput, error) {
	if len(out.Characters) == 0 {
		return setupAgentOutput{}, pkgerr.Validation("characters are required")
	}
	state.mu.Lock()
	defer state.mu.Unlock()
	if len(out.NextAgentSuggestions) == 0 {
		out.NextAgentSuggestions = state.output.NextAgentSuggestions
	}
	if len(out.VisualDraft.NextAgentSuggestions) == 0 {
		out.VisualDraft.NextAgentSuggestions = out.NextAgentSuggestions
	}
	state.output = out
	return state.output, nil
}

func (s *setupRunState) agentOutput() setupAgentOutput {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.output
}
