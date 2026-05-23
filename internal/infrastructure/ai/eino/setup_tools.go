package eino

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"

	"github.com/fishimei/NovelOS/internal/pkgerr"
)

type setupRunState struct {
	mu     sync.Mutex
	output setupAgentOutput
}

type SetSetupAuthorBibleInput struct {
	AuthorBible setupAuthorBibleOutput `json:"author_bible"`
}

type SetSetupWorldStateInput struct {
	WorldState []setupWorldStateOutput `json:"world_state"`
}

type SetSetupCharactersInput struct {
	Characters []setupCharacterOutput `json:"characters"`
}

type SetSetupRelationshipsInput struct {
	Relationships []setupRelationshipOutput `json:"relationships"`
}

type SetSetupVisualDraftInput struct {
	OpenQuestions        []setupQuestionOutput      `json:"open_questions"`
	AssistantSummary     string                     `json:"assistant_summary"`
	VisualDraft          setupVisualDraftOutput     `json:"visual_draft"`
	NextAgentSuggestions []setupNextAgentSuggestion `json:"next_agent_suggestions"`
}

type FinalizeSetupDraftInput struct{}

type setupToolResult struct {
	OK      bool     `json:"ok"`
	Message string   `json:"message"`
	Missing []string `json:"missing,omitempty"`
}

func newSetupTools(state *setupRunState) ([]tool.BaseTool, error) {
	setAuthorBible, err := utils.InferTool("set_setup_author_bible", "提交作者圣经部分：主题、文风、世界规则、审美原则、硬约束、软偏好和禁用套路。只提交 author_bible。", func(ctx context.Context, input SetSetupAuthorBibleInput) (setupToolResult, error) {
		return setSetupAuthorBible(ctx, state, input)
	})
	if err != nil {
		return nil, err
	}
	setWorldState, err := utils.InferTool("set_setup_world_state", "提交世界状态部分。至少 3 项，每项包含 key、value、note、importance、volatility。只提交 world_state。", func(ctx context.Context, input SetSetupWorldStateInput) (setupToolResult, error) {
		return setSetupWorldState(ctx, state, input)
	}, utils.WithUnmarshalArguments(unmarshalSetSetupWorldStateInput))
	if err != nil {
		return nil, err
	}
	setCharacters, err := utils.InferTool("set_setup_characters", "提交主要角色部分。至少 3 个角色，每个角色包含 key、name、role、profile、personality、voice_style、goals、fears、secrets、constraints。只提交 characters。", func(ctx context.Context, input SetSetupCharactersInput) (setupToolResult, error) {
		return setSetupCharacters(ctx, state, input)
	})
	if err != nil {
		return nil, err
	}
	setRelationships, err := utils.InferTool("set_setup_relationships", "提交角色关系部分。至少 2 条关系，character_a_key 和 character_b_key 必须引用 characters 的 key。只提交 relationships。", func(ctx context.Context, input SetSetupRelationshipsInput) (setupToolResult, error) {
		return setSetupRelationships(ctx, state, input)
	})
	if err != nil {
		return nil, err
	}
	setVisualDraft, err := utils.InferTool("set_setup_visual_draft", "提交给用户看的展示草案、待确认问题、摘要和下一步建议。只提交 visual_draft、open_questions、assistant_summary、next_agent_suggestions。", func(ctx context.Context, input SetSetupVisualDraftInput) (setupToolResult, error) {
		return setSetupVisualDraft(ctx, state, input)
	})
	if err != nil {
		return nil, err
	}
	finalizeDraft, err := utils.InferTool("finalize_setup_draft", "确认所有 Setup 草案部分都已提交完毕。只有所有 set_setup_* 工具都成功后才能调用。", func(ctx context.Context, input FinalizeSetupDraftInput) (setupAgentOutput, error) {
		return finalizeSetupDraft(ctx, state, input)
	})
	if err != nil {
		return nil, err
	}
	return []tool.BaseTool{setAuthorBible, setWorldState, setCharacters, setRelationships, setVisualDraft, finalizeDraft}, nil
}

func setSetupAuthorBible(ctx context.Context, state *setupRunState, input SetSetupAuthorBibleInput) (setupToolResult, error) {
	state.mu.Lock()
	defer state.mu.Unlock()
	state.output.AuthorBible = input.AuthorBible
	return setupProgressResult(state.output, "author_bible saved"), nil
}

func setSetupWorldState(ctx context.Context, state *setupRunState, input SetSetupWorldStateInput) (setupToolResult, error) {
	state.mu.Lock()
	defer state.mu.Unlock()
	state.output.WorldState = input.WorldState
	return setupProgressResult(state.output, "world_state saved"), nil
}

func unmarshalSetSetupWorldStateInput(ctx context.Context, arguments string) (any, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal([]byte(arguments), &raw); err == nil {
		if worldState, ok := raw["world_state"]; ok {
			entries, err := unmarshalSetupWorldStateEntries(worldState)
			if err != nil {
				return nil, err
			}
			return SetSetupWorldStateInput{WorldState: entries}, nil
		}
	}
	entries, err := unmarshalSetupWorldStateEntries(json.RawMessage(arguments))
	if err != nil {
		return nil, err
	}
	return SetSetupWorldStateInput{WorldState: entries}, nil
}

func unmarshalSetupWorldStateEntries(raw json.RawMessage) ([]setupWorldStateOutput, error) {
	var items []json.RawMessage
	if err := json.Unmarshal(raw, &items); err == nil {
		entries := make([]setupWorldStateOutput, 0, len(items))
		for idx, item := range items {
			entry, ok, err := unmarshalSetupWorldStateEntry(item, idx)
			if err != nil {
				return nil, err
			}
			if ok {
				entries = append(entries, entry)
			}
		}
		return entries, nil
	}
	entry, ok, err := unmarshalSetupWorldStateEntry(raw, 0)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("world_state is required")
	}
	return []setupWorldStateOutput{entry}, nil
}

func unmarshalSetupWorldStateEntry(raw json.RawMessage, idx int) (setupWorldStateOutput, bool, error) {
	var encoded string
	if err := json.Unmarshal(raw, &encoded); err == nil {
		text := strings.TrimSpace(encoded)
		if text == "" || isSetupWorldStateSchemaToken(text) {
			return setupWorldStateOutput{}, false, nil
		}
		if strings.HasPrefix(text, "{") || strings.HasPrefix(text, "[") {
			entries, err := unmarshalSetupWorldStateEntries(json.RawMessage(text))
			if err != nil {
				return setupWorldStateOutput{}, false, err
			}
			if len(entries) == 0 {
				return setupWorldStateOutput{}, false, nil
			}
			return entries[0], true, nil
		}
		return setupWorldStateOutput{Key: fmt.Sprintf("world_state_%02d", idx+1), Value: text, Note: text, Importance: 3, Volatility: 3}, true, nil
	}
	var entry setupWorldStateOutput
	if err := json.Unmarshal(raw, &entry); err != nil {
		return setupWorldStateOutput{}, false, err
	}
	if strings.TrimSpace(entry.Key) == "" && entry.Value == nil && strings.TrimSpace(entry.Note) == "" {
		return setupWorldStateOutput{}, false, nil
	}
	if strings.TrimSpace(entry.Key) == "" {
		entry.Key = fmt.Sprintf("world_state_%02d", idx+1)
	}
	return entry, true, nil
}

func isSetupWorldStateSchemaToken(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "key", "value", "note", "importance", "volatility", "world_state":
		return true
	default:
		return false
	}
}

func setSetupCharacters(ctx context.Context, state *setupRunState, input SetSetupCharactersInput) (setupToolResult, error) {
	state.mu.Lock()
	defer state.mu.Unlock()
	state.output.Characters = input.Characters
	return setupProgressResult(state.output, "characters saved"), nil
}

func setSetupRelationships(ctx context.Context, state *setupRunState, input SetSetupRelationshipsInput) (setupToolResult, error) {
	state.mu.Lock()
	defer state.mu.Unlock()
	state.output.Relationships = input.Relationships
	return setupProgressResult(state.output, "relationships saved"), nil
}

func setSetupVisualDraft(ctx context.Context, state *setupRunState, input SetSetupVisualDraftInput) (setupToolResult, error) {
	state.mu.Lock()
	defer state.mu.Unlock()
	state.output.OpenQuestions = input.OpenQuestions
	state.output.AssistantSummary = input.AssistantSummary
	state.output.VisualDraft = input.VisualDraft
	state.output.NextAgentSuggestions = firstSuggestions(input.NextAgentSuggestions, input.VisualDraft.NextAgentSuggestions)
	if len(state.output.VisualDraft.OpenQuestions) == 0 {
		state.output.VisualDraft.OpenQuestions = input.OpenQuestions
	}
	if len(state.output.VisualDraft.NextAgentSuggestions) == 0 {
		state.output.VisualDraft.NextAgentSuggestions = state.output.NextAgentSuggestions
	}
	return setupProgressResult(state.output, "visual_draft saved"), nil
}

func finalizeSetupDraft(ctx context.Context, state *setupRunState, input FinalizeSetupDraftInput) (setupAgentOutput, error) {
	state.mu.Lock()
	defer state.mu.Unlock()
	missing := setupDraftMissing(state.output)
	if len(missing) > 0 {
		return setupAgentOutput{}, pkgerr.Validation("setup draft missing: " + strings.Join(missing, ", "))
	}
	if len(state.output.VisualDraft.NextAgentSuggestions) == 0 {
		state.output.VisualDraft.NextAgentSuggestions = state.output.NextAgentSuggestions
	}
	return state.output, nil
}

func setupProgressResult(out setupAgentOutput, message string) setupToolResult {
	return setupToolResult{OK: true, Message: message, Missing: setupDraftMissing(out)}
}

func setupDraftMissing(out setupAgentOutput) []string {
	missing := make([]string, 0)
	if strings.TrimSpace(out.AuthorBible.Theme) == "" || strings.TrimSpace(out.AuthorBible.StyleGuide) == "" {
		missing = append(missing, "author_bible")
	}
	if len(out.WorldState) < 3 {
		missing = append(missing, "world_state")
	}
	if len(out.Characters) < 3 {
		missing = append(missing, "characters")
	}
	if len(out.Relationships) < 2 {
		missing = append(missing, "relationships")
	}
	if strings.TrimSpace(out.AssistantSummary) == "" {
		missing = append(missing, "assistant_summary")
	}
	if strings.TrimSpace(out.VisualDraft.Logline) == "" && strings.TrimSpace(out.VisualDraft.AgentSummary) == "" {
		missing = append(missing, "visual_draft")
	}
	return missing
}

func (s *setupRunState) agentOutput() setupAgentOutput {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.output
}
