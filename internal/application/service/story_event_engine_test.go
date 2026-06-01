package service

import (
	"testing"
	"time"
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
