package service

import (
	"context"
	"testing"
	"time"

	"github.com/fishimei/NovelOS/internal/application/model"
	"github.com/fishimei/NovelOS/internal/domain"
	"github.com/fishimei/NovelOS/internal/pkgerr"
)

type fakeSimulationRepository struct {
	timeline        model.StoryTimeline
	worldMap        model.WorldMap
	mapTiles        []model.MapTile
	locations       []model.LocationState
	factions        []model.FactionInfluence
	characterStates []model.CharacterSimulationState
	tickRuns        map[string]model.StoryTickRun
	events          []model.SimulationEvent
	snapshots       map[string]model.SimulationSnapshot
}

func newFakeSimulationRepository() *fakeSimulationRepository {
	return &fakeSimulationRepository{tickRuns: map[string]model.StoryTickRun{}, snapshots: map[string]model.SimulationSnapshot{}}
}

func (r *fakeSimulationRepository) GetTimelineByProjectID(_ context.Context, projectID string) (model.StoryTimeline, error) {
	if r.timeline.ID == "" {
		return model.StoryTimeline{}, pkgerr.NotFound("story timeline not found")
	}
	return r.timeline, nil
}

func (r *fakeSimulationRepository) UpsertTimeline(_ context.Context, timeline model.StoryTimeline) (model.StoryTimeline, error) {
	r.timeline = timeline
	return r.timeline, nil
}

func (r *fakeSimulationRepository) GetWorldMapByProjectID(_ context.Context, projectID string) (model.WorldMap, error) {
	if r.worldMap.ID == "" {
		return model.WorldMap{}, pkgerr.NotFound("world map not found")
	}
	return r.worldMap, nil
}

func (r *fakeSimulationRepository) UpsertWorldMap(_ context.Context, worldMap model.WorldMap) (model.WorldMap, error) {
	r.worldMap = worldMap
	return r.worldMap, nil
}

func (r *fakeSimulationRepository) ListMapTilesByProjectID(_ context.Context, projectID string) ([]model.MapTile, error) {
	return append([]model.MapTile(nil), r.mapTiles...), nil
}

func (r *fakeSimulationRepository) UpsertMapTiles(_ context.Context, projectID string, tiles []model.MapTile) error {
	r.mapTiles = append([]model.MapTile(nil), tiles...)
	return nil
}

func (r *fakeSimulationRepository) CreateTickRun(_ context.Context, run model.StoryTickRun) (model.StoryTickRun, error) {
	r.tickRuns[run.ID] = run
	return run, nil
}

func (r *fakeSimulationRepository) UpdateTickRun(_ context.Context, run model.StoryTickRun) (model.StoryTickRun, error) {
	r.tickRuns[run.ID] = run
	return run, nil
}

func (r *fakeSimulationRepository) GetTickRunByID(_ context.Context, tickRunID string) (model.StoryTickRun, error) {
	run, ok := r.tickRuns[tickRunID]
	if !ok {
		return model.StoryTickRun{}, pkgerr.NotFound("story tick run not found")
	}
	return run, nil
}

func (r *fakeSimulationRepository) ListLocationsByProjectID(_ context.Context, projectID string) ([]model.LocationState, error) {
	return append([]model.LocationState(nil), r.locations...), nil
}

func (r *fakeSimulationRepository) UpsertLocations(_ context.Context, projectID string, locations []model.LocationState) error {
	for _, location := range locations {
		updated := false
		for i := range r.locations {
			if r.locations[i].ID == location.ID {
				r.locations[i] = location
				updated = true
			}
		}
		if !updated {
			r.locations = append(r.locations, location)
		}
	}
	return nil
}

func (r *fakeSimulationRepository) ListFactionInfluencesByProjectID(_ context.Context, projectID string) ([]model.FactionInfluence, error) {
	return append([]model.FactionInfluence(nil), r.factions...), nil
}

func (r *fakeSimulationRepository) UpsertFactionInfluences(_ context.Context, projectID string, influences []model.FactionInfluence) error {
	r.factions = append([]model.FactionInfluence(nil), influences...)
	return nil
}

func (r *fakeSimulationRepository) ListCharacterStatesByProjectID(_ context.Context, projectID string) ([]model.CharacterSimulationState, error) {
	return append([]model.CharacterSimulationState(nil), r.characterStates...), nil
}

func (r *fakeSimulationRepository) UpsertCharacterStates(_ context.Context, projectID string, states []model.CharacterSimulationState) error {
	for _, state := range states {
		updated := false
		for i := range r.characterStates {
			if r.characterStates[i].CharacterID == state.CharacterID {
				r.characterStates[i] = state
				updated = true
			}
		}
		if !updated {
			r.characterStates = append(r.characterStates, state)
		}
	}
	return nil
}

func (r *fakeSimulationRepository) AppendEvent(_ context.Context, event model.SimulationEvent) (model.SimulationEvent, error) {
	event.Sequence = len(r.events) + 1
	r.events = append(r.events, event)
	return event, nil
}

func (r *fakeSimulationRepository) ListEventsByTickRunID(_ context.Context, tickRunID string) ([]model.SimulationEvent, error) {
	out := make([]model.SimulationEvent, 0)
	for _, event := range r.events {
		if event.TickRunID == tickRunID {
			out = append(out, event)
		}
	}
	return out, nil
}

func (r *fakeSimulationRepository) CreateSnapshot(_ context.Context, snapshot model.SimulationSnapshot) (model.SimulationSnapshot, error) {
	r.snapshots[snapshot.TickRunID] = snapshot
	return snapshot, nil
}

func (r *fakeSimulationRepository) GetSnapshotByTickRunID(_ context.Context, tickRunID string) (model.SimulationSnapshot, error) {
	snapshot, ok := r.snapshots[tickRunID]
	if !ok {
		return model.SimulationSnapshot{}, pkgerr.NotFound("simulation snapshot not found")
	}
	return snapshot, nil
}

type fakeCharacterRepository struct {
	characters []model.Character
}

func (r fakeCharacterRepository) Create(ctx context.Context, projectID string, input model.CreateCharacterInput) (model.Character, error) {
	return model.Character{}, nil
}

func (r fakeCharacterRepository) ListByProjectID(ctx context.Context, projectID string, pageInput model.PageInput) (model.ListResult[model.Character], error) {
	return model.ListResult[model.Character]{Items: r.characters, Total: len(r.characters)}, nil
}

func (r fakeCharacterRepository) GetByID(ctx context.Context, id string) (model.Character, error) {
	for _, character := range r.characters {
		if character.ID == id {
			return character, nil
		}
	}
	return model.Character{}, pkgerr.NotFound("character not found")
}

func (r fakeCharacterRepository) Update(ctx context.Context, id string, input model.UpdateCharacterInput) (model.Character, error) {
	return model.Character{}, nil
}

func (r fakeCharacterRepository) Upsert(ctx context.Context, character model.Character) (model.Character, error) {
	return character, nil
}

type fakeActionDecider struct {
	calls  int
	inputs []model.CharacterActionDecisionInput
}

func (d *fakeActionDecider) Decide(ctx context.Context, input model.CharacterActionDecisionInput) (model.CharacterActionDecision, error) {
	d.calls++
	d.inputs = append(d.inputs, input)
	return model.CharacterActionDecision{ActionType: "patrol", Description: input.Character.Name + "巡视" + input.Location.Name, DurationHours: 2, Rationale: "保持地点控制"}, nil
}

type passthroughTx struct{}

func (passthroughTx) WithinTransaction(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

type storyTickFixedClock struct {
	now time.Time
}

func (c storyTickFixedClock) Now() time.Time { return c.now }

type storyTickIDs struct {
	n int
}

func (g *storyTickIDs) New(prefix string) string {
	g.n++
	return prefix + "_test_" + string(rune('a'+g.n))
}

func TestStoryTickAdvancerStartsActionAndCreatesSnapshot(t *testing.T) {
	now := time.Date(2026, 5, 21, 12, 0, 0, 0, time.UTC)
	repo := newFakeSimulationRepository()
	decider := &fakeActionDecider{}
	advancer := NewStoryTickAdvancer(repo, fakeCharacterRepository{characters: []model.Character{{ID: "character_1", ProjectID: "project_1", Name: "阿青", CreatedAt: now}}}, decider, passthroughTx{}, storyTickFixedClock{now: now}, &storyTickIDs{}, 25)

	result, err := advancer.Advance(context.Background(), "project_1", model.AdvanceStoryTickInput{TickHours: 1})
	if err != nil {
		t.Fatalf("advance story tick: %v", err)
	}
	if decider.calls != 1 {
		t.Fatalf("expected decider to be called once, got %d", decider.calls)
	}
	if result.State.Timeline.Tick != 1 {
		t.Fatalf("expected tick 1, got %d", result.State.Timeline.Tick)
	}
	if len(result.State.CharacterStates) != 1 || result.State.CharacterStates[0].OngoingAction == nil {
		t.Fatalf("expected ongoing action in snapshot: %+v", result.State.CharacterStates)
	}
	if result.State.CharacterStates[0].OngoingAction.EndsAt != now.Add(2*time.Hour) {
		t.Fatalf("unexpected action end: %s", result.State.CharacterStates[0].OngoingAction.EndsAt)
	}
	if result.Snapshot.Snapshot.Timeline.Tick != 1 {
		t.Fatalf("snapshot did not capture advanced state")
	}
}

func TestStoryTickAdvancerPassesNearbyLocationsToDecider(t *testing.T) {
	now := time.Date(2026, 5, 21, 12, 0, 0, 0, time.UTC)
	repo := newFakeSimulationRepository()
	repo.timeline = model.StoryTimeline{ID: "timeline_1", ProjectID: "project_1", CurrentTime: now, Tick: 1}
	repo.worldMap = model.WorldMap{ID: "map_1", ProjectID: "project_1", Name: "Test Map", Width: 100, Height: 100, Status: "active"}
	repo.mapTiles = []model.MapTile{{ID: "tile_1", ProjectID: "project_1", MapID: "map_1", X: 10, Y: 10, Terrain: "plain"}}
	repo.locations = []model.LocationState{
		{ID: "location_origin", ProjectID: "project_1", MapID: "map_1", Name: "原点镇", Status: "active", X: 10, Y: 10},
		{ID: "location_near", ProjectID: "project_1", MapID: "map_1", Name: "近郊", Status: "active", X: 13, Y: 14},
		{ID: "location_far", ProjectID: "project_1", MapID: "map_1", Name: "远山", Status: "active", X: 80, Y: 80},
	}
	repo.factions = []model.FactionInfluence{{ID: "faction_1", ProjectID: "project_1", LocationID: "location_near", FactionName: "近郊议会", Status: "active"}}
	repo.characterStates = []model.CharacterSimulationState{{ID: "state_1", ProjectID: "project_1", CharacterID: "character_1", LocationID: "location_origin", X: 10, Y: 10, Status: "active"}}
	decider := &fakeActionDecider{}
	advancer := NewStoryTickAdvancer(repo, fakeCharacterRepository{characters: []model.Character{{ID: "character_1", ProjectID: "project_1", Name: "阿青", CreatedAt: now}}}, decider, passthroughTx{}, storyTickFixedClock{now: now}, &storyTickIDs{}, 10)

	result, err := advancer.Advance(context.Background(), "project_1", model.AdvanceStoryTickInput{TickHours: 1})
	if err != nil {
		t.Fatalf("advance story tick: %v", err)
	}
	if result.State.Map == nil || result.State.Map.ID != "map_1" || len(result.State.Tiles) != 1 {
		t.Fatalf("expected map and sampled tiles in state: %+v", result.State)
	}
	if len(decider.inputs) != 1 {
		t.Fatalf("expected one decider input, got %d", len(decider.inputs))
	}
	nearby := decider.inputs[0].NearbyLocations
	if len(nearby) != 1 || nearby[0].Location.ID != "location_near" {
		t.Fatalf("unexpected nearby locations: %+v", nearby)
	}
	if len(nearby[0].FactionInfluences) != 1 || nearby[0].FactionInfluences[0].FactionName != "近郊议会" {
		t.Fatalf("expected nearby faction context: %+v", nearby[0].FactionInfluences)
	}
}

func TestStoryTickAdvancerSkipsUnfinishedAction(t *testing.T) {
	now := time.Date(2026, 5, 21, 12, 0, 0, 0, time.UTC)
	repo := newFakeSimulationRepository()
	repo.timeline = model.StoryTimeline{ID: "timeline_1", ProjectID: "project_1", CurrentTime: now, Tick: 3}
	repo.locations = []model.LocationState{{ID: "location_1", ProjectID: "project_1", Name: "东市", Status: "active"}}
	repo.characterStates = []model.CharacterSimulationState{{ID: "state_1", ProjectID: "project_1", CharacterID: "character_1", LocationID: "location_1", Status: "active", OngoingAction: &model.CharacterOngoingAction{ActionType: "search", Description: "搜寻线索", StartedAt: now.Add(-time.Hour), EndsAt: now.Add(time.Hour), Status: domain.CharacterActionStatusOngoing}}}
	decider := &fakeActionDecider{}
	advancer := NewStoryTickAdvancer(repo, fakeCharacterRepository{characters: []model.Character{{ID: "character_1", ProjectID: "project_1", Name: "阿青", CreatedAt: now}}}, decider, passthroughTx{}, storyTickFixedClock{now: now}, &storyTickIDs{}, 25)

	result, err := advancer.Advance(context.Background(), "project_1", model.AdvanceStoryTickInput{TickHours: 1})
	if err != nil {
		t.Fatalf("advance story tick: %v", err)
	}
	if decider.calls != 0 {
		t.Fatalf("expected decider to be skipped, got %d calls", decider.calls)
	}
	if result.State.CharacterStates[0].OngoingAction == nil || result.State.CharacterStates[0].OngoingAction.Description != "搜寻线索" {
		t.Fatalf("expected action to remain ongoing")
	}
	if !hasSimulationEvent(result.Events, domain.EventCharacterSkippedOngoingAction) {
		t.Fatalf("expected skip event, got %+v", result.Events)
	}
}

func TestStoryTickAdvancerCompletesExpiredActionBeforeNewDecision(t *testing.T) {
	now := time.Date(2026, 5, 21, 12, 0, 0, 0, time.UTC)
	repo := newFakeSimulationRepository()
	repo.timeline = model.StoryTimeline{ID: "timeline_1", ProjectID: "project_1", CurrentTime: now, Tick: 3}
	repo.locations = []model.LocationState{{ID: "location_1", ProjectID: "project_1", Name: "东市", Status: "active"}}
	repo.characterStates = []model.CharacterSimulationState{{ID: "state_1", ProjectID: "project_1", CharacterID: "character_1", LocationID: "location_1", Status: "active", OngoingAction: &model.CharacterOngoingAction{ActionType: "search", Description: "搜寻线索", StartedAt: now.Add(-2 * time.Hour), EndsAt: now, Status: domain.CharacterActionStatusOngoing}}}
	decider := &fakeActionDecider{}
	advancer := NewStoryTickAdvancer(repo, fakeCharacterRepository{characters: []model.Character{{ID: "character_1", ProjectID: "project_1", Name: "阿青", CreatedAt: now}}}, decider, passthroughTx{}, storyTickFixedClock{now: now}, &storyTickIDs{}, 25)

	result, err := advancer.Advance(context.Background(), "project_1", model.AdvanceStoryTickInput{TickHours: 1})
	if err != nil {
		t.Fatalf("advance story tick: %v", err)
	}
	if decider.calls != 1 {
		t.Fatalf("expected decider after completion, got %d calls", decider.calls)
	}
	if !hasSimulationEvent(result.Events, domain.EventCharacterActionCompleted) {
		t.Fatalf("expected completion event, got %+v", result.Events)
	}
	if !hasSimulationEvent(result.Events, domain.EventCharacterActionStarted) {
		t.Fatalf("expected new action event, got %+v", result.Events)
	}
}

func hasSimulationEvent(events []model.SimulationEvent, name string) bool {
	for _, event := range events {
		if event.EventName == name {
			return true
		}
	}
	return false
}
