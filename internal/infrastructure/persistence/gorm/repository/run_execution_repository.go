package repository

import (
	"context"
	"time"

	"github.com/fishimei/NovelOS/internal/application/model"
	"github.com/fishimei/NovelOS/internal/application/port"
	"github.com/fishimei/NovelOS/internal/domain"
	persistencemodels "github.com/fishimei/NovelOS/internal/infrastructure/persistence/gorm/models"
)

func (r Repositories) ListRunnableRuns(ctx context.Context, staleBefore time.Time, limit int) ([]model.RunExecutionWork, error) {
	if limit <= 0 {
		limit = 10
	}
	works := make([]model.RunExecutionWork, 0, limit)
	setupLimit := limit
	var setupRows []persistencemodels.SetupRun
	if err := r.SetupSessions.(*setupSessionRepository).dbFor(ctx).
		Where("status = ? OR (status = ? AND updated_at < ?)", domain.RunStatusQueued, domain.RunStatusLoadingState, staleBefore).
		Order("updated_at asc").Limit(setupLimit).Find(&setupRows).Error; err != nil {
		return nil, mapDBError(err, "setup run not found")
	}
	for _, row := range setupRows {
		works = append(works, model.RunExecutionWork{RunKind: port.RunKindSetup, RunID: row.ID, SessionID: row.SessionID, ProjectID: row.ProjectID, Status: row.Status, UpdatedAt: row.UpdatedAt})
	}
	remaining := limit - len(works)
	if remaining <= 0 {
		return works, nil
	}
	var storyRows []persistencemodels.StoryRun
	if err := r.StorySessions.(*storySessionRepository).dbFor(ctx).
		Where("status = ? OR (status = ? AND updated_at < ?)", domain.RunStatusQueued, domain.RunStatusLoadingState, staleBefore).
		Order("updated_at asc").Limit(remaining).Find(&storyRows).Error; err != nil {
		return nil, mapDBError(err, "story run not found")
	}
	for _, row := range storyRows {
		works = append(works, model.RunExecutionWork{RunKind: port.RunKindStory, RunID: row.ID, SessionID: row.SessionID, ProjectID: row.ProjectID, Status: row.Status, UpdatedAt: row.UpdatedAt})
	}
	remaining = limit - len(works)
	if remaining <= 0 {
		return works, nil
	}
	var dialogueRows []persistencemodels.DialogueRun
	if err := r.DialogueSessions.(*dialogueSessionRepository).dbFor(ctx).
		Where("status = ? OR (status = ? AND updated_at < ?)", domain.RunStatusQueued, domain.RunStatusLoadingState, staleBefore).
		Order("updated_at asc").Limit(remaining).Find(&dialogueRows).Error; err != nil {
		return nil, mapDBError(err, "dialogue run not found")
	}
	for _, row := range dialogueRows {
		works = append(works, model.RunExecutionWork{RunKind: port.RunKindDialogue, RunID: row.ID, SessionID: row.SessionID, ProjectID: row.ProjectID, Status: row.Status, UpdatedAt: row.UpdatedAt})
	}
	return works, nil
}

func (r Repositories) ClaimRun(ctx context.Context, work model.RunExecutionWork, staleBefore time.Time) (bool, error) {
	now := r.SetupSessions.(*setupSessionRepository).now()
	updates := map[string]any{"status": domain.RunStatusLoadingState, "current_step": domain.RunStatusLoadingState, "updated_at": now}
	switch work.RunKind {
	case port.RunKindSetup:
		result := r.SetupSessions.(*setupSessionRepository).dbFor(ctx).Model(&persistencemodels.SetupRun{}).
			Where("id = ? AND (status = ? OR (status = ? AND updated_at < ?))", work.RunID, domain.RunStatusQueued, domain.RunStatusLoadingState, staleBefore).
			Updates(updates)
		if result.Error != nil {
			return false, mapDBError(result.Error, "setup run not found")
		}
		return result.RowsAffected > 0, nil
	case port.RunKindStory:
		result := r.StorySessions.(*storySessionRepository).dbFor(ctx).Model(&persistencemodels.StoryRun{}).
			Where("id = ? AND (status = ? OR (status = ? AND updated_at < ?))", work.RunID, domain.RunStatusQueued, domain.RunStatusLoadingState, staleBefore).
			Updates(updates)
		if result.Error != nil {
			return false, mapDBError(result.Error, "story run not found")
		}
		return result.RowsAffected > 0, nil
	case port.RunKindDialogue:
		result := r.DialogueSessions.(*dialogueSessionRepository).dbFor(ctx).Model(&persistencemodels.DialogueRun{}).
			Where("id = ? AND (status = ? OR (status = ? AND updated_at < ?))", work.RunID, domain.RunStatusQueued, domain.RunStatusLoadingState, staleBefore).
			Updates(updates)
		if result.Error != nil {
			return false, mapDBError(result.Error, "dialogue run not found")
		}
		return result.RowsAffected > 0, nil
	default:
		return false, nil
	}
}
