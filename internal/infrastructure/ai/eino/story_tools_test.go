package eino

import (
	"context"
	"testing"

	"github.com/fishimei/NovelOS/internal/application/model"
)

func TestChooseNextStoryActorRecordsTurn(t *testing.T) {
	state := &storyRunState{run: model.StoryRun{RunID: "run_1"}, maxTurns: 25}
	turn, err := chooseNextStoryActor(context.Background(), storyGeneratorDeps{}, state, ChooseNextStoryActorInput{
		ActorID:    "character_1",
		ActorName:  "林澈",
		ActionType: "speak",
		Intent:     "试探对方是否掌握密信",
		Rationale:  "他刚刚受到秘密暴露的压力",
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
