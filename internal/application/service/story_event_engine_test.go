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
