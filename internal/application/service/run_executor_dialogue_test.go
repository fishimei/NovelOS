package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/fishimei/NovelOS/internal/application/model"
	"github.com/fishimei/NovelOS/internal/application/port"
	"github.com/fishimei/NovelOS/internal/domain"
)

func TestRunExecutorDispatchesDialogueRun(t *testing.T) {
	sessions := &runExecutorDialogueSessions{
		session: model.DialogueSession{ID: "dialogue_1", ProjectID: "project_1"},
		run:     model.DialogueRun{RunID: "drun_1", SessionID: "dialogue_1", ProjectID: "project_1", Status: domain.RunStatusLoadingState},
	}
	generator := &runExecutorDialogueGenerator{}
	advancer := NewDialogueSessionAdvancer(sessions, nil, generator, nil)
	repo := &runExecutorRepository{claimed: true}
	executor := NewRunExecutor(repo, nil, nil, advancer, RunExecutorSettings{Enabled: true, BatchSize: 1, RunTimeoutSeconds: 5}, nil)

	executor.handle(context.Background(), model.RunExecutionWork{RunKind: port.RunKindDialogue, RunID: "drun_1"}, time.Now().Add(-time.Minute))

	if !repo.claimCalled {
		t.Fatal("expected dialogue run to be claimed")
	}
	if repo.lease.Owner == "" {
		t.Fatal("expected claim lease owner")
	}
	if !generator.called {
		t.Fatal("expected dialogue generator to be called")
	}
	if generator.lease.Owner != repo.lease.Owner {
		t.Fatalf("generator lease owner = %q, want %q", generator.lease.Owner, repo.lease.Owner)
	}
	if !sessions.savedResult {
		t.Fatal("expected dialogue run result to be saved")
	}
	if sessions.session.Status != domain.SessionStatusIdle {
		t.Fatalf("session status = %q, want %q", sessions.session.Status, domain.SessionStatusIdle)
	}
}

type runExecutorRepository struct {
	claimed     bool
	claimCalled bool
	lease       port.RunLease
}

func (r *runExecutorRepository) ListRunnableRuns(context.Context, time.Time, int) ([]model.RunExecutionWork, error) {
	return nil, errors.New("not implemented")
}

func (r *runExecutorRepository) ClaimRun(_ context.Context, _ model.RunExecutionWork, lease port.RunLease, _ time.Time) (bool, error) {
	r.claimCalled = true
	r.lease = lease
	return r.claimed, nil
}

type runExecutorDialogueGenerator struct {
	called bool
	lease  port.RunLease
}

func (g *runExecutorDialogueGenerator) Generate(ctx context.Context, input port.DialogueRunGenerationInput) (model.DialogueRunResult, error) {
	g.called = true
	lease, ok := port.RunLeaseFromContext(ctx)
	if !ok {
		return model.DialogueRunResult{}, errors.New("missing run lease")
	}
	g.lease = lease
	return model.DialogueRunResult{
		RunID:            input.Run.RunID,
		SessionID:        input.Session.ID,
		Status:           domain.RunStatusCompleted,
		AssistantMessage: "done",
	}, nil
}

type runExecutorDialogueSessions struct {
	session     model.DialogueSession
	run         model.DialogueRun
	savedResult bool
}

func (s *runExecutorDialogueSessions) CreateSession(context.Context, string, model.CreateDialogueSessionInput) (model.DialogueSession, error) {
	return model.DialogueSession{}, errors.New("not implemented")
}

func (s *runExecutorDialogueSessions) ListSessionsByProjectID(context.Context, string, model.PageInput) (model.ListResult[model.DialogueSession], error) {
	return model.ListResult[model.DialogueSession]{}, errors.New("not implemented")
}

func (s *runExecutorDialogueSessions) GetSessionByID(_ context.Context, sessionID string) (model.DialogueSession, error) {
	if s.session.ID == sessionID {
		return s.session, nil
	}
	return model.DialogueSession{}, errors.New("dialogue session not found")
}

func (s *runExecutorDialogueSessions) UpdateSession(_ context.Context, session model.DialogueSession) (model.DialogueSession, error) {
	s.session = session
	return session, nil
}

func (s *runExecutorDialogueSessions) AppendMessage(context.Context, string, string, string, map[string]any) (model.DialogueMessage, error) {
	return model.DialogueMessage{}, nil
}

func (s *runExecutorDialogueSessions) ListMessagesBySessionID(context.Context, string) ([]model.DialogueMessage, error) {
	return nil, errors.New("not implemented")
}

func (s *runExecutorDialogueSessions) CreateRun(context.Context, string, model.AdvanceDialogueSessionInput) (model.DialogueRun, error) {
	return model.DialogueRun{}, errors.New("not implemented")
}

func (s *runExecutorDialogueSessions) GetRunByID(_ context.Context, runID string) (model.DialogueRun, error) {
	if s.run.RunID == runID {
		return s.run, nil
	}
	return model.DialogueRun{}, errors.New("dialogue run not found")
}

func (s *runExecutorDialogueSessions) UpdateRunStatus(_ context.Context, runID string, status string, currentStep string, progress int, errorMessage ...string) error {
	if s.run.RunID != runID {
		return errors.New("dialogue run not found")
	}
	s.run.Status = status
	s.run.CurrentStep = currentStep
	s.run.Progress = progress
	return nil
}

func (s *runExecutorDialogueSessions) UpdateRunHeartbeat(context.Context, string) error {
	return nil
}

func (s *runExecutorDialogueSessions) SaveRunResult(_ context.Context, runID string, result model.DialogueRunResult) error {
	if s.run.RunID != runID {
		return errors.New("dialogue run not found")
	}
	s.savedResult = true
	s.run.Status = result.Status
	return nil
}

func (s *runExecutorDialogueSessions) GetRunResultByID(context.Context, string) (model.DialogueRunResult, error) {
	return model.DialogueRunResult{}, errors.New("not implemented")
}

func (s *runExecutorDialogueSessions) SaveActionOptions(context.Context, []model.DialogueActionOption) error {
	return nil
}

func (s *runExecutorDialogueSessions) ListActionOptionsByRunID(context.Context, string) ([]model.DialogueActionOption, error) {
	return nil, errors.New("not implemented")
}

func (s *runExecutorDialogueSessions) ListPendingActionOptionsBySessionID(context.Context, string) ([]model.DialogueActionOption, error) {
	return nil, errors.New("not implemented")
}

func (s *runExecutorDialogueSessions) GetActionOptionByID(context.Context, string) (model.DialogueActionOption, error) {
	return model.DialogueActionOption{}, errors.New("not implemented")
}

func (s *runExecutorDialogueSessions) UpdateActionOption(context.Context, model.DialogueActionOption) (model.DialogueActionOption, error) {
	return model.DialogueActionOption{}, errors.New("not implemented")
}

func (s *runExecutorDialogueSessions) TryStartActionExecution(context.Context, string) (model.DialogueActionOption, error) {
	return model.DialogueActionOption{}, errors.New("not implemented")
}
