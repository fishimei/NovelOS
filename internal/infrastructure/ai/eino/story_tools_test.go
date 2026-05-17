package eino

import (
	"context"
	"testing"

	"github.com/fishimei/NovelOS/internal/application/model"
)

func TestChooseNextStoryActorRecordsTurn(t *testing.T) {
	state := &storyRunState{run: model.StoryRun{RunID: "run_1"}, maxTurns: 25}
	turn, err := chooseNextStoryActor(context.Background(), storyGeneratorDeps{}, state, ChooseNextStoryActorInput{
		ActorID:        "character_1",
		ActorName:      "林澈",
		ActionType:     "speak",
		Speech:         "密信不在我这里。",
		ActionSummary:  "把袖口压住，避开对方视线。",
		TargetActorIDs: []string{"character_2"},
		Intent:         "试探对方是否掌握密信",
		Rationale:      "他刚刚受到秘密暴露的压力",
	})
	if err != nil {
		t.Fatalf("chooseNextStoryActor returned error: %v", err)
	}
	if turn.TurnIndex != 1 {
		t.Fatalf("expected turn index 1, got %d", turn.TurnIndex)
	}
	if len(state.turns) != 1 {
		t.Fatalf("expected one recorded turn, got %d", len(state.turns))
	}
	if state.turns[0].ActorID != "character_1" {
		t.Fatalf("unexpected actor id %q", state.turns[0].ActorID)
	}
	if state.turns[0].Speech != "密信不在我这里。" || state.turns[0].ActionSummary == "" {
		t.Fatalf("expected visible turn details, got %#v", state.turns[0])
	}
	if len(state.turns[0].TargetActorIDs) != 1 || state.turns[0].TargetActorIDs[0] != "character_2" {
		t.Fatalf("unexpected target actor ids: %#v", state.turns[0].TargetActorIDs)
	}
}

func TestStoryTurnDisplayPayloadHidesRationale(t *testing.T) {
	payload := storyTurnDisplayPayload(StoryTurnPlan{
		TurnIndex:      2,
		ActorID:        "character_1",
		ActorName:      "林澈",
		ActionType:     "action",
		Speech:         "跟我走。",
		ActionSummary:  "推开暗门。",
		TargetActorIDs: []string{"character_2"},
		Intent:         "带对方离开",
		Rationale:      "内部推理不应进入前端实时展示",
	})
	if payload.TurnIndex != 2 || payload.Speech != "跟我走。" || payload.ActionSummary != "推开暗门。" {
		t.Fatalf("unexpected display payload: %#v", payload)
	}
}

func TestChooseNextStoryActorRejectsInvalidActionType(t *testing.T) {
	state := &storyRunState{run: model.StoryRun{RunID: "run_1"}, maxTurns: 25}
	_, err := chooseNextStoryActor(context.Background(), storyGeneratorDeps{}, state, ChooseNextStoryActorInput{
		ActionType: "teleport",
		Intent:     "非法动作",
	})
	if err == nil {
		t.Fatal("expected invalid action type error")
	}
}

func TestDecideStoryStopForcesStopAtMaxTurns(t *testing.T) {
	state := &storyRunState{
		run:      model.StoryRun{RunID: "run_1"},
		maxTurns: 2,
		turns: []StoryTurnPlan{
			{TurnIndex: 1, ActionType: "speak", Intent: "A"},
			{TurnIndex: 2, ActionType: "action", Intent: "B"},
		},
	}
	decision, err := decideStoryStop(context.Background(), storyGeneratorDeps{}, state, DecideStoryStopInput{Stop: false})
	if err != nil {
		t.Fatalf("decideStoryStop returned error: %v", err)
	}
	if !decision.Stop {
		t.Fatal("expected forced stop")
	}
	if decision.Reason == "" {
		t.Fatal("expected stop reason")
	}
}
