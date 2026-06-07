package repository

import (
	"context"

	"github.com/fishimei/NovelOS/internal/application/model"
	persistencemodels "github.com/fishimei/NovelOS/internal/infrastructure/persistence/gorm/models"
	"gorm.io/gorm/clause"
)

type storyAutoRunRepository struct {
	*container
}

func (r *storyAutoRunRepository) Upsert(ctx context.Context, state model.StoryAutoRunState) (model.StoryAutoRunState, error) {
	now := r.now()
	if state.ID == "" {
		state.ID = r.nextID("sauto")
	}
	if state.CreatedAt.IsZero() {
		state.CreatedAt = now
	}
	state.UpdatedAt = now
	row := storyAutoRunStateToRow(state)
	if err := r.dbFor(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "session_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"project_id", "branch_id", "base_event_id", "current_run_id", "status", "stop_requested",
			"iterations", "last_error", "tick_delay_seconds", "updated_at", "last_run_started_at", "last_completed_at",
		}),
	}).Create(&row).Error; err != nil {
		return model.StoryAutoRunState{}, mapDBError(err, "story auto run state not found")
	}
	return r.GetBySessionID(ctx, state.SessionID)
}

func (r *storyAutoRunRepository) GetBySessionID(ctx context.Context, sessionID string) (model.StoryAutoRunState, error) {
	var row persistencemodels.StoryAutoRunState
	if err := r.dbFor(ctx).First(&row, "session_id = ?", sessionID).Error; err != nil {
		return model.StoryAutoRunState{}, mapDBError(err, "story auto run state not found")
	}
	return storyAutoRunStateFromRow(row), nil
}

func (r *storyAutoRunRepository) ListResumable(ctx context.Context) ([]model.StoryAutoRunState, error) {
	var rows []persistencemodels.StoryAutoRunState
	if err := r.dbFor(ctx).Where("status IN ?", []string{"running", "stopping"}).Find(&rows).Error; err != nil {
		return nil, mapDBError(err, "story auto run state not found")
	}
	out := make([]model.StoryAutoRunState, 0, len(rows))
	for _, row := range rows {
		out = append(out, storyAutoRunStateFromRow(row))
	}
	return out, nil
}

func (r *storyAutoRunRepository) Update(ctx context.Context, state model.StoryAutoRunState) (model.StoryAutoRunState, error) {
	state.UpdatedAt = r.now()
	if err := r.dbFor(ctx).Model(&persistencemodels.StoryAutoRunState{}).Where("id = ?", state.ID).Updates(map[string]any{
		"project_id":          state.ProjectID,
		"session_id":          state.SessionID,
		"branch_id":           state.BranchID,
		"base_event_id":       state.BaseEventID,
		"current_run_id":      state.CurrentRunID,
		"status":              state.Status,
		"stop_requested":      state.StopRequested,
		"iterations":          state.Iterations,
		"last_error":          state.LastError,
		"tick_delay_seconds":  state.TickDelaySeconds,
		"updated_at":          state.UpdatedAt,
		"last_run_started_at": state.LastRunStartedAt,
		"last_completed_at":   state.LastCompletedAt,
	}).Error; err != nil {
		return model.StoryAutoRunState{}, mapDBError(err, "story auto run state not found")
	}
	return r.GetBySessionID(ctx, state.SessionID)
}

func storyAutoRunStateToRow(state model.StoryAutoRunState) persistencemodels.StoryAutoRunState {
	return persistencemodels.StoryAutoRunState{
		ID:               state.ID,
		ProjectID:        state.ProjectID,
		SessionID:        state.SessionID,
		BranchID:         state.BranchID,
		BaseEventID:      state.BaseEventID,
		CurrentRunID:     state.CurrentRunID,
		Status:           state.Status,
		StopRequested:    state.StopRequested,
		Iterations:       state.Iterations,
		LastError:        state.LastError,
		TickDelaySeconds: state.TickDelaySeconds,
		CreatedAt:        state.CreatedAt,
		UpdatedAt:        state.UpdatedAt,
		LastRunStartedAt: state.LastRunStartedAt,
		LastCompletedAt:  state.LastCompletedAt,
	}
}

func storyAutoRunStateFromRow(row persistencemodels.StoryAutoRunState) model.StoryAutoRunState {
	return model.StoryAutoRunState{
		ID:               row.ID,
		ProjectID:        row.ProjectID,
		SessionID:        row.SessionID,
		BranchID:         row.BranchID,
		BaseEventID:      row.BaseEventID,
		CurrentRunID:     row.CurrentRunID,
		Status:           row.Status,
		StopRequested:    row.StopRequested,
		Iterations:       row.Iterations,
		LastError:        row.LastError,
		TickDelaySeconds: row.TickDelaySeconds,
		CreatedAt:        row.CreatedAt,
		UpdatedAt:        row.UpdatedAt,
		LastRunStartedAt: row.LastRunStartedAt,
		LastCompletedAt:  row.LastCompletedAt,
	}
}
