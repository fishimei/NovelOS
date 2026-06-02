package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/fishimei/NovelOS/internal/application/model"
	"github.com/fishimei/NovelOS/internal/domain"
	"github.com/fishimei/NovelOS/internal/pkgerr"
)

func TestStoryChapterCutterCutLatestCompletedSpanResolvesRunSpan(t *testing.T) {
	run := model.StoryRun{
		RunID:       "run_1",
		SessionID:   "story_1",
		ProjectID:   "project_1",
		BranchID:    "branch_1",
		BaseEventID: "event_1",
		HeadEventID: "event_2",
		Status:      domain.RunStatusCompleted,
		CreatedAt:   time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	sessions := &latestSpanStorySessions{run: run}
	store := &latestSpanEventStore{
		branch: model.Branch{ID: "branch_1", ProjectID: "project_1", SessionID: "story_1"},
		events: []model.StoryEvent{
			{ID: "event_1", ProjectID: "project_1", SessionID: "story_1", BranchID: "branch_1", Kind: model.EventKindGenesis},
			{
				ID:        "event_2",
				ProjectID: "project_1",
				SessionID: "story_1",
				BranchID:  "branch_1",
				Kind:      model.EventKindSceneResolved,
				Summary:   "scene",
				Payload: map[string]any{"draft": map[string]any{
					"title":          "Resolved",
					"chapter_number": 1,
					"content":        "rendered scene",
					"summary":        "scene summary",
				}},
			},
		},
	}
	cutter := NewStoryChapterCutter(sessions, store, &latestSpanChapters{}, nil, latestSpanTx{}, nil, latestSpanIDs{})

	result, err := cutter.CutLatestCompletedSpan(context.Background(), "run_1", model.CutChapterInput{Title: "Latest"})
	if err != nil {
		t.Fatalf("CutLatestCompletedSpan() error = %v", err)
	}
	if store.span.FromEventID != "event_1" || store.span.ToEventID != "event_2" || store.span.BranchID != "branch_1" {
		t.Fatalf("span not resolved from run: %#v", store.span)
	}
	if result.Chapter.Content != "rendered scene" {
		t.Fatalf("chapter content = %q, want rendered scene", result.Chapter.Content)
	}
	if !sessions.markedCut {
		t.Fatal("expected story run to be marked cut")
	}
}

func TestStoryChapterCutterAllowsSummaryOnlyScene(t *testing.T) {
	run := model.StoryRun{
		RunID:       "run_1",
		SessionID:   "story_1",
		ProjectID:   "project_1",
		BranchID:    "branch_1",
		BaseEventID: "event_1",
		HeadEventID: "event_2",
		Status:      domain.RunStatusCompleted,
		CreatedAt:   time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	sessions := &latestSpanStorySessions{run: run}
	store := &latestSpanEventStore{
		branch: model.Branch{ID: "branch_1", ProjectID: "project_1", SessionID: "story_1"},
		events: []model.StoryEvent{
			{ID: "event_1", ProjectID: "project_1", SessionID: "story_1", BranchID: "branch_1", Kind: model.EventKindGenesis},
			{
				ID:        "event_2",
				ProjectID: "project_1",
				SessionID: "story_1",
				BranchID:  "branch_1",
				Kind:      model.EventKindSceneResolved,
				Summary:   "fallback summary",
				Payload:   map[string]any{"summary": "scene summary"},
			},
		},
	}
	cutter := NewStoryChapterCutter(sessions, store, &latestSpanChapters{}, nil, latestSpanTx{}, nil, latestSpanIDs{})

	result, err := cutter.CutLatestCompletedSpan(context.Background(), "run_1", model.CutChapterInput{Title: "Latest"})
	if err != nil {
		t.Fatalf("CutLatestCompletedSpan() error = %v", err)
	}
	if result.Chapter.Content != "" {
		t.Fatalf("chapter content = %q, want empty", result.Chapter.Content)
	}
	if result.Chapter.Summary != "scene summary" {
		t.Fatalf("chapter summary = %q", result.Chapter.Summary)
	}
}

type latestSpanTx struct{}

func (latestSpanTx) WithinTransaction(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

type latestSpanIDs struct{}

func (latestSpanIDs) New(prefix string) string {
	return prefix + "_1"
}

type latestSpanStorySessions struct {
	run       model.StoryRun
	markedCut bool
}

func (s *latestSpanStorySessions) CreateSession(context.Context, string, model.CreateStorySessionInput) (model.StorySession, error) {
	return model.StorySession{}, errors.New("not implemented")
}

func (s *latestSpanStorySessions) ListSessionsByProjectID(context.Context, string, model.PageInput) (model.ListResult[model.StorySession], error) {
	return model.ListResult[model.StorySession]{}, errors.New("not implemented")
}

func (s *latestSpanStorySessions) GetSessionByID(context.Context, string) (model.StorySession, error) {
	return model.StorySession{}, errors.New("not implemented")
}

func (s *latestSpanStorySessions) UpdateSession(context.Context, model.StorySession) (model.StorySession, error) {
	return model.StorySession{}, errors.New("not implemented")
}

func (s *latestSpanStorySessions) DeleteSession(context.Context, string) error {
	return errors.New("not implemented")
}

func (s *latestSpanStorySessions) AppendMessage(context.Context, string, string, string) (model.ConversationMessage, error) {
	return model.ConversationMessage{}, errors.New("not implemented")
}

func (s *latestSpanStorySessions) CreateRun(context.Context, string, model.AdvanceStorySessionInput) (model.StoryRun, error) {
	return model.StoryRun{}, errors.New("not implemented")
}

func (s *latestSpanStorySessions) GetRunByID(context.Context, string) (model.StoryRun, error) {
	return s.run, nil
}

func (s *latestSpanStorySessions) GetRunResultByID(context.Context, string) (model.StoryRunResult, error) {
	return model.StoryRunResult{}, errors.New("not implemented")
}

func (s *latestSpanStorySessions) SaveRunResult(context.Context, string, model.StoryRunResult) error {
	return errors.New("not implemented")
}

func (s *latestSpanStorySessions) UpdateRunStatus(context.Context, string, string, string, int, ...string) error {
	return errors.New("not implemented")
}

func (s *latestSpanStorySessions) UpdateRunHead(context.Context, string, string) error {
	return errors.New("not implemented")
}

func (s *latestSpanStorySessions) MarkCut(context.Context, string) error {
	s.markedCut = true
	s.run.Status = domain.RunStatusCut
	now := time.Now().UTC()
	s.run.CutAt = &now
	return nil
}

type latestSpanEventStore struct {
	branch model.Branch
	events []model.StoryEvent
	span   model.ChapterEventSpan
}

func (s *latestSpanEventStore) AppendEvent(context.Context, model.StoryEvent) (model.StoryEvent, error) {
	return model.StoryEvent{}, errors.New("not implemented")
}

func (s *latestSpanEventStore) GetEvent(context.Context, string) (model.StoryEvent, error) {
	return model.StoryEvent{}, errors.New("not implemented")
}

func (s *latestSpanEventStore) ListEventsByBranch(context.Context, string) ([]model.StoryEvent, error) {
	return s.events, nil
}

func (s *latestSpanEventStore) ListEventsBySession(context.Context, string) ([]model.StoryEvent, error) {
	return nil, errors.New("not implemented")
}

func (s *latestSpanEventStore) CreateBranch(context.Context, model.Branch) (model.Branch, error) {
	return model.Branch{}, errors.New("not implemented")
}

func (s *latestSpanEventStore) GetBranch(context.Context, string) (model.Branch, error) {
	return s.branch, nil
}

func (s *latestSpanEventStore) ListBranchesBySession(context.Context, string) ([]model.Branch, error) {
	return nil, errors.New("not implemented")
}

func (s *latestSpanEventStore) UpdateBranchHead(context.Context, string, string) error {
	return errors.New("not implemented")
}

func (s *latestSpanEventStore) SetPublishedFrontier(context.Context, string, string) error {
	return nil
}

func (s *latestSpanEventStore) ResolveStateAt(context.Context, string) (model.WorldSnapshot, error) {
	return model.WorldSnapshot{}, errors.New("not implemented")
}

func (s *latestSpanEventStore) InFlightActionsAt(context.Context, string, time.Time) ([]model.OngoingAction, error) {
	return nil, errors.New("not implemented")
}

func (s *latestSpanEventStore) UpsertSnapshot(context.Context, string, string, model.WorldSnapshot) error {
	return errors.New("not implemented")
}

func (s *latestSpanEventStore) InitGenesis(context.Context, string, string, model.WorldSnapshot) (model.StoryEvent, error) {
	return model.StoryEvent{}, errors.New("not implemented")
}

func (s *latestSpanEventStore) GetWorldMapByProjectID(context.Context, string) (model.WorldMap, error) {
	return model.WorldMap{}, errors.New("not implemented")
}

func (s *latestSpanEventStore) UpsertWorldMap(context.Context, model.WorldMap) (model.WorldMap, error) {
	return model.WorldMap{}, errors.New("not implemented")
}

func (s *latestSpanEventStore) ListMapTilesByProjectID(context.Context, string) ([]model.MapTile, error) {
	return nil, errors.New("not implemented")
}

func (s *latestSpanEventStore) UpsertMapTiles(context.Context, string, []model.MapTile) error {
	return errors.New("not implemented")
}

func (s *latestSpanEventStore) ListLocationsByProjectID(context.Context, string) ([]model.LocationState, error) {
	return nil, errors.New("not implemented")
}

func (s *latestSpanEventStore) UpsertLocations(context.Context, string, []model.LocationState) error {
	return errors.New("not implemented")
}

func (s *latestSpanEventStore) ListFactionInfluencesByProjectID(context.Context, string) ([]model.FactionInfluence, error) {
	return nil, errors.New("not implemented")
}

func (s *latestSpanEventStore) UpsertFactionInfluences(context.Context, string, []model.FactionInfluence) error {
	return errors.New("not implemented")
}

func (s *latestSpanEventStore) CreateChapterSpan(_ context.Context, span model.ChapterEventSpan) (model.ChapterEventSpan, error) {
	span.ID = "span_1"
	s.span = span
	return span, nil
}

func (s *latestSpanEventStore) GetChapterSpanByRange(context.Context, string, string, string) (model.ChapterEventSpan, error) {
	return model.ChapterEventSpan{}, pkgerr.NotFound("chapter span not found")
}

type latestSpanChapters struct{}

func (latestSpanChapters) ListByProjectID(context.Context, string, model.PageInput) (model.ListResult[model.Chapter], error) {
	return model.ListResult[model.Chapter]{Total: 0}, nil
}

func (latestSpanChapters) GetByID(context.Context, string) (model.Chapter, error) {
	return model.Chapter{}, errors.New("not implemented")
}

func (latestSpanChapters) Create(_ context.Context, chapter model.Chapter) (model.Chapter, error) {
	if chapter.ID == "" {
		chapter.ID = "chapter_1"
	}
	return chapter, nil
}
