package eino

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/fishimei/NovelOS/internal/application/model"
	"github.com/fishimei/NovelOS/internal/application/port"
)

func TestActionDecisionPromptProjectionStripsPersistenceFields(t *testing.T) {
	input := model.CharacterActionDecisionInput{
		World: model.WorldSnapshot{
			StoryTime: time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC),
			WorldState: map[string]model.WorldStateEntry{
				"gate_lockdown": {ID: "world_1", ProjectID: "project_1", Key: "gate_lockdown", Value: "closed", Note: "城门提前关闭", Status: "active", Importance: 5, Volatility: 4, UpdatedAt: time.Now()},
			},
			Relationships: map[string]model.Relationship{
				"pair_1": {
					Pair:  model.RelationshipPair{ID: "pair_1", ProjectID: "project_1", LeftCharacterID: "character_1", RightCharacterID: "character_2", Summary: "互相试探", TensionPoints: []string{"密信"}},
					Views: []model.RelationshipView{{SourceCharacterID: "character_1", TargetCharacterID: "character_2", PublicAttitude: "冷淡", PrivateAttitude: "警惕", BelievedTargetAttitude: "可能背叛", MaskingStrategy: "装作不知"}},
				},
			},
			Locations: []model.LocationState{{ID: "dock", ProjectID: "project_1", MapID: "map_1", RegionID: "region_1", Name: "旧码头", Type: "dock", Description: "雨夜码头", Status: "active", Properties: map[string]any{"noise": "rain"}}},
		},
		Character:       model.Character{ID: "character_1", ProjectID: "project_1", Name: "林澈", Personality: "谨慎", Goals: []string{"送出密信"}, Secrets: []string{"藏有副本"}, Status: "active", CreatedAt: time.Now(), UpdatedAt: time.Now()},
		CharacterState:  model.CharacterRuntimeState{CharacterID: "character_1", LocationKey: "dock", X: 1, Y: 2, Status: "active"},
		Location:        model.LocationState{ID: "dock", ProjectID: "project_1", Name: "旧码头", Type: "dock", Description: "雨夜码头", Status: "active"},
		NearbyLocations: []model.NearbyLocationContext{{Location: model.LocationState{ID: "gate", ProjectID: "project_1", Name: "城门"}, Distance: 3}},
	}
	sharedPayload, err := json.Marshal(buildActionDecisionSharedPrompt(input.World))
	if err != nil {
		t.Fatal(err)
	}
	characterPayload, err := json.Marshal(buildActionDecisionCharacterPrompt(input))
	if err != nil {
		t.Fatal(err)
	}
	payload := string(sharedPayload) + string(characterPayload)
	for _, forbidden := range []string{"project_id", "created_at", "updated_at", "status", "map_id", "region_id", "properties", "pair_1"} {
		if strings.Contains(payload, forbidden) {
			t.Fatalf("prompt payload leaked %q: %s", forbidden, payload)
		}
	}
	for _, required := range []string{"shared_context", "gate_lockdown", "character_1", "relationships", "private_facts", "target_location_key", "affected_resource_keys"} {
		if required == "shared_context" {
			continue
		}
		if !strings.Contains(payload, required) {
			t.Fatalf("prompt payload missing %q: %s", required, payload)
		}
	}
}

func TestScenePromptProjectionAddsPlannedActionContract(t *testing.T) {
	generator := &StoryRunGenerator{maxTurns: 5}
	ctx := generator.buildSceneContext(portStoryRunInputForPromptTest(), StoryContextSnapshot{
		Characters: []model.Character{{ID: "character_1", ProjectID: "project_1", Name: "林澈", VoiceStyle: "短句", Goals: []string{"送信"}, CreatedAt: time.Now()}},
		WorldState: []model.WorldStateEntry{{ID: "world_1", ProjectID: "project_1", Key: "rain", Note: "雨势遮住行踪", Status: "active", Importance: 4}},
	}, StoryVariablePlan{PlotVariable: StoryNarrativePlotVariable{RelatedCharacterIDs: []string{"character_1"}}}, []ScenePlannedAction{{CharacterID: "character_1", ActionType: "action", Description: "去旧码头", TargetLocationKey: "dock", DurationHours: 1}})
	payloadBytes, err := json.Marshal(buildScenePromptInput(ctx))
	if err != nil {
		t.Fatal(err)
	}
	payload := string(payloadBytes)
	for _, forbidden := range []string{"story_run_id", "project_id", "session_id", "identity", "shared_observable"} {
		if strings.Contains(payload, forbidden) {
			t.Fatalf("scene prompt leaked %q: %s", forbidden, payload)
		}
	}
	for _, required := range []string{"do not emit event records", "planned_actions", "characters", "output_contract"} {
		if !strings.Contains(payload, required) {
			t.Fatalf("scene prompt missing %q: %s", required, payload)
		}
	}
}

func TestReflectionPromptProjectionKeepsPerceptionAndStripsWorldIDs(t *testing.T) {
	ctx := ReflectionContext{
		Scene:           ReflectionScene{PlotVariable: StoryNarrativePlotVariable{CoreChoice: "是否暴露密信"}},
		Characters:      []ReflectionCharacter{{ID: "character_1", Name: "林澈"}},
		PerceptionIndex: []PerceptionIndexEntry{{CharacterID: "character_1", WitnessedTurnIndexes: []int{1}}},
		PriorMemories:   map[string][]string{"character_1": {"旧记忆", "另一条"}},
		Relationships:   []map[string]any{{"pair_id": "pair_1", "left_character_id": "character_1", "right_character_id": "character_2", "summary": "互相试探", "tension_points": []string{"密信"}}},
		WorldState:      []map[string]any{{"key": "rain", "value": "heavy", "note": "雨势遮住行踪", "importance": 4, "project_id": "project_1"}},
	}
	payloadBytes, err := json.Marshal(buildReflectionPromptInput(ctx))
	if err != nil {
		t.Fatal(err)
	}
	payload := string(payloadBytes)
	for _, forbidden := range []string{"project_id", "pair_id"} {
		if strings.Contains(payload, forbidden) {
			t.Fatalf("reflection prompt leaked %q: %s", forbidden, payload)
		}
	}
	for _, required := range []string{"perception_index", "prior_memories", "Deduplicate prior_memories", "world_state_updates"} {
		if !strings.Contains(payload, required) {
			t.Fatalf("reflection prompt missing %q: %s", required, payload)
		}
	}
}

func portStoryRunInputForPromptTest() port.StoryRunGenerationInput {
	return port.StoryRunGenerationInput{
		Run:     model.StoryRun{RunID: "run_1", ProjectID: "project_1", SessionID: "story_1"},
		Session: model.StorySession{ID: "story_1", ProjectID: "project_1", Title: "雨巷", OpeningSituation: "雨巷密信", LastAuthorMessage: "推进试探"},
	}
}
