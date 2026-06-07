package service

import (
	"context"
	"errors"
	"sort"
	"strconv"
	"testing"
	"time"

	"github.com/fishimei/NovelOS/internal/application/model"
)

func TestLocationInspectionEnsureInitializesCurrentAndExpandsChildren(t *testing.T) {
	store := newLocationInspectionStore([]model.LocationState{{
		ID:          "town",
		ProjectID:   "project_1",
		Name:        "Harbor Town",
		Type:        "settlement",
		Scale:       model.LocationScaleSettlement,
		DetailState: model.LocationDetailStub,
		Status:      "active",
	}})
	service := NewLocationInspectionService(store, fixedLocationClock{}, &sequentialLocationIDs{})

	result, err := service.EnsureReachableLocations(context.Background(), model.LocationReachabilityInput{
		ProjectID:         "project_1",
		CurrentLocationID: "town",
	})
	if err != nil {
		t.Fatalf("EnsureReachableLocations() error = %v", err)
	}
	if result.CurrentLocation.DetailState != model.LocationDetailInitialized {
		t.Fatalf("current detail_state = %q, want initialized", result.CurrentLocation.DetailState)
	}
	current := store.locations["town"]
	if current.Properties["children_expanded"] != true {
		t.Fatalf("expected current location to mark children_expanded, got %#v", current.Properties)
	}
	children := store.childrenOf("town")
	if len(children) != 6 {
		t.Fatalf("children = %d, want 6: %#v", len(children), children)
	}
}

func TestLocationInspectionIsIdempotentForInitializedLocation(t *testing.T) {
	store := newLocationInspectionStore([]model.LocationState{
		{
			ID:          "town",
			ProjectID:   "project_1",
			Name:        "Harbor Town",
			Scale:       model.LocationScaleSettlement,
			DetailState: model.LocationDetailInitialized,
			Properties:  map[string]any{"children_expanded": true},
		},
		{
			ID:               "market",
			ProjectID:        "project_1",
			ParentLocationID: "town",
			Name:             "Harbor Town Market",
			Type:             "market",
			Scale:            model.LocationScaleDistrict,
			DetailState:      model.LocationDetailStub,
		},
	})
	service := NewLocationInspectionService(store, fixedLocationClock{}, &sequentialLocationIDs{})

	first, err := service.InspectLocation(context.Background(), model.LocationInspectionInput{
		ProjectID:         "project_1",
		CurrentLocationID: "town",
		LocationID:        "market",
		Reason:            "find a contact",
	})
	if err != nil {
		t.Fatalf("first InspectLocation() error = %v", err)
	}
	second, err := service.InspectLocation(context.Background(), model.LocationInspectionInput{
		ProjectID:         "project_1",
		CurrentLocationID: "town",
		LocationID:        "market",
		Reason:            "repeat",
	})
	if err != nil {
		t.Fatalf("second InspectLocation() error = %v", err)
	}
	if first.InspectedLocation.ID != second.InspectedLocation.ID || first.InspectedLocation.Name != second.InspectedLocation.Name {
		t.Fatalf("inspect changed location identity: first=%#v second=%#v", first, second)
	}
	if second.InspectedLocation.DetailState != model.LocationDetailInitialized {
		t.Fatalf("detail_state = %q, want initialized", second.InspectedLocation.DetailState)
	}
	if store.upsertCalls != 1 {
		t.Fatalf("upsert calls = %d, want one initialization write", store.upsertCalls)
	}
}

func TestLocationInspectionRejectsUnreachableLocation(t *testing.T) {
	store := newLocationInspectionStore([]model.LocationState{
		{
			ID:               "current",
			ProjectID:        "project_1",
			ParentLocationID: "parent_a",
			Name:             "Current Room",
			Scale:            model.LocationScaleRoom,
			DetailState:      model.LocationDetailInitialized,
			Properties:       map[string]any{"children_expanded": true},
		},
		{
			ID:               "elsewhere",
			ProjectID:        "project_1",
			ParentLocationID: "parent_b",
			Name:             "Elsewhere",
			Scale:            model.LocationScaleRoom,
			DetailState:      model.LocationDetailStub,
			X:                100,
			Y:                100,
		},
	})
	service := NewLocationInspectionService(store, fixedLocationClock{}, &sequentialLocationIDs{})

	_, err := service.InspectLocation(context.Background(), model.LocationInspectionInput{
		ProjectID:         "project_1",
		CurrentLocationID: "current",
		LocationID:        "elsewhere",
	})
	if err == nil {
		t.Fatal("expected unreachable inspect to fail")
	}
}

type locationInspectionStore struct {
	forkActionStoryEventStore
	locations   map[string]model.LocationState
	upsertCalls int
}

func newLocationInspectionStore(locations []model.LocationState) *locationInspectionStore {
	store := &locationInspectionStore{locations: map[string]model.LocationState{}}
	for _, location := range locations {
		store.locations[location.ID] = location
	}
	return store
}

func (s *locationInspectionStore) ListLocationsByProjectID(context.Context, string) ([]model.LocationState, error) {
	out := make([]model.LocationState, 0, len(s.locations))
	for _, location := range s.locations {
		out = append(out, location)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func (s *locationInspectionStore) GetLocation(_ context.Context, _ string, locationID string) (model.LocationState, error) {
	location, ok := s.locations[locationID]
	if !ok {
		return model.LocationState{}, errors.New("location not found")
	}
	return location, nil
}

func (s *locationInspectionStore) ListLocationsByParentID(_ context.Context, _ string, parentID string) ([]model.LocationState, error) {
	return s.childrenOf(parentID), nil
}

func (s *locationInspectionStore) UpsertLocations(_ context.Context, projectID string, locations []model.LocationState) error {
	seen := map[string]struct{}{}
	for _, location := range locations {
		if _, ok := seen[location.ID]; ok {
			return errors.New("duplicate location in one upsert batch")
		}
		seen[location.ID] = struct{}{}
		if location.ProjectID == "" {
			location.ProjectID = projectID
		}
		s.locations[location.ID] = location
	}
	s.upsertCalls++
	return nil
}

func (s *locationInspectionStore) ListFactionInfluencesByProjectID(context.Context, string) ([]model.FactionInfluence, error) {
	return nil, nil
}

func (s *locationInspectionStore) childrenOf(parentID string) []model.LocationState {
	children := []model.LocationState{}
	for _, location := range s.locations {
		if location.ParentLocationID == parentID {
			children = append(children, location)
		}
	}
	sort.SliceStable(children, func(i, j int) bool { return children[i].ID < children[j].ID })
	return children
}

type fixedLocationClock struct{}

func (fixedLocationClock) Now() time.Time {
	return time.Date(2026, 6, 6, 10, 0, 0, 0, time.UTC)
}

type sequentialLocationIDs struct {
	next int
}

func (g *sequentialLocationIDs) New(prefix string) string {
	g.next++
	return prefix + "_" + strconv.Itoa(g.next)
}
