package service

import (
	"context"

	"github.com/fishimei/NovelOS/internal/application/model"
	"github.com/fishimei/NovelOS/internal/application/port"
	"github.com/fishimei/NovelOS/internal/pkgerr"
)

type StoryEventLogService struct {
	sessions port.StorySessionRepository
	store    port.StoryEventStore
	advancer *StorySessionAdvancer
	clock    port.Clock
}

func NewStoryEventLogService(sessions port.StorySessionRepository, store port.StoryEventStore, advancer *StorySessionAdvancer, clock port.Clock) *StoryEventLogService {
	return &StoryEventLogService{sessions: sessions, store: store, advancer: advancer, clock: clock}
}

func (s *StoryEventLogService) ListSessionEvents(ctx context.Context, sessionID string) (model.StoryEventLog, error) {
	if s.store == nil {
		return model.StoryEventLog{}, pkgerr.Internal("story event store is required", nil)
	}
	if _, err := s.sessions.GetSessionByID(ctx, sessionID); err != nil {
		return model.StoryEventLog{}, err
	}
	branches, err := s.store.ListBranchesBySession(ctx, sessionID)
	if err != nil {
		return model.StoryEventLog{}, err
	}
	events, err := s.store.ListEventsBySession(ctx, sessionID)
	if err != nil {
		return model.StoryEventLog{}, err
	}
	return model.StoryEventLog{SessionID: sessionID, Branches: branches, Events: events}, nil
}

func (s *StoryEventLogService) GetEvent(ctx context.Context, eventID string) (model.StoryEvent, error) {
	if s.store == nil {
		return model.StoryEvent{}, pkgerr.Internal("story event store is required", nil)
	}
	return s.store.GetEvent(ctx, eventID)
}

func (s *StoryEventLogService) GetEventState(ctx context.Context, eventID string) (model.WorldSnapshot, error) {
	if s.store == nil {
		return model.WorldSnapshot{}, pkgerr.Internal("story event store is required", nil)
	}
	return s.store.ResolveStateAt(ctx, eventID)
}

func (s *StoryEventLogService) ForkEvent(ctx context.Context, eventID string, input model.ForkStoryEventInput) (model.Branch, error) {
	if s.store == nil {
		return model.Branch{}, pkgerr.Internal("story event store is required", nil)
	}
	event, err := s.store.GetEvent(ctx, eventID)
	if err != nil {
		return model.Branch{}, err
	}
	name := input.Name
	if name == "" {
		name = "fork"
	}
	now := currentTime(s.clock)
	return s.store.CreateBranch(ctx, model.Branch{
		ProjectID:   event.ProjectID,
		SessionID:   event.SessionID,
		Name:        name,
		BaseEventID: event.ID,
		HeadEventID: event.ID,
		Status:      "active",
		CreatedAt:   now,
		UpdatedAt:   now,
	})
}

func (s *StoryEventLogService) AdvanceBranch(ctx context.Context, branchID string, input model.AdvanceStorySessionInput) (model.StoryRun, error) {
	if s.store == nil {
		return model.StoryRun{}, pkgerr.Internal("story event store is required", nil)
	}
	if s.advancer == nil {
		return model.StoryRun{}, pkgerr.Internal("story session advancer is required", nil)
	}
	branch, err := s.store.GetBranch(ctx, branchID)
	if err != nil {
		return model.StoryRun{}, err
	}
	input.BranchID = branch.ID
	input.BaseEventID = branch.HeadEventID
	return s.advancer.Advance(ctx, branch.SessionID, input)
}
