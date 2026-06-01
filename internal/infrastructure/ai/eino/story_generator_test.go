package eino

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/fishimei/NovelOS/internal/application/model"
	"github.com/fishimei/NovelOS/internal/application/port"
	"github.com/fishimei/NovelOS/internal/config"
)

type fakeChapterRepository struct{}

func (fakeChapterRepository) ListByProjectID(context.Context, string, model.PageInput) (model.ListResult[model.Chapter], error) {
	return model.ListResult[model.Chapter]{Items: []model.Chapter{{ChapterNumber: 2}}}, nil
}

func (fakeChapterRepository) GetByID(context.Context, string) (model.Chapter, error) {
	return model.Chapter{}, nil
}

func (fakeChapterRepository) Create(context.Context, model.Chapter) (model.Chapter, error) {
	return model.Chapter{}, nil
}

type fakeClock struct{}

func (fakeClock) Now() time.Time {
	return time.Unix(100, 0).UTC()
}

type fakeIDGenerator struct{}

func (fakeIDGenerator) New(prefix string) string {
	return prefix + "_id"
}

func TestBuildResultUsesNarrativeOutput(t *testing.T) {
	generator := &StoryRunGenerator{
		cfg:   config.AIConfig{},
		deps:  storyGeneratorDeps{chapters: fakeChapterRepository{}},
		clock: fakeClock{},
		ids:   fakeIDGenerator{},
	}
	result, err := generator.buildResult(context.Background(), port.StoryRunGenerationInput{
		Run: model.StoryRun{RunID: "run_1", SessionID: "story_1", ProjectID: "project_1"},
		Session: model.StorySession{
			Title:             "第一章",
			OpeningSituation:  "密信被截获",
			LastAuthorMessage: "让角色围绕密信行动",
		},
	}, StoryPlanResult{
		Summary:    "密信引发怀疑",
		StopReason: "怀疑已经形成",
		Turns: []StoryTurnPlan{
			{TurnIndex: 1, ActorID: "character_1", ActorName: "林澈", ActionType: "speak", Intent: "试探对方"},
		},
		EventPlan: []model.StoryEventPlan{
			{ID: "event_1", CharacterID: "character_1", CharacterName: "林澈", LocationKey: "old_dock", LocationName: "旧码头", ActionType: "action", Summary: "林澈抵达旧码头"},
		},
		InteractionAnalysis: model.StoryInteractionAnalysis{
			InteractionGroups: []model.StoryInteractionGroup{{ID: "interaction_1", LocationKey: "old_dock", CharacterIDs: []string{"character_1", "character_2"}, ShouldInteract: true}},
		},
		InteractionTranscripts: []model.StoryInteractionTranscript{
			{
				GroupID:      "interaction_1",
				LocationKey:  "old_dock",
				CharacterIDs: []string{"character_1", "character_2"},
				Turns: []model.StoryInteractionTurn{
					{TurnIndex: 1, InteractionGroupID: "interaction_1", ActorID: "character_1", ActionType: "speak", Speech: "密信不在我这里。"},
				},
			},
		},
	}, StoryNarrativeResult{
		Summary: "密信引发怀疑",
		Content: "林澈压低声音试探对方。",
		PlotVariable: StoryNarrativePlotVariable{
			RelatedCharacterIDs: []string{"character_1"},
		},
		MemoryPatch: StoryNarrativeMemoryPatch{
			CharacterMemoryUpdates: []StoryNarrativeCharacterMemoryUpdate{{CharacterID: "character_1", Content: "林澈试探密信下落", Importance: 4}},
		},
		Review: StoryNarrativeReview{Pass: true},
	}, StoryVariablePlan{})
	if err != nil {
		t.Fatalf("buildResult returned error: %v", err)
	}
	if result.Status != "completed" {
		t.Fatalf("unexpected status %q", result.Status)
	}
	if result.Draft.ChapterNumber != 3 {
		t.Fatalf("expected chapter number 3, got %d", result.Draft.ChapterNumber)
	}
	if !strings.Contains(result.Draft.Content, "林澈压低声音") {
		t.Fatalf("expected draft content to include narrative content, got %q", result.Draft.Content)
	}
	if result.MemoryPatch.ID == "" {
		t.Fatal("expected memory patch id")
	}
	if len(result.MemoryPatch.CharacterMemoryUpdates) != 1 {
		t.Fatalf("unexpected memory updates: %#v", result.MemoryPatch.CharacterMemoryUpdates)
	}
	if len(result.PlotVariable.RelatedCharacterIDs) != 1 || result.PlotVariable.RelatedCharacterIDs[0] != "character_1" {
		t.Fatalf("unexpected related character ids: %#v", result.PlotVariable.RelatedCharacterIDs)
	}
	if len(result.EventPlan) != 1 || result.EventPlan[0].LocationKey != "old_dock" {
		t.Fatalf("unexpected event timeline: %#v", result.EventPlan)
	}
	if len(result.InteractionAnalysis.InteractionGroups) != 1 || result.InteractionAnalysis.InteractionGroups[0].ID != "interaction_1" {
		t.Fatalf("unexpected interaction analysis: %#v", result.InteractionAnalysis)
	}
	if len(result.InteractionTranscripts) != 1 || len(result.InteractionTranscripts[0].Turns) != 1 {
		t.Fatalf("unexpected interaction transcripts: %#v", result.InteractionTranscripts)
	}
}

func TestBuildResultPrefersPreGeneratedVariable(t *testing.T) {
	generator := &StoryRunGenerator{
		deps:  storyGeneratorDeps{chapters: fakeChapterRepository{}},
		clock: fakeClock{},
		ids:   fakeIDGenerator{},
	}
	result, err := generator.buildResult(context.Background(), port.StoryRunGenerationInput{
		Run: model.StoryRun{RunID: "run_1", SessionID: "story_1", ProjectID: "project_1"},
		Session: model.StorySession{
			Title:             "第一章",
			OpeningSituation:  "旧城戒严",
			LastAuthorMessage: "推进戒严压力",
		},
	}, StoryPlanResult{
		Summary: "角色围绕戒严行动",
		Turns:   []StoryTurnPlan{{TurnIndex: 1, ActorID: "character_1", ActorName: "林澈", ActionType: "action", Intent: "寻找突破口"}},
	}, StoryNarrativeResult{
		Summary: "草稿摘要",
		Content: "林澈穿过雨巷。",
		PlotVariable: StoryNarrativePlotVariable{
			CoreChoice: "后置变量不应覆盖",
		},
		Review: StoryNarrativeReview{Pass: true},
	}, StoryVariablePlan{PlotVariable: StoryNarrativePlotVariable{
		PressureSource:      "城门提前关闭",
		FocalCharacterID:    "character_1",
		CoreChoice:          "林澈是否暴露暗线身份换取出城机会",
		OptionA:             "隐藏身份等待",
		OptionB:             "暴露身份突围",
		CostA:               "同伴被困",
		CostB:               "暗线失效",
		IrreversibleEffect:  "守军开始清查暗线",
		RelatedCharacterIDs: []string{"character_1"},
	}})
	if err != nil {
		t.Fatalf("buildResult returned error: %v", err)
	}
	if result.PlotVariable.CoreChoice != "林澈是否暴露暗线身份换取出城机会" {
		t.Fatalf("expected pre-generated variable, got %q", result.PlotVariable.CoreChoice)
	}
	if result.Draft.Summary != result.PlotVariable.CoreChoice {
		t.Fatalf("expected draft summary to follow pre-generated variable, got %q", result.Draft.Summary)
	}
}

func TestRelationshipUpdatesKeepEvents(t *testing.T) {
	updates := toRelationshipUpdates([]StoryNarrativeRelationshipUpdate{
		{
			PairID:       "pair_1",
			TensionDelta: "互相警惕加深",
			Events: []model.RelationshipEvent{
				{EventType: "negotiation", Summary: "围绕密信互相试探"},
			},
		},
	})
	if len(updates) != 1 {
		t.Fatalf("expected one relationship update, got %#v", updates)
	}
	if len(updates[0].Events) != 1 || updates[0].Events[0].EventType != "negotiation" {
		t.Fatalf("expected relationship event to be preserved, got %#v", updates[0].Events)
	}
	if updates[0].TensionDelta != "互相警惕加深" {
		t.Fatalf("expected tension delta, got %#v", updates[0])
	}
}

func TestGenerateStoryVariableUsesBackendState(t *testing.T) {
	generator := &StoryRunGenerator{}
	variable, err := generator.generateStoryVariable(context.Background(), port.StoryRunGenerationInput{
		Session: model.StorySession{
			OpeningSituation:  "城门提前关闭",
			AuthorIntent:      "让林澈被迫决定是否暴露身份",
			LastAuthorMessage: "林澈和沈砚在雨巷交换密信",
		},
	}, StoryContextSnapshot{
		WorldState: []model.WorldStateEntry{
			{Key: "gate_lockdown", Note: "城门提前关闭", Importance: 5, Volatility: 4},
			{Key: "rain", Note: "雨势遮住行踪", Importance: 2, Volatility: 2},
		},
		Characters: []model.Character{
			{ID: "character_1", Name: "林澈"},
			{ID: "character_2", Name: "沈砚"},
		},
		Relationships: []model.Relationship{{Pair: model.RelationshipPair{LeftCharacterID: "character_1", RightCharacterID: "character_2"}, Views: []model.RelationshipView{
			{SourceCharacterID: "character_1", TargetCharacterID: "character_2", BelievedTargetAttitude: "沈砚可能藏着另一封信"},
		}}},
	})
	if err != nil {
		t.Fatalf("generateStoryVariable returned error: %v", err)
	}
	if variable.PlotVariable.FocalCharacterID != "character_1" {
		t.Fatalf("expected focal character to be 林澈, got %#v", variable.PlotVariable)
	}
	if !containsString(variable.PlotVariable.RelatedCharacterIDs, "character_2") {
		t.Fatalf("expected mentioned character to be related, got %#v", variable.PlotVariable.RelatedCharacterIDs)
	}
	if len(variable.PlotVariable.WorldStatePressure) == 0 || variable.PlotVariable.WorldStatePressure[0] != "gate_lockdown" {
		t.Fatalf("expected high-pressure world state, got %#v", variable.PlotVariable.WorldStatePressure)
	}
	if len(variable.CharacterViews) == 0 || len(variable.CharacterViews[0].KnownFacts) == 0 {
		t.Fatalf("expected character views, got %#v", variable.CharacterViews)
	}
}

func TestPerspectiveForTurnOnlyIncludesActorRelationshipViews(t *testing.T) {
	generator := &StoryRunGenerator{}
	perspective := generator.perspectiveForTurn(StoryContextSnapshot{
		Characters: []model.Character{{ID: "character_1", Name: "林澈"}, {ID: "character_2", Name: "沈砚"}},
		Relationships: []model.Relationship{{Views: []model.RelationshipView{
			{SourceCharacterID: "character_1", TargetCharacterID: "character_2", PrivateAttitude: "警惕"},
			{SourceCharacterID: "character_2", TargetCharacterID: "character_1", PrivateAttitude: "嫉妒"},
		}}},
	}, StoryTurnPlan{ActorID: "character_1"}, StoryVariablePlan{})
	if perspective == nil {
		t.Fatal("expected perspective")
	}
	if len(perspective.RelationshipViews) != 1 {
		t.Fatalf("expected one relationship view, got %#v", perspective.RelationshipViews)
	}
	if perspective.RelationshipViews[0].PrivateAttitude != "警惕" {
		t.Fatalf("unexpected private attitude %q", perspective.RelationshipViews[0].PrivateAttitude)
	}
}

func TestPerspectiveForTurnOnlyIncludesActorVariableView(t *testing.T) {
	generator := &StoryRunGenerator{}
	perspective := generator.perspectiveForTurn(StoryContextSnapshot{
		Characters: []model.Character{{ID: "character_1", Name: "林澈"}, {ID: "character_2", Name: "沈砚"}},
	}, StoryTurnPlan{ActorID: "character_1"}, StoryVariablePlan{CharacterViews: []CharacterVariableView{
		{CharacterID: "character_1", KnownFacts: []string{"城门提前关闭"}, EmotionalPressure: "必须决定是否暴露身份"},
		{CharacterID: "character_2", KnownFacts: []string{"林澈行踪异常"}, EmotionalPressure: "怀疑林澈隐瞒"},
	}})
	if perspective == nil {
		t.Fatal("expected perspective")
	}
	if perspective.VariableView == nil {
		t.Fatal("expected variable view")
	}
	if perspective.VariableView.CharacterID != "character_1" {
		t.Fatalf("unexpected variable view: %#v", perspective.VariableView)
	}
	if strings.Contains(strings.Join(perspective.VariableView.KnownFacts, ","), "行踪异常") {
		t.Fatalf("variable view leaked another character view: %#v", perspective.VariableView)
	}
}
