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
	if want := start.Add(time.Hour); !result.NextCompletion.Equal(want) {
		t.Fatalf("next completion = %s, want %s", result.NextCompletion, want)
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

func TestResolveEventClockDetectsResourceCollisionAcrossLocations(t *testing.T) {
	start := time.Date(2026, 6, 1, 11, 0, 0, 0, time.UTC)
	result := ResolveEventClock(start, []TimedAction{
		{CharacterID: "A", LocationKey: "loc:tavern", ResourceKeys: []string{"world:gate_open"}, StartAt: start, ArriveAt: start, EndsAt: start.Add(2 * time.Hour)},
		{CharacterID: "B", LocationKey: "loc:dock", ResourceKeys: []string{"world:gate_open"}, StartAt: start.Add(10 * time.Minute), ArriveAt: start.Add(20 * time.Minute), EndsAt: start.Add(time.Hour)},
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
		ResourceKeys:      []string{"world:gate_open"},
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
	timed.ResourceKeys[0] = "mutated"
	if action.ResourceKeys[0] != "world:gate_open" {
		t.Fatal("timed action resource keys should not alias source action")
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

func TestOngoingActionFromPlanPreservesExplicitTimesAndResources(t *testing.T) {
	start := time.Date(2026, 6, 1, 11, 0, 0, 0, time.UTC)
	arrive := start.Add(30 * time.Minute)
	effect := start.Add(time.Hour)
	end := start.Add(3 * time.Hour)
	action := ongoingActionFromPlan(model.StoryEventPlan{
		ID:            "plan_1",
		CharacterID:   "A",
		LocationKey:   "loc:tavern",
		ActionType:    "action",
		Summary:       "A travels to the tavern",
		DurationHours: 1,
		ArriveAt:      &arrive,
		EffectAt:      &effect,
		EndsAt:        &end,
		ResourceKeys:  []string{"world:gate_open"},
	}, start)

	if action.ID != "plan_1" || !action.ArriveAt.Equal(arrive) || !action.EffectAt.Equal(effect) || !action.EndsAt.Equal(end) {
		t.Fatalf("explicit timing not preserved: %#v", action)
	}
	if !hasString(action.ResourceKeys, "world:gate_open") {
		t.Fatalf("explicit resource key missing: %#v", action.ResourceKeys)
	}
}

func TestStoryEventFromActionDelaysMoveUntilArrival(t *testing.T) {
	start := time.Date(2026, 6, 1, 11, 0, 0, 0, time.UTC)
	event := StoryEventFromAction(model.Branch{ID: "branch_1", ProjectID: "project_1", SessionID: "story_1"}, model.OngoingAction{
		CharacterID:       "A",
		ActionType:        "travel",
		Description:       "A travels",
		TargetLocationKey: "loc:tavern",
		StartAt:           start,
		ArriveAt:          start.Add(time.Hour),
		EndsAt:            start.Add(2 * time.Hour),
		Status:            "ongoing",
	}, "parent_1")

	if len(event.StateDelta.CharacterMoves) != 0 {
		t.Fatalf("scheduled travel should not move immediately: %#v", event.StateDelta.CharacterMoves)
	}
	action, ok := event.Payload["action"].(model.OngoingAction)
	if !ok || action.ID == "" {
		t.Fatalf("scheduled action should carry action id: %#v", event.Payload)
	}
}

func TestValidateRunResultActionTerminalsRejectsOverlap(t *testing.T) {
	action := model.OngoingAction{ID: "action_1", CharacterID: "A"}
	err := validateRunResultActionTerminals(model.StoryRunResult{
		CompletedActions:  []model.OngoingAction{action},
		SupersededActions: []model.OngoingAction{action},
	})
	if err == nil {
		t.Fatal("expected overlapping terminal action to be rejected")
	}
}

func TestStoryEventFromActionCompletionBuildsCompletedPayload(t *testing.T) {
	start := time.Date(2026, 6, 1, 11, 0, 0, 0, time.UTC)
	action := model.OngoingAction{
		CharacterID:       "A",
		ActionType:        "observe",
		Description:       "A watches for B",
		TargetLocationKey: "loc:tavern",
		StartAt:           start,
		EndsAt:            start.Add(time.Hour),
		ResourceKeys:      []string{"character:A"},
		Status:            "ongoing",
	}

	event := StoryEventFromActionCompletion(model.Branch{ID: "branch_1", ProjectID: "project_1", SessionID: "story_1"}, action, "event_parent")

	if event.Kind != model.EventKindActionCompleted || !event.StoryTime.Equal(action.EndsAt) {
		t.Fatalf("unexpected completion event: %#v", event)
	}
	completed, ok := event.Payload["action"].(model.OngoingAction)
	if !ok {
		t.Fatalf("completion action payload missing: %#v", event.Payload)
	}
	if completed.Status != "completed" || action.Status != "ongoing" {
		t.Fatalf("completion status not isolated: completed=%#v source=%#v", completed, action)
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

func TestSceneTimeFromScheduledActionsUsesNextCompletionWithoutCollision(t *testing.T) {
	start := time.Date(2026, 6, 1, 11, 0, 0, 0, time.UTC)
	sceneTime := sceneTimeFromScheduledActions(start, []TimedAction{
		{CharacterID: "A", LocationKey: "loc:tavern", StartAt: start, ArriveAt: start, EndsAt: start.Add(time.Hour)},
		{CharacterID: "B", LocationKey: "loc:dock", StartAt: start, ArriveAt: start, EndsAt: start.Add(2 * time.Hour)},
	}, start)

	if want := start.Add(time.Hour); !sceneTime.Equal(want) {
		t.Fatalf("scene time = %s, want next completion %s", sceneTime, want)
	}
}

func TestSceneTimeFromScheduledActionsUsesCompletionBeforeLaterCollision(t *testing.T) {
	start := time.Date(2026, 6, 1, 11, 0, 0, 0, time.UTC)
	sceneTime := sceneTimeFromScheduledActions(start, []TimedAction{
		{CharacterID: "A", LocationKey: "loc:tavern", StartAt: start, ArriveAt: start, EndsAt: start.Add(time.Hour)},
		{CharacterID: "B", LocationKey: "loc:dock", StartAt: start, ArriveAt: start.Add(90 * time.Minute), EndsAt: start.Add(3 * time.Hour)},
		{CharacterID: "C", LocationKey: "loc:dock", StartAt: start, ArriveAt: start.Add(90 * time.Minute), EndsAt: start.Add(3 * time.Hour)},
	}, start)

	if want := start.Add(time.Hour); !sceneTime.Equal(want) {
		t.Fatalf("scene time = %s, want earliest completion before later collision %s", sceneTime, want)
	}
}

func TestActionsCompletedAtSelectsSameTick(t *testing.T) {
	start := time.Date(2026, 6, 1, 11, 0, 0, 0, time.UTC)
	completed := actionsCompletedAt([]model.OngoingAction{
		{CharacterID: "A", EndsAt: start.Add(time.Hour)},
		{CharacterID: "B", EndsAt: start.Add(2 * time.Hour)},
		{CharacterID: "C", EndsAt: start.Add(time.Hour)},
	}, start.Add(time.Hour))

	if len(completed) != 2 || completed[0].CharacterID != "A" || completed[1].CharacterID != "C" {
		t.Fatalf("completed actions = %#v, want A and C", completed)
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
