package eino

import (
	"fmt"
	"sort"
	"strings"

	"github.com/fishimei/NovelOS/internal/application/model"
	"github.com/fishimei/NovelOS/internal/application/port"
)

const (
	defaultMaxRenderParticipants = 5
	defaultMaxSecondarySummaries = 8
	defaultMaxAmbientActors      = 3
)

type characterSceneCandidate struct {
	characterID string
	tier        string
	locationKey string
	score       int
	reasons     []string
}

func selectCharacterAttention(input port.StoryRunGenerationInput, snapshot StoryContextSnapshot, world model.WorldSnapshot, variable StoryVariablePlan) model.CharacterAttentionSelection {
	if len(input.WakeCharacterIDs) > 0 {
		ids := validExistingCharacterIDs(snapshot.Characters, input.WakeCharacterIDs)
		return model.CharacterAttentionSelection{PrimaryDecisionIDs: ids, SceneCandidateIDs: ids, Rationale: "explicit wake_character_ids"}
	}
	focusLocations := focusLocationKeys(input, world, variable)
	primary := make([]string, 0)
	secondary := make([]string, 0)
	candidates := make([]characterSceneCandidate, 0, len(snapshot.Characters))
	suppressed := make([]string, 0)
	ambient := make([]model.AmbientCharacterHint, 0)
	for _, character := range snapshot.Characters {
		if character.ID == "" {
			continue
		}
		state := runtimeStateForCharacter(world, character)
		if !storyCharacterActive(character, state) {
			suppressed = appendUniqueString(suppressed, character.ID)
			continue
		}
		tier := normalizeCharacterTier(state.Tier, character.Role)
		score, reasons := scoreCharacterForScene(character, state, tier, focusLocations, variable, input)
		switch tier {
		case model.CharacterTierPrimary:
			if characterStateIsIdle(state, world.StoryTime) {
				primary = appendUniqueString(primary, character.ID)
			}
			candidates = append(candidates, characterSceneCandidate{characterID: character.ID, tier: tier, locationKey: state.LocationKey, score: score + 100, reasons: append(reasons, "tier_1")})
		case model.CharacterTierSecondary:
			if score > 0 || state.ContinuityImportance > 0 || state.OngoingAction != nil {
				secondary = appendUniqueString(secondary, character.ID)
				candidates = append(candidates, characterSceneCandidate{characterID: character.ID, tier: tier, locationKey: state.LocationKey, score: score + 35 + state.ContinuityImportance, reasons: append(reasons, "tier_2_relevant")})
			} else {
				suppressed = appendUniqueString(suppressed, character.ID)
			}
		default:
			if score >= 60 && len(ambient) < defaultMaxAmbientActors {
				ambient = append(ambient, model.AmbientCharacterHint{TemporaryID: "ambient_" + character.ID, Name: character.Name, Role: firstText(character.Role, "背景人物"), LocationKey: state.LocationKey, Description: character.Profile, Source: "tier_3_character"})
			} else {
				suppressed = appendUniqueString(suppressed, character.ID)
			}
		}
	}
	sort.SliceStable(candidates, func(i, j int) bool { return candidates[i].score > candidates[j].score })
	sceneCandidates := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		sceneCandidates = appendUniqueString(sceneCandidates, candidate.characterID)
	}
	secondary = firstStrings(secondary, defaultMaxSecondarySummaries)
	return model.CharacterAttentionSelection{
		PrimaryDecisionIDs:     primary,
		SecondarySummaryIDs:    secondary,
		SceneCandidateIDs:      sceneCandidates,
		AmbientCandidateHints:  ambient,
		SuppressedCharacterIDs: suppressed,
		Rationale:              fmt.Sprintf("tier-aware selection: primary=%d secondary=%d candidates=%d ambient=%d", len(primary), len(secondary), len(sceneCandidates), len(ambient)),
	}
}

func storyCharacterActive(character model.Character, state model.CharacterRuntimeState) bool {
	if strings.EqualFold(strings.TrimSpace(character.Status), "inactive") || strings.EqualFold(strings.TrimSpace(state.Status), "inactive") {
		return false
	}
	return true
}

func runtimeStateForCharacter(world model.WorldSnapshot, character model.Character) model.CharacterRuntimeState {
	state := world.Characters[character.ID]
	if state.CharacterID == "" {
		state.CharacterID = character.ID
		state.Status = firstText(character.Status, "active")
	}
	if state.Tier == "" {
		state.Tier = inferStoryCharacterTier(character)
	}
	return state
}

func normalizeCharacterTier(tier string, role string) string {
	tier = strings.ToLower(strings.TrimSpace(tier))
	switch tier {
	case model.CharacterTierPrimary, "primary", "main", "core":
		return model.CharacterTierPrimary
	case model.CharacterTierSecondary, "secondary", "supporting":
		return model.CharacterTierSecondary
	case model.CharacterTierAmbient, "ambient", "minor", "background":
		return model.CharacterTierAmbient
	}
	return inferTierFromRole(role)
}

func inferStoryCharacterTier(character model.Character) string {
	return inferTierFromRole(character.Role)
}

func inferTierFromRole(role string) string {
	role = strings.ToLower(strings.TrimSpace(role))
	switch {
	case strings.Contains(role, "protagonist"), strings.Contains(role, "主角"), strings.Contains(role, "lead"), strings.Contains(role, "核心反派"), strings.Contains(role, "最终"), strings.Contains(role, "boss"):
		return model.CharacterTierPrimary
	case strings.Contains(role, "antagonist"), strings.Contains(role, "反派"), strings.Contains(role, "villain"):
		return model.CharacterTierPrimary
	case strings.Contains(role, "minor"), strings.Contains(role, "background"), strings.Contains(role, "路人"), strings.Contains(role, "背景"):
		return model.CharacterTierAmbient
	default:
		return model.CharacterTierSecondary
	}
}

func scoreCharacterForScene(character model.Character, state model.CharacterRuntimeState, tier string, focusLocations []string, variable StoryVariablePlan, input port.StoryRunGenerationInput) (int, []string) {
	score := 0
	reasons := make([]string, 0)
	if state.LocationKey != "" && containsString(focusLocations, state.LocationKey) {
		score += 60
		reasons = append(reasons, "focus_location")
	}
	if containsString(variable.PlotVariable.RelatedCharacterIDs, character.ID) || variable.PlotVariable.FocalCharacterID == character.ID {
		score += 45
		reasons = append(reasons, "plot_variable")
	}
	if actionListTargetsCharacter(input.InFlightActions, character.ID) || actionListTargetsCharacter(input.CompletedActions, character.ID) || actionListTargetsCharacter(input.SupersededActions, character.ID) {
		score += 35
		reasons = append(reasons, "action_related")
	}
	if state.OngoingAction != nil {
		score += 20
		reasons = append(reasons, "ongoing_action")
	}
	if state.AttentionScore > 0 {
		score += state.AttentionScore
		reasons = append(reasons, "attention_score")
	}
	if tier == model.CharacterTierAmbient {
		score -= 20
	}
	return score, reasons
}

func focusLocationKeys(input port.StoryRunGenerationInput, world model.WorldSnapshot, variable StoryVariablePlan) []string {
	keys := make([]string, 0)
	for _, action := range input.CompletedActions {
		keys = appendUniqueString(keys, action.TargetLocationKey)
	}
	for _, action := range input.SupersededActions {
		keys = appendUniqueString(keys, action.TargetLocationKey)
	}
	for _, state := range world.Characters {
		if variable.PlotVariable.FocalCharacterID != "" && state.CharacterID == variable.PlotVariable.FocalCharacterID {
			keys = appendUniqueString(keys, state.LocationKey)
		}
	}
	for _, state := range world.Characters {
		keys = appendUniqueString(keys, state.LocationKey)
		if len(keys) > 0 {
			break
		}
	}
	return keys
}

func actionListTargetsCharacter(actions []model.OngoingAction, characterID string) bool {
	for _, action := range actions {
		if action.CharacterID == characterID || containsString(action.ParticipantIDs, characterID) {
			return true
		}
	}
	return false
}

func validExistingCharacterIDs(characters []model.Character, ids []string) []string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		for _, character := range characters {
			if character.ID == id || character.Name == id {
				out = appendUniqueString(out, character.ID)
				break
			}
		}
	}
	return out
}
