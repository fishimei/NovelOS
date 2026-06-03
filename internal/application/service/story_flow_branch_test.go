package service

import (
	"context"
	"testing"
	"time"

	"github.com/fishimei/NovelOS/internal/application/model"
	"github.com/fishimei/NovelOS/internal/pkgerr"
)

func TestStorySessionAdvancerCreatesFirstBranchFromProjectGenesis(t *testing.T) {
	genesis := model.StoryEvent{ID: "genesis_1", ProjectID: "project_1", SessionID: "setup_1", Kind: model.EventKindGenesis}
	store := &branchValidationStore{
		events:  map[string]model.StoryEvent{genesis.ID: genesis},
		genesis: genesis,
	}
	advancer := &StorySessionAdvancer{store: store, clock: fixedServiceClock{now: time.Date(2026, 6, 1, 1, 0, 0, 0, time.UTC)}}

	branch, err := advancer.ensureStoryBranch(context.Background(), model.StorySession{ID: "story_1", ProjectID: "project_1"}, model.AdvanceStorySessionInput{})
	if err != nil {
		t.Fatalf("ensureStoryBranch() error = %v", err)
	}
	if branch.BaseEventID != genesis.ID || branch.HeadEventID != genesis.ID {
		t.Fatalf("branch should anchor to project genesis, got %#v", branch)
	}
}

func TestStorySessionAdvancerRejectsExplicitCrossSessionNonGenesisBase(t *testing.T) {
	foreign := model.StoryEvent{ID: "event_other", ProjectID: "project_1", SessionID: "story_2", Kind: model.EventKindSceneResolved}
	store := &branchValidationStore{events: map[string]model.StoryEvent{foreign.ID: foreign}}
	advancer := &StorySessionAdvancer{store: store}

	_, err := advancer.ensureStoryBranch(context.Background(), model.StorySession{ID: "story_1", ProjectID: "project_1"}, model.AdvanceStorySessionInput{BaseEventID: foreign.ID})
	if err == nil {
		t.Fatal("expected cross-session non-genesis base to be rejected")
	}
}

func TestStorySessionAdvancerRejectsUnreachableExistingBranchBase(t *testing.T) {
	genesis := model.StoryEvent{ID: "genesis_1", ProjectID: "project_1", SessionID: "setup_1", Kind: model.EventKindGenesis}
	head := model.StoryEvent{ID: "head_1", ProjectID: "project_1", SessionID: "story_1", BranchID: "branch_1", ParentEventID: genesis.ID, Kind: model.EventKindSceneResolved}
	sibling := model.StoryEvent{ID: "sibling_1", ProjectID: "project_1", SessionID: "story_1", BranchID: "branch_2", ParentEventID: genesis.ID, Kind: model.EventKindSceneResolved}
	store := &branchValidationStore{
		branches: []model.Branch{{ID: "branch_1", ProjectID: "project_1", SessionID: "story_1", BaseEventID: genesis.ID, HeadEventID: head.ID}},
		events:   map[string]model.StoryEvent{genesis.ID: genesis, head.ID: head, sibling.ID: sibling},
		genesis:  genesis,
	}
	advancer := &StorySessionAdvancer{store: store}

	_, err := advancer.ensureStoryBranch(context.Background(), model.StorySession{ID: "story_1", ProjectID: "project_1"}, model.AdvanceStorySessionInput{BaseEventID: sibling.ID})
	if err == nil {
		t.Fatal("expected unreachable sibling base to be rejected")
	}
}

type branchValidationStore struct {
	forkActionStoryEventStore
	branches []model.Branch
	events   map[string]model.StoryEvent
	genesis  model.StoryEvent
}

func (s *branchValidationStore) GetEvent(_ context.Context, id string) (model.StoryEvent, error) {
	event, ok := s.events[id]
	if !ok {
		return model.StoryEvent{}, pkgerr.NotFound("story event not found")
	}
	return event, nil
}

func (s *branchValidationStore) ListBranchesBySession(context.Context, string) ([]model.Branch, error) {
	return append([]model.Branch(nil), s.branches...), nil
}

func (s *branchValidationStore) CreateBranch(_ context.Context, branch model.Branch) (model.Branch, error) {
	if branch.ID == "" {
		branch.ID = "branch_1"
	}
	s.branches = append(s.branches, branch)
	return branch, nil
}

func (s *branchValidationStore) GetProjectGenesis(context.Context, string) (model.StoryEvent, error) {
	if s.genesis.ID == "" {
		return model.StoryEvent{}, pkgerr.NotFound("story event not found")
	}
	return s.genesis, nil
}
