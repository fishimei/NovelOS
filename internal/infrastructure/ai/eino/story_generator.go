package eino

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	llmmodel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"

	"github.com/fishimei/NovelOS/internal/application/model"
	"github.com/fishimei/NovelOS/internal/application/port"
	"github.com/fishimei/NovelOS/internal/config"
	"github.com/fishimei/NovelOS/internal/domain"
)

type StoryRunGeneratorDeps struct {
	Config        config.AIConfig
	Sessions      port.StorySessionRepository
	AuthorBibles  port.AuthorBibleRepository
	WorldState    port.WorldStateRepository
	Characters    port.CharacterRepository
	Relationships port.RelationshipRepository
	Chapters      port.ChapterRepository
	Memories      port.MemoryRepository
	MemoryService port.CharacterMemoryService
	Events        port.GenerationEventStream
	Audit         port.AuditRepository
	Clock         port.Clock
	IDs           port.IDGenerator
}

type StoryRunGenerator struct {
	cfg             config.AIConfig
	model           llmmodel.ToolCallingChatModel
	deps            storyGeneratorDeps
	clock           port.Clock
	ids             port.IDGenerator
	maxTurns        int
	controller      string
	toolPrompt      string
	resultPrompt    string
	narrativePrompt string
	variablePrompt  string
}

func NewStoryRunGenerator(ctx context.Context, deps StoryRunGeneratorDeps) (*StoryRunGenerator, error) {
	maxTurns := deps.Config.StoryAgent.MaxTurns
	if maxTurns <= 0 {
		maxTurns = 25
	}
	if maxTurns > 25 {
		maxTurns = 25
	}
	chatModel, err := newOpenAIChatModel(ctx, deps.Config)
	if err != nil {
		return nil, err
	}
	generator := &StoryRunGenerator{
		cfg:   deps.Config,
		model: chatModel,
		deps: storyGeneratorDeps{
			sessions:      deps.Sessions,
			authorBibles:  deps.AuthorBibles,
			worldState:    deps.WorldState,
			characters:    deps.Characters,
			relationships: deps.Relationships,
			chapters:      deps.Chapters,
			memories:      deps.Memories,
			memoryService: deps.MemoryService,
			events:        deps.Events,
			audit:         deps.Audit,
		},
		clock:           deps.Clock,
		ids:             deps.IDs,
		maxTurns:        maxTurns,
		controller:      deps.Config.StoryAgent.ControllerPrompt,
		toolPrompt:      deps.Config.StoryAgent.ToolPrompt,
		resultPrompt:    deps.Config.StoryAgent.ResultPrompt,
		narrativePrompt: deps.Config.StoryAgent.NarrativePrompt,
		variablePrompt:  deps.Config.StoryAgent.VariablePrompt,
	}
	return generator, nil
}

func (g *StoryRunGenerator) Generate(ctx context.Context, input port.StoryRunGenerationInput) (model.StoryRunResult, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	state := &storyRunState{run: input.Run, session: input.Session, maxTurns: g.maxTurns}
	g.publishStoryOrchestrationStarted(ctx, input)
	snapshot, err := loadStoryContext(ctx, g.deps, state, LoadStoryContextInput{ProjectID: input.Run.ProjectID, SessionID: input.Session.ID})
	if err != nil {
		return model.StoryRunResult{}, err
	}
	updateStoryRunStep(ctx, g.deps, input.Run.RunID, domain.RunStatusGeneratingPlotVariable, 20)
	variable, err := g.generateStoryVariable(ctx, input, snapshot)
	if err != nil {
		return model.StoryRunResult{}, fmt.Errorf("generate story variable: %w", err)
	}
	state.variable = variable
	publishStoryEvent(ctx, g.deps, input.Run.RunID, domain.EventPlotVariable, map[string]any{"plot_variable": variable.PlotVariable})
	tools, err := newStoryTools(g.deps, state)
	if err != nil {
		return model.StoryRunResult{}, err
	}
	agent, err := react.NewAgent(ctx, &react.AgentConfig{
		ToolCallingModel: g.model,
		ToolsConfig:      compose.ToolsNodeConfig{Tools: tools},
		MaxStep:          g.maxTurns*3 + 5,
		ToolReturnDirectly: map[string]struct{}{
			"finalize_story_plan": {},
		},
	})
	if err != nil {
		return model.StoryRunResult{}, err
	}
	updateStoryRunStep(ctx, g.deps, input.Run.RunID, domain.RunStatusSimulatingEvents, 30)
	_, err = agent.Generate(ctx, []*schema.Message{
		schema.SystemMessage(g.systemPrompt()),
		schema.UserMessage(g.userPrompt(input, variable)),
	})
	if err != nil {
		return model.StoryRunResult{}, fmt.Errorf("drive story turns: %w", err)
	}
	plan := state.planResult()
	if len(plan.Turns) == 0 {
		plan.Turns = []StoryTurnPlan{{TurnIndex: 1, ActorName: "旁白", ActionType: "narration", Intent: firstText(variable.PlotVariable.CoreChoice, input.Session.LastAuthorMessage, input.Session.AuthorIntent, input.Session.OpeningSituation, "推进当前故事变量"), Rationale: "模型未产生回合，使用旁白占位"}}
	}
	updateStoryRunStep(ctx, g.deps, input.Run.RunID, domain.RunStatusWritingNarrative, 80)
	narrative, err := g.generateNarrative(ctx, input, snapshot, plan, variable)
	if err != nil {
		return model.StoryRunResult{}, fmt.Errorf("generate narrative: %w", err)
	}
	updateStoryRunStep(ctx, g.deps, input.Run.RunID, domain.RunStatusGeneratingMemoryPatch, 90)
	result, err := g.buildResult(ctx, input, plan, narrative, variable)
	if err != nil {
		return model.StoryRunResult{}, err
	}
	return result, nil
}

func (g *StoryRunGenerator) publishStoryOrchestrationStarted(ctx context.Context, input port.StoryRunGenerationInput) {
	publishStoryEvent(ctx, g.deps, input.Run.RunID, domain.EventStoryOrchestrationStarted, map[string]any{
		"story_run_id":      input.Run.RunID,
		"session_id":        input.Session.ID,
		"author_message":    input.Session.LastAuthorMessage,
		"author_intent":     input.Session.AuthorIntent,
		"opening_situation": input.Session.OpeningSituation,
	})
}

func (g *StoryRunGenerator) systemPrompt() string {
	parts := []string{g.controller, g.toolPrompt, g.resultPrompt}
	return strings.Join(parts, "\n\n")
}

func (g *StoryRunGenerator) userPrompt(input port.StoryRunGenerationInput, variable StoryVariablePlan) string {
	payload, _ := json.Marshal(map[string]any{
		"story_run_id":        input.Run.RunID,
		"project_id":          input.Run.ProjectID,
		"session_id":          input.Session.ID,
		"title":               input.Session.Title,
		"opening_situation":   input.Session.OpeningSituation,
		"author_intent":       input.Session.AuthorIntent,
		"last_author_message": input.Session.LastAuthorMessage,
		"story_variable":      variable,
	})
	return fmt.Sprintf(`%s

你是 NovelOS 的事件模拟和多人物交涉编排阶段，不是最终正文作者。先调用 load_story_context 核对当前状态。然后按顺序推进：
	1. 事件模拟：为与 story_variable 相关的主要角色调用 record_story_event，记录他们本时间片在什么地点做什么。
	2. 同地点分析：record_story_event 返回 same_location_candidates 后，只能从这些候选中调用 select_story_interaction 判断哪些角色会实际交互；不要选择不在同一地点的人。
	3. 多角色交涉：对 should_interact=true 的 interaction_group，用 choose_next_story_actor 记录交涉回合，并带上 interaction_group_id、location_key、phase=negotiation。
	4. 如果没有同地点候选或没有可交互组，就用旁白/行动回合总结事件推进，不要强行制造对话。
	最多 %d 个业务回合内停止。停止时调用 decide_story_stop 与 finalize_story_plan。不要生成完整章节正文。`, string(payload), g.maxTurns)
}

func (g *StoryRunGenerator) generateStoryVariable(ctx context.Context, input port.StoryRunGenerationInput, snapshot StoryContextSnapshot) (StoryVariablePlan, error) {
	plot := StoryNarrativePlotVariable{
		PressureSource:     firstText(input.Session.OpeningSituation, input.Session.LastAuthorMessage, input.Session.AuthorIntent, input.Session.CurrentPlotVariableSummary, "当前故事压力"),
		FocalCharacterID:   storyVariableFocalCharacterID(input, snapshot),
		CoreChoice:         firstText(input.Session.LastAuthorMessage, input.Session.AuthorIntent, input.Session.OpeningSituation, input.Session.CurrentPlotVariableSummary, "推进当前故事变量"),
		OptionA:            "暂时维持当前局面",
		OptionB:            "主动打破当前局面",
		CostA:              "压力继续累积",
		CostB:              "暴露意图或承担代价",
		IrreversibleEffect: firstText(input.Session.CurrentPlotVariableSummary, "本章状态将发生不可逆变化"),
		WorldStatePressure: storyVariableWorldStatePressure(snapshot.WorldState),
	}
	plot.RelatedCharacterIDs = storyVariableRelatedCharacterIDs(plot.FocalCharacterID, input, snapshot)
	variable := StoryVariablePlan{
		PlotVariable:   plot,
		CharacterViews: storyVariableCharacterViews(plot, snapshot),
	}
	return normalizeStoryVariable(variable, input, snapshot), nil
}

func storyVariableFocalCharacterID(input port.StoryRunGenerationInput, snapshot StoryContextSnapshot) string {
	text := strings.Join([]string{input.Session.LastAuthorMessage, input.Session.AuthorIntent, input.Session.OpeningSituation, input.Session.CurrentPlotVariableSummary}, "\n")
	for _, character := range snapshot.Characters {
		if character.Name != "" && strings.Contains(text, character.Name) {
			return character.ID
		}
	}
	if len(snapshot.Characters) > 0 {
		return snapshot.Characters[0].ID
	}
	return ""
}

func storyVariableWorldStatePressure(worldState []model.WorldStateEntry) []string {
	keys := make([]string, 0, 3)
	for _, entry := range worldState {
		if entry.Key == "" || (entry.Importance < 4 && entry.Volatility < 4) {
			continue
		}
		keys = append(keys, entry.Key)
		if len(keys) == 3 {
			return keys
		}
	}
	for _, entry := range worldState {
		if entry.Key == "" || containsString(keys, entry.Key) {
			continue
		}
		keys = append(keys, entry.Key)
		if len(keys) == 3 {
			break
		}
	}
	return keys
}

func storyVariableRelatedCharacterIDs(focalCharacterID string, input port.StoryRunGenerationInput, snapshot StoryContextSnapshot) []string {
	ids := make([]string, 0, 4)
	if focalCharacterID != "" {
		ids = append(ids, focalCharacterID)
	}
	text := strings.Join([]string{input.Session.LastAuthorMessage, input.Session.AuthorIntent, input.Session.OpeningSituation, input.Session.CurrentPlotVariableSummary}, "\n")
	for _, character := range snapshot.Characters {
		if character.ID == focalCharacterID || character.Name == "" || !strings.Contains(text, character.Name) {
			continue
		}
		ids = append(ids, character.ID)
	}
	for _, relationship := range snapshot.Relationships {
		if relationship.Pair.LeftCharacterID == focalCharacterID {
			ids = appendUniqueString(ids, relationship.Pair.RightCharacterID)
		}
		if relationship.Pair.RightCharacterID == focalCharacterID {
			ids = appendUniqueString(ids, relationship.Pair.LeftCharacterID)
		}
		if len(ids) >= 4 {
			break
		}
	}
	return ids
}

func storyVariableCharacterViews(plot StoryNarrativePlotVariable, snapshot StoryContextSnapshot) []CharacterVariableView {
	views := make([]CharacterVariableView, 0, len(plot.RelatedCharacterIDs))
	for _, characterID := range plot.RelatedCharacterIDs {
		views = append(views, CharacterVariableView{
			CharacterID:       characterID,
			KnownFacts:        storyVariableKnownFacts(plot, snapshot, characterID),
			Misreadings:       storyVariableMisreadings(snapshot, characterID),
			EmotionalPressure: plot.CoreChoice,
			ActionBias:        firstText(plot.OptionB, plot.OptionA),
		})
	}
	return views
}

func storyVariableKnownFacts(plot StoryNarrativePlotVariable, snapshot StoryContextSnapshot, characterID string) []string {
	facts := []string{plot.PressureSource}
	for _, entry := range visibleWorldForCharacter(snapshot.WorldState, characterByID(snapshot.Characters, characterID)) {
		facts = append(facts, firstText(entry.Note, entry.Key))
		if len(facts) == 3 {
			break
		}
	}
	return cleanStrings(facts)
}

func storyVariableMisreadings(snapshot StoryContextSnapshot, characterID string) []string {
	for _, relationship := range snapshot.Relationships {
		for _, view := range relationship.Views {
			if view.SourceCharacterID == characterID && view.BelievedTargetAttitude != "" {
				return []string{view.BelievedTargetAttitude}
			}
		}
	}
	return nil
}

func characterByID(characters []model.Character, characterID string) model.Character {
	for _, character := range characters {
		if character.ID == characterID {
			return character
		}
	}
	return model.Character{}
}

func appendUniqueString(values []string, value string) []string {
	if value == "" || containsString(values, value) {
		return values
	}
	return append(values, value)
}

func containsString(values []string, value string) bool {
	for _, existing := range values {
		if existing == value {
			return true
		}
	}
	return false
}

func storyVariablePromptInput(input port.StoryRunGenerationInput, snapshot StoryContextSnapshot) map[string]any {
	return map[string]any{
		"session": map[string]any{
			"title":                 input.Session.Title,
			"opening_situation":     input.Session.OpeningSituation,
			"author_intent":         input.Session.AuthorIntent,
			"last_author_message":   input.Session.LastAuthorMessage,
			"current_plot_variable": input.Session.CurrentPlotVariableSummary,
		},
		"author_bible":    compactAuthorBible(snapshot.AuthorBible),
		"world_state":     compactWorldState(snapshot.WorldState, 6),
		"characters":      compactCharacters(snapshot.Characters, 8),
		"relationships":   compactRelationships(snapshot.Relationships, 8),
		"recent_chapters": compactChapters(snapshot.RecentChapters, 3),
		"recent_memories": compactMemories(snapshot.RecentMemories, 2),
	}
}

func compactAuthorBible(bible *model.AuthorBible) map[string]any {
	if bible == nil {
		return nil
	}
	return map[string]any{
		"theme":                bible.Theme,
		"style_guide":          bible.StyleGuide,
		"world_rules":          firstStrings(bible.WorldRules, 4),
		"hard_constraints":     firstStrings(bible.HardConstraints, 4),
		"forbidden_moves":      firstStrings(bible.ForbiddenMoves, 4),
		"aesthetic_principles": firstStrings(bible.AestheticPrinciples, 4),
	}
}

func compactWorldState(entries []model.WorldStateEntry, limit int) []map[string]any {
	out := make([]map[string]any, 0, minInt(len(entries), limit))
	for _, entry := range firstEntries(entries, limit) {
		out = append(out, map[string]any{
			"key":        entry.Key,
			"value":      entry.Value,
			"note":       entry.Note,
			"importance": entry.Importance,
			"volatility": entry.Volatility,
		})
	}
	return out
}

func compactCharacters(characters []model.Character, limit int) []map[string]any {
	out := make([]map[string]any, 0, minInt(len(characters), limit))
	for _, character := range firstEntries(characters, limit) {
		out = append(out, map[string]any{
			"id":          character.ID,
			"name":        character.Name,
			"role":        character.Role,
			"profile":     character.Profile,
			"goals":       firstStrings(character.Goals, 3),
			"fears":       firstStrings(character.Fears, 3),
			"constraints": firstStrings(character.Constraints, 3),
		})
	}
	return out
}

func compactRelationships(relationships []model.Relationship, limit int) []map[string]any {
	out := make([]map[string]any, 0, minInt(len(relationships), limit))
	for _, relationship := range firstEntries(relationships, limit) {
		out = append(out, map[string]any{
			"pair_id":            relationship.Pair.ID,
			"left_character_id":  relationship.Pair.LeftCharacterID,
			"right_character_id": relationship.Pair.RightCharacterID,
			"summary":            relationship.Pair.Summary,
			"tension_points":     firstStrings(relationship.Pair.TensionPoints, 3),
		})
	}
	return out
}

func compactChapters(chapters []model.Chapter, limit int) []map[string]any {
	out := make([]map[string]any, 0, minInt(len(chapters), limit))
	for _, chapter := range firstEntries(chapters, limit) {
		out = append(out, map[string]any{
			"title":   chapter.Title,
			"summary": chapter.Summary,
		})
	}
	return out
}

func compactMemories(memories map[string][]model.Memory, limitPerCharacter int) map[string][]string {
	out := make(map[string][]string, len(memories))
	for characterID, items := range memories {
		contents := make([]string, 0, minInt(len(items), limitPerCharacter))
		for _, memory := range firstEntries(items, limitPerCharacter) {
			contents = append(contents, memory.Content)
		}
		if len(contents) > 0 {
			out[characterID] = contents
		}
	}
	return out
}

func firstStrings(values []string, limit int) []string {
	return firstEntries(values, limit)
}

func firstEntries[T any](values []T, limit int) []T {
	if limit < 0 || len(values) <= limit {
		return values
	}
	return values[:limit]
}

func minInt(a int, b int) int {
	if a < b {
		return a
	}
	return b
}

func (g *StoryRunGenerator) generateNarrative(ctx context.Context, input port.StoryRunGenerationInput, snapshot StoryContextSnapshot, plan StoryPlanResult, variable StoryVariablePlan) (StoryNarrativeResult, error) {
	return g.generateNarrativeSummary(ctx, input, snapshot, plan, variable), nil
}

func (g *StoryRunGenerator) generateTurnContent(ctx context.Context, input port.StoryRunGenerationInput, snapshot StoryContextSnapshot, plan StoryPlanResult, previousTurns []StoryTurnPlan, turn StoryTurnPlan, variable StoryVariablePlan) (StoryTurnPlan, error) {
	promptInput := map[string]any{
		"session":         input.Session,
		"author_bible":    snapshot.AuthorBible,
		"recent_chapters": snapshot.RecentChapters,
		"plan_summary":    plan.Summary,
		"stop_reason":     plan.StopReason,
		"previous_turns":  previousTurns,
		"current_turn":    turn,
		"story_variable":  variable.PlotVariable,
		"perspective":     g.perspectiveForTurn(snapshot, turn, variable),
	}
	payload, _ := json.Marshal(promptInput)
	msg, err := g.model.Generate(ctx, []*schema.Message{
		schema.SystemMessage(g.turnNarrativePrompt()),
		schema.UserMessage(string(payload)),
	}, maxTokensOption(g.cfg.Model, 1200))
	if err != nil {
		return StoryTurnPlan{}, err
	}
	var out struct {
		Content string `json:"content"`
	}
	if err := decodeModelJSON(msg.Content, &out); err != nil {
		return StoryTurnPlan{}, err
	}
	turn.Content = firstText(out.Content, turn.Intent)
	return turn, nil
}

func (g *StoryRunGenerator) generateNarrativeSummary(ctx context.Context, input port.StoryRunGenerationInput, snapshot StoryContextSnapshot, plan StoryPlanResult, variable StoryVariablePlan) StoryNarrativeResult {
	promptInput := map[string]any{
		"session":                 input.Session,
		"author_bible":            snapshot.AuthorBible,
		"recent_chapters":         snapshot.RecentChapters,
		"world_state_keys":        worldStateKeys(snapshot.WorldState),
		"relationships":           relationshipPublicSummaries(snapshot.Relationships),
		"story_variable":          variable.PlotVariable,
		"event_timeline":          plan.EventTimeline,
		"interaction_analysis":    plan.InteractionAnalysis,
		"interaction_transcripts": plan.InteractionTranscripts,
		"turns":                   plan.Turns,
		"stop_reason":             plan.StopReason,
	}
	payload, _ := json.Marshal(promptInput)
	msg, err := g.model.Generate(ctx, []*schema.Message{
		schema.SystemMessage(g.summaryPrompt()),
		schema.UserMessage(string(payload)),
	}, maxTokensOption(g.cfg.Model, 4000))
	if err == nil {
		var out StoryNarrativeResult
		if decodeModelJSON(msg.Content, &out) == nil && storyNarrativeResultUsable(out) {
			if len(out.Turns) == 0 {
				out.Turns = plan.Turns
			}
			if out.Content == "" {
				out.Content = formatDraftContent(StoryPlanResult{Summary: out.Summary, StopReason: plan.StopReason, Turns: out.Turns})
			}
			return out
		}
	}
	return StoryNarrativeResult{
		Title:   firstText(input.Session.Title, "未命名章节"),
		Summary: plan.Summary,
		Content: formatDraftContent(plan),
		PlotVariable: firstVariable(variable.PlotVariable, StoryNarrativePlotVariable{
			PressureSource:      firstText(input.Session.OpeningSituation, input.Session.LastAuthorMessage, input.Session.AuthorIntent, "当前故事压力"),
			FocalCharacterID:    firstActorID(plan.Turns),
			CoreChoice:          firstText(plan.Summary, input.Session.LastAuthorMessage, input.Session.AuthorIntent, "推进当前故事变量"),
			RelatedCharacterIDs: relatedCharacterIDs(plan.Turns),
		}),
		Review: StoryNarrativeReview{Pass: true},
		Turns:  plan.Turns,
	}
}

func storyNarrativeResultUsable(result StoryNarrativeResult) bool {
	return strings.TrimSpace(result.Title) != "" || strings.TrimSpace(result.Summary) != "" || strings.TrimSpace(result.Content) != "" || storyPlotVariableUsable(result.PlotVariable) || len(result.MemoryPatch.CharacterMemoryUpdates) > 0 || len(result.MemoryPatch.RelationshipUpdates) > 0 || len(result.MemoryPatch.WorldStateUpdates) > 0 || len(result.Turns) > 0
}

func storyPlotVariableUsable(variable StoryNarrativePlotVariable) bool {
	return strings.TrimSpace(variable.PressureSource) != "" || strings.TrimSpace(variable.CoreChoice) != "" || strings.TrimSpace(variable.FocalCharacterID) != "" || len(variable.RelatedCharacterIDs) > 0 || len(variable.WorldStatePressure) > 0
}

func (g *StoryRunGenerator) variableSystemPrompt() string {
	return firstText(g.variablePrompt, "你是 NovelOS 的剧情变量 agent。先生成推动本章状态变化的核心变量，并为角色生成受限视角变量切片。输出 JSON。") + `

输出 JSON 对象，字段包括 plot_variable 和 character_views。plot_variable 包含 pressure_source、focal_character_id、core_choice、option_a、option_b、cost_a、cost_b、irreversible_effect、related_character_ids、world_state_pressure。character_views 的每项包含 character_id、known_facts、misreadings、emotional_pressure、action_bias。只输出 JSON。`
}

func (g *StoryRunGenerator) turnNarrativePrompt() string {
	return firstText(g.narrativePrompt, "你是 NovelOS 的受限视角多角色演绎 agent。只根据当前 actor 的 perspective 生成本回合内容，输出 JSON：{\"content\":\"...\"}。") + `

本次只生成 current_turn 的正文。禁止使用 perspective 之外的秘密、private_attitude 或全局真相。可使用 perspective.variable_view 中该角色可感知的变量切片，但不能使用其他角色切片或全局隐藏信息。speak 写角色台词和必要动作；action 写可观察动作；silence/observe 写沉默或观察；narration 写短叙事。只输出 JSON。`
}

func (g *StoryRunGenerator) summaryPrompt() string {
	return firstText(g.resultPrompt, "整理故事演绎结果。") + `

根据已完成的 event_timeline、interaction_analysis、interaction_transcripts、turns 和 story_variable 输出 JSON 对象，字段包括 title、summary、content、plot_variable、memory_patch、review、turns。运行中的事件和 turns 只是素材，不是章节正文；content 必须在本阶段统一写成连贯完整章节正文，不要只列提纲。memory_patch 只记录本章实际发生且角色会记住/世界会改变的内容。relationship_updates 可填 pair_id、summary、tension_delta、events；交涉造成的关系变化应写入 events，event_type 可用 negotiation 或 interaction_outcome。只输出 JSON。`
}

func (g *StoryRunGenerator) perspectiveForTurn(snapshot StoryContextSnapshot, turn StoryTurnPlan, variable StoryVariablePlan) *CharacterPerspective {
	if turn.ActorID == "" {
		return nil
	}
	for _, character := range snapshot.Characters {
		if character.ID != turn.ActorID {
			continue
		}
		return &CharacterPerspective{
			Character:         character,
			VisibleWorld:      visibleWorldForCharacter(snapshot.WorldState, character),
			RecentMemories:    snapshot.RecentMemories[character.ID],
			RelationshipViews: relationshipViewsForCharacter(snapshot.Relationships, character.ID),
			VariableView:      variableViewForCharacter(variable.CharacterViews, character.ID),
		}
	}
	return nil
}

func (g *StoryRunGenerator) buildResult(ctx context.Context, input port.StoryRunGenerationInput, plan StoryPlanResult, narrative StoryNarrativeResult, variable StoryVariablePlan) (model.StoryRunResult, error) {
	chapterNumber, err := g.nextChapterNumber(ctx, input.Run.ProjectID)
	if err != nil {
		return model.StoryRunResult{}, err
	}
	content := firstText(narrative.Content, formatDraftContent(plan))
	plotVariable := firstVariable(variable.PlotVariable, narrative.PlotVariable)
	focalID := firstText(plotVariable.FocalCharacterID, firstActorID(plan.Turns))
	relatedIDs := plotVariable.RelatedCharacterIDs
	if len(relatedIDs) == 0 {
		relatedIDs = relatedCharacterIDs(plan.Turns)
	}
	coreChoice := firstText(plotVariable.CoreChoice, narrative.Summary, plan.Summary, input.Session.LastAuthorMessage, input.Session.AuthorIntent, "推进当前故事变量")
	return model.StoryRunResult{
		RunID:     input.Run.RunID,
		SessionID: input.Run.SessionID,
		Status:    domain.RunStatusReviewRequired,
		PlotVariable: model.PlotVariable{
			PressureSource:      firstText(plotVariable.PressureSource, input.Session.OpeningSituation, input.Session.LastAuthorMessage, input.Session.AuthorIntent, "当前故事压力"),
			FocalCharacterID:    focalID,
			CoreChoice:          coreChoice,
			OptionA:             firstText(plotVariable.OptionA, "暂时维持当前局面"),
			OptionB:             firstText(plotVariable.OptionB, "主动打破当前局面"),
			CostA:               firstText(plotVariable.CostA, "压力继续累积"),
			CostB:               firstText(plotVariable.CostB, "暴露意图或承担代价"),
			IrreversibleEffect:  firstText(plotVariable.IrreversibleEffect, plan.StopReason, "本轮回合裁决结束"),
			RelatedCharacterIDs: relatedIDs,
			WorldStatePressure:  plotVariable.WorldStatePressure,
		},
		EventTimeline:          plan.EventTimeline,
		InteractionAnalysis:    plan.InteractionAnalysis,
		InteractionTranscripts: plan.InteractionTranscripts,
		Draft: model.Draft{
			ID:            g.newID("draft"),
			Title:         firstText(narrative.Title, input.Session.Title, "未命名章节"),
			ChapterNumber: chapterNumber,
			Content:       content,
			Summary:       coreChoice,
			WordCount:     utf8.RuneCountInString(content),
		},
		Review: model.ReviewReport{
			Pass:             narrative.Review.Pass,
			HardViolations:   narrative.Review.HardViolations,
			ContinuityIssues: narrative.Review.ContinuityIssues,
			StyleIssues:      narrative.Review.StyleIssues,
			SuggestedFixes:   narrative.Review.SuggestedFixes,
		},
		MemoryPatch: model.MemoryPatch{
			ID:                     g.newID("patch"),
			Status:                 "draft",
			CharacterMemoryUpdates: toCharacterMemoryUpdates(narrative.MemoryPatch.CharacterMemoryUpdates),
			RelationshipUpdates:    toRelationshipUpdates(narrative.MemoryPatch.RelationshipUpdates),
			WorldStateUpdates:      toWorldStateUpdates(narrative.MemoryPatch.WorldStateUpdates),
		},
	}, nil
}

func (g *StoryRunGenerator) nextChapterNumber(ctx context.Context, projectID string) (int, error) {
	chapters, err := g.deps.chapters.ListByProjectID(ctx, projectID, model.PageInput{Page: 1, PageSize: 1000})
	if err != nil {
		return 0, err
	}
	maxNumber := 0
	for _, chapter := range chapters.Items {
		if chapter.ChapterNumber > maxNumber {
			maxNumber = chapter.ChapterNumber
		}
	}
	return maxNumber + 1, nil
}

func (g *StoryRunGenerator) newID(prefix string) string {
	if g.ids != nil {
		return g.ids.New(prefix)
	}
	return fmt.Sprintf("%s_%d", prefix, g.now().UnixNano())
}

func (g *StoryRunGenerator) now() time.Time {
	if g.clock != nil {
		return g.clock.Now()
	}
	return time.Now().UTC()
}

func visibleWorldForCharacter(worldState []model.WorldStateEntry, character model.Character) []model.WorldStateEntry {
	visible := make([]model.WorldStateEntry, 0, len(worldState))
	for _, entry := range worldState {
		if entry.Importance >= 4 || entry.Volatility >= 4 || strings.Contains(entry.Note, character.Name) || strings.Contains(entry.Key, character.ID) {
			visible = append(visible, entry)
		}
	}
	if len(visible) == 0 && len(worldState) > 0 {
		limit := 3
		if len(worldState) < limit {
			limit = len(worldState)
		}
		visible = append(visible, worldState[:limit]...)
	}
	return visible
}

func relationshipViewsForCharacter(relationships []model.Relationship, characterID string) []model.RelationshipView {
	views := make([]model.RelationshipView, 0)
	for _, relationship := range relationships {
		for _, view := range relationship.Views {
			if view.SourceCharacterID == characterID {
				views = append(views, view)
			}
		}
	}
	return views
}

func worldStateKeys(worldState []model.WorldStateEntry) []string {
	keys := make([]string, 0, len(worldState))
	for _, entry := range worldState {
		keys = append(keys, entry.Key)
	}
	return keys
}

func normalizeStoryVariable(variable StoryVariablePlan, input port.StoryRunGenerationInput, snapshot StoryContextSnapshot) StoryVariablePlan {
	validCharacters := map[string]struct{}{}
	for _, character := range snapshot.Characters {
		validCharacters[character.ID] = struct{}{}
	}
	plot := variable.PlotVariable
	if _, ok := validCharacters[plot.FocalCharacterID]; !ok {
		plot.FocalCharacterID = ""
	}
	if plot.FocalCharacterID == "" && len(snapshot.Characters) > 0 {
		plot.FocalCharacterID = snapshot.Characters[0].ID
	}
	plot.PressureSource = firstText(plot.PressureSource, input.Session.OpeningSituation, input.Session.LastAuthorMessage, input.Session.AuthorIntent, "当前故事压力")
	plot.CoreChoice = firstText(plot.CoreChoice, input.Session.LastAuthorMessage, input.Session.AuthorIntent, input.Session.OpeningSituation, "推进当前故事变量")
	plot.OptionA = firstText(plot.OptionA, "暂时维持当前局面")
	plot.OptionB = firstText(plot.OptionB, "主动打破当前局面")
	plot.CostA = firstText(plot.CostA, "压力继续累积")
	plot.CostB = firstText(plot.CostB, "暴露意图或承担代价")
	plot.IrreversibleEffect = firstText(plot.IrreversibleEffect, "本章状态将发生不可逆变化")
	plot.RelatedCharacterIDs = validCharacterIDs(plot.RelatedCharacterIDs, validCharacters)
	if len(plot.RelatedCharacterIDs) == 0 && plot.FocalCharacterID != "" {
		plot.RelatedCharacterIDs = []string{plot.FocalCharacterID}
	}
	if len(plot.WorldStatePressure) == 0 {
		plot.WorldStatePressure = worldStateKeys(snapshot.WorldState)
		if len(plot.WorldStatePressure) > 3 {
			plot.WorldStatePressure = plot.WorldStatePressure[:3]
		}
	}
	views := make([]CharacterVariableView, 0, len(variable.CharacterViews))
	seenViews := map[string]struct{}{}
	for _, view := range variable.CharacterViews {
		if _, ok := validCharacters[view.CharacterID]; !ok {
			continue
		}
		if _, ok := seenViews[view.CharacterID]; ok {
			continue
		}
		seenViews[view.CharacterID] = struct{}{}
		views = append(views, view)
	}
	for _, characterID := range plot.RelatedCharacterIDs {
		if _, ok := seenViews[characterID]; ok {
			continue
		}
		views = append(views, CharacterVariableView{
			CharacterID:       characterID,
			KnownFacts:        []string{plot.PressureSource},
			EmotionalPressure: plot.CoreChoice,
			ActionBias:        firstText(plot.OptionA, plot.OptionB),
		})
	}
	return StoryVariablePlan{PlotVariable: plot, CharacterViews: views}
}

func validCharacterIDs(ids []string, validCharacters map[string]struct{}) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		if _, ok := validCharacters[id]; !ok {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

func firstVariable(values ...StoryNarrativePlotVariable) StoryNarrativePlotVariable {
	for _, value := range values {
		if strings.TrimSpace(value.CoreChoice) != "" || strings.TrimSpace(value.PressureSource) != "" {
			return value
		}
	}
	return StoryNarrativePlotVariable{}
}

func variableViewForCharacter(views []CharacterVariableView, characterID string) *CharacterVariableView {
	for _, view := range views {
		if view.CharacterID != characterID {
			continue
		}
		copyView := view
		return &copyView
	}
	return nil
}

func relationshipPublicSummaries(relationships []model.Relationship) []map[string]any {
	items := make([]map[string]any, 0, len(relationships))
	for _, relationship := range relationships {
		items = append(items, map[string]any{
			"pair_id":            relationship.Pair.ID,
			"left_character_id":  relationship.Pair.LeftCharacterID,
			"right_character_id": relationship.Pair.RightCharacterID,
			"summary":            relationship.Pair.Summary,
			"tension_points":     relationship.Pair.TensionPoints,
		})
	}
	return items
}

func toCharacterMemoryUpdates(updates []StoryNarrativeCharacterMemoryUpdate) []model.CharacterMemoryUpdate {
	out := make([]model.CharacterMemoryUpdate, 0, len(updates))
	for _, update := range updates {
		if update.CharacterID == "" || update.Content == "" {
			continue
		}
		out = append(out, model.CharacterMemoryUpdate{CharacterID: update.CharacterID, Type: update.Type, Content: update.Content, Importance: update.Importance})
	}
	return out
}

func toRelationshipUpdates(updates []StoryNarrativeRelationshipUpdate) []model.RelationshipUpdate {
	out := make([]model.RelationshipUpdate, 0, len(updates))
	for _, update := range updates {
		if update.PairID == "" && update.Summary == "" && len(update.Events) == 0 {
			continue
		}
		out = append(out, model.RelationshipUpdate{PairID: update.PairID, Summary: update.Summary, TensionDelta: update.TensionDelta, Events: update.Events})
	}
	return out
}

func toWorldStateUpdates(updates []StoryNarrativeWorldStateUpdate) []model.WorldStateUpdate {
	out := make([]model.WorldStateUpdate, 0, len(updates))
	for _, update := range updates {
		if update.Key == "" {
			continue
		}
		out = append(out, model.WorldStateUpdate{Key: update.Key, Operation: update.Operation, Value: update.Value, Note: update.Note})
	}
	return out
}

func formatDraftContent(plan StoryPlanResult) string {
	var builder strings.Builder
	if plan.Summary != "" {
		builder.WriteString(plan.Summary)
		builder.WriteString("\n\n")
	}
	for _, turn := range plan.Turns {
		actor := firstText(turn.ActorName, turn.ActorID, "旁白")
		content := firstText(turn.Content, turn.Intent)
		builder.WriteString(fmt.Sprintf("【回合 %d】%s（%s）：%s", turn.TurnIndex, actor, turn.ActionType, content))
		if turn.Rationale != "" && turn.Content == "" {
			builder.WriteString("。理由：")
			builder.WriteString(turn.Rationale)
		}
		builder.WriteString("\n")
	}
	if plan.StopReason != "" {
		builder.WriteString("\n停止原因：")
		builder.WriteString(plan.StopReason)
	}
	return builder.String()
}

func firstActorID(turns []StoryTurnPlan) string {
	for _, turn := range turns {
		if turn.ActorID != "" {
			return turn.ActorID
		}
	}
	return ""
}

func relatedCharacterIDs(turns []StoryTurnPlan) []string {
	seen := map[string]struct{}{}
	ids := make([]string, 0)
	for _, turn := range turns {
		if turn.ActorID == "" {
			continue
		}
		if _, ok := seen[turn.ActorID]; ok {
			continue
		}
		seen[turn.ActorID] = struct{}{}
		ids = append(ids, turn.ActorID)
	}
	return ids
}

func firstText(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
