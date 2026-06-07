package repository

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/fishimei/NovelOS/internal/application/model"
	"github.com/fishimei/NovelOS/internal/application/port"
	"github.com/fishimei/NovelOS/internal/domain"
	persistencemodels "github.com/fishimei/NovelOS/internal/infrastructure/persistence/gorm/models"
	"github.com/fishimei/NovelOS/internal/pkgerr"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type setupRunResultPayload struct {
	SetupDraft model.SetupDraft `json:"setup_draft"`
}

const runLeaseLostMessage = "run lease lost"

func runLeaseLostError() error {
	return pkgerr.Conflict(pkgerr.CodeConflict, runLeaseLostMessage)
}

func runLeaseScopedDB(ctx context.Context, db *gorm.DB) (*gorm.DB, bool) {
	lease, ok := port.RunLeaseFromContext(ctx)
	if !ok {
		return db, false
	}
	return db.Where("lease_owner = ?", lease.Owner), true
}

func runLeaseMutationError(ctx context.Context, result *gorm.DB, resource string) error {
	if result.Error != nil {
		return mapDBError(result.Error, resource)
	}
	if _, ok := port.RunLeaseFromContext(ctx); ok && result.RowsAffected == 0 {
		return runLeaseLostError()
	}
	return nil
}

func applyRunLeaseUpdates(ctx context.Context, updates map[string]any, status string, now time.Time) {
	if terminalRunStatus(status) {
		updates["lease_owner"] = ""
		updates["lease_expires_at"] = nil
		return
	}
	lease, ok := port.RunLeaseFromContext(ctx)
	if !ok {
		return
	}
	updates["lease_owner"] = lease.Owner
	updates["lease_expires_at"] = lease.ExpiresAt(now)
}

func terminalRunStatus(status string) bool {
	switch status {
	case domain.RunStatusCompleted, domain.RunStatusCut, domain.RunStatusFailed, domain.RunStatusCancelled, "applied":
		return true
	default:
		return false
	}
}

type storyRunResultPayload struct {
	BranchID               string                             `json:"branch_id,omitempty"`
	BaseEventID            string                             `json:"base_event_id,omitempty"`
	HeadEventID            string                             `json:"head_event_id,omitempty"`
	PlotVariable           model.PlotVariable                 `json:"plot_variable"`
	EventPlan              []model.StoryEventPlan             `json:"event_plan"`
	Turns                  []model.StoryTurn                  `json:"turns"`
	SceneSummary           string                             `json:"scene_summary"`
	InteractionAnalysis    model.StoryInteractionAnalysis     `json:"interaction_analysis"`
	InteractionTranscripts []model.StoryInteractionTranscript `json:"interaction_transcripts"`
	Draft                  model.Draft                        `json:"draft"`
	Review                 model.ReviewReport                 `json:"review"`
	MemoryPatch            model.MemoryPatch                  `json:"memory_patch"`
	Events                 []model.StoryEvent                 `json:"events,omitempty"`
	CompletedActions       []model.OngoingAction              `json:"completed_actions,omitempty"`
	SupersededActions      []model.OngoingAction              `json:"superseded_actions,omitempty"`
	CollisionAt            *time.Time                         `json:"collision_at,omitempty"`
}

type dialogueRunResultPayload struct {
	AssistantMessage    string                       `json:"assistant_message"`
	ActionOptions       []model.DialogueActionOption `json:"action_options"`
	ClarifyingQuestions []model.DialogueQuestion     `json:"clarifying_questions"`
	SuggestedReplies    []string                     `json:"suggested_replies"`
	ContextSummary      string                       `json:"context_summary"`
	ToolTrace           []model.DialogueToolTrace    `json:"tool_trace"`
}

type setupSessionRepository struct {
	*container
}

func (r *setupSessionRepository) CreateSession(ctx context.Context, projectID string, input model.CreateSetupSessionInput) (model.SetupSession, error) {
	now := r.now()
	row := persistencemodels.SetupSession{
		ID:        r.nextID("setup"),
		ProjectID: projectID,
		SeedIdea:  input.SeedIdea,
		Status:    "active",
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := r.dbFor(ctx).Create(&row).Error; err != nil {
		return model.SetupSession{}, mapDBError(err, "setup session not found")
	}
	return r.setupSessionFromRow(ctx, row), nil
}

func (r *setupSessionRepository) ListSessionsByProjectID(ctx context.Context, projectID string, pageInput model.PageInput) (model.ListResult[model.SetupSession], error) {
	db := r.dbFor(ctx)
	var total int64
	if err := db.Model(&persistencemodels.SetupSession{}).Where("project_id = ?", projectID).Count(&total).Error; err != nil {
		return model.ListResult[model.SetupSession]{}, mapDBError(err, "setup session not found")
	}
	if pageInput.Page <= 0 {
		pageInput.Page = 1
	}
	if pageInput.PageSize <= 0 {
		pageInput.PageSize = 20
	}
	var rows []persistencemodels.SetupSession
	if err := db.Where("project_id = ?", projectID).Order("created_at desc").Limit(pageInput.PageSize).Offset((pageInput.Page - 1) * pageInput.PageSize).Find(&rows).Error; err != nil {
		return model.ListResult[model.SetupSession]{}, mapDBError(err, "setup session not found")
	}
	items := make([]model.SetupSession, 0, len(rows))
	for _, row := range rows {
		items = append(items, r.setupSessionFromRow(ctx, row))
	}
	return model.ListResult[model.SetupSession]{Items: items, Total: int(total)}, nil
}

func (r *setupSessionRepository) GetSessionByID(ctx context.Context, sessionID string) (model.SetupSession, error) {
	db := r.dbFor(ctx)
	var row persistencemodels.SetupSession
	if err := db.First(&row, "id = ?", sessionID).Error; err != nil {
		return model.SetupSession{}, mapDBError(err, "setup session not found")
	}
	var messages []persistencemodels.SetupMessage
	if err := db.Where("session_id = ?", sessionID).Order("created_at asc").Find(&messages).Error; err != nil {
		return model.SetupSession{}, mapDBError(err, "setup session not found")
	}
	out := r.setupSessionFromRow(ctx, row)
	out.Messages = make([]model.ConversationMessage, 0, len(messages))
	for _, msg := range messages {
		out.Messages = append(out.Messages, toMessage(msg.ID, msg.SessionID, msg.Role, msg.Content, msg.CreatedAt))
	}
	return out, nil
}

func (r *setupSessionRepository) UpdateSession(ctx context.Context, session model.SetupSession) (model.SetupSession, error) {
	db := r.dbFor(ctx)
	if err := db.Model(&persistencemodels.SetupSession{}).Where("id = ?", session.ID).Updates(map[string]any{
		"last_user_message": session.LastUserMessage,
		"status":            session.Status,
		"updated_at":        r.now(),
	}).Error; err != nil {
		return model.SetupSession{}, mapDBError(err, "setup session not found")
	}
	return r.GetSessionByID(ctx, session.ID)
}

func (r *setupSessionRepository) DeleteSession(ctx context.Context, sessionID string) error {
	err := r.dbFor(ctx).Transaction(func(tx *gorm.DB) error {
		var session persistencemodels.SetupSession
		if err := tx.First(&session, "id = ?", sessionID).Error; err != nil {
			return err
		}

		var runIDs []string
		if err := tx.Model(&persistencemodels.SetupRun{}).Where("session_id = ?", sessionID).Pluck("id", &runIDs).Error; err != nil {
			return err
		}
		if len(runIDs) > 0 {
			if err := tx.Where("run_kind = ? AND run_id IN ?", "setup", runIDs).Delete(&persistencemodels.RunEvent{}).Error; err != nil {
				return err
			}
			if err := tx.Where("run_id IN ?", runIDs).Delete(&persistencemodels.SetupRunResult{}).Error; err != nil {
				return err
			}
			if err := tx.Where("id IN ?", runIDs).Delete(&persistencemodels.SetupRun{}).Error; err != nil {
				return err
			}
		}
		if err := tx.Where("session_id = ?", sessionID).Delete(&persistencemodels.SetupMessage{}).Error; err != nil {
			return err
		}

		var dialogueSessionIDs []string
		if err := tx.Model(&persistencemodels.DialogueSession{}).
			Where("project_id = ? AND title = ?", session.ProjectID, "setup-discussion:"+sessionID).
			Pluck("id", &dialogueSessionIDs).Error; err != nil {
			return err
		}
		if len(dialogueSessionIDs) > 0 {
			var dialogueRunIDs []string
			if err := tx.Model(&persistencemodels.DialogueRun{}).Where("session_id IN ?", dialogueSessionIDs).Pluck("id", &dialogueRunIDs).Error; err != nil {
				return err
			}
			if err := tx.Where("session_id IN ?", dialogueSessionIDs).Delete(&persistencemodels.DialogueActionOption{}).Error; err != nil {
				return err
			}
			if len(dialogueRunIDs) > 0 {
				if err := tx.Where("run_kind = ? AND run_id IN ?", "dialogue", dialogueRunIDs).Delete(&persistencemodels.RunEvent{}).Error; err != nil {
					return err
				}
				if err := tx.Where("run_id IN ?", dialogueRunIDs).Delete(&persistencemodels.DialogueRunResult{}).Error; err != nil {
					return err
				}
				if err := tx.Where("id IN ?", dialogueRunIDs).Delete(&persistencemodels.DialogueRun{}).Error; err != nil {
					return err
				}
			}
			if err := tx.Where("session_id IN ?", dialogueSessionIDs).Delete(&persistencemodels.DialogueMessage{}).Error; err != nil {
				return err
			}
			if err := tx.Where("id IN ?", dialogueSessionIDs).Delete(&persistencemodels.DialogueSession{}).Error; err != nil {
				return err
			}
		}
		return tx.Delete(&session).Error
	})
	if err != nil {
		return mapDBError(err, "setup session not found")
	}
	return nil
}

func (r *setupSessionRepository) setupSessionFromRow(ctx context.Context, row persistencemodels.SetupSession) model.SetupSession {
	session := model.SetupSession{
		ID:              row.ID,
		ProjectID:       row.ProjectID,
		SeedIdea:        row.SeedIdea,
		LastUserMessage: row.LastUserMessage,
		Status:          row.Status,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
	}
	var latestRun persistencemodels.SetupRun
	if err := r.dbFor(ctx).Where("session_id = ?", row.ID).Order("created_at desc").Take(&latestRun).Error; err == nil {
		session.LatestRunID = latestRun.ID
		session.LatestRunStatus = latestRun.Status
		session.LatestRunError = latestRun.Error
	}
	return session
}

func (r *setupSessionRepository) AppendMessage(ctx context.Context, sessionID string, role string, content string) (model.ConversationMessage, error) {
	row := persistencemodels.SetupMessage{
		ID:        r.nextID("smsg"),
		SessionID: sessionID,
		Role:      role,
		Content:   content,
		CreatedAt: r.now(),
	}
	if err := r.dbFor(ctx).Create(&row).Error; err != nil {
		return model.ConversationMessage{}, mapDBError(err, "setup session not found")
	}
	return toMessage(row.ID, row.SessionID, row.Role, row.Content, row.CreatedAt), nil
}

func (r *setupSessionRepository) CreateRun(ctx context.Context, sessionID string, input model.AdvanceSetupSessionInput) (model.SetupRun, error) {
	session, err := r.GetSessionByID(ctx, sessionID)
	if err != nil {
		return model.SetupRun{}, err
	}
	row := persistencemodels.SetupRun{
		ID:          r.nextID("run"),
		SessionID:   sessionID,
		ProjectID:   session.ProjectID,
		Status:      "queued",
		CurrentStep: "collecting_context",
		Progress:    0,
		CreatedAt:   r.now(),
		UpdatedAt:   r.now(),
	}
	if err := r.dbFor(ctx).Create(&row).Error; err != nil {
		return model.SetupRun{}, mapDBError(err, "setup run not found")
	}
	return model.SetupRun{
		RunID:       row.ID,
		SessionID:   row.SessionID,
		ProjectID:   row.ProjectID,
		Status:      row.Status,
		CurrentStep: row.CurrentStep,
		Progress:    row.Progress,
		Error:       row.Error,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}, nil
}

func (r *setupSessionRepository) GetRunByID(ctx context.Context, runID string) (model.SetupRun, error) {
	var row persistencemodels.SetupRun
	if err := r.dbFor(ctx).First(&row, "id = ?", runID).Error; err != nil {
		return model.SetupRun{}, mapDBError(err, "setup run not found")
	}
	return model.SetupRun{
		RunID:       row.ID,
		SessionID:   row.SessionID,
		ProjectID:   row.ProjectID,
		Status:      row.Status,
		CurrentStep: row.CurrentStep,
		Progress:    row.Progress,
		Error:       row.Error,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}, nil
}

func (r *setupSessionRepository) GetRunResultByID(ctx context.Context, runID string) (model.SetupRunResult, error) {
	var row persistencemodels.SetupRunResult
	if err := r.dbFor(ctx).First(&row, "run_id = ?", runID).Error; err != nil {
		return model.SetupRunResult{}, mapDBError(err, "setup run result not found")
	}
	payload, err := decodeJSON[setupRunResultPayload](row.PayloadJSON)
	if err != nil {
		return model.SetupRunResult{}, payloadError("setup run result", err)
	}
	return model.SetupRunResult{
		RunID:      row.RunID,
		SessionID:  row.SessionID,
		Status:     row.Status,
		SetupDraft: payload.SetupDraft,
	}, nil
}

func (r *setupSessionRepository) SaveRunResult(ctx context.Context, runID string, result model.SetupRunResult) error {
	payloadJSON, err := encodeJSON(setupRunResultPayload{SetupDraft: result.SetupDraft})
	if err != nil {
		return payloadError("setup run result", err)
	}
	now := r.now()
	row := persistencemodels.SetupRunResult{
		ID:          r.nextID("sresult"),
		RunID:       runID,
		SessionID:   result.SessionID,
		Status:      result.Status,
		PayloadJSON: payloadJSON,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	updates := map[string]any{
		"status":       result.Status,
		"current_step": "completed",
		"progress":     100,
		"error":        "",
		"updated_at":   now,
	}
	applyRunLeaseUpdates(ctx, updates, result.Status, now)
	db, _ := runLeaseScopedDB(ctx, r.dbFor(ctx).Model(&persistencemodels.SetupRun{}).Where("id = ?", runID))
	if err := runLeaseMutationError(ctx, db.Updates(updates), "setup run not found"); err != nil {
		return err
	}
	return mapDBError(r.dbFor(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "run_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"session_id", "status", "payload_json", "updated_at"}),
	}).Create(&row).Error, "setup run result not found")
}

func (r *setupSessionRepository) UpdateRunStatus(ctx context.Context, runID string, status string, currentStep string, progress int, errorMessage ...string) error {
	now := r.now()
	updates := map[string]any{
		"status":       status,
		"current_step": currentStep,
		"progress":     progress,
		"updated_at":   now,
	}
	if len(errorMessage) > 0 {
		updates["error"] = errorMessage[0]
	}
	applyRunLeaseUpdates(ctx, updates, status, now)
	db, _ := runLeaseScopedDB(ctx, r.dbFor(ctx).Model(&persistencemodels.SetupRun{}).Where("id = ?", runID))
	return runLeaseMutationError(ctx, db.Updates(updates), "setup run not found")
}

func (r *setupSessionRepository) UpdateRunHeartbeat(ctx context.Context, runID string) error {
	now := r.now()
	updates := map[string]any{"updated_at": now}
	applyRunLeaseUpdates(ctx, updates, domain.RunStatusLoadingState, now)
	db, _ := runLeaseScopedDB(ctx, r.dbFor(ctx).Model(&persistencemodels.SetupRun{}).Where("id = ?", runID))
	return runLeaseMutationError(ctx, db.Updates(updates), "setup run not found")
}

func (r *setupSessionRepository) MarkApplied(ctx context.Context, sessionID string, runID string) error {
	now := r.now()
	db := r.dbFor(ctx)
	updates := map[string]any{
		"status":       "applied",
		"current_step": "applied",
		"progress":     100,
		"error":        "",
		"updated_at":   now,
	}
	applyRunLeaseUpdates(ctx, updates, "applied", now)
	if err := db.Model(&persistencemodels.SetupRun{}).Where("id = ?", runID).Updates(updates).Error; err != nil {
		return mapDBError(err, "setup run not found")
	}
	return mapDBError(db.Model(&persistencemodels.SetupSession{}).Where("id = ?", sessionID).Updates(map[string]any{
		"status":     "committed",
		"updated_at": now,
	}).Error, "setup session not found")
}

type dialogueSessionRepository struct {
	*container
}

func (r *dialogueSessionRepository) CreateSession(ctx context.Context, projectID string, input model.CreateDialogueSessionInput) (model.DialogueSession, error) {
	now := r.now()
	row := persistencemodels.DialogueSession{
		ID:        r.nextID("dialogue"),
		ProjectID: projectID,
		Title:     input.Title,
		Status:    domain.SessionStatusIdle,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := r.dbFor(ctx).Create(&row).Error; err != nil {
		return model.DialogueSession{}, mapDBError(err, "dialogue session not found")
	}
	return r.dialogueSessionFromRow(ctx, row), nil
}

func (r *dialogueSessionRepository) ListSessionsByProjectID(ctx context.Context, projectID string, pageInput model.PageInput) (model.ListResult[model.DialogueSession], error) {
	db := r.dbFor(ctx)
	var total int64
	if err := db.Model(&persistencemodels.DialogueSession{}).Where("project_id = ?", projectID).Count(&total).Error; err != nil {
		return model.ListResult[model.DialogueSession]{}, mapDBError(err, "dialogue session not found")
	}
	if pageInput.Page <= 0 {
		pageInput.Page = 1
	}
	if pageInput.PageSize <= 0 {
		pageInput.PageSize = 20
	}
	var rows []persistencemodels.DialogueSession
	if err := db.Where("project_id = ?", projectID).Order("created_at desc").Limit(pageInput.PageSize).Offset((pageInput.Page - 1) * pageInput.PageSize).Find(&rows).Error; err != nil {
		return model.ListResult[model.DialogueSession]{}, mapDBError(err, "dialogue session not found")
	}
	items := make([]model.DialogueSession, 0, len(rows))
	for _, row := range rows {
		items = append(items, r.dialogueSessionFromRow(ctx, row))
	}
	return model.ListResult[model.DialogueSession]{Items: items, Total: int(total)}, nil
}

func (r *dialogueSessionRepository) GetSessionByID(ctx context.Context, sessionID string) (model.DialogueSession, error) {
	var row persistencemodels.DialogueSession
	if err := r.dbFor(ctx).First(&row, "id = ?", sessionID).Error; err != nil {
		return model.DialogueSession{}, mapDBError(err, "dialogue session not found")
	}
	out := r.dialogueSessionFromRow(ctx, row)
	messages, err := r.ListMessagesBySessionID(ctx, sessionID)
	if err != nil {
		return model.DialogueSession{}, err
	}
	out.Messages = messages
	return out, nil
}

func (r *dialogueSessionRepository) UpdateSession(ctx context.Context, session model.DialogueSession) (model.DialogueSession, error) {
	if err := r.dbFor(ctx).Model(&persistencemodels.DialogueSession{}).Where("id = ?", session.ID).Updates(map[string]any{
		"title":             session.Title,
		"last_user_message": session.LastUserMessage,
		"status":            session.Status,
		"updated_at":        r.now(),
	}).Error; err != nil {
		return model.DialogueSession{}, mapDBError(err, "dialogue session not found")
	}
	return r.GetSessionByID(ctx, session.ID)
}

func (r *dialogueSessionRepository) AppendMessage(ctx context.Context, sessionID string, role string, content string, metadata map[string]any) (model.DialogueMessage, error) {
	metadataJSON, err := encodeJSON(metadata)
	if err != nil {
		return model.DialogueMessage{}, payloadError("dialogue message", err)
	}
	row := persistencemodels.DialogueMessage{
		ID:           r.nextID("dmsg"),
		SessionID:    sessionID,
		Role:         role,
		Content:      content,
		MetadataJSON: metadataJSON,
		CreatedAt:    r.now(),
	}
	if err := r.dbFor(ctx).Create(&row).Error; err != nil {
		return model.DialogueMessage{}, mapDBError(err, "dialogue session not found")
	}
	return toDialogueMessage(row)
}

func (r *dialogueSessionRepository) ListMessagesBySessionID(ctx context.Context, sessionID string) ([]model.DialogueMessage, error) {
	var rows []persistencemodels.DialogueMessage
	if err := r.dbFor(ctx).Where("session_id = ?", sessionID).Order("created_at asc").Find(&rows).Error; err != nil {
		return nil, mapDBError(err, "dialogue session not found")
	}
	items := make([]model.DialogueMessage, 0, len(rows))
	for _, row := range rows {
		msg, err := toDialogueMessage(row)
		if err != nil {
			return nil, payloadError("dialogue message", err)
		}
		items = append(items, msg)
	}
	return items, nil
}

func (r *dialogueSessionRepository) CreateRun(ctx context.Context, sessionID string, input model.AdvanceDialogueSessionInput) (model.DialogueRun, error) {
	session, err := r.GetSessionByID(ctx, sessionID)
	if err != nil {
		return model.DialogueRun{}, err
	}
	now := r.now()
	row := persistencemodels.DialogueRun{
		ID:          r.nextID("drun"),
		SessionID:   sessionID,
		ProjectID:   session.ProjectID,
		Status:      domain.RunStatusQueued,
		CurrentStep: domain.RunStatusLoadingState,
		Progress:    0,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := r.dbFor(ctx).Create(&row).Error; err != nil {
		return model.DialogueRun{}, mapDBError(err, "dialogue run not found")
	}
	return toDialogueRun(row), nil
}

func (r *dialogueSessionRepository) GetRunByID(ctx context.Context, runID string) (model.DialogueRun, error) {
	var row persistencemodels.DialogueRun
	if err := r.dbFor(ctx).First(&row, "id = ?", runID).Error; err != nil {
		return model.DialogueRun{}, mapDBError(err, "dialogue run not found")
	}
	return toDialogueRun(row), nil
}

func (r *dialogueSessionRepository) UpdateRunStatus(ctx context.Context, runID string, status string, currentStep string, progress int, errorMessage ...string) error {
	now := r.now()
	updates := map[string]any{
		"status":       status,
		"current_step": currentStep,
		"progress":     progress,
		"updated_at":   now,
	}
	if len(errorMessage) > 0 {
		updates["error"] = errorMessage[0]
	}
	applyRunLeaseUpdates(ctx, updates, status, now)
	db, _ := runLeaseScopedDB(ctx, r.dbFor(ctx).Model(&persistencemodels.DialogueRun{}).Where("id = ?", runID))
	return runLeaseMutationError(ctx, db.Updates(updates), "dialogue run not found")
}

func (r *dialogueSessionRepository) UpdateRunHeartbeat(ctx context.Context, runID string) error {
	now := r.now()
	updates := map[string]any{"updated_at": now}
	applyRunLeaseUpdates(ctx, updates, domain.RunStatusLoadingState, now)
	db, _ := runLeaseScopedDB(ctx, r.dbFor(ctx).Model(&persistencemodels.DialogueRun{}).Where("id = ?", runID))
	return runLeaseMutationError(ctx, db.Updates(updates), "dialogue run not found")
}

func (r *dialogueSessionRepository) SaveRunResult(ctx context.Context, runID string, result model.DialogueRunResult) error {
	payloadJSON, err := encodeJSON(dialogueRunResultPayload{
		AssistantMessage:    result.AssistantMessage,
		ActionOptions:       result.ActionOptions,
		ClarifyingQuestions: result.ClarifyingQuestions,
		SuggestedReplies:    result.SuggestedReplies,
		ContextSummary:      result.ContextSummary,
		ToolTrace:           result.ToolTrace,
	})
	if err != nil {
		return payloadError("dialogue run result", err)
	}
	now := r.now()
	row := persistencemodels.DialogueRunResult{
		ID:          r.nextID("dresult"),
		RunID:       runID,
		SessionID:   result.SessionID,
		Status:      result.Status,
		PayloadJSON: payloadJSON,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	updates := map[string]any{
		"status":       result.Status,
		"current_step": "completed",
		"progress":     100,
		"error":        "",
		"updated_at":   now,
	}
	applyRunLeaseUpdates(ctx, updates, result.Status, now)
	db, _ := runLeaseScopedDB(ctx, r.dbFor(ctx).Model(&persistencemodels.DialogueRun{}).Where("id = ?", runID))
	if err := runLeaseMutationError(ctx, db.Updates(updates), "dialogue run not found"); err != nil {
		return err
	}
	return mapDBError(r.dbFor(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "run_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"session_id", "status", "payload_json", "updated_at"}),
	}).Create(&row).Error, "dialogue run result not found")
}

func (r *dialogueSessionRepository) GetRunResultByID(ctx context.Context, runID string) (model.DialogueRunResult, error) {
	var row persistencemodels.DialogueRunResult
	if err := r.dbFor(ctx).First(&row, "run_id = ?", runID).Error; err != nil {
		return model.DialogueRunResult{}, mapDBError(err, "dialogue run result not found")
	}
	payload, err := decodeJSON[dialogueRunResultPayload](row.PayloadJSON)
	if err != nil {
		return model.DialogueRunResult{}, payloadError("dialogue run result", err)
	}
	options, err := r.ListActionOptionsByRunID(ctx, runID)
	if err != nil {
		return model.DialogueRunResult{}, err
	}
	if len(options) == 0 {
		options = payload.ActionOptions
	}
	return model.DialogueRunResult{
		RunID:               row.RunID,
		SessionID:           row.SessionID,
		Status:              row.Status,
		AssistantMessage:    payload.AssistantMessage,
		ActionOptions:       options,
		ClarifyingQuestions: payload.ClarifyingQuestions,
		SuggestedReplies:    payload.SuggestedReplies,
		ContextSummary:      payload.ContextSummary,
		ToolTrace:           payload.ToolTrace,
	}, nil
}

func (r *dialogueSessionRepository) SaveActionOptions(ctx context.Context, options []model.DialogueActionOption) error {
	for _, option := range options {
		row, err := r.dialogueActionOptionRow(option)
		if err != nil {
			return err
		}
		if err := r.dbFor(ctx).Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "id"}},
			DoNothing: true,
		}).Create(&row).Error; err != nil {
			return mapDBError(err, "dialogue action option not found")
		}
	}
	return nil
}

func (r *dialogueSessionRepository) ListActionOptionsByRunID(ctx context.Context, runID string) ([]model.DialogueActionOption, error) {
	var rows []persistencemodels.DialogueActionOption
	if err := r.dbFor(ctx).Where("run_id = ?", runID).Order("created_at asc").Find(&rows).Error; err != nil {
		return nil, mapDBError(err, "dialogue action option not found")
	}
	return r.dialogueActionOptionsFromRows(rows)
}

func (r *dialogueSessionRepository) ListPendingActionOptionsBySessionID(ctx context.Context, sessionID string) ([]model.DialogueActionOption, error) {
	var rows []persistencemodels.DialogueActionOption
	if err := r.dbFor(ctx).Where("session_id = ? AND status IN ?", sessionID, []string{domain.DialogueActionStatusPending, domain.DialogueActionStatusConfirmed}).Order("created_at asc").Find(&rows).Error; err != nil {
		return nil, mapDBError(err, "dialogue action option not found")
	}
	return r.dialogueActionOptionsFromRows(rows)
}

func (r *dialogueSessionRepository) GetActionOptionByID(ctx context.Context, optionID string) (model.DialogueActionOption, error) {
	var row persistencemodels.DialogueActionOption
	if err := r.dbFor(ctx).First(&row, "id = ?", optionID).Error; err != nil {
		return model.DialogueActionOption{}, mapDBError(err, "dialogue action option not found")
	}
	return toDialogueActionOption(row)
}

func (r *dialogueSessionRepository) UpdateActionOption(ctx context.Context, option model.DialogueActionOption) (model.DialogueActionOption, error) {
	payloadJSON, err := encodeJSON(option.Payload)
	if err != nil {
		return model.DialogueActionOption{}, payloadError("dialogue action option", err)
	}
	resultJSON, err := encodeJSON(option.Result)
	if err != nil {
		return model.DialogueActionOption{}, payloadError("dialogue action option", err)
	}
	updates := map[string]any{
		"label":                 option.Label,
		"description":           option.Description,
		"rationale":             option.Rationale,
		"confirmation_required": option.ConfirmationRequired,
		"payload_json":          payloadJSON,
		"status":                option.Status,
		"result_json":           resultJSON,
		"error":                 option.Error,
		"expires_at":            option.ExpiresAt,
		"updated_at":            r.now(),
	}
	if err := r.dbFor(ctx).Model(&persistencemodels.DialogueActionOption{}).Where("id = ?", option.ID).Updates(updates).Error; err != nil {
		return model.DialogueActionOption{}, mapDBError(err, "dialogue action option not found")
	}
	return r.GetActionOptionByID(ctx, option.ID)
}

func (r *dialogueSessionRepository) TryStartActionExecution(ctx context.Context, optionID string) (model.DialogueActionOption, error) {
	now := r.now()
	result := r.dbFor(ctx).Model(&persistencemodels.DialogueActionOption{}).Where("id = ? AND status IN ?", optionID, []string{domain.DialogueActionStatusPending, domain.DialogueActionStatusConfirmed}).Updates(map[string]any{
		"status":     domain.DialogueActionStatusExecuting,
		"updated_at": now,
	})
	if result.Error != nil {
		return model.DialogueActionOption{}, mapDBError(result.Error, "dialogue action option not found")
	}
	if result.RowsAffected == 0 {
		return model.DialogueActionOption{}, pkgerr.Conflict(pkgerr.CodeConflict, "dialogue action option is not executable")
	}
	return r.GetActionOptionByID(ctx, optionID)
}

func (r *dialogueSessionRepository) dialogueSessionFromRow(ctx context.Context, row persistencemodels.DialogueSession) model.DialogueSession {
	session := model.DialogueSession{
		ID:              row.ID,
		ProjectID:       row.ProjectID,
		Title:           row.Title,
		LastUserMessage: row.LastUserMessage,
		Status:          row.Status,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
	}
	var latestRun persistencemodels.DialogueRun
	if err := r.dbFor(ctx).Where("session_id = ?", row.ID).Order("created_at desc").Take(&latestRun).Error; err == nil {
		session.LatestRunID = latestRun.ID
		session.LatestRunStatus = latestRun.Status
		session.LatestRunError = latestRun.Error
	}
	return session
}

func (r *dialogueSessionRepository) dialogueActionOptionRow(option model.DialogueActionOption) (persistencemodels.DialogueActionOption, error) {
	now := r.now()
	if option.ID == "" {
		option.ID = r.nextID("dopt")
	}
	if option.Status == "" {
		option.Status = domain.DialogueActionStatusPending
	}
	if option.CreatedAt.IsZero() {
		option.CreatedAt = now
	}
	option.UpdatedAt = now
	payloadJSON, err := encodeJSON(option.Payload)
	if err != nil {
		return persistencemodels.DialogueActionOption{}, payloadError("dialogue action option", err)
	}
	resultJSON, err := encodeJSON(option.Result)
	if err != nil {
		return persistencemodels.DialogueActionOption{}, payloadError("dialogue action option", err)
	}
	return persistencemodels.DialogueActionOption{
		ID:                   option.ID,
		SessionID:            option.SessionID,
		RunID:                option.RunID,
		ProjectID:            option.ProjectID,
		ActionType:           option.ActionType,
		Label:                option.Label,
		Description:          option.Description,
		Rationale:            option.Rationale,
		ConfirmationRequired: option.ConfirmationRequired,
		PayloadJSON:          payloadJSON,
		Status:               option.Status,
		ResultJSON:           resultJSON,
		Error:                option.Error,
		ExpiresAt:            option.ExpiresAt,
		CreatedAt:            option.CreatedAt,
		UpdatedAt:            option.UpdatedAt,
	}, nil
}

func (r *dialogueSessionRepository) dialogueActionOptionsFromRows(rows []persistencemodels.DialogueActionOption) ([]model.DialogueActionOption, error) {
	items := make([]model.DialogueActionOption, 0, len(rows))
	for _, row := range rows {
		option, err := toDialogueActionOption(row)
		if err != nil {
			return nil, payloadError("dialogue action option", err)
		}
		items = append(items, option)
	}
	return items, nil
}

func toDialogueMessage(row persistencemodels.DialogueMessage) (model.DialogueMessage, error) {
	metadata, err := decodeJSON[map[string]any](row.MetadataJSON)
	if err != nil {
		return model.DialogueMessage{}, err
	}
	return model.DialogueMessage{
		ID:        row.ID,
		SessionID: row.SessionID,
		Role:      row.Role,
		Content:   row.Content,
		Metadata:  metadata,
		CreatedAt: row.CreatedAt,
	}, nil
}

func toDialogueRun(row persistencemodels.DialogueRun) model.DialogueRun {
	return model.DialogueRun{
		RunID:       row.ID,
		SessionID:   row.SessionID,
		ProjectID:   row.ProjectID,
		Status:      row.Status,
		CurrentStep: row.CurrentStep,
		Progress:    row.Progress,
		Error:       row.Error,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}
}

func toDialogueActionOption(row persistencemodels.DialogueActionOption) (model.DialogueActionOption, error) {
	payload, err := decodeJSON[map[string]any](row.PayloadJSON)
	if err != nil {
		return model.DialogueActionOption{}, err
	}
	result, err := decodeJSON[map[string]any](row.ResultJSON)
	if err != nil {
		return model.DialogueActionOption{}, err
	}
	return model.DialogueActionOption{
		ID:                   row.ID,
		SessionID:            row.SessionID,
		RunID:                row.RunID,
		ProjectID:            row.ProjectID,
		ActionType:           row.ActionType,
		Label:                row.Label,
		Description:          row.Description,
		Rationale:            row.Rationale,
		ConfirmationRequired: row.ConfirmationRequired,
		Payload:              payload,
		Status:               row.Status,
		Result:               result,
		Error:                row.Error,
		ExpiresAt:            row.ExpiresAt,
		CreatedAt:            row.CreatedAt,
		UpdatedAt:            row.UpdatedAt,
	}, nil
}

type storySessionRepository struct {
	*container
}

func (r *storySessionRepository) CreateSession(ctx context.Context, projectID string, input model.CreateStorySessionInput) (model.StorySession, error) {
	now := r.now()
	row := persistencemodels.StorySession{
		ID:                         r.nextID("story"),
		ProjectID:                  projectID,
		Title:                      input.Title,
		OpeningSituation:           input.OpeningSituation,
		AuthorIntent:               input.AuthorIntent,
		CurrentPlotVariableSummary: initialStoryVariableSummary(input),
		Status:                     "active",
		CreatedAt:                  now,
		UpdatedAt:                  now,
	}
	if err := r.dbFor(ctx).Create(&row).Error; err != nil {
		return model.StorySession{}, mapDBError(err, "story session not found")
	}
	return model.StorySession{
		ID:                         row.ID,
		ProjectID:                  row.ProjectID,
		Title:                      row.Title,
		OpeningSituation:           row.OpeningSituation,
		AuthorIntent:               row.AuthorIntent,
		Status:                     row.Status,
		CurrentPlotVariableSummary: row.CurrentPlotVariableSummary,
		CreatedAt:                  row.CreatedAt,
		UpdatedAt:                  row.UpdatedAt,
	}, nil
}

func initialStoryVariableSummary(input model.CreateStorySessionInput) string {
	parts := make([]string, 0, 2)
	if text := strings.TrimSpace(input.OpeningSituation); text != "" {
		parts = append(parts, "initial_situation: "+text)
	}
	if text := strings.TrimSpace(input.AuthorIntent); text != "" {
		parts = append(parts, "author_intent: "+text)
	}
	return strings.Join(parts, "\n")
}

func (r *storySessionRepository) ListSessionsByProjectID(ctx context.Context, projectID string, pageInput model.PageInput) (model.ListResult[model.StorySession], error) {
	db := r.dbFor(ctx)
	var total int64
	if err := db.Model(&persistencemodels.StorySession{}).Where("project_id = ?", projectID).Count(&total).Error; err != nil {
		return model.ListResult[model.StorySession]{}, mapDBError(err, "story session not found")
	}
	if pageInput.Page <= 0 {
		pageInput.Page = 1
	}
	if pageInput.PageSize <= 0 {
		pageInput.PageSize = 20
	}
	var rows []persistencemodels.StorySession
	if err := db.Where("project_id = ?", projectID).Order("created_at desc").Limit(pageInput.PageSize).Offset((pageInput.Page - 1) * pageInput.PageSize).Find(&rows).Error; err != nil {
		return model.ListResult[model.StorySession]{}, mapDBError(err, "story session not found")
	}
	items := make([]model.StorySession, 0, len(rows))
	for _, row := range rows {
		items = append(items, model.StorySession{
			ID:                         row.ID,
			ProjectID:                  row.ProjectID,
			Title:                      row.Title,
			OpeningSituation:           row.OpeningSituation,
			AuthorIntent:               row.AuthorIntent,
			LastAuthorMessage:          row.LastAuthorMessage,
			Status:                     row.Status,
			CurrentPlotVariableSummary: row.CurrentPlotVariableSummary,
			CreatedAt:                  row.CreatedAt,
			UpdatedAt:                  row.UpdatedAt,
		})
	}
	return model.ListResult[model.StorySession]{Items: items, Total: int(total)}, nil
}

func (r *storySessionRepository) GetSessionByID(ctx context.Context, sessionID string) (model.StorySession, error) {
	db := r.dbFor(ctx)
	var row persistencemodels.StorySession
	if err := db.First(&row, "id = ?", sessionID).Error; err != nil {
		return model.StorySession{}, mapDBError(err, "story session not found")
	}
	var messages []persistencemodels.StoryMessage
	if err := db.Where("session_id = ?", sessionID).Order("created_at asc").Find(&messages).Error; err != nil {
		return model.StorySession{}, mapDBError(err, "story session not found")
	}
	out := model.StorySession{
		ID:                         row.ID,
		ProjectID:                  row.ProjectID,
		Title:                      row.Title,
		OpeningSituation:           row.OpeningSituation,
		AuthorIntent:               row.AuthorIntent,
		LastAuthorMessage:          row.LastAuthorMessage,
		Status:                     row.Status,
		CurrentPlotVariableSummary: row.CurrentPlotVariableSummary,
		CreatedAt:                  row.CreatedAt,
		UpdatedAt:                  row.UpdatedAt,
		Messages:                   make([]model.ConversationMessage, 0, len(messages)),
	}
	for _, msg := range messages {
		out.Messages = append(out.Messages, toMessage(msg.ID, msg.SessionID, msg.Role, msg.Content, msg.CreatedAt))
	}
	return out, nil
}

func (r *storySessionRepository) UpdateSession(ctx context.Context, session model.StorySession) (model.StorySession, error) {
	if err := r.dbFor(ctx).Model(&persistencemodels.StorySession{}).Where("id = ?", session.ID).Updates(map[string]any{
		"title":                         session.Title,
		"last_author_message":           session.LastAuthorMessage,
		"status":                        session.Status,
		"current_plot_variable_summary": session.CurrentPlotVariableSummary,
		"updated_at":                    r.now(),
	}).Error; err != nil {
		return model.StorySession{}, mapDBError(err, "story session not found")
	}
	return r.GetSessionByID(ctx, session.ID)
}

func (r *storySessionRepository) DeleteSession(ctx context.Context, sessionID string) error {
	err := r.dbFor(ctx).Transaction(func(tx *gorm.DB) error {
		var session persistencemodels.StorySession
		if err := tx.First(&session, "id = ?", sessionID).Error; err != nil {
			return err
		}
		var cutCount int64
		if err := tx.Model(&persistencemodels.StoryRun{}).Where("session_id = ? AND (status = ? OR cut_at IS NOT NULL)", sessionID, domain.RunStatusCut).Count(&cutCount).Error; err != nil {
			return err
		}
		if cutCount > 0 {
			return pkgerr.Conflict(pkgerr.CodeConflict, "story session has cut chapters and cannot be deleted")
		}

		var runIDs []string
		if err := tx.Model(&persistencemodels.StoryRun{}).Where("session_id = ?", sessionID).Pluck("id", &runIDs).Error; err != nil {
			return err
		}
		var branchIDs []string
		if err := tx.Model(&persistencemodels.StoryBranch{}).Where("session_id = ?", sessionID).Pluck("id", &branchIDs).Error; err != nil {
			return err
		}
		if len(branchIDs) > 0 {
			if err := tx.Where("branch_id IN ?", branchIDs).Delete(&persistencemodels.ChapterEventSpan{}).Error; err != nil {
				return err
			}
			if err := tx.Where("branch_id IN ?", branchIDs).Delete(&persistencemodels.StorySnapshot{}).Error; err != nil {
				return err
			}
			if err := tx.Where("branch_id IN ?", branchIDs).Delete(&persistencemodels.StoryEvent{}).Error; err != nil {
				return err
			}
			if err := tx.Where("id IN ?", branchIDs).Delete(&persistencemodels.StoryBranch{}).Error; err != nil {
				return err
			}
		}
		if len(runIDs) > 0 {
			if err := tx.Where("run_kind = ? AND run_id IN ?", "story", runIDs).Delete(&persistencemodels.RunEvent{}).Error; err != nil {
				return err
			}
			if err := tx.Where("run_id IN ?", runIDs).Delete(&persistencemodels.StoryRunResult{}).Error; err != nil {
				return err
			}
			if err := tx.Where("id IN ?", runIDs).Delete(&persistencemodels.StoryRun{}).Error; err != nil {
				return err
			}
		}
		if err := tx.Where("session_id = ?", sessionID).Delete(&persistencemodels.StoryMessage{}).Error; err != nil {
			return err
		}
		return tx.Delete(&session).Error
	})
	if err != nil {
		if isConflict(err) {
			return err
		}
		return mapDBError(err, "story session not found")
	}
	return nil
}

func (r *storySessionRepository) AppendMessage(ctx context.Context, sessionID string, role string, content string) (model.ConversationMessage, error) {
	row := persistencemodels.StoryMessage{
		ID:        r.nextID("tmsg"),
		SessionID: sessionID,
		Role:      role,
		Content:   content,
		CreatedAt: r.now(),
	}
	if err := r.dbFor(ctx).Create(&row).Error; err != nil {
		return model.ConversationMessage{}, mapDBError(err, "story session not found")
	}
	return toMessage(row.ID, row.SessionID, row.Role, row.Content, row.CreatedAt), nil
}

func (r *storySessionRepository) CreateRun(ctx context.Context, sessionID string, input model.AdvanceStorySessionInput) (model.StoryRun, error) {
	session, err := r.GetSessionByID(ctx, sessionID)
	if err != nil {
		return model.StoryRun{}, err
	}
	row := persistencemodels.StoryRun{
		ID:          r.nextID("run"),
		SessionID:   sessionID,
		ProjectID:   session.ProjectID,
		BranchID:    input.BranchID,
		BaseEventID: input.BaseEventID,
		Status:      "queued",
		CurrentStep: "loading_state",
		Progress:    0,
		CreatedAt:   r.now(),
		UpdatedAt:   r.now(),
	}
	if err := r.dbFor(ctx).Create(&row).Error; err != nil {
		return model.StoryRun{}, mapDBError(err, "story run not found")
	}
	return model.StoryRun{
		RunID:         row.ID,
		SessionID:     row.SessionID,
		ProjectID:     row.ProjectID,
		BranchID:      row.BranchID,
		BaseEventID:   row.BaseEventID,
		HeadEventID:   row.HeadEventID,
		Status:        row.Status,
		CurrentStep:   row.CurrentStep,
		Progress:      row.Progress,
		Error:         row.Error,
		StopRequested: row.StopRequested,
		CreatedAt:     row.CreatedAt,
		UpdatedAt:     row.UpdatedAt,
	}, nil
}

func (r *storySessionRepository) HasActiveRunByBranch(ctx context.Context, branchID string) (bool, error) {
	if branchID == "" {
		return false, nil
	}
	var count int64
	if err := r.dbFor(ctx).Model(&persistencemodels.StoryRun{}).Where("branch_id = ? AND status IN ?", branchID, activeStoryRunStatuses()).Count(&count).Error; err != nil {
		return false, mapDBError(err, "story run not found")
	}
	return count > 0, nil
}

func (r *storySessionRepository) GetRunByID(ctx context.Context, runID string) (model.StoryRun, error) {
	var row persistencemodels.StoryRun
	if err := r.dbFor(ctx).First(&row, "id = ?", runID).Error; err != nil {
		return model.StoryRun{}, mapDBError(err, "story run not found")
	}
	return model.StoryRun{
		RunID:         row.ID,
		SessionID:     row.SessionID,
		ProjectID:     row.ProjectID,
		BranchID:      row.BranchID,
		BaseEventID:   row.BaseEventID,
		HeadEventID:   row.HeadEventID,
		Status:        row.Status,
		CurrentStep:   row.CurrentStep,
		Progress:      row.Progress,
		Error:         row.Error,
		StopRequested: row.StopRequested,
		CutAt:         row.CutAt,
		CreatedAt:     row.CreatedAt,
		UpdatedAt:     row.UpdatedAt,
	}, nil
}

func (r *storySessionRepository) GetRunResultByID(ctx context.Context, runID string) (model.StoryRunResult, error) {
	var row persistencemodels.StoryRunResult
	if err := r.dbFor(ctx).First(&row, "run_id = ?", runID).Error; err != nil {
		return model.StoryRunResult{}, mapDBError(err, "story run result not found")
	}
	payload, err := decodeJSON[storyRunResultPayload](row.PayloadJSON)
	if err != nil {
		return model.StoryRunResult{}, payloadError("story run result", err)
	}
	return model.StoryRunResult{
		RunID:                  row.RunID,
		SessionID:              row.SessionID,
		Status:                 row.Status,
		BranchID:               payload.BranchID,
		BaseEventID:            payload.BaseEventID,
		HeadEventID:            payload.HeadEventID,
		PlotVariable:           payload.PlotVariable,
		EventPlan:              payload.EventPlan,
		Turns:                  payload.Turns,
		SceneSummary:           payload.SceneSummary,
		InteractionAnalysis:    payload.InteractionAnalysis,
		InteractionTranscripts: payload.InteractionTranscripts,
		Draft:                  payload.Draft,
		Review:                 payload.Review,
		MemoryPatch:            payload.MemoryPatch,
		Events:                 payload.Events,
		CompletedActions:       payload.CompletedActions,
		SupersededActions:      payload.SupersededActions,
		CollisionAt:            payload.CollisionAt,
	}, nil
}

func (r *storySessionRepository) SaveRunResult(ctx context.Context, runID string, result model.StoryRunResult) error {
	payloadJSON, err := encodeJSON(storyRunResultPayload{
		BranchID:               result.BranchID,
		BaseEventID:            result.BaseEventID,
		HeadEventID:            result.HeadEventID,
		PlotVariable:           result.PlotVariable,
		EventPlan:              result.EventPlan,
		Turns:                  result.Turns,
		SceneSummary:           result.SceneSummary,
		InteractionAnalysis:    result.InteractionAnalysis,
		InteractionTranscripts: result.InteractionTranscripts,
		Draft:                  result.Draft,
		Review:                 result.Review,
		MemoryPatch:            result.MemoryPatch,
		Events:                 result.Events,
		CompletedActions:       result.CompletedActions,
		SupersededActions:      result.SupersededActions,
		CollisionAt:            result.CollisionAt,
	})
	if err != nil {
		return payloadError("story run result", err)
	}
	now := r.now()
	row := persistencemodels.StoryRunResult{
		ID:          r.nextID("tresult"),
		RunID:       runID,
		SessionID:   result.SessionID,
		Status:      result.Status,
		PayloadJSON: payloadJSON,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	updates := map[string]any{
		"status":        result.Status,
		"current_step":  "completed",
		"progress":      100,
		"error":         "",
		"head_event_id": result.HeadEventID,
		"updated_at":    now,
	}
	applyRunLeaseUpdates(ctx, updates, result.Status, now)
	db, _ := runLeaseScopedDB(ctx, r.dbFor(ctx).Model(&persistencemodels.StoryRun{}).Where("id = ?", runID))
	if err := runLeaseMutationError(ctx, db.Updates(updates), "story run not found"); err != nil {
		return err
	}
	return mapDBError(r.dbFor(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "run_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"session_id", "status", "payload_json", "updated_at"}),
	}).Create(&row).Error, "story run result not found")
}

func (r *storySessionRepository) UpdateRunStatus(ctx context.Context, runID string, status string, currentStep string, progress int, errorMessage ...string) error {
	now := r.now()
	updates := map[string]any{
		"status":       status,
		"current_step": currentStep,
		"progress":     progress,
		"updated_at":   now,
	}
	if len(errorMessage) > 0 {
		updates["error"] = errorMessage[0]
	}
	applyRunLeaseUpdates(ctx, updates, status, now)
	db, _ := runLeaseScopedDB(ctx, r.dbFor(ctx).Model(&persistencemodels.StoryRun{}).Where("id = ?", runID))
	return runLeaseMutationError(ctx, db.Updates(updates), "story run not found")
}

func (r *storySessionRepository) UpdateRunHeartbeat(ctx context.Context, runID string) error {
	now := r.now()
	updates := map[string]any{"updated_at": now}
	applyRunLeaseUpdates(ctx, updates, domain.RunStatusLoadingState, now)
	db, _ := runLeaseScopedDB(ctx, r.dbFor(ctx).Model(&persistencemodels.StoryRun{}).Where("id = ?", runID))
	return runLeaseMutationError(ctx, db.Updates(updates), "story run not found")
}

func (r *storySessionRepository) UpdateRunHead(ctx context.Context, runID string, headEventID string) error {
	now := r.now()
	updates := map[string]any{
		"head_event_id": headEventID,
		"updated_at":    now,
	}
	applyRunLeaseUpdates(ctx, updates, domain.RunStatusLoadingState, now)
	db, _ := runLeaseScopedDB(ctx, r.dbFor(ctx).Model(&persistencemodels.StoryRun{}).Where("id = ?", runID))
	return runLeaseMutationError(ctx, db.Updates(updates), "story run not found")
}

func (r *storySessionRepository) RequestRunStop(ctx context.Context, runID string) error {
	return mapDBError(r.dbFor(ctx).Transaction(func(tx *gorm.DB) error {
		var run persistencemodels.StoryRun
		if err := tx.First(&run, "id = ?", runID).Error; err != nil {
			return err
		}
		now := r.now()
		updates := map[string]any{"stop_requested": true, "updated_at": now}
		if run.Status == domain.RunStatusQueued {
			updates["status"] = domain.RunStatusCancelled
			updates["current_step"] = domain.RunStatusCancelled
			updates["progress"] = 100
			updates["error"] = ""
		}
		return tx.Model(&persistencemodels.StoryRun{}).Where("id = ?", runID).Updates(updates).Error
	}), "story run not found")
}

func (r *storySessionRepository) MarkCut(ctx context.Context, runID string) error {
	now := r.now()
	updates := map[string]any{
		"status":       "cut",
		"current_step": "cut",
		"progress":     100,
		"error":        "",
		"cut_at":       now,
		"updated_at":   now,
	}
	applyRunLeaseUpdates(ctx, updates, domain.RunStatusCut, now)
	return mapDBError(r.dbFor(ctx).Model(&persistencemodels.StoryRun{}).Where("id = ?", runID).Updates(updates).Error, "story run not found")
}

type auditRepository struct {
	*container
}

func (r *auditRepository) AppendRunEvent(ctx context.Context, event model.RunEvent) (model.RunEvent, error) {
	if event.ID == "" {
		event.ID = r.nextID("event")
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = r.now()
	}
	if event.Sequence != 0 {
		row, err := runEventRowFromModel(event, r.now(), event.ID)
		if err != nil {
			return model.RunEvent{}, payloadError("run event", err)
		}
		if err := r.dbFor(ctx).Create(&row).Error; err != nil {
			return model.RunEvent{}, mapDBError(err, "run event not found")
		}
		return toRunEvent(row)
	}

	var saved model.RunEvent
	err := r.dbFor(ctx).Transaction(func(tx *gorm.DB) error {
		sequence, err := allocateRunEventSequence(ctx, tx, event.RunKind, event.RunID, event.CreatedAt)
		if err != nil {
			return err
		}
		event.Sequence = sequence
		row, err := runEventRowFromModel(event, r.now(), event.ID)
		if err != nil {
			return payloadError("run event", err)
		}
		if err := tx.WithContext(ctx).Create(&row).Error; err != nil {
			return mapDBError(err, "run event not found")
		}
		saved, err = toRunEvent(row)
		return err
	})
	if err != nil {
		return model.RunEvent{}, err
	}
	return saved, nil
}

func allocateRunEventSequence(ctx context.Context, db *gorm.DB, runKind string, runID string, updatedAt time.Time) (int, error) {
	current, err := currentSequence(ctx, db, runKind, runID)
	if err != nil {
		return 0, mapDBError(err, "run event not found")
	}
	counter := persistencemodels.RunEventCounter{
		RunKind:      runKind,
		RunID:        runID,
		NextSequence: current + 1,
		UpdatedAt:    updatedAt,
	}
	err = db.WithContext(ctx).Clauses(
		clause.OnConflict{
			Columns: []clause.Column{{Name: "run_kind"}, {Name: "run_id"}},
			DoUpdates: clause.Assignments(map[string]any{
				"next_sequence": gorm.Expr("run_event_counters.next_sequence + 1"),
				"updated_at":    updatedAt,
			}),
		},
		clause.Returning{Columns: []clause.Column{{Name: "next_sequence"}}},
	).Create(&counter).Error
	if err != nil {
		return 0, mapDBError(err, "run event counter not found")
	}
	return counter.NextSequence, nil
}

func (r *auditRepository) ListRunEvents(ctx context.Context, runKind string, runID string) ([]model.RunEvent, error) {
	return r.ListRunEventsAfter(ctx, runKind, runID, 0)
}

func (r *auditRepository) ListRunEventsAfter(ctx context.Context, runKind string, runID string, afterSequence int) ([]model.RunEvent, error) {
	var rows []persistencemodels.RunEvent
	if err := r.dbFor(ctx).Where("run_kind = ? AND run_id = ? AND sequence > ?", runKind, runID, afterSequence).Order("sequence asc").Find(&rows).Error; err != nil {
		return nil, mapDBError(err, "run event not found")
	}
	items := make([]model.RunEvent, 0, len(rows))
	for _, row := range rows {
		event, err := toRunEvent(row)
		if err != nil {
			return nil, payloadError("run event", err)
		}
		items = append(items, event)
	}
	return items, nil
}

func (r *auditRepository) CreateRevision(ctx context.Context, revision model.StateRevision) (model.StateRevision, error) {
	if revision.ID == "" {
		revision.ID = r.nextID("revision")
	}
	if revision.CreatedAt.IsZero() {
		revision.CreatedAt = r.now()
	}
	row, err := revisionRowFromModel(revision, r.now(), revision.ID)
	if err != nil {
		return model.StateRevision{}, payloadError("state revision", err)
	}
	if err := r.dbFor(ctx).Create(&row).Error; err != nil {
		return model.StateRevision{}, mapDBError(err, "state revision not found")
	}
	return toRevision(row)
}

func currentSequence(ctx context.Context, db *gorm.DB, runKind, runID string) (int, error) {
	var row persistencemodels.RunEvent
	if err := db.WithContext(ctx).Where("run_kind = ? AND run_id = ?", runKind, runID).Order("sequence desc").Take(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, nil
		}
		return 0, err
	}
	return row.Sequence, nil
}
