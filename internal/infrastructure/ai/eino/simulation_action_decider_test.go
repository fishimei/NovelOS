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

func TestCharacterActionDeciderUsesForcedToolCall(t *testing.T) {
	chatModel := &fakeStoryChatModel{toolCalls: [][]schema.ToolCall{{{
		ID:   "call_1",
		Type: "function",
		Function: schema.FunctionCall{
			Name:      "submit_character_action",
			Arguments: `{"action_type":"observe","description":"watch the gate","duration_hours":1,"target_location_key":"gate","participant_ids":[],"affected_resource_keys":[],"rationale":"needs current information"}`,
		},
	}}}}
	decider := &CharacterActionDecider{model: chatModel, modelName: "test-model", prompt: "return action JSON"}
	decision, err := decider.Decide(context.Background(), model.CharacterActionDecisionInput{
		Character: model.Character{ID: "character_1", Name: "Lin"},
		World: model.WorldSnapshot{
			Characters: map[string]model.CharacterRuntimeState{"character_1": {CharacterID: "character_1", LocationKey: "gate", Status: "active"}},
		},
		CharacterState: model.CharacterRuntimeState{CharacterID: "character_1", LocationKey: "gate", Status: "active"},
		Location:       model.LocationState{ID: "gate", Name: "Gate"},
	})
	if err != nil {
		t.Fatalf("Decide() error = %v", err)
	}
	if decision.ActionType != "observe" || decision.TargetLocationKey != "gate" || decision.DurationHours != 1 {
		t.Fatalf("unexpected tool decision: %#v", decision)
	}
	if len(chatModel.toolCalls) != 0 {
		t.Fatalf("expected tool call to be consumed, remaining=%d", len(chatModel.toolCalls))
	}
}
