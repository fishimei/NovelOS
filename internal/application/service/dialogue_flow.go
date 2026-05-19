package service

import (
	"context"
	"log"

	"github.com/fishimei/NovelOS/internal/application/model"
	"github.com/fishimei/NovelOS/internal/application/port"
	"github.com/fishimei/NovelOS/internal/domain"
)

type DialogueSessionStarter struct {
	sessions port.DialogueSessionRepository
}

func NewDialogueSessionStarter(sessions port.DialogueSessionRepository) *DialogueSessionStarter {
	return &DialogueSessionStarter{sessions: sessions}
}

func (s *DialogueSessionStarter) Start(ctx context.Context, projectID string, input model.CreateDialogueSessionInput) (model.DialogueSession, error) {
	return s.sessions.CreateSession(ctx, projectID, input)
}

type DialogueSessionAdvancer struct {
	sessions  port.DialogueSessionRepository
	audit     port.AuditRepository
	generator port.DialogueRunGenerator
	events    port.GenerationEventStream
}

func NewDialogueSessionAdvancer(
	sessions port.DialogueSessionRepository,
	audit port.AuditRepository,
	generator port.DialogueRunGenerator,
	events port.GenerationEventStream,
) *DialogueSessionAdvancer {
	return &DialogueSessionAdvancer{sessions: sessions, audit: audit, generator: generator, events: events}
}

func (s *DialogueSessionAdvancer) Advance(ctx context.Context, sessionID string, input model.AdvanceDialogueSessionInput) (model.DialogueRun, error) {
	session, err := s.sessions.GetSessionByID(ctx, sessionID)
	if err != nil {
		return model.DialogueRun{}, err
	}
	if _, err := s.sessions.AppendMessage(ctx, sessionID, "user", input.UserMessage, nil); err != nil {
		return model.DialogueRun{}, err
	}
	session.LastUserMessage = input.UserMessage
	session.Status = domain.SessionStatusAdvancing
	if _, err := s.sessions.UpdateSession(ctx, session); err != nil {
		return model.DialogueRun{}, err
	}
	run, err := s.sessions.CreateRun(ctx, sessionID, input)
	if err != nil {
		return model.DialogueRun{}, err
	}
	if s.generator != nil {
		go s.generate(context.Background(), run.RunID)
	}
	return run, nil
}

func (s *DialogueSessionAdvancer) generate(ctx context.Context, runID string) {
	run, err := s.sessions.GetRunByID(ctx, runID)
	if err != nil {
		log.Printf("dialogue run %s failed before load: %v", runID, err)
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
	s.publish(ctx, runID, domain.EventGenerationStep, map[string]any{"step": domain.RunStatusPlanningActions, "progress": 30})
	if err := s.sessions.UpdateRunStatus(ctx, runID, domain.RunStatusPlanningActions, domain.RunStatusPlanningActions, 30); err != nil {
		s.failRun(ctx, runID, session, err)
		return
	}
	result, err := s.generator.Generate(ctx, port.DialogueRunGenerationInput{Run: run, Session: session})
	if err != nil {
		s.failRun(ctx, runID, session, err)
		return
	}
	if result.Status == "" {
		if len(result.ActionOptions) > 0 {
			result.Status = domain.RunStatusReviewRequired
		} else {
			result.Status = domain.RunStatusCompleted
		}
	}
	if result.AssistantMessage != "" {
		if _, err := s.sessions.AppendMessage(ctx, session.ID, "assistant", result.AssistantMessage, map[string]any{"run_id": runID}); err != nil {
			s.failRun(ctx, runID, session, err)
			return
		}
	}
	if len(result.ActionOptions) > 0 {
		if err := s.sessions.SaveActionOptions(ctx, result.ActionOptions); err != nil {
			s.failRun(ctx, runID, session, err)
			return
		}
	}
	if err := s.sessions.SaveRunResult(ctx, runID, result); err != nil {
		s.failRun(ctx, runID, session, err)
		return
	}
	if len(result.ActionOptions) > 0 {
		session.Status = domain.SessionStatusAwaitingConfirmation
	} else {
		session.Status = domain.SessionStatusIdle
	}
	if _, err := s.sessions.UpdateSession(ctx, session); err != nil {
		s.failRun(ctx, runID, session, err)
		return
	}
	s.publish(ctx, runID, domain.EventReviewRequired, map[string]any{"run_id": runID, "result_available": true, "action_options": len(result.ActionOptions)})
}

func (s *DialogueSessionAdvancer) failRun(ctx context.Context, runID string, session model.DialogueSession, err error) {
	_ = s.sessions.UpdateRunStatus(ctx, runID, domain.RunStatusFailed, domain.RunStatusFailed, 100, err.Error())
	if session.ID != "" {
		session.Status = domain.SessionStatusFailed
		_, _ = s.sessions.UpdateSession(ctx, session)
	}
	log.Printf("dialogue run %s failed: %v", runID, err)
	s.appendAuditEvent(ctx, runID, domain.EventGenerationStep, map[string]any{"step": domain.RunStatusFailed, "error": err.Error()})
	s.publish(ctx, runID, domain.EventGenerationStep, map[string]any{"step": domain.RunStatusFailed, "error": err.Error()})
}

func (s *DialogueSessionAdvancer) publish(ctx context.Context, runID string, name string, data any) {
	if s.events == nil {
		return
	}
	_ = s.events.Publish(ctx, runID, port.GenerationEvent{Name: name, Data: data})
}

func (s *DialogueSessionAdvancer) appendAuditEvent(ctx context.Context, runID string, name string, data map[string]any) {
	if s.audit == nil {
		return
	}
	if _, err := s.audit.AppendRunEvent(ctx, model.RunEvent{
		RunKind:   "dialogue",
		RunID:     runID,
		EventName: name,
		Payload:   data,
	}); err != nil {
		log.Printf("append dialogue run event %s failed: %v", runID, err)
	}
}
