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
	Clock          time.Time
	NextCompletion time.Time
	Collisions     []time.Time
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
		if actionCompletionIsQueued(start, action) && (result.NextCompletion.IsZero() || action.EndsAt.Before(result.NextCompletion)) {
			result.NextCompletion = action.EndsAt
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

func actionCompletionIsQueued(start time.Time, action TimedAction) bool {
	return !action.EndsAt.IsZero() && (action.EndsAt.After(start) || action.EndsAt.Equal(start))
}

func collisionAt(a, b TimedAction) (time.Time, bool) {
	start, _, ok := collisionWindow(a, b)
	if !ok {
		return time.Time{}, false
	}
	if sameActionLocation(a, b) || actionTargetsCharacter(a, b.CharacterID) || actionTargetsCharacter(b, a.CharacterID) {
		return start, true
	}
	return time.Time{}, false
}

func collisionWindow(a, b TimedAction) (time.Time, time.Time, bool) {
	start := maxTime(actionCollisionStart(a), actionCollisionStart(b))
	end := minTime(a.EndsAt, b.EndsAt)
	if start.Before(end) || start.Equal(end) {
		return start, end, true
	}
	return time.Time{}, time.Time{}, false
}

func actionCollisionStart(action TimedAction) time.Time {
	if !action.ArriveAt.IsZero() {
		return action.ArriveAt
	}
	return action.StartAt
}

func sameActionLocation(a, b TimedAction) bool {
	return a.LocationKey != "" && b.LocationKey != "" && a.LocationKey == b.LocationKey
}

func actionTargetsCharacter(action TimedAction, characterID string) bool {
	if characterID == "" {
		return false
	}
	for _, participantID := range action.ParticipantIDs {
		if participantID == characterID {
			return true
		}
	}
	return false
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
