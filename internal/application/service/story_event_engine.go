package service

import (
	"sort"
	"time"

	"github.com/fishimei/NovelOS/internal/application/model"
)

type TimedAction struct {
	CharacterID    string
	LocationKey    string
	ParticipantIDs []string
	StartAt        time.Time
	ArriveAt       time.Time
	EffectAt       time.Time
	EndsAt         time.Time
}

type EventEngineResult struct {
	Clock      time.Time
	Collisions []time.Time
}

func ResolveEventClock(start time.Time, actions []TimedAction) EventEngineResult {
	ordered := append([]TimedAction(nil), actions...)
	sort.SliceStable(ordered, func(i, j int) bool {
		if ordered[i].EndsAt.Equal(ordered[j].EndsAt) {
			return ordered[i].CharacterID < ordered[j].CharacterID
		}
		return ordered[i].EndsAt.Before(ordered[j].EndsAt)
	})
	result := EventEngineResult{Clock: start}
	for i, action := range ordered {
		if action.EndsAt.After(result.Clock) {
			result.Clock = action.EndsAt
		}
		for j := i + 1; j < len(ordered); j++ {
			if t, ok := collisionAt(action, ordered[j]); ok {
				result.Collisions = append(result.Collisions, t)
			}
		}
	}
	sort.SliceStable(result.Collisions, func(i, j int) bool { return result.Collisions[i].Before(result.Collisions[j]) })
	return result
}

func collisionAt(a, b TimedAction) (time.Time, bool) {
	if a.LocationKey == "" || b.LocationKey == "" || a.LocationKey != b.LocationKey {
		return time.Time{}, false
	}
	start := maxTime(a.ArriveAt, b.ArriveAt)
	end := minTime(a.EndsAt, b.EndsAt)
	if start.Before(end) || start.Equal(end) {
		return start, true
	}
	return time.Time{}, false
}

func maxTime(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}

func minTime(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}

func timedActionFrom(action model.OngoingAction) TimedAction {
	return TimedAction{
		CharacterID:    action.CharacterID,
		LocationKey:    action.TargetLocationKey,
		ParticipantIDs: append([]string(nil), action.ParticipantIDs...),
		StartAt:        action.StartAt,
		ArriveAt:       action.ArriveAt,
		EffectAt:       action.EffectAt,
		EndsAt:         action.EndsAt,
	}
}

func StoryEventFromAction(branch model.Branch, action model.OngoingAction, parentEventID string) model.StoryEvent {
	return model.StoryEvent{
		ProjectID:     branch.ProjectID,
		SessionID:     branch.SessionID,
		BranchID:      branch.ID,
		ParentEventID: parentEventID,
		StoryTime:     action.StartAt,
		Kind:          model.EventKindActionScheduled,
		ActorIDs:      []string{action.CharacterID},
		LocationKey:   action.TargetLocationKey,
		ResourceKeys:  action.ResourceKeys,
		Summary:       action.Description,
		Payload:       map[string]any{"action": action},
		StateDelta:    model.EventStateDelta{},
	}
}
