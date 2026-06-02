package service

import (
	"context"
	"strings"

	"github.com/fishimei/NovelOS/internal/application/model"
	"github.com/fishimei/NovelOS/internal/application/port"
	"github.com/fishimei/NovelOS/internal/domain"
	"github.com/fishimei/NovelOS/internal/pkgerr"
)

type DialogueActionExecutor struct {
	dialogueSessions port.DialogueSessionRepository
	setupStarter     *SetupSessionStarter
	setupAdvancer    *SetupSessionAdvancer
	setupApplier     *SetupRunApplier
	storyStarter     *StorySessionStarter
	storyAdvancer    *StorySessionAdvancer
	storyCutter      *StoryChapterCutter
	storyEventLog    *StoryEventLogService
	audit            port.AuditRepository
	validator        *DialogueActionValidator
	clock            port.Clock
}

func NewDialogueActionExecutor(
	dialogueSessions port.DialogueSessionRepository,
	setupStarter *SetupSessionStarter,
	setupAdvancer *SetupSessionAdvancer,
	setupApplier *SetupRunApplier,
	storyStarter *StorySessionStarter,
	storyAdvancer *StorySessionAdvancer,
	storyCutter *StoryChapterCutter,
	storyEventLog *StoryEventLogService,
	audit port.AuditRepository,
	validator *DialogueActionValidator,
	clock port.Clock,
) *DialogueActionExecutor {
	return &DialogueActionExecutor{
		dialogueSessions: dialogueSessions,
		setupStarter:     setupStarter,
		setupAdvancer:    setupAdvancer,
		setupApplier:     setupApplier,
		storyStarter:     storyStarter,
		storyAdvancer:    storyAdvancer,
		storyCutter:      storyCutter,
		storyEventLog:    storyEventLog,
		audit:            audit,
		validator:        validator,
		clock:            clock,
	}
}

func (e *DialogueActionExecutor) ExecuteConfirmed(ctx context.Context, optionID string, input model.ExecuteDialogueActionInput) (model.DialogueActionOption, error) {
	if !input.Confirm {
		return model.DialogueActionOption{}, pkgerr.Validation("confirm must be true")
	}
	input.ExecutionMode = domain.DialoguePolicyManualConfirm
	option, err := e.dialogueSessions.GetActionOptionByID(ctx, optionID)
	if err != nil {
		return model.DialogueActionOption{}, err
	}
	return e.executeActionOption(ctx, option, input, "dialogue_action_executed")
}

func (e *DialogueActionExecutor) ExecuteAutoApproved(ctx context.Context, optionID string, input model.AutoExecuteDialogueActionInput) (model.DialogueActionOption, error) {
	if strings.TrimSpace(input.PolicyReason) == "" {
		return model.DialogueActionOption{}, pkgerr.Validation("auto policy reason is required")
	}
	option, err := e.dialogueSessions.GetActionOptionByID(ctx, optionID)
	if err != nil {
		return model.DialogueActionOption{}, err
	}
	if option.ConfirmationRequired {
		return model.DialogueActionOption{}, pkgerr.Validation("dialogue action option requires manual confirmation")
	}
	if err := validateAutoApprovedDialogueAction(option); err != nil {
		return model.DialogueActionOption{}, err
	}
	if e.validator != nil {
		if err := e.validator.ValidateAutoApproved(ctx, option); err != nil {
			return model.DialogueActionOption{}, err
		}
	}
	return e.executeActionOption(ctx, option, model.ExecuteDialogueActionInput{
		AuthorNote:    input.AuthorNote,
		ExecutionMode: domain.DialoguePolicyAutoPilot,
		PolicyReason:  input.PolicyReason,
	}, "dialogue_action_auto_executed")
}

func (e *DialogueActionExecutor) executeActionOption(ctx context.Context, option model.DialogueActionOption, input model.ExecuteDialogueActionInput, auditEventName string) (model.DialogueActionOption, error) {
	if option.Status != domain.DialogueActionStatusPending && option.Status != domain.DialogueActionStatusConfirmed {
		return model.DialogueActionOption{}, pkgerr.Conflict(pkgerr.CodeConflict, "dialogue action option is not executable")
	}
	if e.validator != nil {
		if err := e.validator.ValidateOption(ctx, option); err != nil {
			return model.DialogueActionOption{}, err
		}
	}
	executing, err := e.dialogueSessions.TryStartActionExecution(ctx, option.ID)
	if err != nil {
		return model.DialogueActionOption{}, err
	}
	executed, execErr := e.execute(ctx, executing, input)
	if execErr != nil {
		executing.Status = domain.DialogueActionStatusFailed
		executing.Error = execErr.Error()
		failed, updateErr := e.dialogueSessions.UpdateActionOption(ctx, executing)
		if updateErr != nil {
			return model.DialogueActionOption{}, updateErr
		}
		return failed, execErr
	}
	executed.Status = domain.DialogueActionStatusExecuted
	executed.Error = ""
	annotateActionResult(&executed, input)
	updated, err := e.dialogueSessions.UpdateActionOption(ctx, executed)
	if err != nil {
		return model.DialogueActionOption{}, err
	}
	_, _ = e.dialogueSessions.AppendMessage(ctx, updated.SessionID, "tool", "executed option: "+updated.Label, map[string]any{"option_id": updated.ID, "action_type": updated.ActionType, "execution_mode": input.ExecutionMode, "result": updated.Result})
	e.appendAudit(ctx, updated, auditEventName, input)
	return updated, nil
}

func (e *DialogueActionExecutor) Reject(ctx context.Context, optionID string, input model.RejectDialogueActionInput) (model.DialogueActionOption, error) {
	option, err := e.dialogueSessions.GetActionOptionByID(ctx, optionID)
	if err != nil {
		return model.DialogueActionOption{}, err
	}
	if option.Status != domain.DialogueActionStatusPending && option.Status != domain.DialogueActionStatusConfirmed {
		return model.DialogueActionOption{}, pkgerr.Conflict(pkgerr.CodeConflict, "dialogue action option is not rejectable")
	}
	option.Status = domain.DialogueActionStatusRejected
	option.Result = map[string]any{"reason": input.Reason}
	updated, err := e.dialogueSessions.UpdateActionOption(ctx, option)
	if err != nil {
		return model.DialogueActionOption{}, err
	}
	_, _ = e.dialogueSessions.AppendMessage(ctx, updated.SessionID, "tool", "rejected option: "+updated.Label, map[string]any{"option_id": updated.ID, "reason": input.Reason})
	return updated, nil
}

func (e *DialogueActionExecutor) execute(ctx context.Context, option model.DialogueActionOption, input model.ExecuteDialogueActionInput) (model.DialogueActionOption, error) {
	switch option.ActionType {
	case domain.DialogueActionSetupStartAndAdvance:
		session, err := e.setupStarter.Start(ctx, option.ProjectID, model.CreateSetupSessionInput{SeedIdea: firstNonEmpty(stringValue(option.Payload, "seed_idea"), stringValue(option.Payload, "user_message"))})
		if err != nil {
			return option, err
		}
		run, err := e.setupAdvancer.Advance(ctx, session.ID, model.AdvanceSetupSessionInput{UserMessage: firstNonEmpty(stringValue(option.Payload, "user_message"), stringValue(option.Payload, "seed_idea"))})
		if err != nil {
			return option, err
		}
		option.Result = map[string]any{"setup_session_id": session.ID, "setup_run_id": run.RunID, "status": run.Status}
		return option, nil
	case domain.DialogueActionSetupAdvance:
		run, err := e.setupAdvancer.Advance(ctx, stringValue(option.Payload, "setup_session_id"), model.AdvanceSetupSessionInput{UserMessage: stringValue(option.Payload, "user_message")})
		if err != nil {
			return option, err
		}
		option.Result = map[string]any{"setup_run_id": run.RunID, "status": run.Status}
		return option, nil
	case domain.DialogueActionSetupApply:
		result, err := e.setupApplier.Apply(ctx, stringValue(option.Payload, "setup_session_id"), model.ApplySetupRunInput{
			RunID:               stringValue(option.Payload, "setup_run_id"),
			AcceptAuthorBible:   boolValue(option.Payload, "accept_author_bible", true),
			AcceptCharacters:    boolValue(option.Payload, "accept_characters", true),
			AcceptRelationships: boolValue(option.Payload, "accept_relationships", true),
			AcceptWorldState:    boolValue(option.Payload, "accept_world_state", true),
			AuthorNote:          firstNonEmpty(input.AuthorNote, stringValue(option.Payload, "author_note")),
		})
		if err != nil {
			return option, err
		}
		option.Result = map[string]any{"project_id": result.ProjectID, "setup_run_id": result.RunID, "status": result.Status}
		return option, nil
	case domain.DialogueActionStoryCreateAndAdvance:
		session, err := e.storyStarter.Start(ctx, option.ProjectID, model.CreateStorySessionInput{
			Title:            stringValue(option.Payload, "title"),
			OpeningSituation: stringValue(option.Payload, "opening_situation"),
			AuthorIntent:     stringValue(option.Payload, "author_intent"),
		})
		if err != nil {
			return option, err
		}
		run, err := e.storyAdvancer.Advance(ctx, session.ID, model.AdvanceStorySessionInput{AuthorMessage: stringValue(option.Payload, "author_message")})
		if err != nil {
			return option, err
		}
		option.Result = map[string]any{"story_session_id": session.ID, "story_run_id": run.RunID, "status": run.Status}
		return option, nil
	case domain.DialogueActionStoryAdvance:
		run, err := e.storyAdvancer.Advance(ctx, stringValue(option.Payload, "story_session_id"), model.AdvanceStorySessionInput{AuthorMessage: stringValue(option.Payload, "author_message")})
		if err != nil {
			return option, err
		}
		option.Result = map[string]any{"story_run_id": run.RunID, "status": run.Status}
		return option, nil
	case domain.DialogueActionStoryCutChapter:
		cutInput := model.CutChapterInput{
			BranchID:    stringValue(option.Payload, "branch_id"),
			FromEventID: stringValue(option.Payload, "from_event_id"),
			ToEventID:   stringValue(option.Payload, "to_event_id"),
			Title:       stringValue(option.Payload, "title"),
			AuthorNote:  firstNonEmpty(input.AuthorNote, stringValue(option.Payload, "author_note")),
		}
		var result model.CutChapterResult
		var err error
		if stringValue(option.Payload, "span_policy") == "latest_completed" {
			result, err = e.storyCutter.CutLatestCompletedSpan(ctx, stringValue(option.Payload, "story_run_id"), cutInput)
		} else {
			result, err = e.storyCutter.CutChapter(ctx, stringValue(option.Payload, "story_run_id"), cutInput)
		}
		if err != nil {
			return option, err
		}
		option.Result = map[string]any{"chapter_id": result.Chapter.ID, "span_id": result.Span.ID, "story_run_id": result.StoryRun.RunID, "status": result.StoryRun.Status}
		return option, nil
	case domain.DialogueActionStoryForkFromEvent:
		if e.storyEventLog == nil {
			return option, pkgerr.Internal("story event log service is required", nil)
		}
		branch, err := e.storyEventLog.ForkEvent(ctx, stringValue(option.Payload, "event_id"), model.ForkStoryEventInput{
			Name:          firstNonEmpty(stringValue(option.Payload, "name"), "dialogue fork"),
			AuthorMessage: firstNonEmpty(input.AuthorNote, stringValue(option.Payload, "author_message")),
		})
		if err != nil {
			return option, err
		}
		option.Result = map[string]any{
			"branch_id":     branch.ID,
			"project_id":    branch.ProjectID,
			"session_id":    branch.SessionID,
			"base_event_id": branch.BaseEventID,
			"head_event_id": branch.HeadEventID,
			"status":        branch.Status,
		}
		return option, nil
	default:
		return option, pkgerr.Validation("unsupported dialogue action type")
	}
}

func annotateActionResult(option *model.DialogueActionOption, input model.ExecuteDialogueActionInput) {
	if input.ExecutionMode == "" && input.PolicyReason == "" {
		return
	}
	if option.Result == nil {
		option.Result = map[string]any{}
	}
	if input.ExecutionMode != "" {
		option.Result["execution_mode"] = input.ExecutionMode
	}
	if input.PolicyReason != "" {
		option.Result["policy_reason"] = input.PolicyReason
	}
}

func (e *DialogueActionExecutor) appendAudit(ctx context.Context, option model.DialogueActionOption, eventName string, input model.ExecuteDialogueActionInput) {
	if e.audit == nil {
		return
	}
	_, _ = e.audit.AppendRunEvent(ctx, model.RunEvent{
		RunKind:   "dialogue",
		RunID:     option.RunID,
		EventName: eventName,
		Payload: map[string]any{
			"option_id":      option.ID,
			"action_type":    option.ActionType,
			"execution_mode": input.ExecutionMode,
			"policy_reason":  input.PolicyReason,
			"result":         option.Result,
		},
		CreatedAt: currentTime(e.clock),
	})
}
