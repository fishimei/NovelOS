package eino

import (
	"fmt"
	"sort"
	"strings"

	"github.com/fishimei/NovelOS/internal/application/model"
	"github.com/fishimei/NovelOS/internal/application/port"
)

func summarizeSecondaryCharacters(input port.StoryRunGenerationInput, snapshot StoryContextSnapshot, world model.WorldSnapshot, ids []string, variable StoryVariablePlan) []model.SecondaryActionSummary {
	summaries := make([]model.SecondaryActionSummary, 0, len(ids))
	for _, id := range ids {
		character := characterByID(snapshot.Characters, id)
		if character.ID == "" {
			continue
		}
		state := runtimeStateForCharacter(world, character)
		score, signals := scoreCharacterForScene(character, state, model.CharacterTierSecondary, focusLocationKeys(input, world, variable), variable, input)
		targetLocation := state.LocationKey
		if state.OngoingAction != nil && strings.TrimSpace(state.OngoingAction.TargetLocationKey) != "" {
			targetLocation = state.OngoingAction.TargetLocationKey
		}
		summary := model.SecondaryActionSummary{
			CharacterID:        character.ID,
			CharacterName:      character.Name,
			CurrentLocationKey: state.LocationKey,
			TargetLocationKey:  targetLocation,
			StatusSummary:      secondaryStatusSummary(character, state),
			IntentSummary:      secondaryIntentSummary(character, state, variable),
			RelevantSignals:    signals,
			MayEnterScene:      score > 0 || state.OngoingAction != nil,
			Rationale:          fmt.Sprintf("tier_2 off-screen summary, relevance_score=%d", score),
		}
		if state.OngoingAction != nil {
			summary.ParticipantIDs = uniqueStoryIDs(state.OngoingAction.ParticipantIDs)
			summary.ResourceKeys = uniqueStoryIDs(state.OngoingAction.ResourceKeys)
		}
		summaries = append(summaries, summary)
	}
	return summaries
}

func arrangeScene(input port.StoryRunGenerationInput, snapshot StoryContextSnapshot, world model.WorldSnapshot, variable StoryVariablePlan, attention model.CharacterAttentionSelection, primaryActions []ScenePlannedAction, secondarySummaries []model.SecondaryActionSummary) model.SceneArrangement {
	focusKey, focusName := chooseFocusLocation(world, variable, primaryActions, secondarySummaries, input)
	render := make([]string, 0, defaultMaxRenderParticipants)
	for _, action := range primaryActions {
		if focusKey == "" || action.TargetLocationKey == "" || action.TargetLocationKey == focusKey {
			render = appendUniqueString(render, action.CharacterID)
			for _, participantID := range action.ParticipantIDs {
				render = appendUniqueString(render, participantID)
			}
		}
	}
	for _, id := range attention.SceneCandidateIDs {
		if len(render) >= defaultMaxRenderParticipants {
			break
		}
		character := characterByID(snapshot.Characters, id)
		if character.ID == "" {
			continue
		}
		state := runtimeStateForCharacter(world, character)
		if normalizeCharacterTier(state.Tier, character.Role) == model.CharacterTierAmbient {
			continue
		}
		if focusKey == "" || state.LocationKey == focusKey || containsString(variable.PlotVariable.RelatedCharacterIDs, id) || variable.PlotVariable.FocalCharacterID == id {
			render = appendUniqueString(render, id)
		}
	}
	if len(render) == 0 {
		render = firstStrings(attention.PrimaryDecisionIDs, defaultMaxRenderParticipants)
	}
	offscreen := make([]string, 0)
	for _, summary := range secondarySummaries {
		if !containsString(render, summary.CharacterID) {
			offscreen = appendUniqueString(offscreen, summary.CharacterID)
		}
	}
	interactionCandidates := buildArrangementInteractionCandidates(focusKey, render, primaryActions, secondarySummaries)
	ambient := attention.AmbientCandidateHints
	if len(ambient) > defaultMaxAmbientActors {
		ambient = ambient[:defaultMaxAmbientActors]
	}
	return model.SceneArrangement{
		FocusLocationKey:      focusKey,
		FocusLocationName:     focusName,
		NearbyLocationKeys:    nearbyLocationKeysForFocus(world, focusKey),
		RenderParticipantIDs:  render,
		OffscreenCharacterIDs: offscreen,
		AmbientActors:         ambient,
		InteractionCandidates: interactionCandidates,
		SelectedReason:        fmt.Sprintf("arranged from tier-aware candidates around location %s", focusKey),
	}
}

func chooseFocusLocation(world model.WorldSnapshot, variable StoryVariablePlan, primaryActions []ScenePlannedAction, secondarySummaries []model.SecondaryActionSummary, input port.StoryRunGenerationInput) (string, string) {
	for _, action := range primaryActions {
		if strings.TrimSpace(action.TargetLocationKey) != "" {
			return action.TargetLocationKey, locationNameByKey(world.Locations, action.TargetLocationKey)
		}
	}
	for _, summary := range secondarySummaries {
		if strings.TrimSpace(summary.TargetLocationKey) != "" && summary.MayEnterScene {
			return summary.TargetLocationKey, locationNameByKey(world.Locations, summary.TargetLocationKey)
		}
	}
	for _, action := range input.CompletedActions {
		if strings.TrimSpace(action.TargetLocationKey) != "" {
			return action.TargetLocationKey, locationNameByKey(world.Locations, action.TargetLocationKey)
		}
	}
	if variable.PlotVariable.FocalCharacterID != "" {
		state := world.Characters[variable.PlotVariable.FocalCharacterID]
		if state.LocationKey != "" {
			return state.LocationKey, locationNameByKey(world.Locations, state.LocationKey)
		}
	}
	for _, state := range world.Characters {
		if state.LocationKey != "" {
			return state.LocationKey, locationNameByKey(world.Locations, state.LocationKey)
		}
	}
	return "", ""
}

func nearbyLocationKeysForFocus(world model.WorldSnapshot, focusKey string) []string {
	focus := locationByKey(world.Locations, focusKey)
	if focus.ID == "" {
		return nil
	}
	type nearby struct {
		key      string
		distance float64
	}
	items := make([]nearby, 0)
	for _, location := range world.Locations {
		if location.ID == "" || location.ID == focusKey {
			continue
		}
		items = append(items, nearby{key: location.ID, distance: locationDistance(focus, location)})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].distance < items[j].distance })
	keys := make([]string, 0, minInt(len(items), 5))
	for _, item := range firstEntries(items, 5) {
		keys = append(keys, item.key)
	}
	return keys
}

func buildArrangementInteractionCandidates(focusKey string, render []string, primaryActions []ScenePlannedAction, secondarySummaries []model.SecondaryActionSummary) []model.InteractionCandidate {
	if len(render) < 2 || focusKey == "" {
		return nil
	}
	candidates := []model.InteractionCandidate{{LocationKey: focusKey, CharacterIDs: append([]string(nil), render...), Reason: "selected render participants share the scene focus", ConflictType: "scene_focus", Priority: 1}}
	for _, action := range primaryActions {
		ids := append([]string{action.CharacterID}, action.ParticipantIDs...)
		ids = uniqueStoryIDs(ids)
		if len(ids) >= 2 {
			candidates = append(candidates, model.InteractionCandidate{LocationKey: firstText(action.TargetLocationKey, focusKey), CharacterIDs: ids, Reason: action.Rationale, ConflictType: "planned_action", Priority: 2})
		}
	}
	for _, summary := range secondarySummaries {
		ids := append([]string{summary.CharacterID}, summary.ParticipantIDs...)
		ids = uniqueStoryIDs(ids)
		if summary.MayEnterScene && len(ids) >= 2 {
			candidates = append(candidates, model.InteractionCandidate{LocationKey: firstText(summary.TargetLocationKey, focusKey), CharacterIDs: ids, Reason: summary.Rationale, ConflictType: "secondary_action", Priority: 3})
		}
	}
	return candidates
}

func secondaryStatusSummary(character model.Character, state model.CharacterRuntimeState) string {
	if state.OngoingAction != nil && state.OngoingAction.Description != "" {
		return state.OngoingAction.Description
	}
	return fmt.Sprintf("%s 维持当前状态，仍在世界中按既有目标行动。", firstText(character.Name, character.ID))
}

func secondaryIntentSummary(character model.Character, state model.CharacterRuntimeState, variable StoryVariablePlan) string {
	if len(character.Goals) > 0 {
		return character.Goals[0]
	}
	return firstText(variable.PlotVariable.CoreChoice, "保持生存并观察局势")
}

func locationNameByKey(locations []model.LocationState, key string) string {
	location := locationByKey(locations, key)
	return location.Name
}
