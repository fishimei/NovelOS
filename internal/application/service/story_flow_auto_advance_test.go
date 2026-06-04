package service

import (
	"context"
	"testing"

	"github.com/fishimei/NovelOS/internal/application/model"
	"github.com/fishimei/NovelOS/internal/domain"
)

func TestStorySessionAdvancerAutoAdvanceSkipsAuthorMessage(t *testing.T) {
	sessions := &autoAdvanceStorySessions{session: model.StorySession{ID: "story_1", ProjectID: "project_1", LastAuthorMessage: "previous author note"}}
	store := &autoAdvanceStoryStore{branches: []model.Branch{{ID: "branch_1", ProjectID: "project_1", SessionID: "story_1", HeadEventID: "event_head"}}}
	audit := &autoAdvanceAudit{}
	advancer := &StorySessionAdvancer{sessions: sessions, store: store, audit: audit}

	run, err := advancer.Advance(context.Background(), "story_1", model.AdvanceStorySessionInput{AdvanceMode: "auto"})
	if err != nil {
		t.Fatalf("Advance() error = %v", err)
	}
	if run.RunID != "run_1" {
		t.Fatalf("run = %#v", run)
	}
	if sessions.appendedMessages != 0 {
		t.Fatalf("auto advance should not append author message, got %d", sessions.appendedMessages)
	}
	if sessions.updatedSession.LastAuthorMessage != "previous author note" {
		t.Fatalf("auto advance cleared last author message: %#v", sessions.updatedSession)
	}
	if sessions.createdInput.AdvanceMode != "auto" || sessions.createdInput.BranchID != "branch_1" || sessions.createdInput.BaseEventID != "event_head" {
		t.Fatalf("unexpected created input: %#v", sessions.createdInput)
	}
	if len(audit.events) != 1 || audit.events[0].Payload["advance_mode"] != "auto" {
		t.Fatalf("missing auto advance audit payload: %#v", audit.events)
	}
}

func TestStorySessionAdvancerManualAdvanceKeepsAuthorMessage(t *testing.T) {
	sessions := &autoAdvanceStorySessions{session: model.StorySession{ID: "story_1", ProjectID: "project_1"}}
	store := &autoAdvanceStoryStore{branches: []model.Branch{{ID: "branch_1", ProjectID: "project_1", SessionID: "story_1", HeadEventID: "event_head"}}}
	advancer := &StorySessionAdvancer{sessions: sessions, store: store, audit: &autoAdvanceAudit{}}

	_, err := advancer.Advance(context.Background(), "story_1", model.AdvanceStorySessionInput{AuthorMessage: "push the scene"})
	if err != nil {
		t.Fatalf("Advance() error = %v", err)
	}
	if sessions.appendedMessages != 1 || sessions.appendedContent != "push the scene" {
		t.Fatalf("manual advance did not append author message: count=%d content=%q", sessions.appendedMessages, sessions.appendedContent)
	}
	if sessions.updatedSession.LastAuthorMessage != "push the scene" {
		t.Fatalf("manual advance did not update last author message: %#v", sessions.updatedSession)
	}
}

type autoAdvanceStorySessions struct {
	session          model.StorySession
	updatedSession   model.StorySession
	createdInput     model.AdvanceStorySessionInput
	appendedMessages int
	appendedContent  string
}

func (s *autoAdvanceStorySessions) CreateSession(context.Context, string, model.CreateStorySessionInput) (model.StorySession, error) {
	return model.StorySession{}, nil
}

func (s *autoAdvanceStorySessions) ListSessionsByProjectID(context.Context, string, model.PageInput) (model.ListResult[model.StorySession], error) {
	return model.ListResult[model.StorySession]{}, nil
}

func (s *autoAdvanceStorySessions) GetSessionByID(context.Context, string) (model.StorySession, error) {
	return s.session, nil
}

func (s *autoAdvanceStorySessions) UpdateSession(_ context.Context, session model.StorySession) (model.StorySession, error) {
	s.updatedSession = session
	return session, nil
}

func (s *autoAdvanceStorySessions) DeleteSession(context.Context, string) error { return nil }

func (s *autoAdvanceStorySessions) AppendMessage(_ context.Context, _ string, _ string, content string) (model.ConversationMessage, error) {
	s.appendedMessages++
	s.appendedContent = content
	return model.ConversationMessage{Content: content}, nil
}

func (s *autoAdvanceStorySessions) CreateRun(_ context.Context, sessionID string, input model.AdvanceStorySessionInput) (model.StoryRun, error) {
	s.createdInput = input
	return model.StoryRun{RunID: "run_1", SessionID: sessionID, ProjectID: s.session.ProjectID, BranchID: input.BranchID, BaseEventID: input.BaseEventID, Status: domain.RunStatusQueued}, nil
}

func (s *autoAdvanceStorySessions) HasActiveRunByBranch(context.Context, string) (bool, error) {
	return false, nil
}

func (s *autoAdvanceStorySessions) GetRunByID(context.Context, string) (model.StoryRun, error) {
	return model.StoryRun{}, nil
}

func (s *autoAdvanceStorySessions) GetRunResultByID(context.Context, string) (model.StoryRunResult, error) {
	return model.StoryRunResult{}, nil
}

func (s *autoAdvanceStorySessions) SaveRunResult(context.Context, string, model.StoryRunResult) error {
	return nil
}

func (s *autoAdvanceStorySessions) UpdateRunStatus(context.Context, string, string, string, int, ...string) error {
	return nil
}

func (s *autoAdvanceStorySessions) UpdateRunHeartbeat(context.Context, string) error    { return nil }
func (s *autoAdvanceStorySessions) UpdateRunHead(context.Context, string, string) error { return nil }
func (s *autoAdvanceStorySessions) RequestRunStop(context.Context, string) error        { return nil }
func (s *autoAdvanceStorySessions) MarkCut(context.Context, string) error               { return nil }

type autoAdvanceStoryStore struct {
	forkActionStoryEventStore
	branches []model.Branch
}

func (s *autoAdvanceStoryStore) ListBranchesBySession(context.Context, string) ([]model.Branch, error) {
	return append([]model.Branch(nil), s.branches...), nil
}

type autoAdvanceAudit struct {
	events []model.RunEvent
}

func (a *autoAdvanceAudit) AppendRunEvent(_ context.Context, event model.RunEvent) (model.RunEvent, error) {
	a.events = append(a.events, event)
	return event, nil
}

func (a *autoAdvanceAudit) ListRunEvents(context.Context, string, string) ([]model.RunEvent, error) {
	return nil, nil
}

func (a *autoAdvanceAudit) ListRunEventsAfter(context.Context, string, string, int) ([]model.RunEvent, error) {
	return nil, nil
}

func (a *autoAdvanceAudit) CreateRevision(context.Context, model.StateRevision) (model.StateRevision, error) {
	return model.StateRevision{}, nil
}
