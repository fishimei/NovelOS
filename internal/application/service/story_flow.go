package service

import (
	"context"
	"log"

	"github.com/fishimei/NovelOS/internal/application/model"
	"github.com/fishimei/NovelOS/internal/application/port"
	"github.com/fishimei/NovelOS/internal/domain"
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
	audit     port.AuditRepository
	generator port.StoryRunGenerator
	events    port.GenerationEventStream
}

func NewStorySessionAdvancer(
	sessions port.StorySessionRepository,
	audit port.AuditRepository,
	generator port.StoryRunGenerator,
	events port.GenerationEventStream,
) *StorySessionAdvancer {
	return &StorySessionAdvancer{sessions: sessions, audit: audit, generator: generator, events: events}
}

// Advance 添加作者消息并创建新的故事运行。
func (s *StorySessionAdvancer) Advance(ctx context.Context, sessionID string, input model.AdvanceStorySessionInput) (model.StoryRun, error) {
	session, err := s.sessions.GetSessionByID(ctx, sessionID)
	if err != nil {
		return model.StoryRun{}, err
	}
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
