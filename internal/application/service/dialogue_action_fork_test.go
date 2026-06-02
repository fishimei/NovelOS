package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/fishimei/NovelOS/internal/application/model"
	"github.com/fishimei/NovelOS/internal/domain"
)

func TestDialogueActionValidatorValidatesStoryForkEventScope(t *testing.T) {
	store := &forkActionStoryEventStore{event: model.StoryEvent{ID: "event_1", ProjectID: "project_1", SessionID: "story_1"}}
	validator := NewDialogueActionValidator(nil, nil, store)

	err := validator.ValidateOption(context.Background(), model.DialogueActionOption{
		ProjectID:  "other_project",
		ActionType: domain.DialogueActionStoryForkFromEvent,
		Payload:    map[string]any{"event_id": "event_1"},
	})
	if err == nil {
		t.Fatal("expected cross-project fork event to be rejected")
	}

	err = validator.ValidateOption(context.Background(), model.DialogueActionOption{
		ProjectID:  "project_1",
		ActionType: domain.DialogueActionStoryForkFromEvent,
		Payload:    map[string]any{"event_id": "event_1", "story_session_id": "story_1", "name": "branch A"},
	})
	if err != nil {
		t.Fatalf("ValidateOption returned error: %v", err)
	}
}

func TestDialogueActionExecutorForksStoryEvent(t *testing.T) {
	store := &forkActionStoryEventStore{event: model.StoryEvent{ID: "event_1", ProjectID: "project_1", SessionID: "story_1"}}
	executor := &DialogueActionExecutor{
		storyEventLog: NewStoryEventLogService(nil, store, nil, nil),
	}

	option, err := executor.execute(context.Background(), model.DialogueActionOption{
		ProjectID:  "project_1",
		ActionType: domain.DialogueActionStoryForkFromEvent,
		Payload:    map[string]any{"event_id": "event_1", "name": "branch A"},
	}, model.ExecuteDialogueActionInput{AuthorNote: "keep the alternate path"})
	if err != nil {
		t.Fatalf("execute returned error: %v", err)
	}
	if store.createdBranch.BaseEventID != "event_1" || store.createdBranch.HeadEventID != "event_1" {
		t.Fatalf("unexpected branch base/head: %#v", store.createdBranch)
	}
	if option.Result["branch_id"] != "branch_1" {
		t.Fatalf("expected branch result, got %#v", option.Result)
	}
}

type forkActionStoryEventStore struct {
	event         model.StoryEvent
	createdBranch model.Branch
}

func (s *forkActionStoryEventStore) AppendEvent(context.Context, model.StoryEvent) (model.StoryEvent, error) {
	return model.StoryEvent{}, errors.New("not implemented")
}

func (s *forkActionStoryEventStore) GetEvent(_ context.Context, id string) (model.StoryEvent, error) {
	if s.event.ID == id {
		return s.event, nil
	}
	return model.StoryEvent{}, errors.New("event not found")
}

func (s *forkActionStoryEventStore) ListEventsByBranch(context.Context, string) ([]model.StoryEvent, error) {
	return nil, errors.New("not implemented")
}

func (s *forkActionStoryEventStore) ListEventsBySession(context.Context, string) ([]model.StoryEvent, error) {
	return nil, errors.New("not implemented")
}

func (s *forkActionStoryEventStore) CreateBranch(_ context.Context, branch model.Branch) (model.Branch, error) {
	if branch.ID == "" {
		branch.ID = "branch_1"
	}
	s.createdBranch = branch
	return branch, nil
}

func (s *forkActionStoryEventStore) GetBranch(context.Context, string) (model.Branch, error) {
	return model.Branch{}, errors.New("not implemented")
}

func (s *forkActionStoryEventStore) ListBranchesBySession(context.Context, string) ([]model.Branch, error) {
	return nil, errors.New("not implemented")
}

func (s *forkActionStoryEventStore) UpdateBranchHead(context.Context, string, string) error {
	return errors.New("not implemented")
}

func (s *forkActionStoryEventStore) SetPublishedFrontier(context.Context, string, string) error {
	return errors.New("not implemented")
}

func (s *forkActionStoryEventStore) ResolveStateAt(context.Context, string) (model.WorldSnapshot, error) {
	return model.WorldSnapshot{}, errors.New("not implemented")
}

func (s *forkActionStoryEventStore) InFlightActionsAt(context.Context, string, time.Time) ([]model.OngoingAction, error) {
	return nil, errors.New("not implemented")
}

func (s *forkActionStoryEventStore) UpsertSnapshot(context.Context, string, string, model.WorldSnapshot) error {
	return errors.New("not implemented")
}

func (s *forkActionStoryEventStore) InitGenesis(context.Context, string, string, model.WorldSnapshot) (model.StoryEvent, error) {
	return model.StoryEvent{}, errors.New("not implemented")
}

func (s *forkActionStoryEventStore) GetWorldMapByProjectID(context.Context, string) (model.WorldMap, error) {
	return model.WorldMap{}, errors.New("not implemented")
}

func (s *forkActionStoryEventStore) UpsertWorldMap(context.Context, model.WorldMap) (model.WorldMap, error) {
	return model.WorldMap{}, errors.New("not implemented")
}

func (s *forkActionStoryEventStore) ListMapTilesByProjectID(context.Context, string) ([]model.MapTile, error) {
	return nil, errors.New("not implemented")
}

func (s *forkActionStoryEventStore) UpsertMapTiles(context.Context, string, []model.MapTile) error {
	return errors.New("not implemented")
}

func (s *forkActionStoryEventStore) ListLocationsByProjectID(context.Context, string) ([]model.LocationState, error) {
	return nil, errors.New("not implemented")
}

func (s *forkActionStoryEventStore) UpsertLocations(context.Context, string, []model.LocationState) error {
	return errors.New("not implemented")
}

func (s *forkActionStoryEventStore) ListFactionInfluencesByProjectID(context.Context, string) ([]model.FactionInfluence, error) {
	return nil, errors.New("not implemented")
}

func (s *forkActionStoryEventStore) UpsertFactionInfluences(context.Context, string, []model.FactionInfluence) error {
	return errors.New("not implemented")
}

func (s *forkActionStoryEventStore) CreateChapterSpan(context.Context, model.ChapterEventSpan) (model.ChapterEventSpan, error) {
	return model.ChapterEventSpan{}, errors.New("not implemented")
}

func (s *forkActionStoryEventStore) GetChapterSpanByRange(context.Context, string, string, string) (model.ChapterEventSpan, error) {
	return model.ChapterEventSpan{}, errors.New("not implemented")
}
