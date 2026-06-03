package service

import (
	"sort"
	"strings"
	"time"

	"github.com/fishimei/NovelOS/internal/application/model"
)

type TimedAction struct {
	Action         model.OngoingAction
	CharacterID    string
	LocationKey    string
	ParticipantIDs []string
	ResourceKeys   []string
	StartAt        time.Time
	ArriveAt       time.Time
	EffectAt       time.Time
	EndsAt         time.Time
}

type ActionCollision struct {
	At      time.Time
	Actions []model.OngoingAction
}

type EventEngineResult struct {
	Clock            time.Time
	NextCompletion   time.Time
	Collisions       []time.Time
	ActionCollisions []ActionCollision
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
				result.ActionCollisions = append(result.ActionCollisions, ActionCollision{At: t, Actions: []model.OngoingAction{action.Action, ordered[j].Action}})
			}
		}
	}
	sort.SliceStable(result.Collisions, func(i, j int) bool { return result.Collisions[i].Before(result.Collisions[j]) })
	sort.SliceStable(result.ActionCollisions, func(i, j int) bool {
		if result.ActionCollisions[i].At.Equal(result.ActionCollisions[j].At) {
			return collisionActionKey(result.ActionCollisions[i].Actions) < collisionActionKey(result.ActionCollisions[j].Actions)
		}
		return result.ActionCollisions[i].At.Before(result.ActionCollisions[j].At)
	})
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
	if sameActionLocation(a, b) || actionTargetsCharacter(a, b.CharacterID) || actionTargetsCharacter(b, a.CharacterID) || actionsShareResource(a, b) {
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

func actionsShareResource(a, b TimedAction) bool {
	if len(a.ResourceKeys) == 0 || len(b.ResourceKeys) == 0 {
		return false
	}
	seen := make(map[string]struct{}, len(a.ResourceKeys))
	for _, key := range a.ResourceKeys {
		if key != "" {
			seen[key] = struct{}{}
		}
	}
	for _, key := range b.ResourceKeys {
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
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
		Action:         action,
		CharacterID:    action.CharacterID,
		LocationKey:    action.TargetLocationKey,
		ParticipantIDs: append([]string(nil), action.ParticipantIDs...),
		ResourceKeys:   append([]string(nil), action.ResourceKeys...),
		StartAt:        action.StartAt,
		ArriveAt:       action.ArriveAt,
		EffectAt:       action.EffectAt,
		EndsAt:         action.EndsAt,
	}
}

func collisionActionKey(actions []model.OngoingAction) string {
	ids := make([]string, 0, len(actions))
	for _, action := range actions {
		ids = append(ids, action.CharacterID)
	}
	sort.Strings(ids)
	return strings.Join(ids, ":")
}

func StoryEventFromAction(branch model.Branch, action model.OngoingAction, parentEventID string) model.StoryEvent {
	if action.ID == "" {
		action.ID = fallbackActionID(branch, action)
	}
	stateDelta := model.EventStateDelta{}
	if actionArrivesImmediately(action) {
		stateDelta.CharacterMoves = []model.CharacterMove{{
			CharacterID: action.CharacterID,
			LocationKey: action.TargetLocationKey,
		}}
	}
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
		StateDelta:    stateDelta,
	}
}

func fallbackActionID(branch model.Branch, action model.OngoingAction) string {
	parts := []string{"action", branch.ID, action.CharacterID, action.StartAt.UTC().Format(time.RFC3339Nano), action.ActionType}
	return strings.Join(parts, ":")
}

func actionArrivesImmediately(action model.OngoingAction) bool {
	return action.ArriveAt.IsZero() || action.StartAt.IsZero() || !action.ArriveAt.After(action.StartAt)
}

func StoryEventFromActionCompletion(branch model.Branch, action model.OngoingAction, parentEventID string) model.StoryEvent {
	completed := action
	completed.Status = "completed"
	return model.StoryEvent{
		ProjectID:     branch.ProjectID,
		SessionID:     branch.SessionID,
		BranchID:      branch.ID,
		ParentEventID: parentEventID,
		StoryTime:     action.EndsAt,
		Kind:          model.EventKindActionCompleted,
		ActorIDs:      []string{action.CharacterID},
		LocationKey:   action.TargetLocationKey,
		ResourceKeys:  action.ResourceKeys,
		Summary:       action.Description,
		Payload:       map[string]any{"action": completed},
		StateDelta:    model.EventStateDelta{},
	}
}

func StoryEventFromActionSuperseded(branch model.Branch, action model.OngoingAction, parentEventID string, at time.Time, reason string) model.StoryEvent {
	superseded := action
	superseded.Status = "superseded"
	return model.StoryEvent{
		ProjectID:     branch.ProjectID,
		SessionID:     branch.SessionID,
		BranchID:      branch.ID,
		ParentEventID: parentEventID,
		StoryTime:     at,
		Kind:          model.EventKindActionSuperseded,
		ActorIDs:      []string{action.CharacterID},
		LocationKey:   action.TargetLocationKey,
		ResourceKeys:  action.ResourceKeys,
		Summary:       action.Description,
		Payload: map[string]any{
			"action": superseded,
			"reason": reason,
		},
		StateDelta: model.EventStateDelta{},
	}
}
