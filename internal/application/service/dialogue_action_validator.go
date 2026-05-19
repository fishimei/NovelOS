package service

import (
	"context"

	"github.com/fishimei/NovelOS/internal/application/model"
	"github.com/fishimei/NovelOS/internal/application/port"
	"github.com/fishimei/NovelOS/internal/domain"
	"github.com/fishimei/NovelOS/internal/pkgerr"
)

type DialogueActionValidator struct {
	setupSessions port.SetupSessionRepository
	storySessions port.StorySessionRepository
}

func NewDialogueActionValidator(setupSessions port.SetupSessionRepository, storySessions port.StorySessionRepository) *DialogueActionValidator {
	return &DialogueActionValidator{setupSessions: setupSessions, storySessions: storySessions}
}

func (v *DialogueActionValidator) ValidateOption(ctx context.Context, option model.DialogueActionOption) error {
	switch option.ActionType {
	case domain.DialogueActionSetupStartAndAdvance, domain.DialogueActionStoryCreateAndAdvance:
		return nil
	case domain.DialogueActionSetupAdvance:
		sessionID := stringValue(option.Payload, "setup_session_id")
		if sessionID == "" {
			return pkgerr.Validation("setup session id is required")
		}
		session, err := v.setupSessions.GetSessionByID(ctx, sessionID)
		if err != nil {
			return err
		}
		if session.ProjectID != option.ProjectID {
			return pkgerr.Validation("setup session does not belong to project")
		}
		return nil
	case domain.DialogueActionSetupApply:
		sessionID := stringValue(option.Payload, "setup_session_id")
		runID := stringValue(option.Payload, "setup_run_id")
		if sessionID == "" || runID == "" {
			return pkgerr.Validation("setup session id and run id are required")
		}
		session, err := v.setupSessions.GetSessionByID(ctx, sessionID)
		if err != nil {
			return err
		}
		if session.ProjectID != option.ProjectID {
			return pkgerr.Validation("setup session does not belong to project")
		}
		run, err := v.setupSessions.GetRunByID(ctx, runID)
		if err != nil {
			return err
		}
		if run.SessionID != sessionID || run.ProjectID != option.ProjectID {
			return pkgerr.Validation("setup run does not belong to session")
		}
		if _, err := v.setupSessions.GetRunResultByID(ctx, runID); err != nil {
			return err
		}
		return nil
	case domain.DialogueActionStoryAdvance:
		sessionID := stringValue(option.Payload, "story_session_id")
		if sessionID == "" {
			return pkgerr.Validation("story session id is required")
		}
		session, err := v.storySessions.GetSessionByID(ctx, sessionID)
		if err != nil {
			return err
		}
		if session.ProjectID != option.ProjectID {
			return pkgerr.Validation("story session does not belong to project")
		}
		return nil
	case domain.DialogueActionStoryCommit:
		runID := stringValue(option.Payload, "story_run_id")
		if runID == "" {
			return pkgerr.Validation("story run id is required")
		}
		run, err := v.storySessions.GetRunByID(ctx, runID)
		if err != nil {
			return err
		}
		if run.ProjectID != option.ProjectID {
			return pkgerr.Validation("story run does not belong to project")
		}
		if run.Status == domain.RunStatusCommitted || run.CommittedAt != nil {
			return pkgerr.Conflict(pkgerr.CodeRunAlreadyCommitted, "story run already committed")
		}
		result, err := v.storySessions.GetRunResultByID(ctx, runID)
		if err != nil {
			return err
		}
		if draftID := stringValue(option.Payload, "draft_id"); draftID != "" && draftID != result.Draft.ID {
			return pkgerr.Validation("draft id mismatch")
		}
		if patchID := stringValue(option.Payload, "memory_patch_id"); patchID != "" && patchID != result.MemoryPatch.ID {
			return pkgerr.Validation("memory patch id mismatch")
		}
		return nil
	default:
		return pkgerr.Validation("unsupported dialogue action type")
	}
}

func stringValue(payload map[string]any, key string) string {
	if payload == nil {
		return ""
	}
	value, ok := payload[key]
	if !ok || value == nil {
		return ""
	}
	if s, ok := value.(string); ok {
		return s
	}
	return ""
}

func boolValue(payload map[string]any, key string, fallback bool) bool {
	if payload == nil {
		return fallback
	}
	value, ok := payload[key]
	if !ok || value == nil {
		return fallback
	}
	if b, ok := value.(bool); ok {
		return b
	}
	return fallback
}
