package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/fishimei/NovelOS/internal/application/model"
)

func TestWorldHandlerGetMapChecksProjectScope(t *testing.T) {
	gin.SetMode(gin.TestMode)
	projects := &worldProjectRepository{}
	events := &worldEventStore{worldMap: model.WorldMap{ID: "map_1", ProjectID: "project_1"}}
	handler := NewWorldHandler(projects, events)
	router := gin.New()
	router.GET("/projects/:project_id/world-map", handler.GetMap)

	req := httptest.NewRequest(http.MethodGet, "/projects/project_1/world-map", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if projects.getDetailProjectID != "project_1" {
		t.Fatalf("expected project scope check, got %q", projects.getDetailProjectID)
	}
	if events.worldMapProjectID != "project_1" {
		t.Fatalf("expected world map lookup for project_1, got %q", events.worldMapProjectID)
	}
}

func TestWorldHandlerInFlightRequiresStoryTime(t *testing.T) {
	gin.SetMode(gin.TestMode)
	events := &worldEventStore{branch: model.Branch{ID: "branch_1"}}
	handler := NewWorldHandler(nil, events)
	router := gin.New()
	router.GET("/story-branches/:branch_id/in-flight-actions", handler.ListInFlightActions)

	req := httptest.NewRequest(http.MethodGet, "/story-branches/branch_1/in-flight-actions", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
	if events.inFlightCalled {
		t.Fatal("expected missing at query to avoid in-flight lookup")
	}
}

type worldProjectRepository struct {
	getDetailProjectID string
}

func (r *worldProjectRepository) Create(context.Context, model.CreateProjectInput) (model.Project, error) {
	return model.Project{}, errors.New("not implemented")
}

func (r *worldProjectRepository) GetByID(context.Context, string) (model.Project, error) {
	return model.Project{}, errors.New("not implemented")
}

func (r *worldProjectRepository) Update(context.Context, string, model.UpdateProjectInput) (model.Project, error) {
	return model.Project{}, errors.New("not implemented")
}

func (r *worldProjectRepository) GetDetail(_ context.Context, id string) (model.ProjectDetail, error) {
	r.getDetailProjectID = id
	return model.ProjectDetail{Project: model.Project{ID: id}}, nil
}

type worldEventStore struct {
	worldMap          model.WorldMap
	worldMapProjectID string
	branch            model.Branch
	inFlightCalled    bool
}

func (s *worldEventStore) AppendEvent(context.Context, model.StoryEvent) (model.StoryEvent, error) {
	return model.StoryEvent{}, errors.New("not implemented")
}

func (s *worldEventStore) GetEvent(context.Context, string) (model.StoryEvent, error) {
	return model.StoryEvent{}, errors.New("not implemented")
}

func (s *worldEventStore) ListEventsByBranch(context.Context, string) ([]model.StoryEvent, error) {
	return nil, errors.New("not implemented")
}

func (s *worldEventStore) ListEventsBySession(context.Context, string) ([]model.StoryEvent, error) {
	return nil, errors.New("not implemented")
}

func (s *worldEventStore) CreateBranch(context.Context, model.Branch) (model.Branch, error) {
	return model.Branch{}, errors.New("not implemented")
}

func (s *worldEventStore) GetBranch(_ context.Context, id string) (model.Branch, error) {
	if s.branch.ID == id {
		return s.branch, nil
	}
	return model.Branch{}, errors.New("branch not found")
}

func (s *worldEventStore) ListBranchesBySession(context.Context, string) ([]model.Branch, error) {
	return nil, errors.New("not implemented")
}

func (s *worldEventStore) UpdateBranchHead(context.Context, string, string) error {
	return errors.New("not implemented")
}

func (s *worldEventStore) SetPublishedFrontier(context.Context, string, string) error {
	return errors.New("not implemented")
}

func (s *worldEventStore) ResolveStateAt(context.Context, string) (model.WorldSnapshot, error) {
	return model.WorldSnapshot{}, errors.New("not implemented")
}

func (s *worldEventStore) InFlightActionsAt(context.Context, string, time.Time) ([]model.OngoingAction, error) {
	s.inFlightCalled = true
	return nil, nil
}

func (s *worldEventStore) UpsertSnapshot(context.Context, string, string, model.WorldSnapshot) error {
	return errors.New("not implemented")
}

func (s *worldEventStore) InitGenesis(context.Context, string, string, model.WorldSnapshot) (model.StoryEvent, error) {
	return model.StoryEvent{}, errors.New("not implemented")
}

func (s *worldEventStore) GetProjectGenesis(context.Context, string) (model.StoryEvent, error) {
	return model.StoryEvent{}, errors.New("not implemented")
}

func (s *worldEventStore) GetWorldMapByProjectID(_ context.Context, projectID string) (model.WorldMap, error) {
	s.worldMapProjectID = projectID
	return s.worldMap, nil
}

func (s *worldEventStore) UpsertWorldMap(context.Context, model.WorldMap) (model.WorldMap, error) {
	return model.WorldMap{}, errors.New("not implemented")
}

func (s *worldEventStore) ListMapTilesByProjectID(context.Context, string) ([]model.MapTile, error) {
	return nil, errors.New("not implemented")
}

func (s *worldEventStore) UpsertMapTiles(context.Context, string, []model.MapTile) error {
	return errors.New("not implemented")
}

func (s *worldEventStore) ListLocationsByProjectID(context.Context, string) ([]model.LocationState, error) {
	return nil, errors.New("not implemented")
}

func (s *worldEventStore) UpsertLocations(context.Context, string, []model.LocationState) error {
	return errors.New("not implemented")
}

func (s *worldEventStore) ListFactionInfluencesByProjectID(context.Context, string) ([]model.FactionInfluence, error) {
	return nil, errors.New("not implemented")
}

func (s *worldEventStore) UpsertFactionInfluences(context.Context, string, []model.FactionInfluence) error {
	return errors.New("not implemented")
}

func (s *worldEventStore) CreateChapterSpan(context.Context, model.ChapterEventSpan) (model.ChapterEventSpan, error) {
	return model.ChapterEventSpan{}, errors.New("not implemented")
}

func (s *worldEventStore) GetChapterSpanByRange(context.Context, string, string, string) (model.ChapterEventSpan, error) {
	return model.ChapterEventSpan{}, errors.New("not implemented")
}
