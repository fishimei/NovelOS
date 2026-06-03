package service

import (
	"context"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/fishimei/NovelOS/internal/application/model"
	"github.com/fishimei/NovelOS/internal/application/port"
	"github.com/fishimei/NovelOS/internal/domain"
	"github.com/fishimei/NovelOS/internal/pkgerr"
)

type StorySessionStarter struct {
	sessions port.StorySessionRepository
}

func NewStorySessionStarter(sessions port.StorySessionRepository) *StorySessionStarter {
	return &StorySessionStarter{sessions: sessions}
}

func (s *StorySessionStarter) Start(ctx context.Context, projectID string, input model.CreateStorySessionInput) (model.StorySession, error) {
	return s.sessions.CreateSession(ctx, projectID, input)
}

type StorySessionAdvancer struct {
	sessions  port.StorySessionRepository
	store     port.StoryEventStore
	audit     port.AuditRepository
	generator port.StoryRunGenerator
	memory    port.CharacterMemoryService
	events    port.GenerationEventStream
	clock     port.Clock
	ids       port.IDGenerator
}

func NewStorySessionAdvancer(
	sessions port.StorySessionRepository,
	store port.StoryEventStore,
	audit port.AuditRepository,
	generator port.StoryRunGenerator,
	memory port.CharacterMemoryService,
	events port.GenerationEventStream,
	clock port.Clock,
	ids port.IDGenerator,
) *StorySessionAdvancer {
	return &StorySessionAdvancer{sessions: sessions, store: store, audit: audit, generator: generator, memory: memory, events: events, clock: clock, ids: ids}
}

func (s *StorySessionAdvancer) Advance(ctx context.Context, sessionID string, input model.AdvanceStorySessionInput) (model.StoryRun, error) {
	session, err := s.sessions.GetSessionByID(ctx, sessionID)
	if err != nil {
		return model.StoryRun{}, err
	}
	branch, err := s.ensureStoryBranch(ctx, session, input)
	if err != nil {
		return model.StoryRun{}, err
	}
	if input.BaseEventID == "" {
		input.BaseEventID = branch.HeadEventID
	}
	input.BranchID = branch.ID
	if _, err := s.sessions.AppendMessage(ctx, sessionID, "user", input.AuthorMessage); err != nil {
		return model.StoryRun{}, err
	}
	session.LastAuthorMessage = input.AuthorMessage
	session.Status = domain.SessionStatusAdvancing
	if _, err := s.sessions.UpdateSession(ctx, session); err != nil {
		return model.StoryRun{}, err
	}
	run, err := s.sessions.CreateRun(ctx, sessionID, input)
	if err != nil {
		return model.StoryRun{}, err
	}
	s.appendAuditEvent(ctx, run.RunID, domain.EventGenerationStep, map[string]any{"step": domain.RunStatusQueued, "progress": 0})
	return run, nil
}

func (s *StorySessionAdvancer) ensureStoryBranch(ctx context.Context, session model.StorySession, input model.AdvanceStorySessionInput) (model.Branch, error) {
	if s.store == nil {
		return model.Branch{}, nil
	}
	if input.BranchID != "" {
		branch, err := s.store.GetBranch(ctx, input.BranchID)
		if err != nil {
			return model.Branch{}, err
		}
		if branch.SessionID != session.ID {
			return model.Branch{}, pkgerr.Validation("branch does not belong to story session")
		}
		if input.BaseEventID != "" {
			if _, err := s.store.GetEvent(ctx, input.BaseEventID); err != nil {
				return model.Branch{}, err
			}
		}
		return branch, nil
	}
	branches, err := s.store.ListBranchesBySession(ctx, session.ID)
	if err != nil {
		return model.Branch{}, err
	}
	if len(branches) > 0 {
		return branches[0], nil
	}
	now := currentTime(s.clock)
	return s.store.CreateBranch(ctx, model.Branch{
		ProjectID:   session.ProjectID,
		SessionID:   session.ID,
		Name:        "main",
		BaseEventID: input.BaseEventID,
		HeadEventID: input.BaseEventID,
		Status:      "active",
		CreatedAt:   now,
		UpdatedAt:   now,
	})
}

func (s *StorySessionAdvancer) Generate(ctx context.Context, runID string) {
	run, err := s.sessions.GetRunByID(ctx, runID)
	if err != nil {
		log.Printf("story run %s failed before load: %v", runID, err)
		s.appendAuditEvent(ctx, runID, domain.EventGenerationStep, map[string]any{"step": domain.RunStatusFailed, "error": err.Error()})
		s.publish(ctx, runID, domain.EventGenerationStep, map[string]any{"step": domain.RunStatusFailed, "error": err.Error()})
		return
	}
	session, err := s.sessions.GetSessionByID(ctx, run.SessionID)
	if err != nil {
		s.failRun(ctx, runID, session, err)
		return
	}
	s.appendAuditEvent(ctx, runID, domain.EventGenerationStep, map[string]any{"step": domain.RunStatusLoadingState, "progress": 10})
	s.publish(ctx, runID, domain.EventGenerationStep, map[string]any{"step": domain.RunStatusLoadingState, "progress": 10})
	if err := s.sessions.UpdateRunStatus(ctx, runID, domain.RunStatusLoadingState, domain.RunStatusLoadingState, 10); err != nil {
		s.failRun(ctx, runID, session, err)
		return
	}
	result, err := s.generator.Generate(ctx, port.StoryRunGenerationInput{Run: run, Session: session})
	if err != nil {
		s.failRun(ctx, runID, session, err)
		return
	}
	result.BranchID = run.BranchID
	result.BaseEventID = run.BaseEventID
	result.Status = domain.RunStatusCompleted
	result, err = s.persistResultEvents(ctx, run, result)
	if err != nil {
		s.failRun(ctx, runID, session, err)
		return
	}
	s.commitCharacterMemories(ctx, run, result)
	if err := s.sessions.SaveRunResult(ctx, runID, result); err != nil {
		s.failRun(ctx, runID, session, err)
		return
	}
	session.Status = domain.SessionStatusIdle
	session.CurrentPlotVariableSummary = firstNonEmpty(result.SceneSummary, result.PlotVariable.CoreChoice)
	if _, err := s.sessions.UpdateSession(ctx, session); err != nil {
		s.failRun(ctx, runID, session, err)
		return
	}
	s.appendAuditEvent(ctx, runID, domain.EventGenerationStep, map[string]any{"step": domain.RunStatusCompleted, "result_available": true})
	s.publish(ctx, runID, domain.EventGenerationStep, map[string]any{"step": domain.RunStatusCompleted, "result_available": true})
}

func (s *StorySessionAdvancer) RequestStop(ctx context.Context, runID string) (model.StoryRun, error) {
	if err := s.sessions.RequestRunStop(ctx, runID); err != nil {
		return model.StoryRun{}, err
	}
	run, err := s.sessions.GetRunByID(ctx, runID)
	if err != nil {
		return model.StoryRun{}, err
	}
	event := map[string]any{"step": "stop_requested", "run_id": runID}
	s.appendAuditEvent(ctx, runID, domain.EventGenerationStep, event)
	s.publish(ctx, runID, domain.EventGenerationStep, event)
	return run, nil
}

func (s *StorySessionAdvancer) persistResultEvents(ctx context.Context, run model.StoryRun, result model.StoryRunResult) (model.StoryRunResult, error) {
	if s.store == nil || run.BranchID == "" {
		return result, nil
	}
	parentEventID := run.BaseEventID
	baseStoryTime := run.CreatedAt
	if parentEventID != "" {
		parent, err := s.store.GetEvent(ctx, parentEventID)
		if err != nil {
			return model.StoryRunResult{}, err
		}
		baseStoryTime = parent.StoryTime
	}
	storyTime := baseStoryTime
	written := make([]model.StoryEvent, 0, len(result.EventPlan)*2+1)
	branch := model.Branch{ID: run.BranchID, ProjectID: run.ProjectID, SessionID: run.SessionID}
	timedActions := make([]TimedAction, 0, len(result.EventPlan))
	scheduledActions := make([]model.OngoingAction, 0, len(result.EventPlan))
	for _, planned := range orderedStoryEventPlans(result.EventPlan) {
		eventTime := storyEventPlanTime(baseStoryTime, planned)
		action := ongoingActionFromPlan(planned, eventTime)
		eventInput := StoryEventFromAction(branch, action, parentEventID)
		eventInput.Payload["event_plan"] = planned
		eventInput.Payload["source_run_id"] = run.RunID
		eventInput.CreatedAt = currentTime(s.clock)
		event, err := s.store.AppendEvent(ctx, eventInput)
		if err != nil {
			return model.StoryRunResult{}, err
		}
		written = append(written, event)
		scheduledActions = append(scheduledActions, action)
		timedActions = append(timedActions, timedActionFrom(action))
		parentEventID = event.ID
		storyTime = event.StoryTime
	}
	sceneTime := sceneTimeFromScheduledActions(baseStoryTime, timedActions, storyTime)
	if sceneTime.IsZero() {
		sceneTime = run.CreatedAt
	}
	for _, action := range actionsCompletedAt(scheduledActions, sceneTime) {
		eventInput := StoryEventFromActionCompletion(branch, action, parentEventID)
		eventInput.Payload["source_run_id"] = run.RunID
		eventInput.CreatedAt = currentTime(s.clock)
		event, err := s.store.AppendEvent(ctx, eventInput)
		if err != nil {
			return model.StoryRunResult{}, err
		}
		written = append(written, event)
		parentEventID = event.ID
	}
	if !storyRunResultHasRenderedScene(result) {
		result.Events = written
		result.HeadEventID = parentEventID
		if parentEventID == "" {
			return result, nil
		}
		if err := s.store.UpdateBranchHead(ctx, run.BranchID, parentEventID); err != nil {
			return model.StoryRunResult{}, err
		}
		if err := s.sessions.UpdateRunHead(ctx, run.RunID, parentEventID); err != nil {
			return model.StoryRunResult{}, err
		}
		return result, nil
	}
	sceneEvent, err := s.store.AppendEvent(ctx, model.StoryEvent{
		ProjectID:     run.ProjectID,
		SessionID:     run.SessionID,
		BranchID:      run.BranchID,
		ParentEventID: parentEventID,
		StoryTime:     sceneTime,
		Kind:          model.EventKindSceneResolved,
		ActorIDs:      actorIDsForResult(result),
		LocationKey:   primaryLocationKey(result),
		ResourceKeys:  resourceKeysForResult(result),
		Summary:       firstNonEmpty(result.SceneSummary, result.PlotVariable.CoreChoice),
		Payload: map[string]any{
			"draft":                   result.Draft,
			"summary":                 result.SceneSummary,
			"turns":                   result.Turns,
			"memory_patch":            result.MemoryPatch,
			"memory_scope":            domain.MemoryPatchStatusRunLocal,
			"interaction_analysis":    result.InteractionAnalysis,
			"interaction_transcripts": result.InteractionTranscripts,
			"plot_variable":           result.PlotVariable,
			"source_run_id":           run.RunID,
		},
		StateDelta: model.EventStateDelta{MemoryPatch: result.MemoryPatch},
		CreatedAt:  currentTime(s.clock),
	})
	if err != nil {
		return model.StoryRunResult{}, err
	}
	written = append(written, sceneEvent)
	result.Events = written
	result.HeadEventID = sceneEvent.ID
	if err := s.store.UpdateBranchHead(ctx, run.BranchID, sceneEvent.ID); err != nil {
		return model.StoryRunResult{}, err
	}
	if err := s.sessions.UpdateRunHead(ctx, run.RunID, sceneEvent.ID); err != nil {
		return model.StoryRunResult{}, err
	}
	return result, nil
}

func storyRunResultHasRenderedScene(result model.StoryRunResult) bool {
	return strings.TrimSpace(result.Draft.Content) != "" ||
		len(result.Turns) > 0 ||
		len(result.InteractionAnalysis.LocationGroups) > 0 ||
		len(result.InteractionAnalysis.InteractionGroups) > 0 ||
		len(result.InteractionTranscripts) > 0 ||
		storyRunMemoryPatchHasUpdates(result.MemoryPatch)
}

func storyRunMemoryPatchHasUpdates(patch model.MemoryPatch) bool {
	return len(patch.CharacterMemoryUpdates) > 0 || len(patch.RelationshipUpdates) > 0 || len(patch.WorldStateUpdates) > 0
}

func (s *StorySessionAdvancer) commitCharacterMemories(ctx context.Context, run model.StoryRun, result model.StoryRunResult) {
	if s.memory == nil || len(result.MemoryPatch.CharacterMemoryUpdates) == 0 {
		return
	}
	memories := make([]model.Memory, 0, len(result.MemoryPatch.CharacterMemoryUpdates))
	for _, update := range result.MemoryPatch.CharacterMemoryUpdates {
		if update.CharacterID == "" || update.Content == "" {
			continue
		}
		memories = append(memories, model.Memory{
			ID:            generatedID(s.ids, s.clock, "memory"),
			CharacterID:   update.CharacterID,
			Content:       update.Content,
			SourceRunID:   run.RunID,
			BranchID:      run.BranchID,
			SourceEventID: result.HeadEventID,
			Importance:    update.Importance,
			Note:          domain.MemoryScopeExternalCommitted + ":" + domain.MemoryCommitTriggerRunCompletion,
			Status:        "active",
			CreatedAt:     currentTime(s.clock),
		})
	}
	if len(memories) == 0 {
		return
	}
	err := s.memory.Commit(ctx, port.CharacterMemoryCommitInput{
		ProjectID: run.ProjectID,
		RunID:     run.RunID,
		Memories:  memories,
	})
	if err != nil {
		log.Printf("story run %s external memory commit failed: %v", run.RunID, err)
		s.appendAuditEvent(ctx, run.RunID, domain.EventGenerationStep, map[string]any{
			"step":  "external_memory_flush_failed",
			"error": err.Error(),
		})
		return
	}
	s.appendAuditEvent(ctx, run.RunID, "external_memory_committed", map[string]any{
		"memory_count": len(memories),
		"scope":        domain.MemoryScopeExternalCommitted,
		"trigger":      domain.MemoryCommitTriggerRunCompletion,
	})
}

func orderedStoryEventPlans(events []model.StoryEventPlan) []model.StoryEventPlan {
	out := append([]model.StoryEventPlan(nil), events...)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].TimeIndex == out[j].TimeIndex {
			return out[i].ID < out[j].ID
		}
		return out[i].TimeIndex < out[j].TimeIndex
	})
	return out
}

func storyEventPlanTime(baseStoryTime time.Time, event model.StoryEventPlan) time.Time {
	return baseStoryTime.Add(time.Duration(maxInt(event.TimeIndex, 0)) * time.Hour)
}

func ongoingActionFromPlan(event model.StoryEventPlan, eventTime time.Time) model.OngoingAction {
	duration := time.Duration(maxInt(event.DurationHours, 1)) * time.Hour
	return model.OngoingAction{
		CharacterID:       event.CharacterID,
		ActionType:        firstNonEmpty(event.ActionType, "action"),
		Description:       event.Summary,
		TargetLocationKey: event.LocationKey,
		ParticipantIDs:    uniqueStrings(event.TargetActorIDs),
		StartAt:           eventTime,
		ArriveAt:          eventTime,
		EffectAt:          eventTime,
		EndsAt:            eventTime.Add(duration),
		ResourceKeys:      resourceKeysForPlan(event),
		Status:            "ongoing",
		Rationale:         event.Intent,
	}
}

func sceneTimeFromScheduledActions(start time.Time, actions []TimedAction, fallback time.Time) time.Time {
	if len(actions) == 0 {
		return fallback
	}
	result := ResolveEventClock(start, actions)
	next := result.NextCompletion
	if len(result.Collisions) > 0 && (next.IsZero() || result.Collisions[0].Before(next)) {
		next = result.Collisions[0]
	}
	if !next.IsZero() {
		return next
	}
	return result.Clock
}

func actionsCompletedAt(actions []model.OngoingAction, at time.Time) []model.OngoingAction {
	if at.IsZero() {
		return nil
	}
	out := make([]model.OngoingAction, 0)
	for _, action := range actions {
		if action.CharacterID == "" || action.EndsAt.IsZero() {
			continue
		}
		if action.EndsAt.Equal(at) {
			out = append(out, action)
		}
	}
	return out
}

func actorIDsForPlan(event model.StoryEventPlan) []string {
	ids := make([]string, 0, 1+len(event.TargetActorIDs))
	if event.CharacterID != "" {
		ids = append(ids, event.CharacterID)
	}
	ids = append(ids, event.TargetActorIDs...)
	return uniqueStrings(ids)
}

func actorIDsForResult(result model.StoryRunResult) []string {
	ids := make([]string, 0)
	for _, event := range result.EventPlan {
		ids = append(ids, actorIDsForPlan(event)...)
	}
	for _, transcript := range result.InteractionTranscripts {
		ids = append(ids, transcript.CharacterIDs...)
	}
	for _, turn := range result.Turns {
		if turn.ActorID != "" {
			ids = append(ids, turn.ActorID)
		}
		ids = append(ids, turn.TargetActorIDs...)
	}
	return uniqueStrings(ids)
}

func primaryLocationKey(result model.StoryRunResult) string {
	for _, event := range result.EventPlan {
		if event.LocationKey != "" {
			return event.LocationKey
		}
	}
	for _, transcript := range result.InteractionTranscripts {
		if transcript.LocationKey != "" {
			return transcript.LocationKey
		}
	}
	for _, turn := range result.Turns {
		if turn.LocationKey != "" {
			return turn.LocationKey
		}
	}
	return ""
}

func resourceKeysForPlan(event model.StoryEventPlan) []string {
	keys := make([]string, 0, 2+len(event.TargetActorIDs))
	if event.CharacterID != "" {
		keys = append(keys, "character:"+event.CharacterID)
	}
	for _, actorID := range event.TargetActorIDs {
		if actorID != "" {
			keys = append(keys, "character:"+actorID)
		}
	}
	if event.LocationKey != "" {
		keys = append(keys, "location:"+event.LocationKey)
	}
	return uniqueStrings(keys)
}

func resourceKeysForResult(result model.StoryRunResult) []string {
	keys := make([]string, 0)
	for _, event := range result.EventPlan {
		keys = append(keys, resourceKeysForPlan(event)...)
	}
	for _, turn := range result.Turns {
		if turn.ActorID != "" {
			keys = append(keys, "character:"+turn.ActorID)
		}
		for _, actorID := range turn.TargetActorIDs {
			if actorID != "" {
				keys = append(keys, "character:"+actorID)
			}
		}
		if turn.LocationKey != "" {
			keys = append(keys, "location:"+turn.LocationKey)
		}
	}
	return uniqueStrings(keys)
}

func uniqueStrings(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (s *StorySessionAdvancer) failRun(ctx context.Context, runID string, session model.StorySession, err error) {
	_ = s.sessions.UpdateRunStatus(ctx, runID, domain.RunStatusFailed, domain.RunStatusFailed, 100, err.Error())
	if session.ID != "" {
		session.Status = domain.SessionStatusFailed
		_, _ = s.sessions.UpdateSession(ctx, session)
	}
	log.Printf("story run %s failed: %v", runID, err)
	s.appendAuditEvent(ctx, runID, domain.EventGenerationStep, map[string]any{"step": domain.RunStatusFailed, "error": err.Error()})
	s.publish(ctx, runID, domain.EventGenerationStep, map[string]any{"step": domain.RunStatusFailed, "error": err.Error()})
}

func (s *StorySessionAdvancer) publish(ctx context.Context, runID string, name string, data any) {
	if s.events == nil {
		return
	}
	_ = s.events.Publish(ctx, runID, port.GenerationEvent{Name: name, Data: data})
}

func (s *StorySessionAdvancer) appendAuditEvent(ctx context.Context, runID string, name string, data map[string]any) {
	if s.audit == nil {
		return
	}
	if _, err := s.audit.AppendRunEvent(ctx, model.RunEvent{
		RunKind:   "story",
		RunID:     runID,
		EventName: name,
		Payload:   data,
	}); err != nil {
		log.Printf("append story run event %s failed: %v", runID, err)
	}
}
