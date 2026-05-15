package service

import (
	"context"

	"github.com/fishimei/NovelOS/internal/application/model"
	"github.com/fishimei/NovelOS/internal/application/port"
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
	sessions port.SetupSessionRepository
}

func NewSetupSessionAdvancer(sessions port.SetupSessionRepository) *SetupSessionAdvancer {
	return &SetupSessionAdvancer{sessions: sessions}
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
	session.Status = "advancing"
	if _, err := s.sessions.UpdateSession(ctx, session); err != nil {
		return model.SetupRun{}, err
	}
	return s.sessions.CreateRun(ctx, sessionID, input)
}
