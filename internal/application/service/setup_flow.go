package service

import (
	"context"
	"log"
	"time"

	"github.com/fishimei/NovelOS/internal/application/model"
	"github.com/fishimei/NovelOS/internal/application/port"
	"github.com/fishimei/NovelOS/internal/domain"
)

// SetupSessionStarter 负责创建新的设置会话。
// 管理设置流程的初始化状态。
type SetupSessionStarter struct {
	sessions port.SetupSessionRepository
}

func NewSetupSessionStarter(sessions port.SetupSessionRepository) *SetupSessionStarter {
	return &SetupSessionStarter{sessions: sessions}
}

// Start 创建设置会话并添加种子想法作为初始消息。
func (s *SetupSessionStarter) Start(ctx context.Context, projectID string, input model.CreateSetupSessionInput) (model.SetupSession, error) {
	session, err := s.sessions.CreateSession(ctx, projectID, input)
	if err != nil {
		return model.SetupSession{}, err
	}
	if _, err := s.sessions.AppendMessage(ctx, session.ID, "user", input.SeedIdea); err != nil {
		return model.SetupSession{}, err
	}
	return s.sessions.GetSessionByID(ctx, session.ID)
}

// SetupSessionAdvancer 负责推进设置会话。
// 管理从用户提示到设置运行的转换。
type SetupSessionAdvancer struct {
	sessions  port.SetupSessionRepository
	audit     port.AuditRepository
	generator port.SetupRunGenerator
	events    port.GenerationEventStream
}

func NewSetupSessionAdvancer(
	sessions port.SetupSessionRepository,
	audit port.AuditRepository,
	generator port.SetupRunGenerator,
	events port.GenerationEventStream,
) *SetupSessionAdvancer {
	return &SetupSessionAdvancer{sessions: sessions, audit: audit, generator: generator, events: events}
}

// Advance 添加用户消息并创建新的设置运行。
func (s *SetupSessionAdvancer) Advance(ctx context.Context, sessionID string, input model.AdvanceSetupSessionInput) (model.SetupRun, error) {
	session, err := s.sessions.GetSessionByID(ctx, sessionID)
	if err != nil {
		return model.SetupRun{}, err
	}
	if _, err := s.sessions.AppendMessage(ctx, sessionID, "user", input.UserMessage); err != nil {
		return model.SetupRun{}, err
	}
	session.LastUserMessage = input.UserMessage
	session.Status = domain.SessionStatusAdvancing
	if _, err := s.sessions.UpdateSession(ctx, session); err != nil {
		return model.SetupRun{}, err
	}
	run, err := s.sessions.CreateRun(ctx, sessionID, input)
	if err != nil {
		return model.SetupRun{}, err
	}
	s.appendAuditEvent(ctx, run.RunID, domain.EventGenerationStep, map[string]any{"step": domain.RunStatusQueued, "progress": 0})
	return run, nil
}

func (s *SetupSessionAdvancer) Generate(ctx context.Context, runID string) {
	run, err := s.sessions.GetRunByID(ctx, runID)
	if err != nil {
		log.Printf("setup run %s failed before load: %v", runID, err)
		s.appendAuditEvent(ctx, runID, domain.EventGenerationStep, map[string]any{"step": domain.RunStatusFailed, "error": err.Error()})
		s.publish(ctx, runID, domain.EventGenerationStep, map[string]any{"step": domain.RunStatusFailed, "error": err.Error()})
		return
	}
	session, err := s.sessions.GetSessionByID(ctx, run.SessionID)
	if err != nil {
		s.failRun(ctx, runID, session, err)
		return
	}
	s.appendAuditEvent(ctx, runID, domain.EventGenerationStep, map[string]any{"step": "inferring_setup", "progress": 20})
	s.publish(ctx, runID, domain.EventGenerationStep, map[string]any{"step": "inferring_setup", "progress": 20})
	if err := s.sessions.UpdateRunStatus(ctx, runID, domain.RunStatusLoadingState, "inferring_setup", 20); err != nil {
		s.failRun(ctx, runID, session, err)
		return
	}
	heartbeatCtx, stopHeartbeat := context.WithCancel(ctx)
	defer stopHeartbeat()
	go s.heartbeatRun(heartbeatCtx, runID)
	result, err := s.generator.Generate(ctx, port.SetupRunGenerationInput{Run: run, Session: session})
	if err != nil {
		s.failRun(ctx, runID, session, err)
		return
	}
	if err := s.sessions.SaveRunResult(ctx, runID, result); err != nil {
		s.failRun(ctx, runID, session, err)
		return
	}
	session.Status = domain.SessionStatusReviewing
	if _, err := s.sessions.UpdateSession(ctx, session); err != nil {
		s.failRun(ctx, runID, session, err)
		return
	}
	s.appendAuditEvent(ctx, runID, domain.EventGenerationStep, map[string]any{"step": domain.RunStatusCompleted, "run_id": runID, "result_available": true})
	s.publish(ctx, runID, domain.EventGenerationStep, map[string]any{"step": domain.RunStatusCompleted, "run_id": runID, "result_available": true})
}

func (s *SetupSessionAdvancer) heartbeatRun(ctx context.Context, runID string) {
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
				log.Printf("setup run %s heartbeat failed: %v", runID, err)
			}
		}
	}
}

func (s *SetupSessionAdvancer) failRun(ctx context.Context, runID string, session model.SetupSession, err error) {
	if isRunLeaseLost(err) {
		log.Printf("setup run %s lease lost; not marking failed", runID)
		return
	}
	if updateErr := s.sessions.UpdateRunStatus(ctx, runID, domain.RunStatusFailed, domain.RunStatusFailed, 100, err.Error()); isRunLeaseLost(updateErr) {
		log.Printf("setup run %s lease lost; not marking failed", runID)
		return
	}
	if session.ID != "" {
		session.Status = domain.SessionStatusFailed
		_, _ = s.sessions.UpdateSession(ctx, session)
	}
	log.Printf("setup run %s failed: %v", runID, err)
	s.appendAuditEvent(ctx, runID, domain.EventGenerationStep, map[string]any{"step": domain.RunStatusFailed, "error": err.Error()})
	s.publish(ctx, runID, domain.EventGenerationStep, map[string]any{"step": domain.RunStatusFailed, "error": err.Error()})
}

func (s *SetupSessionAdvancer) publish(ctx context.Context, runID string, name string, data any) {
	if s.events == nil {
		return
	}
	_ = s.events.Publish(ctx, runID, port.GenerationEvent{Name: name, Data: data})
}

func (s *SetupSessionAdvancer) appendAuditEvent(ctx context.Context, runID string, name string, data map[string]any) {
	if s.audit == nil {
		return
	}
	if _, err := s.audit.AppendRunEvent(ctx, model.RunEvent{
		RunKind:   "setup",
		RunID:     runID,
		EventName: name,
		Payload:   data,
	}); err != nil {
		log.Printf("append setup run event %s failed: %v", runID, err)
	}
}
