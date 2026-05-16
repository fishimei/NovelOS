package service

import (
	"context"

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
	generator port.SetupRunGenerator
	events    port.GenerationEventStream
}

func NewSetupSessionAdvancer(sessions port.SetupSessionRepository, generator port.SetupRunGenerator, events port.GenerationEventStream) *SetupSessionAdvancer {
	return &SetupSessionAdvancer{sessions: sessions, generator: generator, events: events}
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
	if s.generator != nil {
		go s.generate(context.Background(), run.RunID)
	}
	return run, nil
}

func (s *SetupSessionAdvancer) generate(ctx context.Context, runID string) {
	run, err := s.sessions.GetRunByID(ctx, runID)
	if err != nil {
		s.publish(ctx, runID, domain.EventGenerationStep, map[string]any{"step": domain.RunStatusFailed, "error": err.Error()})
		return
	}
	session, err := s.sessions.GetSessionByID(ctx, run.SessionID)
	if err != nil {
		s.failRun(ctx, runID, session, err)
		return
	}
	s.publish(ctx, runID, domain.EventGenerationStep, map[string]any{"step": "inferring_setup", "progress": 20})
	if err := s.sessions.UpdateRunStatus(ctx, runID, domain.RunStatusLoadingState, "inferring_setup", 20); err != nil {
		s.failRun(ctx, runID, session, err)
		return
	}
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
	s.publish(ctx, runID, domain.EventReviewRequired, map[string]any{"run_id": runID, "result_available": true})
}

func (s *SetupSessionAdvancer) failRun(ctx context.Context, runID string, session model.SetupSession, err error) {
	_ = s.sessions.UpdateRunStatus(ctx, runID, domain.RunStatusFailed, domain.RunStatusFailed, 100)
	if session.ID != "" {
		session.Status = domain.SessionStatusFailed
		_, _ = s.sessions.UpdateSession(ctx, session)
	}
	s.publish(ctx, runID, domain.EventGenerationStep, map[string]any{"step": domain.RunStatusFailed, "error": err.Error()})
}

func (s *SetupSessionAdvancer) publish(ctx context.Context, runID string, name string, data any) {
	if s.events == nil {
		return
	}
	_ = s.events.Publish(ctx, runID, port.GenerationEvent{Name: name, Data: data})
}
