package eino

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	llmmodel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"

	"github.com/fishimei/NovelOS/internal/application/model"
	"github.com/fishimei/NovelOS/internal/application/port"
	"github.com/fishimei/NovelOS/internal/config"
	"github.com/fishimei/NovelOS/internal/domain"
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

type fakeCharacterActionDecider struct {
	inputs   []model.CharacterActionDecisionInput
	decision model.CharacterActionDecision
}

func (d *fakeCharacterActionDecider) Decide(_ context.Context, input model.CharacterActionDecisionInput) (model.CharacterActionDecision, error) {
	d.inputs = append(d.inputs, input)
	return d.decision, nil
}

type fakeStoryChatModel struct {
	responses     []string
	streamChunks  []string
	streamErr     error
	systemPrompts []string
}

func (m *fakeStoryChatModel) Generate(ctx context.Context, input []*schema.Message, opts ...llmmodel.Option) (*schema.Message, error) {
	if len(m.responses) == 0 {
		return nil, errors.New("unexpected model call")
	}
	if len(input) > 0 {
		m.systemPrompts = append(m.systemPrompts, input[0].Content)
	}
	content := m.responses[0]
	m.responses = m.responses[1:]
	return schema.AssistantMessage(content, nil), nil
}

func (m *fakeStoryChatModel) Stream(ctx context.Context, input []*schema.Message, opts ...llmmodel.Option) (*schema.StreamReader[*schema.Message], error) {
	if m.streamErr != nil {
		return nil, m.streamErr
	}
	if len(m.streamChunks) == 0 {
		return nil, errors.New("stream is not supported")
	}
	if len(input) > 0 {
		m.systemPrompts = append(m.systemPrompts, input[0].Content)
	}
	stream, writer := schema.Pipe[*schema.Message](len(m.streamChunks))
	go func() {
		defer writer.Close()
		for _, chunk := range m.streamChunks {
			writer.Send(schema.AssistantMessage(chunk, nil), nil)
		}
	}()
	return stream, nil
}

func (m *fakeStoryChatModel) WithTools(tools []*schema.ToolInfo) (llmmodel.ToolCallingChatModel, error) {
	return m, nil
}

func TestAssembleStoryRunResultUsesReflectionOutput(t *testing.T) {
	generator := &StoryRunGenerator{clock: fakeClock{}, ids: fakeIDGenerator{}}
	result := generator.assembleStoryRunResult(port.StoryRunGenerationInput{
		Run: model.StoryRun{RunID: "run_1", SessionID: "story_1", ProjectID: "project_1"},
		Session: model.StorySession{
			Title:             "Rain Docks",
			OpeningSituation:  "a letter was intercepted",
			LastAuthorMessage: "advance the intercepted letter scene",
		},
	}, StoryPlanResult{
		StopReason:       "the test closes",
		ContinuityIssues: []string{"ignored v1-only NDJSON record type: result"},
		Turns: []StoryTurnPlan{
			{TurnIndex: 1, ActorID: "character_1", ActorName: "Lin", ActionType: "speak", Speech: "The letter is not with me.", Intent: "probe Shen", TargetActorIDs: []string{"character_2"}, InteractionGroupID: "interaction_1", LocationKey: "old_dock", Phase: "negotiation"},
		},
		EventPlan: []model.StoryEventPlan{
			{ID: "event_1", CharacterID: "character_1", CharacterName: "Lin", LocationKey: "old_dock", LocationName: "Old Dock", ActionType: "action", Summary: "Lin reaches the dock"},
		},
		InteractionAnalysis: model.StoryInteractionAnalysis{
			InteractionGroups: []model.StoryInteractionGroup{{ID: "interaction_1", LocationKey: "old_dock", CharacterIDs: []string{"character_1", "character_2"}, ShouldInteract: true}},
		},
		InteractionTranscripts: []model.StoryInteractionTranscript{{
			GroupID:      "interaction_1",
			LocationKey:  "old_dock",
			CharacterIDs: []string{"character_1", "character_2"},
			Turns: []model.StoryInteractionTurn{
				{TurnIndex: 1, InteractionGroupID: "interaction_1", ActorID: "character_1", ActionType: "speak", Speech: "The letter is not with me."},
			},
		}},
	}, SceneReflectionResult{
		Summary: "Lin and Shen test each other at the dock.",
		MemoryPatch: StoryNarrativeMemoryPatch{
			CharacterMemoryUpdates: []StoryNarrativeCharacterMemoryUpdate{{CharacterID: "character_1", Type: "episodic", Content: "Lin denied carrying the letter at the dock.", Importance: 4}},
		},
	}, StoryVariablePlan{PlotVariable: StoryNarrativePlotVariable{
		PressureSource:      "intercepted letter",
		FocalCharacterID:    "character_1",
		CoreChoice:          "whether Lin reveals the letter trail",
		RelatedCharacterIDs: []string{"character_1", "character_2"},
	}})
	if result.Status != domain.RunStatusCompleted {
		t.Fatalf("unexpected status %q", result.Status)
	}
	if result.SceneSummary != "Lin and Shen test each other at the dock." {
		t.Fatalf("scene summary = %q", result.SceneSummary)
	}
	if result.Draft.Content != "" || result.Draft.WordCount != 0 {
		t.Fatalf("draft should be empty prose shell, got %#v", result.Draft)
	}
	if result.Draft.Summary != result.SceneSummary {
		t.Fatalf("draft summary = %q, want scene summary", result.Draft.Summary)
	}
	if len(result.Turns) != 1 || result.Turns[0].Speech == "" {
		t.Fatalf("expected persisted turn, got %#v", result.Turns)
	}
	if len(result.MemoryPatch.CharacterMemoryUpdates) != 1 {
		t.Fatalf("unexpected memory updates: %#v", result.MemoryPatch.CharacterMemoryUpdates)
	}
	if len(result.Review.ContinuityIssues) != 1 {
		t.Fatalf("expected continuity issue to survive, got %#v", result.Review)
	}
}

func TestAssembleStoryRunResultPrefersSeedVariable(t *testing.T) {
	generator := &StoryRunGenerator{clock: fakeClock{}, ids: fakeIDGenerator{}}
	result := generator.assembleStoryRunResult(port.StoryRunGenerationInput{
		Run: model.StoryRun{RunID: "run_1", SessionID: "story_1", ProjectID: "project_1"},
		Session: model.StorySession{
			Title:             "Gate",
			OpeningSituation:  "the city gate closes early",
			LastAuthorMessage: "advance the pressure",
		},
	}, StoryPlanResult{
		Turns: []StoryTurnPlan{{TurnIndex: 1, ActorID: "character_1", ActorName: "Lin", ActionType: "action", Intent: "find a way out"}},
	}, SceneReflectionResult{Summary: "Lin looks for a way out."}, StoryVariablePlan{PlotVariable: StoryNarrativePlotVariable{
		PressureSource:      "the city gate closes early",
		FocalCharacterID:    "character_1",
		CoreChoice:          "whether Lin exposes his hidden identity to leave the city",
		OptionA:             "hide and wait",
		OptionB:             "expose himself and break out",
		CostA:               "his ally remains trapped",
		CostB:               "the hidden line is burned",
		IrreversibleEffect:  "guards begin searching the hidden line",
		RelatedCharacterIDs: []string{"character_1"},
	}})
	if result.PlotVariable.CoreChoice != "whether Lin exposes his hidden identity to leave the city" {
		t.Fatalf("expected pre-generated variable, got %q", result.PlotVariable.CoreChoice)
	}
	if result.Draft.Summary != result.SceneSummary {
		t.Fatalf("expected draft summary to follow scene summary, got %q", result.Draft.Summary)
	}
}

func TestSimulateSceneConsumesV2NDJSONStream(t *testing.T) {
	chatModel := &fakeStoryChatModel{streamChunks: []string{
		`{"type":"plot_variable","plot_variable":{"pressure_source":"intercepted letter","focal_character_id":"character_1","core_choice":"whether Lin reveals the letter trail","option_a":"hide","option_b":"show hand","cost_a":"suspicion rises","cost_b":"exposure","irreversible_effect":"they test each other","related_character_ids":["character_1","character_2"],"world_state_pressure":["gate_lockdown"]}}` + "\n",
		`{"type":"event","event":{"time_index":1,"character_id":"character_1","character_name":"Lin","location_key":"old_dock","location_name":"Old Dock","action_type":"action","summary":"Lin reaches the dock","intent":"find the contact"}}` + "\n",
		`{"type":"event","event":{"time_index":2,"character_id":"character_2","character_name":"Shen","location_key":"old_dock","location_name":"Old Dock","action_type":"observe","summary":"Shen waits under the awning","intent":"confirm Lin's movement"}}` + "\n",
		`{"type":"interaction","interaction_group":{"id":"interaction_1","location_key":"old_dock","location_name":"Old Dock","character_ids":["character_1","character_2"],"should_interact":true,"interaction_type":"confrontation","stakes":"whether the letter trail is exposed","priority":1}}` + "\n",
		`{"type":"turn","turn":{"turn_index":1,"actor_id":"character_1","actor_name":"Lin","action_type":"speak","speech":"The letter is not with me.","intent":"probe Shen","target_actor_ids":["character_2"],"interaction_group_id":"interaction_1","location_key":"old_dock","phase":"negotiation"}}` + "\n",
		`{"type":"stop","stop_reason":"the test is complete"}` + "\n",
	}}
	generator := &StoryRunGenerator{
		cfg:      config.AIConfig{Model: "test-model"},
		model:    chatModel,
		deps:     storyGeneratorDeps{},
		maxTurns: 5,
	}
	input := port.StoryRunGenerationInput{
		Run:     model.StoryRun{RunID: "run_1", SessionID: "story_1", ProjectID: "project_1"},
		Session: modelStorySession(),
	}
	snapshot := StoryContextSnapshot{
		Characters: []model.Character{{ID: "character_1", Name: "Lin"}, {ID: "character_2", Name: "Shen"}},
		WorldState: []model.WorldStateEntry{{Key: "gate_lockdown", Note: "city gate closes early", Importance: 5, Volatility: 4}},
	}
	seed := StoryVariablePlan{PlotVariable: StoryNarrativePlotVariable{
		PressureSource:      "intercepted letter",
		FocalCharacterID:    "character_1",
		CoreChoice:          "whether Lin reveals the letter trail",
		RelatedCharacterIDs: []string{"character_1", "character_2"},
	}}
	state := &storyRunState{run: input.Run, session: input.Session, maxTurns: 5, characters: snapshot.Characters, variable: seed}
	plan, variable, err := generator.simulateScene(context.Background(), input, snapshot, state, generator.buildSceneContext(input, snapshot, seed, nil), seed)
	if err != nil {
		t.Fatalf("simulateScene returned error: %v", err)
	}
	if variable.PlotVariable.CoreChoice != "whether Lin reveals the letter trail" {
		t.Fatalf("unexpected plot variable: %#v", variable.PlotVariable)
	}
	if plan.StopReason != "the test is complete" {
		t.Fatalf("stop reason = %q", plan.StopReason)
	}
	if len(plan.EventPlan) != 2 {
		t.Fatalf("expected two events, got %#v", plan.EventPlan)
	}
	if len(plan.InteractionAnalysis.InteractionGroups) != 1 || plan.InteractionAnalysis.InteractionGroups[0].ID != "interaction_1" {
		t.Fatalf("expected selected interaction, got %#v", plan.InteractionAnalysis)
	}
	if len(plan.InteractionTranscripts) != 1 || len(plan.InteractionTranscripts[0].Turns) != 1 {
		t.Fatalf("expected transcript from turn, got %#v", plan.InteractionTranscripts)
	}
	if len(chatModel.systemPrompts) != 1 {
		t.Fatalf("expected one model stream call, got %d", len(chatModel.systemPrompts))
	}
	if !strings.Contains(chatModel.systemPrompts[0], "single scene simulator") {
		t.Fatalf("expected scene prompt, got %q", chatModel.systemPrompts[0])
	}
}

func TestSimulateSceneHandlesFencedAndSplitNDJSONStream(t *testing.T) {
	chatModel := &fakeStoryChatModel{streamChunks: []string{
		"```json\n",
		`{"type":"plot_variable","plot_variable":{"core_choice":"whether Lin hides the coin","related_character_ids":["character_1"]}}` + "\n",
		"```\n",
		`{"type":"event","event":{"time_index":1,"character_id":"character_1",` + "\n",
		`"character_name":"Lin","location_key":"old_dock","location_name":"Old Dock","action_type":"action","summary":"Lin hides the coin","intent":"cover the payment"}}` + "\n",
		`{"type":"turn","turn":{"turn_index":1,"actor_id":"character_1","action_type":"action","action_summary":"Lin wipes water from the coin","intent":"hide the payment","location_key":"old_dock"}}`,
	}}
	generator := &StoryRunGenerator{
		cfg:      config.AIConfig{Model: "test-model"},
		model:    chatModel,
		deps:     storyGeneratorDeps{},
		maxTurns: 5,
	}
	input := port.StoryRunGenerationInput{
		Run:     model.StoryRun{RunID: "run_1", SessionID: "story_1", ProjectID: "project_1"},
		Session: modelStorySession(),
	}
	snapshot := StoryContextSnapshot{Characters: []model.Character{{ID: "character_1", Name: "Lin"}}}
	seed := StoryVariablePlan{PlotVariable: StoryNarrativePlotVariable{CoreChoice: "whether Lin hides the coin", RelatedCharacterIDs: []string{"character_1"}}}
	state := &storyRunState{run: input.Run, session: input.Session, maxTurns: 5, characters: snapshot.Characters, variable: seed}

	plan, variable, err := generator.simulateScene(context.Background(), input, snapshot, state, generator.buildSceneContext(input, snapshot, seed, nil), seed)
	if err != nil {
		t.Fatalf("simulateScene returned error: %v", err)
	}
	if variable.PlotVariable.CoreChoice != "whether Lin hides the coin" {
		t.Fatalf("unexpected plot variable: %#v", variable.PlotVariable)
	}
	if len(plan.EventPlan) != 1 || plan.EventPlan[0].Summary != "Lin hides the coin" {
		t.Fatalf("expected split event to be consumed, got %#v", plan.EventPlan)
	}
	if len(plan.Turns) != 1 || plan.Turns[0].ActionSummary != "Lin wipes water from the coin" {
		t.Fatalf("expected final un-newlined turn to be consumed, got %#v", plan.Turns)
	}
}

func TestSimulateSceneFallsBackWhenStreamCannotBeParsed(t *testing.T) {
	chatModel := &fakeStoryChatModel{
		streamChunks: []string{`{"type":"event","event":`},
		responses: []string{
			`{"event_plan":[{"time_index":1,"character_id":"character_1","location_key":"old_dock","action_type":"action","summary":"Fallback event"}],"turns":[{"turn_index":1,"actor_id":"character_1","action_type":"action","action_summary":"Fallback turn","intent":"recover","location_key":"old_dock"}],"stop_reason":"fallback complete"}`,
		},
	}
	generator := &StoryRunGenerator{
		cfg:          config.AIConfig{Model: "test-model"},
		model:        chatModel,
		deps:         storyGeneratorDeps{},
		maxTurns:     5,
		resultPrompt: defaultSceneFallbackPrompt(),
	}
	input := port.StoryRunGenerationInput{
		Run:     model.StoryRun{RunID: "run_1", SessionID: "story_1", ProjectID: "project_1"},
		Session: modelStorySession(),
	}
	snapshot := StoryContextSnapshot{Characters: []model.Character{{ID: "character_1", Name: "Lin"}}}
	seed := StoryVariablePlan{PlotVariable: StoryNarrativePlotVariable{CoreChoice: "recover from invalid stream", RelatedCharacterIDs: []string{"character_1"}}}
	state := &storyRunState{run: input.Run, session: input.Session, maxTurns: 5, characters: snapshot.Characters, variable: seed}

	plan, _, err := generator.simulateScene(context.Background(), input, snapshot, state, generator.buildSceneContext(input, snapshot, seed, nil), seed)
	if err != nil {
		t.Fatalf("simulateScene returned error: %v", err)
	}
	if plan.StopReason != "fallback complete" {
		t.Fatalf("stop reason = %q", plan.StopReason)
	}
	if len(plan.EventPlan) != 1 || plan.EventPlan[0].Summary != "Fallback event" {
		t.Fatalf("expected fallback event plan, got %#v", plan.EventPlan)
	}
	if len(chatModel.systemPrompts) != 2 {
		t.Fatalf("expected stream prompt plus fallback prompt, got %d", len(chatModel.systemPrompts))
	}
	if !strings.Contains(chatModel.systemPrompts[1], "non-streaming scene simulation fallback") {
		t.Fatalf("expected fallback prompt, got %q", chatModel.systemPrompts[1])
	}
}

func TestConsumeSceneBatchEnforcesInteractionInvariants(t *testing.T) {
	generator := &StoryRunGenerator{maxTurns: 5}
	input := port.StoryRunGenerationInput{
		Run:     model.StoryRun{RunID: "run_1", SessionID: "story_1", ProjectID: "project_1"},
		Session: modelStorySession(),
	}
	snapshot := StoryContextSnapshot{Characters: []model.Character{
		{ID: "character_1", Name: "Lin"},
		{ID: "character_2", Name: "Shen"},
		{ID: "character_3", Name: "Yan"},
	}}
	seed := StoryVariablePlan{PlotVariable: StoryNarrativePlotVariable{CoreChoice: "whether the dock confrontation holds", RelatedCharacterIDs: []string{"character_1", "character_2", "character_3"}}}
	state := &storyRunState{run: input.Run, session: input.Session, maxTurns: 5, characters: snapshot.Characters, variable: seed}
	batch := sceneBatchResult{
		Events: []model.StoryEventPlan{
			{TimeIndex: 1, CharacterID: "character_1", LocationKey: "old_dock", LocationName: "Old Dock", ActionType: "action", Summary: "Lin reaches the dock"},
			{TimeIndex: 2, CharacterID: "character_2", LocationKey: "old_dock", LocationName: "Old Dock", ActionType: "observe", Summary: "Shen waits at the dock"},
			{TimeIndex: 3, CharacterID: "character_3", LocationKey: "hall", LocationName: "Hall", ActionType: "observe", Summary: "Yan watches the hall"},
		},
		InteractionGroups: []model.StoryInteractionGroup{
			{ID: "bad_cross_location", LocationKey: "old_dock", CharacterIDs: []string{"character_1", "character_3"}, ShouldInteract: true},
			{ID: "valid_group", LocationKey: "old_dock", CharacterIDs: []string{"character_1", "character_2"}, ShouldInteract: true},
		},
		Turns: []StoryTurnPlan{
			{TurnIndex: 1, ActorID: "character_1", ActionType: "speak", Speech: "You followed me.", Intent: "challenge Shen", TargetActorIDs: []string{"character_2"}, InteractionGroupID: "valid_group", LocationKey: "old_dock"},
			{TurnIndex: 2, ActorID: "character_3", ActionType: "speak", Speech: "I heard everything.", Intent: "intrude", TargetActorIDs: []string{"character_1"}, InteractionGroupID: "valid_group", LocationKey: "hall"},
		},
	}

	plan, _, err := generator.consumeSceneBatch(context.Background(), input, snapshot, state, batch, seed)
	if err != nil {
		t.Fatalf("consumeSceneBatch returned error: %v", err)
	}
	if len(plan.InteractionAnalysis.InteractionGroups) != 1 || plan.InteractionAnalysis.InteractionGroups[0].ID != "valid_group" {
		t.Fatalf("expected only the valid interaction group, got %#v", plan.InteractionAnalysis.InteractionGroups)
	}
	if len(plan.Turns) != 1 || plan.Turns[0].ActorID != "character_1" {
		t.Fatalf("expected invalid turn to be dropped, got %#v", plan.Turns)
	}
	if !containsIssue(plan.ContinuityIssues, "跳过跨地点交涉组") {
		t.Fatalf("expected cross-location issue, got %#v", plan.ContinuityIssues)
	}
	if !containsIssue(plan.ContinuityIssues, "story turn actor must belong to interaction group") {
		t.Fatalf("expected invalid turn issue, got %#v", plan.ContinuityIssues)
	}
}

func TestConsumeSceneBatchTruncatesMaxTurnsAndInfersEvents(t *testing.T) {
	generator := &StoryRunGenerator{maxTurns: 1}
	input := port.StoryRunGenerationInput{
		Run:     model.StoryRun{RunID: "run_1", SessionID: "story_1", ProjectID: "project_1"},
		Session: modelStorySession(),
	}
	snapshot := StoryContextSnapshot{Characters: []model.Character{{ID: "character_1", Name: "Lin"}}}
	seed := StoryVariablePlan{PlotVariable: StoryNarrativePlotVariable{CoreChoice: "whether Lin escapes", RelatedCharacterIDs: []string{"character_1"}}}
	state := &storyRunState{run: input.Run, session: input.Session, maxTurns: 1, characters: snapshot.Characters, variable: seed}
	batch := sceneBatchResult{Turns: []StoryTurnPlan{
		{TurnIndex: 1, ActorID: "character_1", ActionType: "action", ActionSummary: "Lin checks the locked gate", Intent: "find a way out", LocationKey: "gate"},
		{TurnIndex: 2, ActorID: "character_1", ActionType: "action", ActionSummary: "Lin tries the alley", Intent: "escape", LocationKey: "gate"},
	}}

	plan, _, err := generator.consumeSceneBatch(context.Background(), input, snapshot, state, batch, seed)
	if err != nil {
		t.Fatalf("consumeSceneBatch returned error: %v", err)
	}
	if len(plan.Turns) != 1 {
		t.Fatalf("expected max_turns truncation, got %#v", plan.Turns)
	}
	if plan.StopReason != "达到最大回合数" {
		t.Fatalf("stop reason = %q", plan.StopReason)
	}
	if !containsIssue(plan.ContinuityIssues, "超过 max_turns") {
		t.Fatalf("expected max_turns issue, got %#v", plan.ContinuityIssues)
	}
	if len(plan.EventPlan) != 1 || plan.EventPlan[0].Summary != "Lin checks the locked gate" {
		t.Fatalf("expected event inferred from surviving turn, got %#v", plan.EventPlan)
	}
}

func TestReflectSceneUsesPerceptionIndex(t *testing.T) {
	chatModel := &fakeStoryChatModel{responses: []string{
		`{"summary":"Lin and Shen test each other.","character_takeaways":[{"character_id":"character_2","summary":"Shen suspects Lin."}],"memory_patch":{"character_memory_updates":[{"character_id":"character_2","type":"belief","content":"Lin denied carrying the letter.","importance":4}],"relationship_updates":[],"world_state_updates":[]}}`,
	}}
	generator := &StoryRunGenerator{cfg: config.AIConfig{Model: "test-model"}, model: chatModel, maxReflectTokens: 128}
	input := port.StoryRunGenerationInput{Run: model.StoryRun{RunID: "run_1", ProjectID: "project_1"}, Session: modelStorySession()}
	snapshot := StoryContextSnapshot{
		Characters:     []model.Character{{ID: "character_1", Name: "Lin"}, {ID: "character_2", Name: "Shen"}},
		RecentMemories: map[string][]model.Memory{"character_2": {{CharacterID: "character_2", Content: "Shen already distrusts Lin."}}},
	}
	plan := StoryPlanResult{
		EventPlan: []model.StoryEventPlan{{ID: "private_1", CharacterID: "character_1", LocationKey: "old_dock", ActionType: "action", Summary: "Lin palms the letter", Visibility: "private"}},
		Turns:     []StoryTurnPlan{{TurnIndex: 1, ActorID: "character_1", ActorName: "Lin", ActionType: "speak", Speech: "The letter is not with me.", TargetActorIDs: []string{"character_2"}, LocationKey: "old_dock"}},
	}
	variable := StoryVariablePlan{PlotVariable: StoryNarrativePlotVariable{CoreChoice: "whether Lin reveals the letter trail"}}
	reflectionContext := generator.buildReflectionContext(input, snapshot, plan, variable)
	shen := perceptionEntryByID(reflectionContext.PerceptionIndex, "character_2")
	if shen == nil {
		t.Fatal("expected Shen perception entry")
	}
	if containsString(shen.WitnessedEventIDs, "private_1") {
		t.Fatalf("private event leaked into Shen perception: %#v", shen)
	}
	if len(shen.WitnessedTurnIndexes) != 1 || shen.WitnessedTurnIndexes[0] != 1 {
		t.Fatalf("expected Shen to witness addressed turn, got %#v", shen)
	}
	if !containsString(reflectionContext.PriorMemories["character_2"], "Shen already distrusts Lin.") {
		t.Fatalf("expected prior memories in reflection context, got %#v", reflectionContext.PriorMemories)
	}
	if !strings.Contains(generator.reflectSystemPrompt(), "Deduplicate prior_memories") {
		t.Fatalf("reflect prompt should require prior memory dedupe, got %q", generator.reflectSystemPrompt())
	}
	reflection, err := generator.reflectScene(context.Background(), input, snapshot, plan, variable)
	if err != nil {
		t.Fatalf("reflectScene returned error: %v", err)
	}
	if reflection.Summary == "" || len(reflection.MemoryPatch.CharacterMemoryUpdates) != 1 {
		t.Fatalf("unexpected reflection: %#v", reflection)
	}
}

func perceptionEntryByID(entries []PerceptionIndexEntry, characterID string) *PerceptionIndexEntry {
	for i := range entries {
		if entries[i].CharacterID == characterID {
			return &entries[i]
		}
	}
	return nil
}
func modelStorySession() model.StorySession {
	return model.StorySession{
		Title:             "雨巷密信",
		OpeningSituation:  "雨巷里有人截获密信",
		LastAuthorMessage: "推进雨巷密信试探",
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
	variable, err := generator.seedPlotVariable(context.Background(), port.StoryRunGenerationInput{
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
		t.Fatalf("seedPlotVariable returned error: %v", err)
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

func TestBuildSceneContextOnlyIncludesOwnRelationshipView(t *testing.T) {
	generator := &StoryRunGenerator{maxTurns: 25}
	sceneContext := generator.buildSceneContext(port.StoryRunGenerationInput{
		Run:     model.StoryRun{RunID: "run_1", SessionID: "story_1", ProjectID: "project_1"},
		Session: modelStorySession(),
	}, StoryContextSnapshot{
		Characters: []model.Character{{ID: "character_1", Name: "林澈"}, {ID: "character_2", Name: "沈砚"}},
		Relationships: []model.Relationship{{Views: []model.RelationshipView{
			{SourceCharacterID: "character_1", TargetCharacterID: "character_2", PrivateAttitude: "警惕"},
			{SourceCharacterID: "character_2", TargetCharacterID: "character_1", PrivateAttitude: "嫉妒"},
		}}},
	}, StoryVariablePlan{PlotVariable: StoryNarrativePlotVariable{RelatedCharacterIDs: []string{"character_1", "character_2"}}}, nil)
	view := sceneCharacterViewByID(sceneContext.CharacterViews, "character_1")
	if view == nil {
		t.Fatal("expected character view")
	}
	if len(view.PrivateAttitude) != 1 {
		t.Fatalf("expected one private attitude row, got %#v", view.PrivateAttitude)
	}
	if view.PrivateAttitude[0]["private_attitude"] != "警惕" {
		t.Fatalf("unexpected private attitude %#v", view.PrivateAttitude[0])
	}
}

func TestBuildSceneContextOnlyIncludesOwnVariableView(t *testing.T) {
	generator := &StoryRunGenerator{maxTurns: 25}
	sceneContext := generator.buildSceneContext(port.StoryRunGenerationInput{
		Run:     model.StoryRun{RunID: "run_1", SessionID: "story_1", ProjectID: "project_1"},
		Session: modelStorySession(),
	}, StoryContextSnapshot{
		Characters: []model.Character{{ID: "character_1", Name: "林澈"}, {ID: "character_2", Name: "沈砚"}},
	}, StoryVariablePlan{
		PlotVariable: StoryNarrativePlotVariable{RelatedCharacterIDs: []string{"character_1", "character_2"}},
		CharacterViews: []CharacterVariableView{
			{CharacterID: "character_1", KnownFacts: []string{"城门提前关闭"}, EmotionalPressure: "必须决定是否暴露身份"},
			{CharacterID: "character_2", KnownFacts: []string{"林澈行踪异常"}, EmotionalPressure: "怀疑林澈隐瞒"},
		},
	}, nil)
	view := sceneCharacterViewByID(sceneContext.CharacterViews, "character_1")
	if view == nil {
		t.Fatal("expected character view")
	}
	if !containsString(view.KnownFacts, "城门提前关闭") {
		t.Fatalf("expected own known fact, got %#v", view.KnownFacts)
	}
	if strings.Contains(strings.Join(view.KnownFacts, ","), "行踪异常") {
		t.Fatalf("variable view leaked another character view: %#v", view)
	}
}

func TestPlanCharacterActionsUsesVisibleDecisionInput(t *testing.T) {
	decider := &fakeCharacterActionDecider{decision: model.CharacterActionDecision{
		ActionType:        "action",
		Description:       "去码头找沈砚摊牌",
		TargetLocationKey: "old_dock",
		DurationHours:     2,
		ParticipantIDs:    []string{"沈砚"},
		Rationale:         "必须确认密信去向",
	}}
	generator := &StoryRunGenerator{actionDecider: decider, clock: fakeClock{}}
	input := port.StoryRunGenerationInput{
		Run:     model.StoryRun{RunID: "run_1", SessionID: "story_1", ProjectID: "project_1"},
		Session: modelStorySession(),
	}
	snapshot := StoryContextSnapshot{
		WorldState: []model.WorldStateEntry{{Key: "gate_lockdown", Note: "城门提前关闭", Importance: 5}},
		Characters: []model.Character{
			{ID: "character_1", Name: "林澈", Status: "active"},
			{ID: "character_2", Name: "沈砚", Status: "active"},
		},
		Relationships: []model.Relationship{{
			Pair: model.RelationshipPair{ID: "rel_1", LeftCharacterID: "character_1", RightCharacterID: "character_2", Summary: "旧识互疑"},
			Views: []model.RelationshipView{
				{SourceCharacterID: "character_1", TargetCharacterID: "character_2", PrivateAttitude: "警惕"},
				{SourceCharacterID: "character_2", TargetCharacterID: "character_1", PrivateAttitude: "嫉妒"},
			},
		}},
	}

	planned, err := generator.planCharacterActions(context.Background(), input, snapshot, StoryVariablePlan{
		PlotVariable: StoryNarrativePlotVariable{RelatedCharacterIDs: []string{"character_1"}},
	})
	if err != nil {
		t.Fatalf("planCharacterActions() error = %v", err)
	}
	if len(planned) != 1 {
		t.Fatalf("planned actions = %#v, want one", planned)
	}
	if len(planned[0].ParticipantIDs) != 1 || planned[0].ParticipantIDs[0] != "character_2" {
		t.Fatalf("participant ids = %#v, want character_2", planned[0].ParticipantIDs)
	}
	if len(decider.inputs) != 1 {
		t.Fatalf("decider calls = %d, want 1", len(decider.inputs))
	}
	relationship := decider.inputs[0].World.Relationships["rel_1"]
	if len(relationship.Views) != 1 {
		t.Fatalf("visible relationship views = %#v, want one own view", relationship.Views)
	}
	if relationship.Views[0].PrivateAttitude != "警惕" {
		t.Fatalf("visible relationship leaked wrong private attitude: %#v", relationship.Views)
	}
}

func sceneCharacterViewByID(views []SceneCharacterView, characterID string) *SceneCharacterView {
	for i := range views {
		if views[i].CharacterID == characterID {
			return &views[i]
		}
	}
	return nil
}

func containsIssue(issues []string, want string) bool {
	for _, issue := range issues {
		if strings.Contains(issue, want) {
			return true
		}
	}
	return false
}
