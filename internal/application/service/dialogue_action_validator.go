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
	storyEvents   port.StoryEventStore
}

func NewDialogueActionValidator(setupSessions port.SetupSessionRepository, storySessions port.StorySessionRepository, storyEvents port.StoryEventStore) *DialogueActionValidator {
	return &DialogueActionValidator{setupSessions: setupSessions, storySessions: storySessions, storyEvents: storyEvents}
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
	case domain.DialogueActionStoryCutChapter:
		return v.validateStoryCutChapter(ctx, option)
	case domain.DialogueActionStoryForkFromEvent:
		eventID := stringValue(option.Payload, "event_id")
		if eventID == "" {
			return pkgerr.Validation("event id is required")
		}
		if v.storyEvents == nil {
			return pkgerr.Internal("story event store is required", nil)
		}
		event, err := v.storyEvents.GetEvent(ctx, eventID)
		if err != nil {
			return err
		}
		if event.ProjectID != option.ProjectID {
			return pkgerr.Validation("story event does not belong to project")
		}
		sessionID := stringValue(option.Payload, "story_session_id")
		if sessionID != "" && event.SessionID != sessionID {
			return pkgerr.Validation("story event does not belong to session")
		}
		if len(stringValue(option.Payload, "name")) > 100 {
			return pkgerr.Validation("fork name is too long")
		}
		if len(stringValue(option.Payload, "author_message")) > 2000 {
			return pkgerr.Validation("fork author message is too long")
		}
		return nil
	default:
		return pkgerr.Validation("unsupported dialogue action type")
	}
}

func (v *DialogueActionValidator) ValidateAutoApproved(_ context.Context, option model.DialogueActionOption) error {
	return validateAutoApprovedDialogueAction(option)
}

func validateAutoApprovedDialogueAction(option model.DialogueActionOption) error {
	if option.ConfirmationRequired {
		return pkgerr.Validation("dialogue action option requires manual confirmation")
	}
	switch option.ActionType {
	case domain.DialogueActionStoryAdvance:
		return nil
	case domain.DialogueActionStoryCutChapter:
		if stringValue(option.Payload, "span_policy") != "latest_completed" {
			return pkgerr.Validation("auto story cut must use latest_completed span policy")
		}
		if stringValue(option.Payload, "branch_id") != "" || stringValue(option.Payload, "from_event_id") != "" || stringValue(option.Payload, "to_event_id") != "" {
			return pkgerr.Validation("auto story cut latest completed span must not include event ids")
		}
		return nil
	default:
		return pkgerr.Validation("dialogue action type requires manual confirmation")
	}
}

func (v *DialogueActionValidator) validateStoryCutChapter(ctx context.Context, option model.DialogueActionOption) error {
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
	if run.Status == domain.RunStatusCut || run.CutAt != nil {
		return pkgerr.Conflict(pkgerr.CodeConflict, "story run is already cut")
	}
	if stringValue(option.Payload, "span_policy") == "latest_completed" {
		if run.Status != domain.RunStatusCompleted {
			return pkgerr.Validation("latest completed span requires a completed story run")
		}
		if stringValue(option.Payload, "branch_id") != "" || stringValue(option.Payload, "from_event_id") != "" || stringValue(option.Payload, "to_event_id") != "" {
			return pkgerr.Validation("latest completed span is resolved by service and must not include event ids")
		}
	}
	if stringValue(option.Payload, "branch_id") == "" && run.BranchID == "" {
		return pkgerr.Validation("branch id is required")
	}
	if stringValue(option.Payload, "from_event_id") == "" && run.BaseEventID == "" {
		return pkgerr.Validation("from event id is required")
	}
	if stringValue(option.Payload, "to_event_id") == "" && run.HeadEventID == "" {
		return pkgerr.Validation("to event id is required")
	}
	return nil
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
