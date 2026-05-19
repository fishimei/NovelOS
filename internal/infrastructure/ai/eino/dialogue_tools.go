package eino

import (
	"context"
	"sync"
	"time"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"

	"github.com/fishimei/NovelOS/internal/application/model"
	"github.com/fishimei/NovelOS/internal/application/port"
	"github.com/fishimei/NovelOS/internal/domain"
	"github.com/fishimei/NovelOS/internal/pkgerr"
)

type dialogueGeneratorDeps struct {
	projects         port.ProjectRepository
	authorBibles     port.AuthorBibleRepository
	worldState       port.WorldStateRepository
	characters       port.CharacterRepository
	relationships    port.RelationshipRepository
	setupSessions    port.SetupSessionRepository
	storySessions    port.StorySessionRepository
	chapters         port.ChapterRepository
	dialogueSessions port.DialogueSessionRepository
	actionExecutor   port.DialogueConfirmedActionExecutor
	optionValidator  port.DialogueActionOptionValidator
	events           port.GenerationEventStream
	clock            port.Clock
	ids              port.IDGenerator
}

type dialogueRunState struct {
	mu              sync.Mutex
	run             model.DialogueRun
	session         model.DialogueSession
	context         *DialogueContextSnapshot
	proposedOptions []model.DialogueActionOption
	finalResponse   FinalizeDialogueResponseInput
	toolTrace       []model.DialogueToolTrace
}

func newDialogueTools(deps dialogueGeneratorDeps, state *dialogueRunState) ([]tool.BaseTool, error) {
	loadContext, err := utils.InferTool("load_dialogue_context", "读取统一对话入口需要的项目状态、setup/story 会话、最近章节和待确认选项。每轮必须先调用。", func(ctx context.Context, input DialogueContextInput) (DialogueContextSnapshot, error) {
		return loadDialogueContext(ctx, deps, state, input)
	})
	if err != nil {
		return nil, err
	}
	inspectSetup, err := utils.InferTool("inspect_setup_run_result", "读取 setup run result 摘要，用于判断是否可提出 setup.apply 选项。", func(ctx context.Context, input InspectSetupRunResultInput) (SetupRunResultInspection, error) {
		return inspectSetupRunResult(ctx, deps, state, input)
	})
	if err != nil {
		return nil, err
	}
	inspectStory, err := utils.InferTool("inspect_story_run_result", "读取 story run result 摘要、draft_id 和 memory_patch_id，用于提出 story.commit 选项。", func(ctx context.Context, input InspectStoryRunResultInput) (StoryRunResultInspection, error) {
		return inspectStoryRunResult(ctx, deps, state, input)
	})
	if err != nil {
		return nil, err
	}
	listPending, err := utils.InferTool("list_pending_action_options", "列出当前对话会话中等待确认或已确认的 action options。", func(ctx context.Context, input ListPendingDialogueOptionsInput) ([]model.DialogueActionOption, error) {
		return listPendingDialogueOptions(ctx, deps, state, input)
	})
	if err != nil {
		return nil, err
	}
	proposeSetupStart, err := utils.InferTool("propose_setup_start_and_advance", "提出创建 setup session 并推进 setup agent 的待确认选项。只创建 option，不执行。", func(ctx context.Context, input ProposeSetupStartAndAdvanceInput) (dialogueToolResult, error) {
		return proposeDialogueAction(ctx, deps, state, domain.DialogueActionSetupStartAndAdvance, input.Label, input.Description, input.Rationale, map[string]any{"seed_idea": input.SeedIdea, "user_message": input.UserMessage})
	})
	if err != nil {
		return nil, err
	}
	proposeSetupAdvance, err := utils.InferTool("propose_setup_advance", "提出推进已有 setup session 的待确认选项。只创建 option，不执行。", func(ctx context.Context, input ProposeSetupAdvanceInput) (dialogueToolResult, error) {
		return proposeDialogueAction(ctx, deps, state, domain.DialogueActionSetupAdvance, input.Label, input.Description, input.Rationale, map[string]any{"setup_session_id": input.SetupSessionID, "user_message": input.UserMessage})
	})
	if err != nil {
		return nil, err
	}
	proposeSetupApply, err := utils.InferTool("propose_setup_apply", "提出应用 setup 草案的待确认选项。只创建 option，不执行。", func(ctx context.Context, input ProposeSetupApplyInput) (dialogueToolResult, error) {
		return proposeDialogueAction(ctx, deps, state, domain.DialogueActionSetupApply, input.Label, input.Description, input.Rationale, map[string]any{"setup_session_id": input.SetupSessionID, "setup_run_id": input.SetupRunID, "accept_author_bible": input.AcceptAuthorBible, "accept_characters": input.AcceptCharacters, "accept_relationships": input.AcceptRelationships, "accept_world_state": input.AcceptWorldState, "author_note": input.AuthorNote})
	})
	if err != nil {
		return nil, err
	}
	proposeStoryCreate, err := utils.InferTool("propose_story_create_and_advance", "提出创建 story session 并推进剧情编排的待确认选项。只创建 option，不执行。", func(ctx context.Context, input ProposeStoryCreateAndAdvanceInput) (dialogueToolResult, error) {
		return proposeDialogueAction(ctx, deps, state, domain.DialogueActionStoryCreateAndAdvance, input.Label, input.Description, input.Rationale, map[string]any{"title": input.Title, "opening_situation": input.OpeningSituation, "author_intent": input.AuthorIntent, "author_message": input.AuthorMessage})
	})
	if err != nil {
		return nil, err
	}
	proposeStoryAdvance, err := utils.InferTool("propose_story_advance", "提出推进已有 story session 的待确认选项。只创建 option，不执行。", func(ctx context.Context, input ProposeStoryAdvanceInput) (dialogueToolResult, error) {
		return proposeDialogueAction(ctx, deps, state, domain.DialogueActionStoryAdvance, input.Label, input.Description, input.Rationale, map[string]any{"story_session_id": input.StorySessionID, "author_message": input.AuthorMessage})
	})
	if err != nil {
		return nil, err
	}
	proposeStoryCommit, err := utils.InferTool("propose_story_commit", "提出提交 story run 草稿的待确认选项。只创建 option，不执行。", func(ctx context.Context, input ProposeStoryCommitInput) (dialogueToolResult, error) {
		return proposeDialogueAction(ctx, deps, state, domain.DialogueActionStoryCommit, input.Label, input.Description, input.Rationale, map[string]any{"story_run_id": input.StoryRunID, "draft_id": input.DraftID, "memory_patch_id": input.MemoryPatchID, "author_note": input.AuthorNote})
	})
	if err != nil {
		return nil, err
	}
	executeConfirmed, err := utils.InferTool("execute_confirmed_action", "执行已经被用户确认的 action option。只接受 option_id；不会接受模型传入的业务 payload。", func(ctx context.Context, input ExecuteConfirmedDialogueActionInput) (model.DialogueActionOption, error) {
		return executeConfirmedDialogueAction(ctx, deps, state, input)
	})
	if err != nil {
		return nil, err
	}
	finalizeResponse, err := utils.InferTool("finalize_dialogue_response", "提交本轮对话回复、澄清问题、建议回复和上下文摘要。结束本轮时必须调用。", func(ctx context.Context, input FinalizeDialogueResponseInput) (FinalizeDialogueResponseInput, error) {
		return finalizeDialogueResponse(ctx, state, input)
	})
	if err != nil {
		return nil, err
	}
	return []tool.BaseTool{loadContext, inspectSetup, inspectStory, listPending, proposeSetupStart, proposeSetupAdvance, proposeSetupApply, proposeStoryCreate, proposeStoryAdvance, proposeStoryCommit, executeConfirmed, finalizeResponse}, nil
}

func loadDialogueContext(ctx context.Context, deps dialogueGeneratorDeps, state *dialogueRunState, input DialogueContextInput) (DialogueContextSnapshot, error) {
	projectID := firstText(input.ProjectID, state.run.ProjectID, state.session.ProjectID)
	sessionID := firstText(input.SessionID, state.session.ID)
	snapshot := DialogueContextSnapshot{}
	project, err := deps.projects.GetDetail(ctx, projectID)
	if err != nil {
		return snapshot, err
	}
	snapshot.Project = project
	if bible, err := deps.authorBibles.GetByProjectID(ctx, projectID); err == nil {
		snapshot.HasAuthorBible = true
		snapshot.AuthorBibleSummary = firstText(bible.Theme, bible.StyleGuide)
	} else if !isNotFound(err) {
		return snapshot, err
	}
	worldState, err := deps.worldState.ListByProjectID(ctx, projectID)
	if err != nil {
		return snapshot, err
	}
	snapshot.WorldStateCount = len(worldState)
	characters, err := deps.characters.ListByProjectID(ctx, projectID, model.PageInput{Page: 1, PageSize: 50})
	if err != nil {
		return snapshot, err
	}
	snapshot.CharacterCount = len(characters.Items)
	relationships, err := deps.relationships.ListByProjectID(ctx, projectID, model.PageInput{Page: 1, PageSize: 50})
	if err != nil {
		return snapshot, err
	}
	snapshot.RelationshipCount = len(relationships.Items)
	chapters, err := deps.chapters.ListByProjectID(ctx, projectID, model.PageInput{Page: 1, PageSize: 5})
	if err == nil {
		snapshot.RecentChapters = chapters.Items
	} else if !isNotFound(err) {
		return snapshot, err
	}
	setupSessions, err := deps.setupSessions.ListSessionsByProjectID(ctx, projectID, model.PageInput{Page: 1, PageSize: 5})
	if err != nil {
		return snapshot, err
	}
	snapshot.SetupSessions = setupSessions.Items
	storySessions, err := deps.storySessions.ListSessionsByProjectID(ctx, projectID, model.PageInput{Page: 1, PageSize: 5})
	if err != nil {
		return snapshot, err
	}
	snapshot.StorySessions = storySessions.Items
	pending, err := deps.dialogueSessions.ListPendingActionOptionsBySessionID(ctx, sessionID)
	if err != nil {
		return snapshot, err
	}
	snapshot.PendingOptions = pending
	state.mu.Lock()
	state.context = &snapshot
	state.addTraceLocked("load_dialogue_context", "读取项目上下文", true)
	state.mu.Unlock()
	return snapshot, nil
}

func inspectSetupRunResult(ctx context.Context, deps dialogueGeneratorDeps, state *dialogueRunState, input InspectSetupRunResultInput) (SetupRunResultInspection, error) {
	run, err := deps.setupSessions.GetRunByID(ctx, input.RunID)
	if err != nil {
		return SetupRunResultInspection{}, err
	}
	result, err := deps.setupSessions.GetRunResultByID(ctx, input.RunID)
	if err != nil {
		return SetupRunResultInspection{}, err
	}
	inspection := SetupRunResultInspection{
		RunID:                      run.RunID,
		SessionID:                  run.SessionID,
		ProjectID:                  run.ProjectID,
		Status:                     run.Status,
		AssistantSummary:           result.SetupDraft.AssistantSummary,
		CharacterCount:             len(result.SetupDraft.Characters),
		RelationshipCount:          len(result.SetupDraft.Relationships),
		WorldStateCount:            len(result.SetupDraft.WorldState),
		AcceptAuthorBibleDefault:   true,
		AcceptCharactersDefault:    true,
		AcceptRelationshipsDefault: true,
		AcceptWorldStateDefault:    true,
	}
	state.mu.Lock()
	state.addTraceLocked("inspect_setup_run_result", "读取 setup run result", true)
	state.mu.Unlock()
	return inspection, nil
}

func inspectStoryRunResult(ctx context.Context, deps dialogueGeneratorDeps, state *dialogueRunState, input InspectStoryRunResultInput) (StoryRunResultInspection, error) {
	run, err := deps.storySessions.GetRunByID(ctx, input.RunID)
	if err != nil {
		return StoryRunResultInspection{}, err
	}
	result, err := deps.storySessions.GetRunResultByID(ctx, input.RunID)
	if err != nil {
		return StoryRunResultInspection{}, err
	}
	inspection := StoryRunResultInspection{
		RunID:         run.RunID,
		SessionID:     run.SessionID,
		ProjectID:     run.ProjectID,
		Status:        run.Status,
		DraftID:       result.Draft.ID,
		MemoryPatchID: result.MemoryPatch.ID,
		Title:         result.Draft.Title,
		Summary:       result.Draft.Summary,
		WordCount:     result.Draft.WordCount,
		Committed:     run.Status == domain.RunStatusCommitted || run.CommittedAt != nil,
	}
	state.mu.Lock()
	state.addTraceLocked("inspect_story_run_result", "读取 story run result", true)
	state.mu.Unlock()
	return inspection, nil
}

func listPendingDialogueOptions(ctx context.Context, deps dialogueGeneratorDeps, state *dialogueRunState, input ListPendingDialogueOptionsInput) ([]model.DialogueActionOption, error) {
	sessionID := firstText(input.SessionID, state.session.ID)
	options, err := deps.dialogueSessions.ListPendingActionOptionsBySessionID(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	state.mu.Lock()
	state.addTraceLocked("list_pending_action_options", "读取待确认选项", true)
	state.mu.Unlock()
	return options, nil
}

func proposeDialogueAction(ctx context.Context, deps dialogueGeneratorDeps, state *dialogueRunState, actionType string, label string, description string, rationale string, payload map[string]any) (dialogueToolResult, error) {
	option := model.DialogueActionOption{
		ID:                   deps.ids.New("dopt"),
		SessionID:            state.session.ID,
		RunID:                state.run.RunID,
		ProjectID:            state.run.ProjectID,
		ActionType:           actionType,
		Label:                firstText(label, defaultDialogueActionLabel(actionType)),
		Description:          description,
		Rationale:            rationale,
		ConfirmationRequired: true,
		Payload:              payload,
		Status:               domain.DialogueActionStatusPending,
		CreatedAt:            currentTimeFromPort(deps.clock),
		UpdatedAt:            currentTimeFromPort(deps.clock),
	}
	if deps.optionValidator != nil {
		if err := deps.optionValidator.ValidateOption(ctx, option); err != nil {
			return dialogueToolResult{}, err
		}
	}
	if err := deps.dialogueSessions.SaveActionOptions(ctx, []model.DialogueActionOption{option}); err != nil {
		return dialogueToolResult{}, err
	}
	state.mu.Lock()
	state.proposedOptions = append(state.proposedOptions, option)
	state.addTraceLocked("propose_"+actionType, "创建待确认选项", true)
	state.mu.Unlock()
	return dialogueToolResult{OK: true, Message: "option proposed", Option: option}, nil
}

func executeConfirmedDialogueAction(ctx context.Context, deps dialogueGeneratorDeps, state *dialogueRunState, input ExecuteConfirmedDialogueActionInput) (model.DialogueActionOption, error) {
	if deps.actionExecutor == nil {
		return model.DialogueActionOption{}, pkgerr.Internal("dialogue action executor is required", nil)
	}
	option, err := deps.dialogueSessions.GetActionOptionByID(ctx, input.OptionID)
	if err != nil {
		return model.DialogueActionOption{}, err
	}
	if option.Status != domain.DialogueActionStatusConfirmed {
		return model.DialogueActionOption{}, pkgerr.Validation("dialogue action option must be confirmed before execution")
	}
	result, err := deps.actionExecutor.ExecuteConfirmed(ctx, input.OptionID, model.ExecuteDialogueActionInput{Confirm: true})
	state.mu.Lock()
	state.addTraceLocked("execute_confirmed_action", "执行已确认选项", err == nil)
	state.mu.Unlock()
	return result, err
}

func finalizeDialogueResponse(ctx context.Context, state *dialogueRunState, input FinalizeDialogueResponseInput) (FinalizeDialogueResponseInput, error) {
	state.mu.Lock()
	defer state.mu.Unlock()
	state.finalResponse = input
	state.addTraceLocked("finalize_dialogue_response", "提交对话回复", true)
	return input, nil
}

func (s *dialogueRunState) result() model.DialogueRunResult {
	s.mu.Lock()
	defer s.mu.Unlock()
	options := make([]model.DialogueActionOption, len(s.proposedOptions))
	copy(options, s.proposedOptions)
	trace := make([]model.DialogueToolTrace, len(s.toolTrace))
	copy(trace, s.toolTrace)
	status := domain.RunStatusCompleted
	if len(options) > 0 {
		status = domain.RunStatusReviewRequired
	}
	return model.DialogueRunResult{
		RunID:               s.run.RunID,
		SessionID:           s.session.ID,
		Status:              status,
		AssistantMessage:    s.finalResponse.AssistantMessage,
		ActionOptions:       options,
		ClarifyingQuestions: s.finalResponse.ClarifyingQuestions,
		SuggestedReplies:    s.finalResponse.SuggestedReplies,
		ContextSummary:      s.finalResponse.ContextSummary,
		ToolTrace:           trace,
	}
}

func (s *dialogueRunState) addTraceLocked(toolName string, summary string, ok bool) {
	s.toolTrace = append(s.toolTrace, model.DialogueToolTrace{ToolName: toolName, Summary: summary, OK: ok, CreatedAt: time.Now().UTC()})
}

func defaultDialogueActionLabel(actionType string) string {
	switch actionType {
	case domain.DialogueActionSetupStartAndAdvance:
		return "生成项目设定草案"
	case domain.DialogueActionSetupAdvance:
		return "继续完善设定草案"
	case domain.DialogueActionSetupApply:
		return "应用设定草案"
	case domain.DialogueActionStoryCreateAndAdvance:
		return "开始剧情编排"
	case domain.DialogueActionStoryAdvance:
		return "继续剧情编排"
	case domain.DialogueActionStoryCommit:
		return "提交章节草稿"
	default:
		return "执行下一步"
	}
}

func currentTimeFromPort(clock port.Clock) time.Time {
	if clock != nil {
		return clock.Now().UTC()
	}
	return time.Now().UTC()
}
