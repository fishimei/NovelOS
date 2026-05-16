package repository

import (
	"context"

	"github.com/fishimei/NovelOS/internal/application/model"
	persistencemodels "github.com/fishimei/NovelOS/internal/infrastructure/persistence/gorm/models"
	"gorm.io/gorm/clause"
)

type relationshipRepository struct {
	*container
}

func (r *relationshipRepository) Create(ctx context.Context, projectID string, input model.CreateRelationshipInput) (model.Relationship, error) {
	leftID, rightID := normalizePair(input.CharacterAID, input.CharacterBID)
	now := r.now()

	pair := model.RelationshipPair{
		ID:               r.nextID("pair"),
		ProjectID:        projectID,
		LeftCharacterID:  leftID,
		RightCharacterID: rightID,
		Summary:          input.Summary,
		Anchors:          input.Anchors,
		TensionPoints:    input.TensionPoints,
		SharedHistory:    input.SharedHistory,
		Volatility:       input.Volatility,
		Status:           "active",
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	pair, err := r.UpsertPair(ctx, pair)
	if err != nil {
		return model.Relationship{}, err
	}

	views := []model.RelationshipView{
		{
			ID:                     r.nextID("view"),
			ProjectID:              projectID,
			PairID:                 pair.ID,
			SourceCharacterID:      input.CharacterAID,
			TargetCharacterID:      input.CharacterBID,
			PublicAttitude:         input.CharacterAView.PublicAttitude,
			PrivateAttitude:        input.CharacterAView.PrivateAttitude,
			BelievedTargetAttitude: input.CharacterAView.BelievedTargetAttitude,
			MaskingStrategy:        input.CharacterAView.MaskingStrategy,
			Status:                 "active",
			CreatedAt:              now,
			UpdatedAt:              now,
		},
		{
			ID:                     r.nextID("view"),
			ProjectID:              projectID,
			PairID:                 pair.ID,
			SourceCharacterID:      input.CharacterBID,
			TargetCharacterID:      input.CharacterAID,
			PublicAttitude:         input.CharacterBView.PublicAttitude,
			PrivateAttitude:        input.CharacterBView.PrivateAttitude,
			BelievedTargetAttitude: input.CharacterBView.BelievedTargetAttitude,
			MaskingStrategy:        input.CharacterBView.MaskingStrategy,
			Status:                 "active",
			CreatedAt:              now,
			UpdatedAt:              now,
		},
	}
	if err := r.UpsertViews(ctx, pair.ID, views); err != nil {
		return model.Relationship{}, err
	}

	return r.GetByID(ctx, pair.ID)
}

func (r *relationshipRepository) ListByProjectID(ctx context.Context, projectID string, pageInput model.PageInput) (model.ListResult[model.Relationship], error) {
	db := r.dbFor(ctx)
	var total int64
	if err := db.Model(&persistencemodels.RelationshipPair{}).Where("project_id = ?", projectID).Count(&total).Error; err != nil {
		return model.ListResult[model.Relationship]{}, mapDBError(err, "relationship not found")
	}
	if pageInput.Page <= 0 {
		pageInput.Page = 1
	}
	if pageInput.PageSize <= 0 {
		pageInput.PageSize = 20
	}
	var rows []persistencemodels.RelationshipPair
	if err := db.Where("project_id = ?", projectID).Order("created_at asc").Limit(pageInput.PageSize).Offset((pageInput.Page - 1) * pageInput.PageSize).Find(&rows).Error; err != nil {
		return model.ListResult[model.Relationship]{}, mapDBError(err, "relationship not found")
	}
	items := make([]model.Relationship, 0, len(rows))
	for _, row := range rows {
		relationship, err := r.GetByID(ctx, row.ID)
		if err != nil {
			return model.ListResult[model.Relationship]{}, err
		}
		items = append(items, relationship)
	}
	return model.ListResult[model.Relationship]{Items: items, Total: int(total)}, nil
}

func (r *relationshipRepository) GetByID(ctx context.Context, id string) (model.Relationship, error) {
	db := r.dbFor(ctx)
	var pairRow persistencemodels.RelationshipPair
	if err := db.First(&pairRow, "id = ?", id).Error; err != nil {
		return model.Relationship{}, mapDBError(err, "relationship not found")
	}
	pair, err := toRelationshipPair(pairRow)
	if err != nil {
		return model.Relationship{}, payloadError("relationship pair", err)
	}

	var viewRows []persistencemodels.RelationshipView
	if err := db.Where("pair_id = ?", id).Order("created_at asc").Find(&viewRows).Error; err != nil {
		return model.Relationship{}, mapDBError(err, "relationship not found")
	}
	views := make([]model.RelationshipView, 0, len(viewRows))
	for _, row := range viewRows {
		views = append(views, toRelationshipView(row))
	}

	var eventRows []persistencemodels.RelationshipEvent
	if err := db.Where("pair_id = ?", id).Order("created_at desc").Limit(10).Find(&eventRows).Error; err != nil {
		return model.Relationship{}, mapDBError(err, "relationship not found")
	}
	events := make([]model.RelationshipEvent, 0, len(eventRows))
	for _, row := range eventRows {
		event, err := toRelationshipEvent(row)
		if err != nil {
			return model.Relationship{}, payloadError("relationship event", err)
		}
		events = append(events, event)
	}

	result := model.Relationship{
		Pair:         pair,
		Views:        views,
		RecentEvents: events,
	}
	for i := range views {
		view := views[i]
		if view.SourceCharacterID == pair.LeftCharacterID && view.TargetCharacterID == pair.RightCharacterID {
			result.CharacterAView = &view
		}
		if view.SourceCharacterID == pair.RightCharacterID && view.TargetCharacterID == pair.LeftCharacterID {
			result.CharacterBView = &view
		}
	}
	return result, nil
}

func (r *relationshipRepository) Update(ctx context.Context, id string, input model.UpdateRelationshipInput) (model.Relationship, error) {
	current, err := r.GetByID(ctx, id)
	if err != nil {
		return model.Relationship{}, err
	}
	current.Pair.Summary = input.Summary
	current.Pair.Anchors = input.Anchors
	current.Pair.TensionPoints = input.TensionPoints
	current.Pair.SharedHistory = input.SharedHistory
	current.Pair.Volatility = input.Volatility
	current.Pair.UpdatedAt = r.now()
	if _, err := r.UpsertPair(ctx, current.Pair); err != nil {
		return model.Relationship{}, err
	}
	return r.GetByID(ctx, id)
}

func (r *relationshipRepository) UpsertPair(ctx context.Context, pair model.RelationshipPair) (model.RelationshipPair, error) {
	if pair.ID == "" {
		pair.ID = r.nextID("pair")
	}
	pair.LeftCharacterID, pair.RightCharacterID = normalizePair(pair.LeftCharacterID, pair.RightCharacterID)
	if pair.Status == "" {
		pair.Status = "active"
	}
	row, err := relationshipPairRowFromModel(pair, r.now(), pair.ID)
	if err != nil {
		return model.RelationshipPair{}, payloadError("relationship pair", err)
	}
	if err := r.dbFor(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "project_id"}, {Name: "left_character_id"}, {Name: "right_character_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"summary", "anchors_json", "tension_json", "shared_history_json", "volatility", "status", "updated_at"}),
	}).Create(&row).Error; err != nil {
		return model.RelationshipPair{}, mapDBError(err, "relationship not found")
	}
	result, err := r.GetByID(ctx, pair.ID)
	if err == nil {
		return result.Pair, nil
	}
	var pairRow persistencemodels.RelationshipPair
	if err := r.dbFor(ctx).Where("project_id = ? AND left_character_id = ? AND right_character_id = ?", pair.ProjectID, pair.LeftCharacterID, pair.RightCharacterID).First(&pairRow).Error; err != nil {
		return model.RelationshipPair{}, mapDBError(err, "relationship not found")
	}
	return toRelationshipPair(pairRow)
}

func (r *relationshipRepository) UpsertViews(ctx context.Context, pairID string, views []model.RelationshipView) error {
	if len(views) == 0 {
		return nil
	}
	now := r.now()
	rows := make([]persistencemodels.RelationshipView, 0, len(views))
	for _, view := range views {
		if view.ID == "" {
			view.ID = r.nextID("view")
		}
		if view.PairID == "" {
			view.PairID = pairID
		}
		if view.Status == "" {
			view.Status = "active"
		}
		rows = append(rows, relationshipViewRowFromModel(view, now, view.ID))
	}
	return mapDBError(r.dbFor(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "project_id"}, {Name: "source_character_id"}, {Name: "target_character_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"pair_id", "public_attitude", "private_attitude", "believed_target_attitude", "masking_strategy", "status", "updated_at"}),
	}).Create(&rows).Error, "relationship not found")
}

func (r *relationshipRepository) AddEvent(ctx context.Context, event model.RelationshipEvent) (model.RelationshipEvent, error) {
	if event.ID == "" {
		event.ID = r.nextID("revent")
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = r.now()
	}
	row, err := relationshipEventRowFromModel(event, r.now(), event.ID)
	if err != nil {
		return model.RelationshipEvent{}, payloadError("relationship event", err)
	}
	if err := r.dbFor(ctx).Create(&row).Error; err != nil {
		return model.RelationshipEvent{}, mapDBError(err, "relationship event not found")
	}
	return toRelationshipEvent(row)
}
