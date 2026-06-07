package service

import (
	"context"
	"errors"
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
	events    port.GenerationEventStream
	tx        port.TxManager
	clock     port.Clock
	ids       port.IDGenerator
}

func NewStorySessionAdvancer(
	sessions port.StorySessionRepository,
	store port.StoryEventStore,
	audit port.AuditRepository,
	generator port.StoryRunGenerator,
	events port.GenerationEventStream,
	tx port.TxManager,
	clock port.Clock,
	ids port.IDGenerator,
) *StorySessionAdvancer {
	return &StorySessionAdvancer{sessions: sessions, store: store, audit: audit, generator: generator, events: events, tx: tx, clock: clock, ids: ids}
}

func normalizeStoryAdvanceMode(input model.AdvanceStorySessionInput) string {
	mode := strings.TrimSpace(input.AdvanceMode)
	if mode != "" {
		return mode
	}
	return "auto"
}

func (s *StorySessionAdvancer) Advance(ctx context.Context, sessionID string, input model.AdvanceStorySessionInput) (model.StoryRun, error) {
	return s.createRun(ctx, sessionID, input)
}

func (s *StorySessionAdvancer) CreateAutoRun(ctx context.Context, sessionID string, input model.AdvanceStorySessionInput) (model.StoryRun, error) {
	input.AdvanceMode = "auto"
	return s.createRun(ctx, sessionID, input)
}

func (s *StorySessionAdvancer) createRun(ctx context.Context, sessionID string, input model.AdvanceStorySessionInput) (model.StoryRun, error) {
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
	advanceMode := normalizeStoryAdvanceMode(input)
	var run model.StoryRun
	createRun := func(txCtx context.Context) error {
		active, err := s.sessions.HasActiveRunByBranch(txCtx, branch.ID)
		if err != nil {
			return err
		}
		if active {
			return pkgerr.Conflict(pkgerr.CodeConflict, "story branch already has an active run")
		}
		session.Status = domain.SessionStatusAdvancing
		if _, err := s.sessions.UpdateSession(txCtx, session); err != nil {
			return err
		}
		run, err = s.sessions.CreateRun(txCtx, sessionID, input)
		if err != nil {
			return err
		}
		s.appendAuditEvent(txCtx, run.RunID, domain.EventGenerationStep, map[string]any{"step": domain.RunStatusQueued, "progress": 0, "advance_mode": advanceMode, "branch_id": input.BranchID, "base_event_id": input.BaseEventID})
		return nil
	}
	if s.tx != nil {
		if err := s.tx.WithinTransaction(ctx, createRun); err != nil {
			return model.StoryRun{}, err
		}
		return run, nil
	}
	if err := createRun(ctx); err != nil {
		return model.StoryRun{}, err
	}
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
		if branch.SessionID != session.ID || branch.ProjectID != session.ProjectID {
			return model.Branch{}, pkgerr.Validation("branch does not belong to story session")
		}
		if input.BaseEventID != "" {
			if err := s.validateBranchBaseEvent(ctx, branch, input.BaseEventID); err != nil {
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
		branch := branches[0]
		if input.BaseEventID != "" {
			if err := s.validateBranchBaseEvent(ctx, branch, input.BaseEventID); err != nil {
				return model.Branch{}, err
			}
		}
		return branch, nil
	}
	baseEventID := input.BaseEventID
	if baseEventID == "" {
		genesis, err := s.store.GetProjectGenesis(ctx, session.ProjectID)
		if err != nil && !isNotFound(err) {
			return model.Branch{}, err
		}
		if err == nil {
			baseEventID = genesis.ID
		}
	}
	if baseEventID != "" {
		if err := s.validateNewBranchBaseEvent(ctx, session, baseEventID); err != nil {
			return model.Branch{}, err
		}
	}
	now := currentTime(s.clock)
	return s.store.CreateBranch(ctx, model.Branch{
		ProjectID:   session.ProjectID,
		SessionID:   session.ID,
		Name:        "main",
		BaseEventID: baseEventID,
		HeadEventID: baseEventID,
		Status:      "active",
		CreatedAt:   now,
		UpdatedAt:   now,
	})
}

func (s *StorySessionAdvancer) validateNewBranchBaseEvent(ctx context.Context, session model.StorySession, eventID string) error {
	event, err := s.store.GetEvent(ctx, eventID)
	if err != nil {
		if isNotFound(err) {
			return pkgerr.Validation("base event does not exist")
		}
		return err
	}
	if event.ProjectID != session.ProjectID {
		return pkgerr.Validation("base event does not belong to story session project")
	}
	if event.Kind == model.EventKindGenesis {
		if err := s.validateProjectGenesis(ctx, session.ProjectID, event.ID); err != nil {
			return err
		}
		return nil
	}
	if event.SessionID != session.ID {
		return pkgerr.Validation("base event does not belong to story session")
	}
	return nil
}

func (s *StorySessionAdvancer) validateBranchBaseEvent(ctx context.Context, branch model.Branch, eventID string) error {
	event, err := s.store.GetEvent(ctx, eventID)
	if err != nil {
		if isNotFound(err) {
			return pkgerr.Validation("base event does not exist")
		}
		return err
	}
	if event.ProjectID != branch.ProjectID {
		return pkgerr.Validation("base event does not belong to branch project")
	}
	if event.Kind == model.EventKindGenesis {
		if err := s.validateProjectGenesis(ctx, branch.ProjectID, event.ID); err != nil {
			return err
		}
	} else if event.SessionID != branch.SessionID {
		return pkgerr.Validation("base event does not belong to branch session")
	}
	if branch.HeadEventID == "" {
		if event.Kind == model.EventKindGenesis && event.ID == branch.BaseEventID {
			return nil
		}
		return pkgerr.Validation("base event is not reachable from empty branch")
	}
	if eventID == branch.HeadEventID {
		return nil
	}
	if ok, err := s.eventIsAncestorOf(ctx, eventID, branch.HeadEventID); err != nil {
		return err
	} else if !ok {
		return pkgerr.Validation("base event is not reachable from branch head")
	}
	return nil
}

func (s *StorySessionAdvancer) validateProjectGenesis(ctx context.Context, projectID, eventID string) error {
	genesis, err := s.store.GetProjectGenesis(ctx, projectID)
	if err != nil {
		if isNotFound(err) {
			return pkgerr.Validation("project genesis does not exist")
		}
		return err
	}
	if genesis.ID != eventID {
		return pkgerr.Validation("base genesis is not the project genesis")
	}
	return nil
}

func (s *StorySessionAdvancer) eventIsAncestorOf(ctx context.Context, ancestorID, headID string) (bool, error) {
	for headID != "" {
		if headID == ancestorID {
			return true, nil
		}
		event, err := s.store.GetEvent(ctx, headID)
		if err != nil {
			return false, err
		}
		headID = event.ParentEventID
	}
	return false, nil
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
	input, err := s.prepareStoryGenerationInput(ctx, run, session)
	if err != nil {
		s.failRun(ctx, runID, session, err)
		return
	}
	heartbeatCtx, stopHeartbeat := context.WithCancel(ctx)
	defer stopHeartbeat()
	go s.heartbeatRun(heartbeatCtx, runID)
	result, err := s.generator.Generate(ctx, input)
	if err != nil {
		if errors.Is(err, port.ErrRunStopRequested) {
			s.cancelRun(ctx, runID, session, "stop_requested")
			return
		}
		s.failRun(ctx, runID, session, err)
		return
	}
	latestRun, err := s.sessions.GetRunByID(ctx, runID)
	if err != nil {
		s.failRun(ctx, runID, session, err)
		return
	}
	if latestRun.StopRequested && !storyRunResultHasRenderedScene(result) {
		s.cancelRun(ctx, runID, session, "stop_requested")
		return
	}
	result.BranchID = run.BranchID
	result.BaseEventID = run.BaseEventID
	result.Status = domain.RunStatusCompleted
	result, err = s.persistCompletedRun(ctx, run, session, result)
	if err != nil {
		s.failRun(ctx, runID, session, err)
		return
	}
	s.appendAuditEvent(ctx, runID, domain.EventGenerationStep, map[string]any{"step": domain.RunStatusCompleted, "result_available": true})
	s.publish(ctx, runID, domain.EventGenerationStep, map[string]any{"step": domain.RunStatusCompleted, "result_available": true})
}

func (s *StorySessionAdvancer) heartbeatRun(ctx context.Context, runID string) {
	if s.sessions == nil {
		return
	}
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.sessions.UpdateRunHeartbeat(ctx, runID); err != nil {
				log.Printf("story run %s heartbeat failed: %v", runID, err)
			}
		}
	}
}

func (s *StorySessionAdvancer) prepareStoryGenerationInput(ctx context.Context, run model.StoryRun, session model.StorySession) (port.StoryRunGenerationInput, error) {
	input := port.StoryRunGenerationInput{Run: run, Session: session}
	if s.store == nil || run.BranchID == "" || run.BaseEventID == "" {
		return input, nil
	}
	world, err := s.store.ResolveStateAt(ctx, run.BaseEventID)
	if err != nil {
		return input, err
	}
	input.World = world
	inFlight, err := s.store.InFlightActionsAt(ctx, run.BranchID, world.StoryTime)
	if err != nil {
		return input, err
	}
	input.InFlightActions = inFlight
	if len(inFlight) == 0 {
		return input, nil
	}
	clock := ResolveEventClock(world.StoryTime, timedActionsFromOngoing(inFlight))
	if collision := firstCollisionBeforeOrAtCompletion(clock); collision != nil {
		input.CollisionAt = collision.At
		input.SupersededActions = uniqueActions(collision.Actions)
		input.WakeCharacterIDs = actionCharacterIDsForService(input.SupersededActions)
		return input, nil
	}
	if clock.NextCompletion.IsZero() {
		return input, nil
	}
	completed := actionsCompletedAt(inFlight, clock.NextCompletion)
	input.CompletedActions = completed
	input.WakeCharacterIDs = actionCharacterIDsForService(completed)
	return input, nil
}

func (s *StorySessionAdvancer) RequestStop(ctx context.Context, runID string) (model.StoryRun, error) {
	if err := s.sessions.RequestRunStop(ctx, runID); err != nil {
		return model.StoryRun{}, err
	}
	run, err := s.sessions.GetRunByID(ctx, runID)
	if err != nil {
		return model.StoryRun{}, err
	}
	step := "stop_requested"
	if run.Status == domain.RunStatusCancelled {
		step = domain.RunStatusCancelled
	}
	event := map[string]any{"step": step, "run_id": runID}
	s.appendAuditEvent(ctx, runID, domain.EventGenerationStep, event)
	s.publish(ctx, runID, domain.EventGenerationStep, event)
	return run, nil
}

func (s *StorySessionAdvancer) persistCompletedRun(ctx context.Context, run model.StoryRun, session model.StorySession, result model.StoryRunResult) (model.StoryRunResult, error) {
	persist := func(txCtx context.Context) error {
		var err error
		result, err = s.persistResultEvents(txCtx, run, result)
		if err != nil {
			return err
		}
		if err := s.sessions.SaveRunResult(txCtx, run.RunID, result); err != nil {
			return err
		}
		session.Status = domain.SessionStatusIdle
		session.CurrentPlotVariableSummary = firstNonEmpty(result.SceneSummary, result.PlotVariable.CoreChoice)
		_, err = s.sessions.UpdateSession(txCtx, session)
		return err
	}
	if s.tx != nil {
		if err := s.tx.WithinTransaction(ctx, persist); err != nil {
			return model.StoryRunResult{}, err
		}
		return result, nil
	}
	if err := persist(ctx); err != nil {
		return model.StoryRunResult{}, err
	}
	return result, nil
}

func (s *StorySessionAdvancer) persistResultEvents(ctx context.Context, run model.StoryRun, result model.StoryRunResult) (model.StoryRunResult, error) {
	if s.store == nil || run.BranchID == "" {
		return result, nil
	}
	if err := validateRunResultActionTerminals(result); err != nil {
		return model.StoryRunResult{}, err
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
	written := make([]model.StoryEvent, 0, len(result.EventPlan)*2+len(result.CompletedActions)+len(result.SupersededActions)+1)
	branch := model.Branch{ID: run.BranchID, ProjectID: run.ProjectID, SessionID: run.SessionID}
	for _, action := range orderedOngoingActions(result.SupersededActions) {
		at := baseStoryTime
		if result.CollisionAt != nil && !result.CollisionAt.IsZero() {
			at = *result.CollisionAt
		}
		eventInput := StoryEventFromActionSuperseded(branch, action, parentEventID, at, "collision_rearranged")
		eventInput.Payload["source_run_id"] = run.RunID
		eventInput.CreatedAt = currentTime(s.clock)
		event, err := s.store.AppendEvent(ctx, eventInput)
		if err != nil {
			return model.StoryRunResult{}, err
		}
		written = append(written, event)
		parentEventID = event.ID
		storyTime = event.StoryTime
	}
	for _, action := range orderedOngoingActions(result.CompletedActions) {
		eventInput := StoryEventFromActionCompletion(branch, action, parentEventID)
		eventInput.Payload["source_run_id"] = run.RunID
		eventInput.CreatedAt = currentTime(s.clock)
		event, err := s.store.AppendEvent(ctx, eventInput)
		if err != nil {
			return model.StoryRunResult{}, err
		}
		written = append(written, event)
		parentEventID = event.ID
		storyTime = event.StoryTime
	}
	scheduleBaseTime := storyTime
	plannedEvents := orderedStoryEventPlans(result.EventPlan, scheduleBaseTime)
	if err := validateNoOverlappingCharacterPlans(plannedEvents, scheduleBaseTime); err != nil {
		return model.StoryRunResult{}, err
	}
	timedActions := make([]TimedAction, 0, len(plannedEvents))
	scheduledActions := make([]model.OngoingAction, 0, len(plannedEvents))
	for _, planned := range plannedEvents {
		eventTime := storyEventPlanTime(scheduleBaseTime, planned)
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
	sceneTime := storyTime
	if len(result.CompletedActions) == 0 && len(result.SupersededActions) == 0 {
		sceneTime = sceneTimeFromScheduledActions(baseStoryTime, timedActions, storyTime)
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
			storyTime = event.StoryTime
		}
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
			"completed_actions":       result.CompletedActions,
			"superseded_actions":      result.SupersededActions,
			"collision_at":            result.CollisionAt,
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

func validateNoOverlappingCharacterPlans(events []model.StoryEventPlan, baseStoryTime time.Time) error {
	byCharacter := map[string][]model.OngoingAction{}
	for _, event := range events {
		if event.CharacterID == "" {
			continue
		}
		action := ongoingActionFromPlan(event, storyEventPlanTime(baseStoryTime, event))
		for _, existing := range byCharacter[action.CharacterID] {
			if actionIntervalsOverlap(existing, action) {
				return pkgerr.Validation("story event plan schedules overlapping actions for the same character")
			}
		}
		byCharacter[action.CharacterID] = append(byCharacter[action.CharacterID], action)
	}
	return nil
}

func validateRunResultActionTerminals(result model.StoryRunResult) error {
	completed := map[string]struct{}{}
	for _, action := range result.CompletedActions {
		key := terminalActionKey(action)
		if key != "" {
			completed[key] = struct{}{}
		}
	}
	for _, action := range result.SupersededActions {
		key := terminalActionKey(action)
		if key == "" {
			continue
		}
		if _, ok := completed[key]; ok {
			return pkgerr.Validation("story run result completes and supersedes the same action")
		}
	}
	return nil
}

func terminalActionKey(action model.OngoingAction) string {
	if strings.TrimSpace(action.ID) != "" {
		return "id:" + strings.TrimSpace(action.ID)
	}
	if strings.TrimSpace(action.CharacterID) != "" {
		return "legacy-character:" + strings.TrimSpace(action.CharacterID)
	}
	return ""
}

func actionIntervalsOverlap(a, b model.OngoingAction) bool {
	if a.StartAt.IsZero() || a.EndsAt.IsZero() || b.StartAt.IsZero() || b.EndsAt.IsZero() {
		return false
	}
	return a.StartAt.Before(b.EndsAt) && b.StartAt.Before(a.EndsAt)
}

func orderedOngoingActions(actions []model.OngoingAction) []model.OngoingAction {
	out := append([]model.OngoingAction(nil), actions...)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].EndsAt.Equal(out[j].EndsAt) {
			return out[i].CharacterID < out[j].CharacterID
		}
		return out[i].EndsAt.Before(out[j].EndsAt)
	})
	return out
}

func timedActionsFromOngoing(actions []model.OngoingAction) []TimedAction {
	out := make([]TimedAction, 0, len(actions))
	for _, action := range actions {
		out = append(out, timedActionFrom(action))
	}
	return out
}

func firstCollisionBeforeOrAtCompletion(clock EventEngineResult) *ActionCollision {
	for _, collision := range clock.ActionCollisions {
		if clock.NextCompletion.IsZero() || collision.At.Before(clock.NextCompletion) || collision.At.Equal(clock.NextCompletion) {
			copyCollision := collision
			return &copyCollision
		}
	}
	return nil
}

func uniqueActions(actions []model.OngoingAction) []model.OngoingAction {
	seen := map[string]struct{}{}
	out := make([]model.OngoingAction, 0, len(actions))
	for _, action := range actions {
		key := firstNonEmpty(action.ID, action.CharacterID)
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, action)
	}
	return out
}

func actionCharacterIDsForService(actions []model.OngoingAction) []string {
	ids := make([]string, 0, len(actions)*2)
	for _, action := range actions {
		ids = appendUnique(ids, action.CharacterID)
		for _, participantID := range action.ParticipantIDs {
			ids = appendUnique(ids, participantID)
		}
	}
	return ids
}

func appendUnique(values []string, value string) []string {
	if value == "" {
		return values
	}
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
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

func orderedStoryEventPlans(events []model.StoryEventPlan, baseStoryTime time.Time) []model.StoryEventPlan {
	out := append([]model.StoryEventPlan(nil), events...)
	sort.SliceStable(out, func(i, j int) bool {
		left := storyEventPlanTime(baseStoryTime, out[i])
		right := storyEventPlanTime(baseStoryTime, out[j])
		if left.Equal(right) {
			return out[i].ID < out[j].ID
		}
		return left.Before(right)
	})
	return out
}

func storyEventPlanTime(baseStoryTime time.Time, event model.StoryEventPlan) time.Time {
	if event.StartAt != nil && !event.StartAt.IsZero() {
		return *event.StartAt
	}
	return baseStoryTime.Add(time.Duration(maxInt(event.TimeIndex, 0)) * time.Hour)
}

func ongoingActionFromPlan(event model.StoryEventPlan, eventTime time.Time) model.OngoingAction {
	duration := time.Duration(maxInt(event.DurationHours, 1)) * time.Hour
	startAt := eventTime
	arriveAt := explicitPlanTime(event.ArriveAt, startAt)
	effectAt := explicitPlanTime(event.EffectAt, arriveAt)
	endsAt := explicitPlanTime(event.EndsAt, startAt.Add(duration))
	if endsAt.Before(startAt) {
		endsAt = startAt.Add(duration)
	}
	return model.OngoingAction{
		ID:                event.ID,
		CharacterID:       event.CharacterID,
		ActionType:        firstNonEmpty(event.ActionType, "action"),
		Description:       event.Summary,
		TargetLocationKey: event.LocationKey,
		ParticipantIDs:    uniqueStrings(event.TargetActorIDs),
		StartAt:           startAt,
		ArriveAt:          arriveAt,
		EffectAt:          effectAt,
		EndsAt:            endsAt,
		ResourceKeys:      resourceKeysForPlan(event),
		Status:            "ongoing",
		Rationale:         event.Intent,
	}
}

func explicitPlanTime(value *time.Time, fallback time.Time) time.Time {
	if value == nil || value.IsZero() {
		return fallback
	}
	return *value
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
	keys := make([]string, 0, len(event.ResourceKeys)+2+len(event.TargetActorIDs))
	keys = append(keys, event.ResourceKeys...)
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
	if isRunLeaseLost(err) {
		log.Printf("story run %s lease lost; not marking failed", runID)
		return
	}
	if updateErr := s.sessions.UpdateRunStatus(ctx, runID, domain.RunStatusFailed, domain.RunStatusFailed, 100, err.Error()); isRunLeaseLost(updateErr) {
		log.Printf("story run %s lease lost; not marking failed", runID)
		return
	}
	if session.ID != "" {
		session.Status = domain.SessionStatusFailed
		_, _ = s.sessions.UpdateSession(ctx, session)
	}
	log.Printf("story run %s failed: %v", runID, err)
	s.appendAuditEvent(ctx, runID, domain.EventGenerationStep, map[string]any{"step": domain.RunStatusFailed, "error": err.Error()})
	s.publish(ctx, runID, domain.EventGenerationStep, map[string]any{"step": domain.RunStatusFailed, "error": err.Error()})
}

func (s *StorySessionAdvancer) cancelRun(ctx context.Context, runID string, session model.StorySession, reason string) {
	if err := s.sessions.UpdateRunStatus(ctx, runID, domain.RunStatusCancelled, domain.RunStatusCancelled, 100, reason); isRunLeaseLost(err) {
		log.Printf("story run %s lease lost; not marking cancelled", runID)
		return
	}
	if session.ID != "" {
		session.Status = domain.SessionStatusIdle
		_, _ = s.sessions.UpdateSession(ctx, session)
	}
	s.appendAuditEvent(ctx, runID, domain.EventGenerationStep, map[string]any{"step": domain.RunStatusCancelled, "reason": reason})
	s.publish(ctx, runID, domain.EventGenerationStep, map[string]any{"step": domain.RunStatusCancelled, "reason": reason})
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
