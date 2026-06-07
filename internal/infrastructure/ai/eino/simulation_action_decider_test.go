package eino

import (
	"context"
	"testing"

	"github.com/cloudwego/eino/schema"

	"github.com/fishimei/NovelOS/internal/application/model"
)

func TestDecodeModelJSONWithMetaReportsMalformedObject(t *testing.T) {
	var decision model.CharacterActionDecision
	meta, err := decodeModelJSONWithMeta("```json\n{\"action_type\":\"observe\"\n```", &decision)
	if err == nil {
		t.Fatal("expected malformed JSON to fail")
	}
	if meta.RawLength == 0 || meta.ExtractedLength == 0 {
		t.Fatalf("expected parse metadata lengths, got %#v", meta)
	}
	if !meta.FencedRemoved {
		t.Fatalf("expected fenced JSON metadata, got %#v", meta)
	}
	summary := modelJSONErrorSummary(err)
	if summary["raw_prefix"] == "" || summary["parse_error"] == "" {
		t.Fatalf("expected audit summary to include parse details, got %#v", summary)
	}
}

func TestCharacterActionDeciderUsesInspectAndSubmitToolsWithoutForcedChoice(t *testing.T) {
	chatModel := &fakeStoryChatModel{toolCalls: [][]schema.ToolCall{
		{{
			ID:   "call_1",
			Type: "function",
			Function: schema.FunctionCall{
				Name:      "inspect_location",
				Arguments: `{"location_id":"dock","reason":"needs detail before moving"}`,
			},
		}},
		{{
			ID:   "call_2",
			Type: "function",
			Function: schema.FunctionCall{
				Name:      "submit_character_action",
				Arguments: `{"action_type":"move","description":"go to the dock","duration_hours":1,"target_location_key":"dock","participant_ids":[],"affected_resource_keys":[],"rationale":"the dock is now inspected"}`,
			},
		}},
	}}
	inspector := &fakeLocationInspectionService{location: model.LocationState{ID: "dock", Name: "Dock", DetailState: model.LocationDetailInitialized}}
	decider := &CharacterActionDecider{model: chatModel, modelName: "test-model", prompt: "use tools", locationInspector: inspector}
	decision, err := decider.Decide(context.Background(), model.CharacterActionDecisionInput{
		Character: model.Character{ID: "character_1", ProjectID: "project_1", Name: "Lin"},
		World: model.WorldSnapshot{
			Characters: map[string]model.CharacterRuntimeState{"character_1": {CharacterID: "character_1", LocationKey: "gate", Status: "active"}},
		},
		CharacterState: model.CharacterRuntimeState{CharacterID: "character_1", LocationKey: "gate", Status: "active"},
		Location:       model.LocationState{ID: "gate", ProjectID: "project_1", Name: "Gate"},
		NearbyLocations: []model.NearbyLocationContext{{
			Location: model.LocationState{ID: "dock", ProjectID: "project_1", Name: "Dock", DetailState: model.LocationDetailStub},
		}},
	})
	if err != nil {
		t.Fatalf("Decide() error = %v", err)
	}
	if decision.ActionType != "move" || decision.TargetLocationKey != "dock" || decision.DurationHours != 1 {
		t.Fatalf("unexpected tool decision: %#v", decision)
	}
	if len(inspector.inputs) != 1 || inspector.inputs[0].LocationID != "dock" || inspector.inputs[0].CurrentLocationID != "gate" {
		t.Fatalf("unexpected inspect inputs: %#v", inspector.inputs)
	}
	if len(chatModel.boundTools) != 0 {
		t.Fatalf("expected action decider to avoid model.WithTools binding, got %#v", chatModel.boundTools)
	}
	if len(chatModel.callTools) != 2 || !hasToolNames(chatModel.callTools[0], "inspect_location", "submit_character_action") || !hasToolNames(chatModel.callTools[1], "inspect_location", "submit_character_action") {
		t.Fatalf("expected both action tools to be passed per Generate call, got %#v", chatModel.callTools)
	}
	for _, choice := range chatModel.toolChoices {
		if choice != nil {
			t.Fatalf("expected no forced tool choice, got %v", *choice)
		}
	}
}

func hasToolNames(actual []string, expected ...string) bool {
	if len(actual) != len(expected) {
		return false
	}
	seen := map[string]struct{}{}
	for _, name := range actual {
		seen[name] = struct{}{}
	}
	for _, name := range expected {
		if _, ok := seen[name]; !ok {
			return false
		}
	}
	return true
}

func TestValidateCharacterActionDecisionRejectsInvalidLocationAndParticipants(t *testing.T) {
	input := model.CharacterActionDecisionInput{
		World: model.WorldSnapshot{
			Characters: map[string]model.CharacterRuntimeState{
				"character_1": {CharacterID: "character_1", LocationKey: "gate", Status: "active"},
				"character_2": {CharacterID: "character_2", LocationKey: "gate", Status: "active"},
			},
		},
		Character:      model.Character{ID: "character_1", Name: "Lin"},
		CharacterState: model.CharacterRuntimeState{CharacterID: "character_1", LocationKey: "gate", Status: "active"},
		Location:       model.LocationState{ID: "gate", Name: "Gate", DetailState: model.LocationDetailInitialized},
		NearbyLocations: []model.NearbyLocationContext{{
			Location: model.LocationState{ID: "dock", Name: "Dock", DetailState: model.LocationDetailStub},
		}},
	}

	err := validateCharacterActionDecision(input, model.CharacterActionDecision{
		ActionType:        "observe",
		Description:       "watch nowhere",
		TargetLocationKey: "unknown",
		DurationHours:     1,
	}, nil)
	if err == nil {
		t.Fatal("expected unreachable target to be rejected")
	}

	err = validateCharacterActionDecision(input, model.CharacterActionDecision{
		ActionType:        "action",
		Description:       "go to the dock",
		TargetLocationKey: "dock",
		DurationHours:     1,
	}, nil)
	if err == nil {
		t.Fatal("expected uninspected action target to be rejected")
	}

	err = validateCharacterActionDecision(input, model.CharacterActionDecision{
		ActionType:        "observe",
		Description:       "watch the gate",
		TargetLocationKey: "gate",
		DurationHours:     1,
		ParticipantIDs:    []string{"missing_character"},
	}, nil)
	if err == nil {
		t.Fatal("expected unknown participant to be rejected")
	}
}
