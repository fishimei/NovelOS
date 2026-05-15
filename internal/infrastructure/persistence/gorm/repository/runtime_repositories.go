package repository

import (
	"context"
	"errors"

	"github.com/fishimei/NovelOS/internal/application/model"
	persistencemodels "github.com/fishimei/NovelOS/internal/infrastructure/persistence/gorm/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type setupRunResultPayload struct {
	SetupDraft model.SetupDraft `json:"setup_draft"`
}

type storyRunResultPayload struct {
	PlotVariable model.PlotVariable `json:"plot_variable"`
	Draft        model.Draft        `json:"draft"`
	Review       model.ReviewReport `json:"review"`
	MemoryPatch  model.MemoryPatch  `json:"memory_patch"`
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
	return model.SetupSession{
		ID:        row.ID,
		ProjectID: row.ProjectID,
		SeedIdea:  row.SeedIdea,
		Status:    row.Status,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}, nil
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
		items = append(items, model.SetupSession{
			ID:              row.ID,
			ProjectID:       row.ProjectID,
			SeedIdea:        row.SeedIdea,
			LastUserMessage: row.LastUserMessage,
			Status:          row.Status,
			CreatedAt:       row.CreatedAt,
			UpdatedAt:       row.UpdatedAt,
		})
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
	out := model.SetupSession{
		ID:              row.ID,
		ProjectID:       row.ProjectID,
		SeedIdea:        row.SeedIdea,
		LastUserMessage: row.LastUserMessage,
		Status:          row.Status,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
		Messages:        make([]model.ConversationMessage, 0, len(messages)),
	}
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
	if err := r.dbFor(ctx).Model(&persistencemodels.SetupRun{}).Where("id = ?", runID).Updates(map[string]any{
		"status":       result.Status,
		"current_step": "completed",
		"progress":     100,
		"updated_at":   now,
	}).Error; err != nil {
		return mapDBError(err, "setup run not found")
	}
	return mapDBError(r.dbFor(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "run_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"session_id", "status", "payload_json", "updated_at"}),
	}).Create(&row).Error, "setup run result not found")
}

type storySessionRepository struct {
	*container
}

func (r *storySessionRepository) CreateSession(ctx context.Context, projectID string, input model.CreateStorySessionInput) (model.StorySession, error) {
	now := r.now()
	row := persistencemodels.StorySession{
		ID:               r.nextID("story"),
		ProjectID:        projectID,
		Title:            input.Title,
		OpeningSituation: input.OpeningSituation,
		AuthorIntent:     input.AuthorIntent,
		Status:           "active",
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if err := r.dbFor(ctx).Create(&row).Error; err != nil {
		return model.StorySession{}, mapDBError(err, "story session not found")
	}
	return model.StorySession{
		ID:               row.ID,
		ProjectID:        row.ProjectID,
		Title:            row.Title,
		OpeningSituation: row.OpeningSituation,
		AuthorIntent:     row.AuthorIntent,
		Status:           row.Status,
		CreatedAt:        row.CreatedAt,
		UpdatedAt:        row.UpdatedAt,
	}, nil
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
		"last_author_message":           session.LastAuthorMessage,
		"status":                        session.Status,
		"current_plot_variable_summary": session.CurrentPlotVariableSummary,
		"updated_at":                    r.now(),
	}).Error; err != nil {
		return model.StorySession{}, mapDBError(err, "story session not found")
	}
	return r.GetSessionByID(ctx, session.ID)
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
		RunID:       row.ID,
		SessionID:   row.SessionID,
		ProjectID:   row.ProjectID,
		Status:      row.Status,
		CurrentStep: row.CurrentStep,
		Progress:    row.Progress,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}, nil
}

func (r *storySessionRepository) GetRunByID(ctx context.Context, runID string) (model.StoryRun, error) {
	var row persistencemodels.StoryRun
	if err := r.dbFor(ctx).First(&row, "id = ?", runID).Error; err != nil {
		return model.StoryRun{}, mapDBError(err, "story run not found")
	}
	return model.StoryRun{
		RunID:       row.ID,
		SessionID:   row.SessionID,
		ProjectID:   row.ProjectID,
		Status:      row.Status,
		CurrentStep: row.CurrentStep,
		Progress:    row.Progress,
		CommittedAt: row.CommittedAt,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
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
		RunID:        row.RunID,
		SessionID:    row.SessionID,
		Status:       row.Status,
		PlotVariable: payload.PlotVariable,
		Draft:        payload.Draft,
		Review:       payload.Review,
		MemoryPatch:  payload.MemoryPatch,
	}, nil
}

func (r *storySessionRepository) SaveRunResult(ctx context.Context, runID string, result model.StoryRunResult) error {
	payloadJSON, err := encodeJSON(storyRunResultPayload{
		PlotVariable: result.PlotVariable,
		Draft:        result.Draft,
		Review:       result.Review,
		MemoryPatch:  result.MemoryPatch,
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
	if err := r.dbFor(ctx).Model(&persistencemodels.StoryRun{}).Where("id = ?", runID).Updates(map[string]any{
		"status":       result.Status,
		"current_step": "review_required",
		"progress":     100,
		"updated_at":   now,
	}).Error; err != nil {
		return mapDBError(err, "story run not found")
	}
	return mapDBError(r.dbFor(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "run_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"session_id", "status", "payload_json", "updated_at"}),
	}).Create(&row).Error, "story run result not found")
}

func (r *storySessionRepository) UpdateRunStatus(ctx context.Context, runID string, status string, currentStep string, progress int) error {
	return mapDBError(r.dbFor(ctx).Model(&persistencemodels.StoryRun{}).Where("id = ?", runID).Updates(map[string]any{
		"status":       status,
		"current_step": currentStep,
		"progress":     progress,
		"updated_at":   r.now(),
	}).Error, "story run not found")
}

func (r *storySessionRepository) MarkCommitted(ctx context.Context, runID string) error {
	now := r.now()
	return mapDBError(r.dbFor(ctx).Model(&persistencemodels.StoryRun{}).Where("id = ?", runID).Updates(map[string]any{
		"status":       "committed",
		"current_step": "committed",
		"progress":     100,
		"committed_at": now,
		"updated_at":   now,
	}).Error, "story run not found")
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
	row, err := runEventRowFromModel(event, r.now(), event.ID)
	if err != nil {
		return model.RunEvent{}, payloadError("run event", err)
	}
	if err := r.dbFor(ctx).Create(&row).Error; err != nil {
		return model.RunEvent{}, mapDBError(err, "run event not found")
	}
	return toRunEvent(row)
}

func (r *auditRepository) ListRunEvents(ctx context.Context, runKind string, runID string) ([]model.RunEvent, error) {
	var rows []persistencemodels.RunEvent
	if err := r.dbFor(ctx).Where("run_kind = ? AND run_id = ?", runKind, runID).Order("sequence asc").Find(&rows).Error; err != nil {
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
