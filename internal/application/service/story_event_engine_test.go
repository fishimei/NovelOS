package service

import (
	"testing"
	"time"

	"github.com/fishimei/NovelOS/internal/application/model"
)

func TestResolveEventClockUsesActionEnds(t *testing.T) {
	start := time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC)
	result := ResolveEventClock(start, []TimedAction{
		{CharacterID: "a", LocationKey: "loc", StartAt: start, ArriveAt: start, EffectAt: start.Add(time.Hour), EndsAt: start.Add(time.Hour)},
		{CharacterID: "b", LocationKey: "loc2", StartAt: start.Add(time.Hour), ArriveAt: start.Add(time.Hour), EffectAt: start.Add(3 * time.Hour), EndsAt: start.Add(3 * time.Hour)},
		{CharacterID: "c", LocationKey: "loc3", StartAt: start.Add(3 * time.Hour), ArriveAt: start.Add(3 * time.Hour), EffectAt: start.Add(210 * time.Minute), EndsAt: start.Add(210 * time.Minute)},
	})
	if want := start.Add(210 * time.Minute); !result.Clock.Equal(want) {
		t.Fatalf("clock = %s, want %s", result.Clock, want)
	}
}

func TestResolveEventClockDetectsAppendixCollision(t *testing.T) {
	start := time.Date(2026, 6, 1, 11, 0, 0, 0, time.UTC)
	result := ResolveEventClock(start, []TimedAction{
		{CharacterID: "A", LocationKey: "loc:tavern", StartAt: start, ArriveAt: start, EndsAt: start.Add(time.Hour)},
		{CharacterID: "B", LocationKey: "loc:tavern", StartAt: start.Add(10 * time.Minute), ArriveAt: start.Add(30 * time.Minute), EndsAt: start.Add(50 * time.Minute)},
	})
	if len(result.Collisions) != 1 {
		t.Fatalf("collisions = %d, want 1", len(result.Collisions))
	}
	if want := start.Add(30 * time.Minute); !result.Collisions[0].Equal(want) {
		t.Fatalf("collision = %s, want %s", result.Collisions[0], want)
	}
}

func TestResolveEventClockDetectsIntentCollisionAcrossLocations(t *testing.T) {
	start := time.Date(2026, 6, 1, 11, 0, 0, 0, time.UTC)
	result := ResolveEventClock(start, []TimedAction{
		{CharacterID: "A", LocationKey: "loc:tavern", ParticipantIDs: []string{"B"}, StartAt: start, ArriveAt: start.Add(20 * time.Minute), EndsAt: start.Add(2 * time.Hour)},
		{CharacterID: "B", LocationKey: "loc:dock", StartAt: start, ArriveAt: start, EndsAt: start.Add(time.Hour)},
		{CharacterID: "C", LocationKey: "loc:gate", StartAt: start, ArriveAt: start, EndsAt: start.Add(time.Hour)},
	})
	if len(result.Collisions) != 1 {
		t.Fatalf("collisions = %d, want 1", len(result.Collisions))
	}
	if want := start.Add(20 * time.Minute); !result.Collisions[0].Equal(want) {
		t.Fatalf("collision = %s, want %s", result.Collisions[0], want)
	}
}

func TestResolveEventClockIgnoresUnrelatedCrossLocationActions(t *testing.T) {
	start := time.Date(2026, 6, 1, 11, 0, 0, 0, time.UTC)
	result := ResolveEventClock(start, []TimedAction{
		{CharacterID: "A", LocationKey: "loc:tavern", ParticipantIDs: []string{"B"}, StartAt: start, ArriveAt: start, EndsAt: start.Add(time.Hour)},
		{CharacterID: "C", LocationKey: "loc:dock", StartAt: start, ArriveAt: start, EndsAt: start.Add(time.Hour)},
	})
	if len(result.Collisions) != 0 {
		t.Fatalf("collisions = %d, want 0", len(result.Collisions))
	}
}

func TestResolveEventClockNoCollisionAfterActionEnds(t *testing.T) {
	start := time.Date(2026, 6, 1, 11, 0, 0, 0, time.UTC)
	result := ResolveEventClock(start, []TimedAction{
		{CharacterID: "A", LocationKey: "loc:tavern", StartAt: start, ArriveAt: start, EndsAt: start.Add(time.Hour)},
		{CharacterID: "B", LocationKey: "loc:tavern", StartAt: start.Add(70 * time.Minute), ArriveAt: start.Add(90 * time.Minute), EndsAt: start.Add(110 * time.Minute)},
	})
	if len(result.Collisions) != 0 {
		t.Fatalf("collisions = %d, want 0", len(result.Collisions))
	}
}

func TestTimedActionFromOngoingActionCarriesIntent(t *testing.T) {
	start := time.Date(2026, 6, 1, 11, 0, 0, 0, time.UTC)
	action := model.OngoingAction{
		CharacterID:       "A",
		TargetLocationKey: "loc:tavern",
		ParticipantIDs:    []string{"B"},
		StartAt:           start,
		ArriveAt:          start.Add(10 * time.Minute),
		EffectAt:          start.Add(20 * time.Minute),
		EndsAt:            start.Add(time.Hour),
	}

	timed := timedActionFrom(action)
	if timed.CharacterID != action.CharacterID || timed.LocationKey != action.TargetLocationKey {
		t.Fatalf("timed action identity = (%q, %q), want (%q, %q)", timed.CharacterID, timed.LocationKey, action.CharacterID, action.TargetLocationKey)
	}
	if len(timed.ParticipantIDs) != 1 || timed.ParticipantIDs[0] != "B" {
		t.Fatalf("participant ids = %#v, want [B]", timed.ParticipantIDs)
	}
	timed.ParticipantIDs[0] = "mutated"
	if action.ParticipantIDs[0] != "B" {
		t.Fatal("timed action participant ids should not alias source action")
	}
}

func TestOngoingActionFromPlanBuildsActionPayload(t *testing.T) {
	start := time.Date(2026, 6, 1, 11, 0, 0, 0, time.UTC)
	action := ongoingActionFromPlan(model.StoryEventPlan{
		CharacterID:    "A",
		LocationKey:    "loc:tavern",
		ActionType:     "observe",
		Summary:        "A watches for B",
		Intent:         "confirm B's arrival",
		DurationHours:  2,
		TargetActorIDs: []string{"B", "B"},
	}, start)

	if action.CharacterID != "A" || action.ActionType != "observe" || action.TargetLocationKey != "loc:tavern" {
		t.Fatalf("unexpected action identity: %#v", action)
	}
	if len(action.ParticipantIDs) != 1 || action.ParticipantIDs[0] != "B" {
		t.Fatalf("participant ids = %#v, want [B]", action.ParticipantIDs)
	}
	if !action.EndsAt.Equal(start.Add(2*time.Hour)) || action.Status != "ongoing" {
		t.Fatalf("unexpected action timing/status: %#v", action)
	}
	if !hasString(action.ResourceKeys, "character:A") || !hasString(action.ResourceKeys, "character:B") || !hasString(action.ResourceKeys, "location:loc:tavern") {
		t.Fatalf("resource keys missing action targets: %#v", action.ResourceKeys)
	}
}

func TestStoryEventPlanTimeUsesBaseClock(t *testing.T) {
	base := time.Date(2026, 6, 1, 11, 0, 0, 0, time.UTC)
	first := storyEventPlanTime(base, model.StoryEventPlan{TimeIndex: 1})
	second := storyEventPlanTime(base, model.StoryEventPlan{TimeIndex: 2})

	if !first.Equal(base.Add(time.Hour)) {
		t.Fatalf("first event time = %s, want %s", first, base.Add(time.Hour))
	}
	if !second.Equal(base.Add(2 * time.Hour)) {
		t.Fatalf("second event time = %s, want %s", second, base.Add(2*time.Hour))
	}
}

func TestSceneTimeFromScheduledActionsUsesCollision(t *testing.T) {
	start := time.Date(2026, 6, 1, 11, 0, 0, 0, time.UTC)
	sceneTime := sceneTimeFromScheduledActions(start, []TimedAction{
		{CharacterID: "A", LocationKey: "loc:tavern", StartAt: start, ArriveAt: start.Add(15 * time.Minute), EndsAt: start.Add(time.Hour)},
		{CharacterID: "B", LocationKey: "loc:tavern", StartAt: start, ArriveAt: start.Add(30 * time.Minute), EndsAt: start.Add(time.Hour)},
	}, start)

	if want := start.Add(30 * time.Minute); !sceneTime.Equal(want) {
		t.Fatalf("scene time = %s, want collision %s", sceneTime, want)
	}
}

func TestSceneTimeFromScheduledActionsUsesEngineClockWithoutCollision(t *testing.T) {
	start := time.Date(2026, 6, 1, 11, 0, 0, 0, time.UTC)
	sceneTime := sceneTimeFromScheduledActions(start, []TimedAction{
		{CharacterID: "A", LocationKey: "loc:tavern", StartAt: start, ArriveAt: start, EndsAt: start.Add(time.Hour)},
		{CharacterID: "B", LocationKey: "loc:dock", StartAt: start, ArriveAt: start, EndsAt: start.Add(2 * time.Hour)},
	}, start)

	if want := start.Add(2 * time.Hour); !sceneTime.Equal(want) {
		t.Fatalf("scene time = %s, want engine clock %s", sceneTime, want)
	}
}

func hasString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
