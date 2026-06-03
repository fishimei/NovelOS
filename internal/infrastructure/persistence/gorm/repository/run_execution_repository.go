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
	now := r.SetupSessions.(*setupSessionRepository).now()
	works := make([]model.RunExecutionWork, 0, limit)
	setupLimit := limit
	var setupRows []persistencemodels.SetupRun
	if err := r.SetupSessions.(*setupSessionRepository).dbFor(ctx).
		Where(runnableRunWhereClause(), domain.RunStatusQueued, activeRunStatuses(), domain.RunStatusQueued, staleBefore, now).
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
		Where("("+runnableRunWhereClause()+") AND stop_requested = ?", domain.RunStatusQueued, activeRunStatuses(), domain.RunStatusQueued, staleBefore, now, false).
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
		Where(runnableRunWhereClause(), domain.RunStatusQueued, activeRunStatuses(), domain.RunStatusQueued, staleBefore, now).
		Order("updated_at asc").Limit(remaining).Find(&dialogueRows).Error; err != nil {
		return nil, mapDBError(err, "dialogue run not found")
	}
	for _, row := range dialogueRows {
		works = append(works, model.RunExecutionWork{RunKind: port.RunKindDialogue, RunID: row.ID, SessionID: row.SessionID, ProjectID: row.ProjectID, Status: row.Status, UpdatedAt: row.UpdatedAt})
	}
	return works, nil
}

func (r Repositories) ClaimRun(ctx context.Context, work model.RunExecutionWork, lease port.RunLease, staleBefore time.Time) (bool, error) {
	if !lease.Valid() {
		lease = port.RunLease{Owner: "legacy:" + work.RunKind + ":" + work.RunID, Duration: 10 * time.Minute}
	}
	now := r.SetupSessions.(*setupSessionRepository).now()
	updates := map[string]any{
		"status":           domain.RunStatusLoadingState,
		"current_step":     domain.RunStatusLoadingState,
		"updated_at":       now,
		"lease_owner":      lease.Owner,
		"lease_expires_at": lease.ExpiresAt(now),
	}
	switch work.RunKind {
	case port.RunKindSetup:
		result := r.SetupSessions.(*setupSessionRepository).dbFor(ctx).Model(&persistencemodels.SetupRun{}).
			Where("id = ? AND ("+runnableRunWhereClause()+")", work.RunID, domain.RunStatusQueued, activeRunStatuses(), domain.RunStatusQueued, staleBefore, now).
			Updates(updates)
		if result.Error != nil {
			return false, mapDBError(result.Error, "setup run not found")
		}
		return result.RowsAffected > 0, nil
	case port.RunKindStory:
		db := r.StorySessions.(*storySessionRepository).dbFor(ctx)
		var run persistencemodels.StoryRun
		if err := db.First(&run, "id = ?", work.RunID).Error; err != nil {
			return false, mapDBError(err, "story run not found")
		}
		if run.StopRequested || !runIsRunnable(run.Status, run.LeaseOwner, run.LeaseExpiresAt, run.UpdatedAt, staleBefore, now) {
			return false, nil
		}
		var otherActive int64
		if err := db.Model(&persistencemodels.StoryRun{}).Where("branch_id = ? AND id <> ? AND status IN ?", run.BranchID, run.ID, activeStoryRunStatuses()).Count(&otherActive).Error; err != nil {
			return false, mapDBError(err, "story run not found")
		}
		if otherActive > 0 {
			return false, nil
		}
		result := db.Model(&persistencemodels.StoryRun{}).
			Where("id = ? AND ("+runnableRunWhereClause()+") AND stop_requested = ?", work.RunID, domain.RunStatusQueued, activeRunStatuses(), domain.RunStatusQueued, staleBefore, now, false).
			Updates(updates)
		if result.Error != nil {
			return false, mapDBError(result.Error, "story run not found")
		}
		return result.RowsAffected > 0, nil
	case port.RunKindDialogue:
		result := r.DialogueSessions.(*dialogueSessionRepository).dbFor(ctx).Model(&persistencemodels.DialogueRun{}).
			Where("id = ? AND ("+runnableRunWhereClause()+")", work.RunID, domain.RunStatusQueued, activeRunStatuses(), domain.RunStatusQueued, staleBefore, now).
			Updates(updates)
		if result.Error != nil {
			return false, mapDBError(result.Error, "dialogue run not found")
		}
		return result.RowsAffected > 0, nil
	default:
		return false, nil
	}
}

func runnableRunWhereClause() string {
	return "status = ? OR (status IN ? AND status <> ? AND ((lease_owner = '' AND updated_at < ?) OR lease_expires_at < ?))"
}

func runIsRunnable(status string, leaseOwner string, leaseExpiresAt *time.Time, updatedAt time.Time, staleBefore time.Time, now time.Time) bool {
	if status == domain.RunStatusQueued {
		return true
	}
	if !containsString(activeRunStatuses(), status) {
		return false
	}
	if leaseOwner == "" && updatedAt.Before(staleBefore) {
		return true
	}
	return leaseExpiresAt != nil && leaseExpiresAt.Before(now)
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
