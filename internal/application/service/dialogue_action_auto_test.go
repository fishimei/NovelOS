package service

import (
	"context"
	"errors"
	"testing"

	"github.com/fishimei/NovelOS/internal/application/model"
	"github.com/fishimei/NovelOS/internal/domain"
)

func TestDialogueActionValidatorAutoPolicyAllowsOnlyLowRiskActions(t *testing.T) {
	validator := NewDialogueActionValidator(nil, nil, nil)

	if err := validator.ValidateAutoApproved(context.Background(), model.DialogueActionOption{
		ActionType:           domain.DialogueActionStoryAdvance,
		ConfirmationRequired: false,
	}); err != nil {
		t.Fatalf("story advance should be auto-approved: %v", err)
	}

	if err := validator.ValidateAutoApproved(context.Background(), model.DialogueActionOption{
		ActionType:           domain.DialogueActionStoryCutChapter,
		ConfirmationRequired: false,
		Payload:              map[string]any{"story_run_id": "run_1", "span_policy": "latest_completed"},
	}); err != nil {
		t.Fatalf("latest completed cut should be auto-approved: %v", err)
	}

	if err := validator.ValidateAutoApproved(context.Background(), model.DialogueActionOption{
		ActionType:           domain.DialogueActionStoryForkFromEvent,
		ConfirmationRequired: false,
	}); err == nil {
		t.Fatal("fork should require manual confirmation")
	}

	if err := validator.ValidateAutoApproved(context.Background(), model.DialogueActionOption{
		ActionType:           domain.DialogueActionStoryCutChapter,
		ConfirmationRequired: false,
		Payload:              map[string]any{"story_run_id": "run_1", "span_policy": "latest_completed", "to_event_id": "event_2"},
	}); err == nil {
		t.Fatal("auto latest completed cut should reject agent-supplied event ids")
	}
}

func TestDialogueActionExecutorAutoApprovedRejectsManualOption(t *testing.T) {
	repo := &autoActionDialogueSessions{option: model.DialogueActionOption{
		ID:                   "dopt_1",
		ActionType:           domain.DialogueActionStoryAdvance,
		ConfirmationRequired: true,
		Status:               domain.DialogueActionStatusPending,
	}}
	executor := &DialogueActionExecutor{dialogueSessions: repo, validator: NewDialogueActionValidator(nil, nil, nil)}

	if _, err := executor.ExecuteAutoApproved(context.Background(), "dopt_1", model.AutoExecuteDialogueActionInput{PolicyReason: "test"}); err == nil {
		t.Fatal("expected auto execution to reject confirmation-required option")
	}
	if repo.started {
		t.Fatal("manual option should not start execution through auto policy")
	}
}

func TestDialogueActionExecutorAutoApprovedRequiresPolicyReason(t *testing.T) {
	repo := &autoActionDialogueSessions{option: model.DialogueActionOption{
		ID:                   "dopt_1",
		ActionType:           domain.DialogueActionStoryAdvance,
		ConfirmationRequired: false,
		Status:               domain.DialogueActionStatusPending,
	}}
	executor := &DialogueActionExecutor{dialogueSessions: repo, validator: NewDialogueActionValidator(nil, nil, nil)}

	if _, err := executor.ExecuteAutoApproved(context.Background(), "dopt_1", model.AutoExecuteDialogueActionInput{}); err == nil {
		t.Fatal("expected auto execution to require policy reason")
	}
	if repo.started {
		t.Fatal("option without policy reason should not start execution")
	}
}

func TestDialogueActionExecutorAutoApprovedRejectsHighRiskAction(t *testing.T) {
	repo := &autoActionDialogueSessions{option: model.DialogueActionOption{
		ID:                   "dopt_1",
		ActionType:           domain.DialogueActionSetupApply,
		ConfirmationRequired: false,
		Status:               domain.DialogueActionStatusPending,
	}}
	executor := &DialogueActionExecutor{dialogueSessions: repo, validator: NewDialogueActionValidator(nil, nil, nil)}

	if _, err := executor.ExecuteAutoApproved(context.Background(), "dopt_1", model.AutoExecuteDialogueActionInput{PolicyReason: "test"}); err == nil {
		t.Fatal("expected auto execution to reject setup apply")
	}
	if repo.started {
		t.Fatal("high-risk option should not start execution through auto policy")
	}
}

type autoActionDialogueSessions struct {
	option  model.DialogueActionOption
	started bool
}

func (s *autoActionDialogueSessions) CreateSession(context.Context, string, model.CreateDialogueSessionInput) (model.DialogueSession, error) {
	return model.DialogueSession{}, errors.New("not implemented")
}

func (s *autoActionDialogueSessions) ListSessionsByProjectID(context.Context, string, model.PageInput) (model.ListResult[model.DialogueSession], error) {
	return model.ListResult[model.DialogueSession]{}, errors.New("not implemented")
}

func (s *autoActionDialogueSessions) GetSessionByID(context.Context, string) (model.DialogueSession, error) {
	return model.DialogueSession{}, errors.New("not implemented")
}

func (s *autoActionDialogueSessions) UpdateSession(context.Context, model.DialogueSession) (model.DialogueSession, error) {
	return model.DialogueSession{}, errors.New("not implemented")
}

func (s *autoActionDialogueSessions) AppendMessage(context.Context, string, string, string, map[string]any) (model.DialogueMessage, error) {
	return model.DialogueMessage{}, nil
}

func (s *autoActionDialogueSessions) ListMessagesBySessionID(context.Context, string) ([]model.DialogueMessage, error) {
	return nil, errors.New("not implemented")
}

func (s *autoActionDialogueSessions) CreateRun(context.Context, string, model.AdvanceDialogueSessionInput) (model.DialogueRun, error) {
	return model.DialogueRun{}, errors.New("not implemented")
}

func (s *autoActionDialogueSessions) GetRunByID(context.Context, string) (model.DialogueRun, error) {
	return model.DialogueRun{}, errors.New("not implemented")
}

func (s *autoActionDialogueSessions) UpdateRunStatus(context.Context, string, string, string, int, ...string) error {
	return errors.New("not implemented")
}

func (s *autoActionDialogueSessions) UpdateRunHeartbeat(context.Context, string) error {
	return errors.New("not implemented")
}

func (s *autoActionDialogueSessions) SaveRunResult(context.Context, string, model.DialogueRunResult) error {
	return errors.New("not implemented")
}

func (s *autoActionDialogueSessions) GetRunResultByID(context.Context, string) (model.DialogueRunResult, error) {
	return model.DialogueRunResult{}, errors.New("not implemented")
}

func (s *autoActionDialogueSessions) SaveActionOptions(context.Context, []model.DialogueActionOption) error {
	return errors.New("not implemented")
}

func (s *autoActionDialogueSessions) ListActionOptionsByRunID(context.Context, string) ([]model.DialogueActionOption, error) {
	return nil, errors.New("not implemented")
}

func (s *autoActionDialogueSessions) ListPendingActionOptionsBySessionID(context.Context, string) ([]model.DialogueActionOption, error) {
	return nil, errors.New("not implemented")
}

func (s *autoActionDialogueSessions) GetActionOptionByID(context.Context, string) (model.DialogueActionOption, error) {
	return s.option, nil
}

func (s *autoActionDialogueSessions) UpdateActionOption(context.Context, model.DialogueActionOption) (model.DialogueActionOption, error) {
	return model.DialogueActionOption{}, errors.New("not implemented")
}

func (s *autoActionDialogueSessions) TryStartActionExecution(context.Context, string) (model.DialogueActionOption, error) {
	s.started = true
	return s.option, nil
}
