package repository

import (
	"context"

	"github.com/fishimei/NovelOS/internal/application/model"
	persistencemodels "github.com/fishimei/NovelOS/internal/infrastructure/persistence/gorm/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type storyTimelineRepository struct {
	*container
}

func (r *storyTimelineRepository) CreateBranch(ctx context.Context, branch model.StoryBranch) (model.StoryBranch, error) {
	if branch.ID == "" {
		branch.ID = r.nextID("branch")
	}
	if branch.Status == "" {
		branch.Status = "active"
	}
	if branch.Name == "" {
		branch.Name = "main"
	}
	if branch.CreatedAt.IsZero() {
		branch.CreatedAt = r.now()
	}
	branch.UpdatedAt = r.now()
	row := storyBranchRow(branch)
	if err := r.dbFor(ctx).Create(&row).Error; err != nil {
		return model.StoryBranch{}, mapDBError(err, "story branch not found")
	}
	return toStoryBranch(row), nil
}

func (r *storyTimelineRepository) GetBranchByID(ctx context.Context, branchID string) (model.StoryBranch, error) {
	var row persistencemodels.StoryBranch
	if err := r.dbFor(ctx).First(&row, "id = ?", branchID).Error; err != nil {
		return model.StoryBranch{}, mapDBError(err, "story branch not found")
	}
	return toStoryBranch(row), nil
}

func (r *storyTimelineRepository) ListBranchesBySessionID(ctx context.Context, sessionID string) ([]model.StoryBranch, error) {
	var rows []persistencemodels.StoryBranch
	if err := r.dbFor(ctx).Where("session_id = ?", sessionID).Order("created_at asc").Find(&rows).Error; err != nil {
		return nil, mapDBError(err, "story branch not found")
	}
	items := make([]model.StoryBranch, 0, len(rows))
	for _, row := range rows {
		items = append(items, toStoryBranch(row))
	}
	return items, nil
}

func (r *storyTimelineRepository) UpdateBranchHead(ctx context.Context, branchID string, headTickID string) error {
	return mapDBError(r.dbFor(ctx).Model(&persistencemodels.StoryBranch{}).Where("id = ?", branchID).Updates(map[string]any{
		"head_tick_id": headTickID,
		"updated_at":   r.now(),
	}).Error, "story branch not found")
}

func (r *storyTimelineRepository) AppendTick(ctx context.Context, tick model.StoryTick, refs []model.StoryTickStateRef, versions []model.StoryStateVersion) (model.StoryTick, error) {
	if tick.ID == "" {
		tick.ID = r.nextID("tick")
	}
	if tick.CreatedAt.IsZero() {
		tick.CreatedAt = r.now()
	}
	row, err := storyTickRow(tick)
	if err != nil {
		return model.StoryTick{}, payloadError("story tick", err)
	}
	err = r.dbFor(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
		for _, version := range versions {
			if version.ID == "" {
				version.ID = r.nextID("sversion")
			}
			if version.SourceTickID == "" {
				version.SourceTickID = tick.ID
			}
			if version.ProjectID == "" {
				version.ProjectID = tick.ProjectID
			}
			if version.SourceRunID == "" {
				version.SourceRunID = tick.SourceRunID
			}
			if version.CreatedAt.IsZero() {
				version.CreatedAt = r.now()
			}
			versionRow, err := storyStateVersionRow(version)
			if err != nil {
				return err
			}
			if err := tx.Create(&versionRow).Error; err != nil {
				return err
			}
		}
		for _, ref := range refs {
			if ref.TickID == "" {
				ref.TickID = tick.ID
			}
			if ref.ProjectID == "" {
				ref.ProjectID = tick.ProjectID
			}
			refRow := storyTickStateRefRow(ref)
			if err := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "tick_id"}, {Name: "entity_type"}, {Name: "entity_id"}},
				DoUpdates: clause.AssignmentColumns([]string{"project_id", "version_id"}),
			}).Create(&refRow).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return model.StoryTick{}, mapDBError(err, "story tick not found")
	}
	return toStoryTick(row)
}

func (r *storyTimelineRepository) GetTickByID(ctx context.Context, tickID string) (model.StoryTick, error) {
	var row persistencemodels.StoryTick
	if err := r.dbFor(ctx).First(&row, "id = ?", tickID).Error; err != nil {
		return model.StoryTick{}, mapDBError(err, "story tick not found")
	}
	return toStoryTick(row)
}

func (r *storyTimelineRepository) ListTicksByBranchID(ctx context.Context, branchID string) ([]model.StoryTick, error) {
	var rows []persistencemodels.StoryTick
	if err := r.dbFor(ctx).Where("branch_id = ?", branchID).Order("sequence asc").Find(&rows).Error; err != nil {
		return nil, mapDBError(err, "story tick not found")
	}
	items := make([]model.StoryTick, 0, len(rows))
	for _, row := range rows {
		tick, err := toStoryTick(row)
		if err != nil {
			return nil, payloadError("story tick", err)
		}
		items = append(items, tick)
	}
	return items, nil
}

func (r *storyTimelineRepository) ListTickStateRefs(ctx context.Context, tickID string) ([]model.StoryTickStateRef, error) {
	var rows []persistencemodels.StoryTickStateRef
	if err := r.dbFor(ctx).Where("tick_id = ?", tickID).Find(&rows).Error; err != nil {
		return nil, mapDBError(err, "story tick state ref not found")
	}
	items := make([]model.StoryTickStateRef, 0, len(rows))
	for _, row := range rows {
		items = append(items, toStoryTickStateRef(row))
	}
	return items, nil
}

func (r *storyTimelineRepository) ResolveTickState(ctx context.Context, tickID string) (model.StoryTickState, error) {
	tick, err := r.GetTickByID(ctx, tickID)
	if err != nil {
		return model.StoryTickState{}, err
	}
	refsByEntity := map[string]model.StoryTickStateRef{}
	for currentID := tickID; currentID != ""; {
		refs, err := r.ListTickStateRefs(ctx, currentID)
		if err != nil {
			return model.StoryTickState{}, err
		}
		for _, ref := range refs {
			key := stateRefKey(ref.EntityType, ref.EntityID)
			if _, ok := refsByEntity[key]; !ok {
				refsByEntity[key] = ref
			}
		}
		current, err := r.GetTickByID(ctx, currentID)
		if err != nil {
			return model.StoryTickState{}, err
		}
		currentID = current.ParentTickID
	}
	refs := make([]model.StoryTickStateRef, 0, len(refsByEntity))
	versionIDs := make([]string, 0, len(refsByEntity))
	for _, ref := range refsByEntity {
		refs = append(refs, ref)
		versionIDs = append(versionIDs, ref.VersionID)
	}
	versions, err := r.storyStateVersionsByID(ctx, versionIDs)
	if err != nil {
		return model.StoryTickState{}, err
	}
	return model.StoryTickState{Tick: tick, Refs: refs, Versions: versions}, nil
}

func (r *storyTimelineRepository) storyStateVersionsByID(ctx context.Context, versionIDs []string) ([]model.StoryStateVersion, error) {
	if len(versionIDs) == 0 {
		return nil, nil
	}
	var rows []persistencemodels.StoryStateVersion
	if err := r.dbFor(ctx).Where("id IN ?", versionIDs).Find(&rows).Error; err != nil {
		return nil, mapDBError(err, "story state version not found")
	}
	items := make([]model.StoryStateVersion, 0, len(rows))
	for _, row := range rows {
		version, err := toStoryStateVersion(row)
		if err != nil {
			return nil, payloadError("story state version", err)
		}
		items = append(items, version)
	}
	return items, nil
}

func stateRefKey(entityType string, entityID string) string {
	return entityType + ":" + entityID
}

func storyBranchRow(branch model.StoryBranch) persistencemodels.StoryBranch {
	return persistencemodels.StoryBranch{
		ID:               branch.ID,
		ProjectID:        branch.ProjectID,
		SessionID:        branch.SessionID,
		Name:             branch.Name,
		BaseTickID:       branch.BaseTickID,
		HeadTickID:       branch.HeadTickID,
		Status:           branch.Status,
		CreatedFromRunID: branch.CreatedFromRunID,
		CreatedAt:        branch.CreatedAt,
		UpdatedAt:        branch.UpdatedAt,
	}
}

func toStoryBranch(row persistencemodels.StoryBranch) model.StoryBranch {
	return model.StoryBranch{
		ID:               row.ID,
		ProjectID:        row.ProjectID,
		SessionID:        row.SessionID,
		Name:             row.Name,
		BaseTickID:       row.BaseTickID,
		HeadTickID:       row.HeadTickID,
		Status:           row.Status,
		CreatedFromRunID: row.CreatedFromRunID,
		CreatedAt:        row.CreatedAt,
		UpdatedAt:        row.UpdatedAt,
	}
}

func storyTickRow(tick model.StoryTick) (persistencemodels.StoryTick, error) {
	payloadJSON, err := encodeJSON(tick.Payload)
	if err != nil {
		return persistencemodels.StoryTick{}, err
	}
	return persistencemodels.StoryTick{
		ID:           tick.ID,
		ProjectID:    tick.ProjectID,
		SessionID:    tick.SessionID,
		BranchID:     tick.BranchID,
		ParentTickID: tick.ParentTickID,
		SourceRunID:  tick.SourceRunID,
		Sequence:     tick.Sequence,
		Kind:         tick.Kind,
		Summary:      tick.Summary,
		PayloadJSON:  payloadJSON,
		CreatedAt:    tick.CreatedAt,
	}, nil
}

func toStoryTick(row persistencemodels.StoryTick) (model.StoryTick, error) {
	payload, err := decodeJSON[map[string]any](row.PayloadJSON)
	if err != nil {
		return model.StoryTick{}, err
	}
	return model.StoryTick{
		ID:           row.ID,
		ProjectID:    row.ProjectID,
		SessionID:    row.SessionID,
		BranchID:     row.BranchID,
		ParentTickID: row.ParentTickID,
		SourceRunID:  row.SourceRunID,
		Sequence:     row.Sequence,
		Kind:         row.Kind,
		Summary:      row.Summary,
		Payload:      payload,
		CreatedAt:    row.CreatedAt,
	}, nil
}

func storyStateVersionRow(version model.StoryStateVersion) (persistencemodels.StoryStateVersion, error) {
	snapshotJSON, err := encodeJSON(version.Snapshot)
	if err != nil {
		return persistencemodels.StoryStateVersion{}, err
	}
	return persistencemodels.StoryStateVersion{
		ID:              version.ID,
		ProjectID:       version.ProjectID,
		EntityType:      version.EntityType,
		EntityID:        version.EntityID,
		ParentVersionID: version.ParentVersionID,
		SourceTickID:    version.SourceTickID,
		SourceRunID:     version.SourceRunID,
		SnapshotJSON:    snapshotJSON,
		CreatedAt:       version.CreatedAt,
	}, nil
}

func toStoryStateVersion(row persistencemodels.StoryStateVersion) (model.StoryStateVersion, error) {
	snapshot, err := decodeJSON[map[string]any](row.SnapshotJSON)
	if err != nil {
		return model.StoryStateVersion{}, err
	}
	return model.StoryStateVersion{
		ID:              row.ID,
		ProjectID:       row.ProjectID,
		EntityType:      row.EntityType,
		EntityID:        row.EntityID,
		ParentVersionID: row.ParentVersionID,
		SourceTickID:    row.SourceTickID,
		SourceRunID:     row.SourceRunID,
		Snapshot:        snapshot,
		CreatedAt:       row.CreatedAt,
	}, nil
}

func storyTickStateRefRow(ref model.StoryTickStateRef) persistencemodels.StoryTickStateRef {
	return persistencemodels.StoryTickStateRef{
		TickID:     ref.TickID,
		ProjectID:  ref.ProjectID,
		EntityType: ref.EntityType,
		EntityID:   ref.EntityID,
		VersionID:  ref.VersionID,
	}
}

func toStoryTickStateRef(row persistencemodels.StoryTickStateRef) model.StoryTickStateRef {
	return model.StoryTickStateRef{
		TickID:     row.TickID,
		ProjectID:  row.ProjectID,
		EntityType: row.EntityType,
		EntityID:   row.EntityID,
		VersionID:  row.VersionID,
	}
}
