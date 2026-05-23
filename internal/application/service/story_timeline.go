package service

import (
	"context"

	"github.com/fishimei/NovelOS/internal/application/model"
	"github.com/fishimei/NovelOS/internal/application/port"
	"github.com/fishimei/NovelOS/internal/pkgerr"
)

type StoryTimelineService struct {
	sessions          port.StorySessionRepository
	narrativeTimeline port.StoryTimelineRepository
	advancer          *StorySessionAdvancer
	clock             port.Clock
}

func NewStoryTimelineService(sessions port.StorySessionRepository, narrativeTimeline port.StoryTimelineRepository, advancer *StorySessionAdvancer, clock port.Clock) *StoryTimelineService {
	return &StoryTimelineService{sessions: sessions, narrativeTimeline: narrativeTimeline, advancer: advancer, clock: clock}
}

func (s *StoryTimelineService) ListSessionTimeline(ctx context.Context, sessionID string) (model.StorySessionTimeline, error) {
	if s.narrativeTimeline == nil {
		return model.StorySessionTimeline{}, pkgerr.Internal("narrative timeline repository is required", nil)
	}
	if _, err := s.sessions.GetSessionByID(ctx, sessionID); err != nil {
		return model.StorySessionTimeline{}, err
	}
	branches, err := s.narrativeTimeline.ListBranchesBySessionID(ctx, sessionID)
	if err != nil {
		return model.StorySessionTimeline{}, err
	}
	ticks := []model.StoryTick{}
	for _, branch := range branches {
		branchTicks, err := s.narrativeTimeline.ListTicksByBranchID(ctx, branch.ID)
		if err != nil {
			return model.StorySessionTimeline{}, err
		}
		ticks = append(ticks, branchTicks...)
	}
	return model.StorySessionTimeline{SessionID: sessionID, Branches: branches, Ticks: ticks}, nil
}

func (s *StoryTimelineService) GetTick(ctx context.Context, tickID string) (model.StoryTick, error) {
	if s.narrativeTimeline == nil {
		return model.StoryTick{}, pkgerr.Internal("narrative timeline repository is required", nil)
	}
	return s.narrativeTimeline.GetTickByID(ctx, tickID)
}

func (s *StoryTimelineService) GetTickState(ctx context.Context, tickID string) (model.StoryTickState, error) {
	if s.narrativeTimeline == nil {
		return model.StoryTickState{}, pkgerr.Internal("narrative timeline repository is required", nil)
	}
	return s.narrativeTimeline.ResolveTickState(ctx, tickID)
}

func (s *StoryTimelineService) ForkTick(ctx context.Context, tickID string, input model.ForkStoryTickInput) (model.StoryBranch, error) {
	if s.narrativeTimeline == nil {
		return model.StoryBranch{}, pkgerr.Internal("narrative timeline repository is required", nil)
	}
	tick, err := s.narrativeTimeline.GetTickByID(ctx, tickID)
	if err != nil {
		return model.StoryBranch{}, err
	}
	name := input.Name
	if name == "" {
		name = "fork"
	}
	return s.narrativeTimeline.CreateBranch(ctx, model.StoryBranch{
		ProjectID:        tick.ProjectID,
		SessionID:        tick.SessionID,
		Name:             name,
		BaseTickID:       tick.ID,
		HeadTickID:       tick.ID,
		Status:           "active",
		CreatedFromRunID: tick.SourceRunID,
		CreatedAt:        currentTime(s.clock),
		UpdatedAt:        currentTime(s.clock),
	})
}

func (s *StoryTimelineService) AdvanceBranch(ctx context.Context, branchID string, input model.AdvanceStorySessionInput) (model.StoryRun, error) {
	if s.narrativeTimeline == nil {
		return model.StoryRun{}, pkgerr.Internal("narrative timeline repository is required", nil)
	}
	if s.advancer == nil {
		return model.StoryRun{}, pkgerr.Internal("story session advancer is required", nil)
	}
	branch, err := s.narrativeTimeline.GetBranchByID(ctx, branchID)
	if err != nil {
		return model.StoryRun{}, err
	}
	input.BranchID = branch.ID
	input.BaseTickID = branch.HeadTickID
	return s.advancer.Advance(ctx, branch.SessionID, input)
}
