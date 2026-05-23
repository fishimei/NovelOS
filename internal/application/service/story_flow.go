package service

import (
	"context"
	"log"

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

// StorySessionAdvancer 负责推进故事会话。
// 管理从作者提示到故事运行的转换。
type StorySessionAdvancer struct {
	sessions  port.StorySessionRepository
	timeline  port.StoryTimelineRepository
	audit     port.AuditRepository
	generator port.StoryRunGenerator
	events    port.GenerationEventStream
	clock     port.Clock
	ids       port.IDGenerator
}

func NewStorySessionAdvancer(
	sessions port.StorySessionRepository,
	timeline port.StoryTimelineRepository,
	audit port.AuditRepository,
	generator port.StoryRunGenerator,
	events port.GenerationEventStream,
	clock port.Clock,
	ids port.IDGenerator,
) *StorySessionAdvancer {
	return &StorySessionAdvancer{sessions: sessions, timeline: timeline, audit: audit, generator: generator, events: events, clock: clock, ids: ids}
}

// Advance 添加作者消息并创建新的故事运行。
func (s *StorySessionAdvancer) Advance(ctx context.Context, sessionID string, input model.AdvanceStorySessionInput) (model.StoryRun, error) {
	session, err := s.sessions.GetSessionByID(ctx, sessionID)
	if err != nil {
		return model.StoryRun{}, err
	}
	branch, err := s.ensureStoryBranch(ctx, session, input)
	if err != nil {
		return model.StoryRun{}, err
	}
	if input.BaseTickID == "" {
		input.BaseTickID = branch.HeadTickID
	}
	input.BranchID = branch.ID
	if _, err := s.sessions.AppendMessage(ctx, sessionID, "user", input.AuthorMessage); err != nil {
		return model.StoryRun{}, err
	}
	session.LastAuthorMessage = input.AuthorMessage
	session.Status = "advancing"
	if _, err := s.sessions.UpdateSession(ctx, session); err != nil {
		return model.StoryRun{}, err
	}
	run, err := s.sessions.CreateRun(ctx, sessionID, input)
	if err != nil {
		return model.StoryRun{}, err
	}
	if s.generator != nil {
		go s.generate(context.Background(), run.RunID)
	}
	return run, nil
}

func (s *StorySessionAdvancer) ensureStoryBranch(ctx context.Context, session model.StorySession, input model.AdvanceStorySessionInput) (model.StoryBranch, error) {
	if s.timeline == nil {
		return model.StoryBranch{}, nil
	}
	if input.BranchID != "" {
		branch, err := s.timeline.GetBranchByID(ctx, input.BranchID)
		if err != nil {
			return model.StoryBranch{}, err
		}
		if branch.SessionID != session.ID {
			return model.StoryBranch{}, pkgerr.Validation("branch does not belong to story session")
		}
		if input.BaseTickID != "" && input.BaseTickID != branch.HeadTickID {
			if _, err := s.timeline.GetTickByID(ctx, input.BaseTickID); err != nil {
				return model.StoryBranch{}, err
			}
		}
		return branch, nil
	}
	branches, err := s.timeline.ListBranchesBySessionID(ctx, session.ID)
	if err != nil {
		return model.StoryBranch{}, err
	}
	if len(branches) > 0 {
		return branches[0], nil
	}
	return s.timeline.CreateBranch(ctx, model.StoryBranch{
		ProjectID: session.ProjectID,
		SessionID: session.ID,
		Name:      "main",
		Status:    "active",
		CreatedAt: currentTime(s.clock),
		UpdatedAt: currentTime(s.clock),
	})
}

func (s *StorySessionAdvancer) generate(ctx context.Context, runID string) {
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
	result.BaseTickID = run.BaseTickID
	result, err = s.persistResultTimeline(ctx, run, result)
	if err != nil {
		s.failRun(ctx, runID, session, err)
		return
	}
	if err := s.sessions.SaveRunResult(ctx, runID, result); err != nil {
		s.failRun(ctx, runID, session, err)
		return
	}
	session.Status = domain.SessionStatusReviewing
	session.CurrentPlotVariableSummary = result.PlotVariable.CoreChoice
	if _, err := s.sessions.UpdateSession(ctx, session); err != nil {
		s.failRun(ctx, runID, session, err)
		return
	}
	s.publish(ctx, runID, domain.EventReviewRequired, map[string]any{"run_id": runID, "result_available": true})
}

func (s *StorySessionAdvancer) persistResultTimeline(ctx context.Context, run model.StoryRun, result model.StoryRunResult) (model.StoryRunResult, error) {
	if s.timeline == nil || run.BranchID == "" {
		return result, nil
	}
	parentTickID := run.BaseTickID
	sequence, err := s.nextTickSequence(ctx, run.BranchID, parentTickID)
	if err != nil {
		return model.StoryRunResult{}, err
	}
	for _, event := range result.EventTimeline {
		tick, err := s.appendStoryTick(ctx, run, parentTickID, sequence, "event", event.Summary, map[string]any{"event": event}, nil, nil)
		if err != nil {
			return model.StoryRunResult{}, err
		}
		parentTickID = tick.ID
		sequence++
	}
	for _, transcript := range result.InteractionTranscripts {
		summary := transcript.OutcomeSummary
		if summary == "" {
			summary = transcript.GroupID
		}
		tick, err := s.appendStoryTick(ctx, run, parentTickID, sequence, "interaction", summary, map[string]any{"transcript": transcript}, nil, nil)
		if err != nil {
			return model.StoryRunResult{}, err
		}
		parentTickID = tick.ID
		sequence++
	}
	versions, refs := s.storyStateRefsForPatch(run, result.MemoryPatch)
	draftPayload := map[string]any{"draft": result.Draft, "memory_patch": result.MemoryPatch}
	draftTick, err := s.appendStoryTick(ctx, run, parentTickID, sequence, "draft", result.Draft.Summary, draftPayload, refs, versions)
	if err != nil {
		return model.StoryRunResult{}, err
	}
	result.HeadTickID = draftTick.ID
	if err := s.sessions.UpdateRunTimeline(ctx, run.RunID, draftTick.ID); err != nil {
		return model.StoryRunResult{}, err
	}
	return result, s.timeline.UpdateBranchHead(ctx, run.BranchID, draftTick.ID)
}

func (s *StorySessionAdvancer) nextTickSequence(ctx context.Context, branchID string, parentTickID string) (int, error) {
	if parentTickID != "" {
		parent, err := s.timeline.GetTickByID(ctx, parentTickID)
		if err != nil {
			return 0, err
		}
		return parent.Sequence + 1, nil
	}
	ticks, err := s.timeline.ListTicksByBranchID(ctx, branchID)
	if err != nil {
		return 0, err
	}
	if len(ticks) == 0 {
		return 1, nil
	}
	return ticks[len(ticks)-1].Sequence + 1, nil
}

func (s *StorySessionAdvancer) appendStoryTick(ctx context.Context, run model.StoryRun, parentTickID string, sequence int, kind string, summary string, payload map[string]any, refs []model.StoryTickStateRef, versions []model.StoryStateVersion) (model.StoryTick, error) {
	tickID := generatedID(s.ids, s.clock, "tick")
	for i := range versions {
		if versions[i].SourceTickID == "" {
			versions[i].SourceTickID = tickID
		}
	}
	for i := range refs {
		if refs[i].TickID == "" {
			refs[i].TickID = tickID
		}
	}
	return s.timeline.AppendTick(ctx, model.StoryTick{
		ID:           tickID,
		ProjectID:    run.ProjectID,
		SessionID:    run.SessionID,
		BranchID:     run.BranchID,
		ParentTickID: parentTickID,
		SourceRunID:  run.RunID,
		Sequence:     sequence,
		Kind:         kind,
		Summary:      summary,
		Payload:      payload,
		CreatedAt:    currentTime(s.clock),
	}, refs, versions)
}

func (s *StorySessionAdvancer) storyStateRefsForPatch(run model.StoryRun, patch model.MemoryPatch) ([]model.StoryStateVersion, []model.StoryTickStateRef) {
	versions := []model.StoryStateVersion{}
	refs := []model.StoryTickStateRef{}
	add := func(entityType string, entityID string, snapshot map[string]any) {
		if entityID == "" {
			return
		}
		versionID := generatedID(s.ids, s.clock, "sversion")
		versions = append(versions, model.StoryStateVersion{
			ID:          versionID,
			ProjectID:   run.ProjectID,
			EntityType:  entityType,
			EntityID:    entityID,
			SourceRunID: run.RunID,
			Snapshot:    snapshot,
			CreatedAt:   currentTime(s.clock),
		})
		refs = append(refs, model.StoryTickStateRef{
			ProjectID:  run.ProjectID,
			EntityType: entityType,
			EntityID:   entityID,
			VersionID:  versionID,
		})
	}
	for _, update := range patch.CharacterMemoryUpdates {
		add("character_memory", generatedID(s.ids, s.clock, "memory_state"), map[string]any{"patch": update})
	}
	for _, update := range patch.WorldStateUpdates {
		add("world_state", update.Key, map[string]any{"patch": update})
	}
	for _, update := range patch.RelationshipUpdates {
		entityID := update.PairID
		if entityID == "" && update.Pair != nil {
			entityID = update.Pair.ID
		}
		add("relationship", entityID, map[string]any{"patch": update})
	}
	return versions, refs
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
