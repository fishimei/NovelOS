package eino

import (
	"testing"

	"github.com/fishimei/NovelOS/internal/application/model"
	"github.com/fishimei/NovelOS/internal/application/port"
)

type uniqueSetupIDGenerator struct {
	n int
}

func (g *uniqueSetupIDGenerator) New(prefix string) string {
	g.n++
	return prefix + "_id_" + string(rune('0'+g.n))
}

func TestSetupBuildDraftMapsCharacterKeysToRelationshipIDs(t *testing.T) {
	generator := &SetupRunGenerator{ids: &uniqueSetupIDGenerator{}}
	draft, err := generator.buildDraft(port.SetupRunGenerationInput{
		Run:     model.SetupRun{RunID: "run_1", SessionID: "setup_1", ProjectID: "project_1"},
		Session: model.SetupSession{SeedIdea: "权谋仙侠"},
	}, setupAgentOutput{
		AuthorBible: setupAuthorBibleOutput{Theme: "权力与长生的代价"},
		WorldState:  []setupWorldStateOutput{{Key: "sect_pressure", Value: "宗门大比将至", Importance: 5, Volatility: 4}},
		Characters: []setupCharacterOutput{
			{Key: "protagonist", Name: "林澈", Role: "protagonist", Goals: []string{"保住师妹"}},
			{Key: "rival", Name: "沈砚", Role: "rival", Goals: []string{"夺得首席"}},
		},
		Relationships: []setupRelationshipOutput{{
			CharacterAKey:  "protagonist",
			CharacterBKey:  "rival",
			Summary:        "表面同门，暗中互相试探",
			CharacterAView: setupRelationshipViewOutput{PrivateAttitude: "警惕"},
			CharacterBView: setupRelationshipViewOutput{PrivateAttitude: "嫉妒"},
		}},
		AssistantSummary: "围绕宗门压力启动故事。",
	})
	if err != nil {
		t.Fatalf("buildDraft returned error: %v", err)
	}
	if draft.AuthorBible.Theme != "权力与长生的代价" {
		t.Fatalf("unexpected theme %q", draft.AuthorBible.Theme)
	}
	if len(draft.Characters) != 2 {
		t.Fatalf("expected two characters, got %d", len(draft.Characters))
	}
	if len(draft.Relationships) != 1 {
		t.Fatalf("expected one relationship, got %d", len(draft.Relationships))
	}
	pair := draft.Relationships[0].Pair
	if pair.LeftCharacterID == "protagonist" || pair.RightCharacterID == "rival" {
		t.Fatalf("expected relationship to use generated ids, got %#v", pair)
	}
	if len(draft.WorldState) != 1 || draft.WorldState[0].Key != "sect_pressure" {
		t.Fatalf("unexpected world state: %#v", draft.WorldState)
	}
}
