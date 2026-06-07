package repository

import (
	"context"
	"errors"
	"sort"
	"time"

	"github.com/fishimei/NovelOS/internal/application/model"
	persistencemodels "github.com/fishimei/NovelOS/internal/infrastructure/persistence/gorm/models"
	"github.com/fishimei/NovelOS/internal/pkgerr"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type storyEventStore struct {
	*container
}

func (s *storyEventStore) AppendEvent(ctx context.Context, e model.StoryEvent) (model.StoryEvent, error) {
	now := s.now()
	if e.ID == "" {
		e.ID = s.nextID("event")
	}
	if e.Kind == "" {
		e.Kind = model.EventKindSceneResolved
	}
	if e.CreatedAt.IsZero() {
		e.CreatedAt = now
	}
	if e.StoryTime.IsZero() {
		e.StoryTime = e.CreatedAt
	}
	if e.Sequence == 0 {
		next, err := s.nextEventSequence(ctx, e.BranchID)
		if err != nil {
			return model.StoryEvent{}, err
		}
		e.Sequence = next
	}
	if e.ParentEventID == "" && e.Kind != model.EventKindGenesis {
		branch, err := s.GetBranch(ctx, e.BranchID)
		if err != nil {
			return model.StoryEvent{}, err
		}
		e.ParentEventID = branch.HeadEventID
	}
	row, err := storyEventRow(e)
	if err != nil {
		return model.StoryEvent{}, payloadError("story event", err)
	}
	if err := s.dbFor(ctx).Create(&row).Error; err != nil {
		return model.StoryEvent{}, mapDBError(err, "story event not found")
	}
	return toStoryEvent(row)
}

func (s *storyEventStore) GetEvent(ctx context.Context, id string) (model.StoryEvent, error) {
	var row persistencemodels.StoryEvent
	if err := s.dbFor(ctx).First(&row, "id = ?", id).Error; err != nil {
		return model.StoryEvent{}, mapDBError(err, "story event not found")
	}
	return toStoryEvent(row)
}

func (s *storyEventStore) ListEventsByBranch(ctx context.Context, branchID string) ([]model.StoryEvent, error) {
	var rows []persistencemodels.StoryEvent
	if err := s.dbFor(ctx).Where("branch_id = ?", branchID).Order("sequence asc, id asc").Find(&rows).Error; err != nil {
		return nil, mapDBError(err, "story event not found")
	}
	return toStoryEvents(rows)
}

func (s *storyEventStore) ListEventsBySession(ctx context.Context, sessionID string) ([]model.StoryEvent, error) {
	var rows []persistencemodels.StoryEvent
	if err := s.dbFor(ctx).Where("session_id = ?", sessionID).Order("branch_id asc, sequence asc, id asc").Find(&rows).Error; err != nil {
		return nil, mapDBError(err, "story event not found")
	}
	return toStoryEvents(rows)
}

func (s *storyEventStore) CreateBranch(ctx context.Context, b model.Branch) (model.Branch, error) {
	now := s.now()
	if b.ID == "" {
		b.ID = s.nextID("branch")
	}
	if b.Name == "" {
		b.Name = "main"
	}
	if b.Status == "" {
		b.Status = "active"
	}
	if b.CreatedAt.IsZero() {
		b.CreatedAt = now
	}
	b.UpdatedAt = now
	row := storyBranchRow(b)
	if err := s.dbFor(ctx).Create(&row).Error; err != nil {
		return model.Branch{}, mapDBError(err, "story branch not found")
	}
	return toBranch(row), nil
}

func (s *storyEventStore) GetBranch(ctx context.Context, id string) (model.Branch, error) {
	var row persistencemodels.StoryBranch
	if err := s.dbFor(ctx).First(&row, "id = ?", id).Error; err != nil {
		return model.Branch{}, mapDBError(err, "story branch not found")
	}
	return toBranch(row), nil
}

func (s *storyEventStore) ListBranchesBySession(ctx context.Context, sessionID string) ([]model.Branch, error) {
	var rows []persistencemodels.StoryBranch
	if err := s.dbFor(ctx).Where("session_id = ?", sessionID).Order("created_at asc, id asc").Find(&rows).Error; err != nil {
		return nil, mapDBError(err, "story branch not found")
	}
	items := make([]model.Branch, 0, len(rows))
	for _, row := range rows {
		items = append(items, toBranch(row))
	}
	return items, nil
}

func (s *storyEventStore) UpdateBranchHead(ctx context.Context, branchID, headEventID string) error {
	return mapDBError(s.dbFor(ctx).Model(&persistencemodels.StoryBranch{}).Where("id = ?", branchID).Updates(map[string]any{
		"head_event_id": headEventID,
		"updated_at":    s.now(),
	}).Error, "story branch not found")
}

func (s *storyEventStore) SetPublishedFrontier(ctx context.Context, branchID, eventID string) error {
	branch, err := s.GetBranch(ctx, branchID)
	if err != nil {
		return err
	}
	events, err := s.ListEventsByBranch(ctx, branchID)
	if err != nil {
		return err
	}
	found := false
	eventIDs := make([]string, 0)
	for _, event := range events {
		eventIDs = append(eventIDs, event.ID)
		if event.ID == eventID {
			found = true
			break
		}
	}
	if !found && eventID != "" {
		return pkgerr.Validation("published frontier event does not belong to branch")
	}
	return mapDBError(s.dbFor(ctx).Transaction(func(tx *gorm.DB) error {
		if len(eventIDs) > 0 {
			if err := tx.Model(&persistencemodels.StoryEvent{}).Where("id IN ?", eventIDs).Update("published", true).Error; err != nil {
				return err
			}
		}
		return tx.Model(&persistencemodels.StoryBranch{}).Where("id = ?", branch.ID).Updates(map[string]any{
			"published_frontier_event_id": eventID,
			"updated_at":                  s.now(),
		}).Error
	}), "story branch not found")
}

func (s *storyEventStore) ResolveStateAt(ctx context.Context, eventID string) (model.WorldSnapshot, error) {
	event, err := s.GetEvent(ctx, eventID)
	if err != nil {
		return model.WorldSnapshot{}, err
	}
	events, err := s.ancestorEvents(ctx, event)
	if err != nil {
		return model.WorldSnapshot{}, err
	}
	if len(events) == 0 {
		return model.WorldSnapshot{}, pkgerr.NotFound("story event not found")
	}
	state := snapshotFromDelta(events[0])
	applyActionRuntime(&state, events[0])
	for _, event := range events[1:] {
		applyDelta(&state, event)
		applyActionRuntime(&state, event)
	}
	if err := s.mergeProjectWorldDirectory(ctx, &state, event.ProjectID); err != nil {
		return model.WorldSnapshot{}, err
	}
	state.AtEventID = event.ID
	state.StoryTime = event.StoryTime
	return state, nil
}

func (s *storyEventStore) mergeProjectWorldDirectory(ctx context.Context, snapshot *model.WorldSnapshot, projectID string) error {
	areas, err := s.ListMapAreasByProjectID(ctx, projectID)
	if err != nil {
		return err
	}
	if len(areas) > 0 {
		snapshot.Areas = areas
	}
	locations, err := s.ListLocationsByProjectID(ctx, projectID)
	if err != nil {
		return err
	}
	if len(locations) > 0 {
		snapshot.Locations = locations
	}
	influences, err := s.ListFactionInfluencesByProjectID(ctx, projectID)
	if err != nil {
		return err
	}
	if len(influences) > 0 {
		snapshot.Factions = influences
	}
	return nil
}

func (s *storyEventStore) InFlightActionsAt(ctx context.Context, branchID string, at time.Time) ([]model.OngoingAction, error) {
	branch, err := s.GetBranch(ctx, branchID)
	if err != nil {
		return nil, err
	}
	events, err := s.reachableEventsAt(ctx, branch.HeadEventID)
	if err != nil {
		return nil, err
	}
	actionsByID := map[string]model.OngoingAction{}
	actionIDsByCharacter := map[string][]string{}
	for _, event := range events {
		if event.StoryTime.After(at) {
			continue
		}
		switch event.Kind {
		case model.EventKindActionScheduled:
			action, ok := actionFromEvent(event)
			if !ok {
				continue
			}
			if action.ID == "" {
				action.ID = event.ID
			}
			if !action.StartAt.After(at) && action.EndsAt.After(at) && action.Status == "ongoing" {
				actionsByID[action.ID] = action
				actionIDsByCharacter[action.CharacterID] = appendUniqueString(actionIDsByCharacter[action.CharacterID], action.ID)
			}
		case model.EventKindActionCompleted, model.EventKindActionVoided, model.EventKindActionSuperseded:
			action, ok := actionFromEvent(event)
			if ok {
				if action.ID != "" {
					delete(actionsByID, action.ID)
				}
				if action.ID == "" {
					for _, actionID := range actionIDsByCharacter[action.CharacterID] {
						delete(actionsByID, actionID)
					}
				}
				continue
			}
			for _, actorID := range event.ActorIDs {
				for _, actionID := range actionIDsByCharacter[actorID] {
					delete(actionsByID, actionID)
				}
			}
		case model.EventKindSceneResolved:
			for _, actorID := range event.ActorIDs {
				for _, actionID := range actionIDsByCharacter[actorID] {
					delete(actionsByID, actionID)
				}
			}
		}
	}
	out := make([]model.OngoingAction, 0, len(actionsByID))
	for _, action := range actionsByID {
		out = append(out, action)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].StartAt.Equal(out[j].StartAt) {
			if out[i].CharacterID == out[j].CharacterID {
				return out[i].ID < out[j].ID
			}
			return out[i].CharacterID < out[j].CharacterID
		}
		return out[i].StartAt.Before(out[j].StartAt)
	})
	return out, nil
}

func appendUniqueString(values []string, value string) []string {
	if value == "" {
		return values
	}
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

func (s *storyEventStore) UpsertSnapshot(ctx context.Context, branchID, eventID string, snapshot model.WorldSnapshot) error {
	raw, err := encodeJSON(snapshot)
	if err != nil {
		return payloadError("story snapshot", err)
	}
	row := persistencemodels.StorySnapshot{BranchID: branchID, EventID: eventID, SnapshotJSON: raw, UpdatedAt: s.now()}
	return mapDBError(s.dbFor(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "branch_id"}, {Name: "event_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"snapshot_json", "updated_at"}),
	}).Create(&row).Error, "story snapshot not found")
}

func (s *storyEventStore) InitGenesis(ctx context.Context, projectID, sessionID string, t0 model.WorldSnapshot) (model.StoryEvent, error) {
	existing, err := s.ListBranchesBySession(ctx, sessionID)
	if err == nil && len(existing) > 0 {
		if existing[0].HeadEventID == "" {
			return model.StoryEvent{}, pkgerr.Conflict(pkgerr.CodeConflict, "story branch has no genesis event")
		}
		return s.GetEvent(ctx, existing[0].HeadEventID)
	}
	if err != nil && !isNotFound(err) {
		return model.StoryEvent{}, err
	}

	now := s.now()
	if t0.StoryTime.IsZero() {
		t0.StoryTime = now
	}
	var out model.StoryEvent
	err = s.dbFor(ctx).Transaction(func(tx *gorm.DB) error {
		txCtx := ctx
		store := &storyEventStore{container: &container{db: tx, ids: s.ids, clock: s.clock}}
		branch, err := store.CreateBranch(txCtx, model.Branch{
			ProjectID: projectID,
			SessionID: sessionID,
			Name:      "main",
			Status:    "active",
			CreatedAt: now,
			UpdatedAt: now,
		})
		if err != nil {
			return err
		}
		event, err := store.AppendEvent(txCtx, model.StoryEvent{
			ProjectID:  projectID,
			SessionID:  sessionID,
			BranchID:   branch.ID,
			Sequence:   1,
			StoryTime:  t0.StoryTime,
			Kind:       model.EventKindGenesis,
			Summary:    "初始化故事世界",
			Payload:    map[string]any{"world_snapshot": t0},
			StateDelta: genesisDelta(t0),
			CreatedAt:  now,
		})
		if err != nil {
			return err
		}
		t0.AtEventID = event.ID
		if err := store.UpdateBranchHead(txCtx, branch.ID, event.ID); err != nil {
			return err
		}
		if err := store.UpsertSnapshot(txCtx, branch.ID, event.ID, t0); err != nil {
			return err
		}
		out = event
		return nil
	})
	if err != nil {
		return model.StoryEvent{}, err
	}
	return out, nil
}

func (s *storyEventStore) GetProjectGenesis(ctx context.Context, projectID string) (model.StoryEvent, error) {
	var row persistencemodels.StoryEvent
	if err := s.dbFor(ctx).Where("project_id = ? AND kind = ?", projectID, model.EventKindGenesis).Order("created_at asc, sequence asc, id asc").Take(&row).Error; err != nil {
		return model.StoryEvent{}, mapDBError(err, "story event not found")
	}
	return toStoryEvent(row)
}

func (s *storyEventStore) CreateChapterSpan(ctx context.Context, span model.ChapterEventSpan) (model.ChapterEventSpan, error) {
	if span.ID == "" {
		span.ID = s.nextID("span")
	}
	if span.CreatedAt.IsZero() {
		span.CreatedAt = s.now()
	}
	row := persistencemodels.ChapterEventSpan{
		ID:          span.ID,
		ProjectID:   span.ProjectID,
		ChapterID:   span.ChapterID,
		BranchID:    span.BranchID,
		FromEventID: span.FromEventID,
		ToEventID:   span.ToEventID,
		CreatedAt:   span.CreatedAt,
	}
	if err := s.dbFor(ctx).Create(&row).Error; err != nil {
		return model.ChapterEventSpan{}, mapDBError(err, "chapter event span not found")
	}
	return toChapterEventSpan(row), nil
}

func (s *storyEventStore) GetChapterSpanByRange(ctx context.Context, branchID, fromEventID, toEventID string) (model.ChapterEventSpan, error) {
	var row persistencemodels.ChapterEventSpan
	if err := s.dbFor(ctx).First(&row, "branch_id = ? AND from_event_id = ? AND to_event_id = ?", branchID, fromEventID, toEventID).Error; err != nil {
		return model.ChapterEventSpan{}, mapDBError(err, "chapter event span not found")
	}
	return toChapterEventSpan(row), nil
}

func (s *storyEventStore) GetWorldMapByProjectID(ctx context.Context, projectID string) (model.WorldMap, error) {
	var row persistencemodels.WorldMap
	if err := s.dbFor(ctx).First(&row, "project_id = ?", projectID).Error; err != nil {
		return model.WorldMap{}, mapDBError(err, "world map not found")
	}
	return toWorldMap(row)
}

func (s *storyEventStore) UpsertWorldMap(ctx context.Context, worldMap model.WorldMap) (model.WorldMap, error) {
	now := s.now()
	if worldMap.ID == "" {
		worldMap.ID = s.nextID("map")
	}
	if worldMap.CreatedAt.IsZero() {
		worldMap.CreatedAt = now
	}
	if worldMap.UpdatedAt.IsZero() {
		worldMap.UpdatedAt = now
	}
	row, err := worldMapRowFromModel(worldMap, now)
	if err != nil {
		return model.WorldMap{}, payloadError("world map", err)
	}
	if err := s.dbFor(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "project_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"name", "seed", "width", "height", "status", "properties_json", "updated_at"}),
	}).Create(&row).Error; err != nil {
		return model.WorldMap{}, mapDBError(err, "world map not found")
	}
	return s.GetWorldMapByProjectID(ctx, worldMap.ProjectID)
}

func (s *storyEventStore) ListMapAreasByProjectID(ctx context.Context, projectID string) ([]model.MapArea, error) {
	var rows []persistencemodels.MapArea
	if err := s.dbFor(ctx).Where("project_id = ?", projectID).Order("level asc, created_at asc").Find(&rows).Error; err != nil {
		return nil, mapDBError(err, "map area not found")
	}
	areas := make([]model.MapArea, 0, len(rows))
	for _, row := range rows {
		area, err := toMapArea(row)
		if err != nil {
			return nil, payloadError("map area", err)
		}
		areas = append(areas, area)
	}
	return areas, nil
}

func (s *storyEventStore) UpsertMapAreas(ctx context.Context, projectID string, areas []model.MapArea) error {
	if len(areas) == 0 {
		return nil
	}
	now := s.now()
	rows := make([]persistencemodels.MapArea, 0, len(areas))
	for _, area := range areas {
		if area.ID == "" {
			area.ID = s.nextID("area")
		}
		area.ProjectID = projectID
		if area.Status == "" {
			area.Status = "active"
		}
		row, err := mapAreaRowFromModel(area, now)
		if err != nil {
			return payloadError("map area", err)
		}
		rows = append(rows, row)
	}
	return mapDBError(s.dbFor(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"project_id", "map_id", "parent_area_id", "name", "level", "min_x", "min_y", "max_x", "max_y", "center_x", "center_y", "dominant_terrain", "status", "properties_json", "updated_at"}),
	}).Create(&rows).Error, "map area not found")
}

func (s *storyEventStore) ListMapTilesByProjectID(ctx context.Context, projectID string) ([]model.MapTile, error) {
	var rows []persistencemodels.MapTile
	if err := s.dbFor(ctx).Where("project_id = ?", projectID).Order("y asc, x asc").Find(&rows).Error; err != nil {
		return nil, mapDBError(err, "map tile not found")
	}
	tiles := make([]model.MapTile, 0, len(rows))
	for _, row := range rows {
		tile, err := toMapTile(row)
		if err != nil {
			return nil, payloadError("map tile", err)
		}
		tiles = append(tiles, tile)
	}
	return tiles, nil
}

func (s *storyEventStore) UpsertMapTiles(ctx context.Context, projectID string, tiles []model.MapTile) error {
	if len(tiles) == 0 {
		return nil
	}
	now := s.now()
	rows := make([]persistencemodels.MapTile, 0, len(tiles))
	for _, tile := range tiles {
		if tile.ID == "" {
			tile.ID = s.nextID("tile")
		}
		tile.ProjectID = projectID
		row, err := mapTileRowFromModel(tile, now)
		if err != nil {
			return payloadError("map tile", err)
		}
		rows = append(rows, row)
	}
	return mapDBError(s.dbFor(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "map_id"}, {Name: "x"}, {Name: "y"}},
		DoUpdates: clause.AssignmentColumns([]string{"project_id", "altitude", "temperature", "humidity", "is_ocean", "terrain", "properties_json", "updated_at"}),
	}).Create(&rows).Error, "map tile not found")
}

func (s *storyEventStore) ListLocationsByProjectID(ctx context.Context, projectID string) ([]model.LocationState, error) {
	var rows []persistencemodels.LocationState
	if err := s.dbFor(ctx).Where("project_id = ?", projectID).Order("created_at asc").Find(&rows).Error; err != nil {
		return nil, mapDBError(err, "location state not found")
	}
	return locationStatesFromRows(rows)
}

func (s *storyEventStore) GetLocation(ctx context.Context, projectID string, locationID string) (model.LocationState, error) {
	var row persistencemodels.LocationState
	if err := s.dbFor(ctx).Where("project_id = ? AND id = ?", projectID, locationID).First(&row).Error; err != nil {
		return model.LocationState{}, mapDBError(err, "location state not found")
	}
	return toLocationState(row)
}

func (s *storyEventStore) ListLocationsByParentID(ctx context.Context, projectID string, parentLocationID string) ([]model.LocationState, error) {
	var rows []persistencemodels.LocationState
	if err := s.dbFor(ctx).
		Where("project_id = ? AND parent_location_id = ?", projectID, parentLocationID).
		Order("created_at asc").
		Find(&rows).Error; err != nil {
		return nil, mapDBError(err, "location state not found")
	}
	return locationStatesFromRows(rows)
}

func locationStatesFromRows(rows []persistencemodels.LocationState) ([]model.LocationState, error) {
	locations := make([]model.LocationState, 0, len(rows))
	for _, row := range rows {
		location, err := toLocationState(row)
		if err != nil {
			return nil, payloadError("location state", err)
		}
		locations = append(locations, location)
	}
	return locations, nil
}

func (s *storyEventStore) UpsertLocations(ctx context.Context, projectID string, locations []model.LocationState) error {
	if len(locations) == 0 {
		return nil
	}
	now := s.now()
	rows := make([]persistencemodels.LocationState, 0, len(locations))
	for _, location := range locations {
		if location.ID == "" {
			location.ID = s.nextID("location")
		}
		location.ProjectID = projectID
		if location.Status == "" {
			location.Status = "active"
		}
		row, err := locationStateRowFromModel(location, now)
		if err != nil {
			return payloadError("location state", err)
		}
		rows = append(rows, row)
	}
	return mapDBError(s.dbFor(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"project_id", "map_id", "area_id", "region_id", "parent_location_id", "name", "type", "scale", "detail_state", "description", "x", "y", "radius", "status", "properties_json", "updated_at"}),
	}).Create(&rows).Error, "location state not found")
}

func (s *storyEventStore) ListFactionInfluencesByProjectID(ctx context.Context, projectID string) ([]model.FactionInfluence, error) {
	var rows []persistencemodels.FactionInfluence
	if err := s.dbFor(ctx).Where("project_id = ?", projectID).Order("created_at asc").Find(&rows).Error; err != nil {
		return nil, mapDBError(err, "faction influence not found")
	}
	influences := make([]model.FactionInfluence, 0, len(rows))
	for _, row := range rows {
		influences = append(influences, toFactionInfluence(row))
	}
	return influences, nil
}

func (s *storyEventStore) UpsertFactionInfluences(ctx context.Context, projectID string, influences []model.FactionInfluence) error {
	if len(influences) == 0 {
		return nil
	}
	now := s.now()
	rows := make([]persistencemodels.FactionInfluence, 0, len(influences))
	for _, influence := range influences {
		if influence.ID == "" {
			influence.ID = s.nextID("faction")
		}
		influence.ProjectID = projectID
		if influence.Status == "" {
			influence.Status = "active"
		}
		rows = append(rows, factionInfluenceRowFromModel(influence, now))
	}
	return mapDBError(s.dbFor(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"project_id", "location_id", "faction_name", "influence", "attitude", "description", "status", "updated_at"}),
	}).Create(&rows).Error, "faction influence not found")
}

func (s *storyEventStore) nextEventSequence(ctx context.Context, branchID string) (int, error) {
	var row persistencemodels.StoryEvent
	err := s.dbFor(ctx).Where("branch_id = ?", branchID).Order("sequence desc, id desc").Take(&row).Error
	if err == nil {
		return row.Sequence + 1, nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return 1, nil
	}
	return 0, mapDBError(err, "story event not found")
}

func (s *storyEventStore) ancestorEvents(ctx context.Context, event model.StoryEvent) ([]model.StoryEvent, error) {
	chain := []model.StoryEvent{event}
	for event.ParentEventID != "" {
		parent, err := s.GetEvent(ctx, event.ParentEventID)
		if err != nil {
			return nil, err
		}
		chain = append(chain, parent)
		event = parent
	}
	for i, j := 0, len(chain)-1; i < j; i, j = i+1, j-1 {
		chain[i], chain[j] = chain[j], chain[i]
	}
	return chain, nil
}

func (s *storyEventStore) reachableEventsAt(ctx context.Context, eventID string) ([]model.StoryEvent, error) {
	if eventID == "" {
		return nil, nil
	}
	event, err := s.GetEvent(ctx, eventID)
	if err != nil {
		return nil, err
	}
	return s.ancestorEvents(ctx, event)
}

func storyEventRow(e model.StoryEvent) (persistencemodels.StoryEvent, error) {
	actorIDs, err := encodeJSON(e.ActorIDs)
	if err != nil {
		return persistencemodels.StoryEvent{}, err
	}
	resources, err := encodeJSON(e.ResourceKeys)
	if err != nil {
		return persistencemodels.StoryEvent{}, err
	}
	payload, err := encodeJSON(e.Payload)
	if err != nil {
		return persistencemodels.StoryEvent{}, err
	}
	delta, err := encodeJSON(e.StateDelta)
	if err != nil {
		return persistencemodels.StoryEvent{}, err
	}
	return persistencemodels.StoryEvent{
		ID:             e.ID,
		ProjectID:      e.ProjectID,
		SessionID:      e.SessionID,
		BranchID:       e.BranchID,
		ParentEventID:  e.ParentEventID,
		Sequence:       e.Sequence,
		StoryTime:      e.StoryTime,
		Kind:           e.Kind,
		ActorIDsJSON:   actorIDs,
		LocationKey:    e.LocationKey,
		ResourceJSON:   resources,
		Summary:        e.Summary,
		PayloadJSON:    payload,
		StateDeltaJSON: delta,
		Published:      e.Published,
		CreatedAt:      e.CreatedAt,
	}, nil
}

func toStoryEvent(row persistencemodels.StoryEvent) (model.StoryEvent, error) {
	actorIDs, err := decodeJSON[[]string](row.ActorIDsJSON)
	if err != nil {
		return model.StoryEvent{}, err
	}
	resources, err := decodeJSON[[]string](row.ResourceJSON)
	if err != nil {
		return model.StoryEvent{}, err
	}
	payload, err := decodeJSON[map[string]any](row.PayloadJSON)
	if err != nil {
		return model.StoryEvent{}, err
	}
	delta, err := decodeJSON[model.EventStateDelta](row.StateDeltaJSON)
	if err != nil {
		return model.StoryEvent{}, err
	}
	return model.StoryEvent{
		ID:            row.ID,
		ProjectID:     row.ProjectID,
		SessionID:     row.SessionID,
		BranchID:      row.BranchID,
		ParentEventID: row.ParentEventID,
		Sequence:      row.Sequence,
		StoryTime:     row.StoryTime,
		Kind:          row.Kind,
		ActorIDs:      actorIDs,
		LocationKey:   row.LocationKey,
		ResourceKeys:  resources,
		Summary:       row.Summary,
		Payload:       payload,
		StateDelta:    delta,
		Published:     row.Published,
		CreatedAt:     row.CreatedAt,
	}, nil
}

func toStoryEvents(rows []persistencemodels.StoryEvent) ([]model.StoryEvent, error) {
	events := make([]model.StoryEvent, 0, len(rows))
	for _, row := range rows {
		event, err := toStoryEvent(row)
		if err != nil {
			return nil, payloadError("story event", err)
		}
		events = append(events, event)
	}
	return events, nil
}

func storyBranchRow(branch model.Branch) persistencemodels.StoryBranch {
	return persistencemodels.StoryBranch{
		ID:                       branch.ID,
		ProjectID:                branch.ProjectID,
		SessionID:                branch.SessionID,
		Name:                     branch.Name,
		BaseEventID:              branch.BaseEventID,
		HeadEventID:              branch.HeadEventID,
		PublishedFrontierEventID: branch.PublishedFrontierEventID,
		Status:                   branch.Status,
		CreatedAt:                branch.CreatedAt,
		UpdatedAt:                branch.UpdatedAt,
	}
}

func toBranch(row persistencemodels.StoryBranch) model.Branch {
	return model.Branch{
		ID:                       row.ID,
		ProjectID:                row.ProjectID,
		SessionID:                row.SessionID,
		Name:                     row.Name,
		BaseEventID:              row.BaseEventID,
		HeadEventID:              row.HeadEventID,
		PublishedFrontierEventID: row.PublishedFrontierEventID,
		Status:                   row.Status,
		CreatedAt:                row.CreatedAt,
		UpdatedAt:                row.UpdatedAt,
	}
}

func toChapterEventSpan(row persistencemodels.ChapterEventSpan) model.ChapterEventSpan {
	return model.ChapterEventSpan{
		ID:          row.ID,
		ProjectID:   row.ProjectID,
		ChapterID:   row.ChapterID,
		BranchID:    row.BranchID,
		FromEventID: row.FromEventID,
		ToEventID:   row.ToEventID,
		CreatedAt:   row.CreatedAt,
	}
}

func worldMapRowFromModel(worldMap model.WorldMap, now time.Time) (persistencemodels.WorldMap, error) {
	propertiesJSON, err := encodeJSON(worldMap.Properties)
	if err != nil {
		return persistencemodels.WorldMap{}, err
	}
	if worldMap.CreatedAt.IsZero() {
		worldMap.CreatedAt = now
	}
	if worldMap.UpdatedAt.IsZero() {
		worldMap.UpdatedAt = now
	}
	return persistencemodels.WorldMap{
		ID:             worldMap.ID,
		ProjectID:      worldMap.ProjectID,
		Name:           worldMap.Name,
		Seed:           worldMap.Seed,
		Width:          worldMap.Width,
		Height:         worldMap.Height,
		Status:         worldMap.Status,
		PropertiesJSON: propertiesJSON,
		CreatedAt:      worldMap.CreatedAt,
		UpdatedAt:      worldMap.UpdatedAt,
	}, nil
}

func toWorldMap(row persistencemodels.WorldMap) (model.WorldMap, error) {
	properties, err := decodeJSON[map[string]any](row.PropertiesJSON)
	if err != nil {
		return model.WorldMap{}, err
	}
	return model.WorldMap{
		ID:         row.ID,
		ProjectID:  row.ProjectID,
		Name:       row.Name,
		Seed:       row.Seed,
		Width:      row.Width,
		Height:     row.Height,
		Status:     row.Status,
		Properties: properties,
		CreatedAt:  row.CreatedAt,
		UpdatedAt:  row.UpdatedAt,
	}, nil
}

func mapTileRowFromModel(tile model.MapTile, now time.Time) (persistencemodels.MapTile, error) {
	propertiesJSON, err := encodeJSON(tile.Properties)
	if err != nil {
		return persistencemodels.MapTile{}, err
	}
	if tile.CreatedAt.IsZero() {
		tile.CreatedAt = now
	}
	if tile.UpdatedAt.IsZero() {
		tile.UpdatedAt = now
	}
	return persistencemodels.MapTile{
		ID:             tile.ID,
		ProjectID:      tile.ProjectID,
		MapID:          tile.MapID,
		X:              tile.X,
		Y:              tile.Y,
		Altitude:       tile.Altitude,
		Temperature:    tile.Temperature,
		Humidity:       tile.Humidity,
		IsOcean:        tile.IsOcean,
		Terrain:        tile.Terrain,
		PropertiesJSON: propertiesJSON,
		CreatedAt:      tile.CreatedAt,
		UpdatedAt:      tile.UpdatedAt,
	}, nil
}

func toMapTile(row persistencemodels.MapTile) (model.MapTile, error) {
	properties, err := decodeJSON[map[string]any](row.PropertiesJSON)
	if err != nil {
		return model.MapTile{}, err
	}
	return model.MapTile{
		ID:          row.ID,
		ProjectID:   row.ProjectID,
		MapID:       row.MapID,
		X:           row.X,
		Y:           row.Y,
		Altitude:    row.Altitude,
		Temperature: row.Temperature,
		Humidity:    row.Humidity,
		IsOcean:     row.IsOcean,
		Terrain:     row.Terrain,
		Properties:  properties,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}, nil
}

func mapAreaRowFromModel(area model.MapArea, now time.Time) (persistencemodels.MapArea, error) {
	propertiesJSON, err := encodeJSON(area.Properties)
	if err != nil {
		return persistencemodels.MapArea{}, err
	}
	if area.CreatedAt.IsZero() {
		area.CreatedAt = now
	}
	if area.UpdatedAt.IsZero() {
		area.UpdatedAt = now
	}
	return persistencemodels.MapArea{
		ID:              area.ID,
		ProjectID:       area.ProjectID,
		MapID:           area.MapID,
		ParentAreaID:    area.ParentAreaID,
		Name:            area.Name,
		Level:           area.Level,
		MinX:            area.MinX,
		MinY:            area.MinY,
		MaxX:            area.MaxX,
		MaxY:            area.MaxY,
		CenterX:         area.CenterX,
		CenterY:         area.CenterY,
		DominantTerrain: area.DominantTerrain,
		Status:          area.Status,
		PropertiesJSON:  propertiesJSON,
		CreatedAt:       area.CreatedAt,
		UpdatedAt:       area.UpdatedAt,
	}, nil
}

func toMapArea(row persistencemodels.MapArea) (model.MapArea, error) {
	properties, err := decodeJSON[map[string]any](row.PropertiesJSON)
	if err != nil {
		return model.MapArea{}, err
	}
	return model.MapArea{
		ID:              row.ID,
		ProjectID:       row.ProjectID,
		MapID:           row.MapID,
		ParentAreaID:    row.ParentAreaID,
		Name:            row.Name,
		Level:           row.Level,
		MinX:            row.MinX,
		MinY:            row.MinY,
		MaxX:            row.MaxX,
		MaxY:            row.MaxY,
		CenterX:         row.CenterX,
		CenterY:         row.CenterY,
		DominantTerrain: row.DominantTerrain,
		Status:          row.Status,
		Properties:      properties,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
	}, nil
}

func locationStateRowFromModel(location model.LocationState, now time.Time) (persistencemodels.LocationState, error) {
	propertiesJSON, err := encodeJSON(location.Properties)
	if err != nil {
		return persistencemodels.LocationState{}, err
	}
	if location.CreatedAt.IsZero() {
		location.CreatedAt = now
	}
	if location.UpdatedAt.IsZero() {
		location.UpdatedAt = now
	}
	return persistencemodels.LocationState{
		ID:               location.ID,
		ProjectID:        location.ProjectID,
		MapID:            location.MapID,
		AreaID:           location.AreaID,
		RegionID:         location.RegionID,
		ParentLocationID: location.ParentLocationID,
		Name:             location.Name,
		Type:             location.Type,
		Scale:            location.Scale,
		DetailState:      location.DetailState,
		Description:      location.Description,
		X:                location.X,
		Y:                location.Y,
		Radius:           location.Radius,
		Status:           location.Status,
		PropertiesJSON:   propertiesJSON,
		CreatedAt:        location.CreatedAt,
		UpdatedAt:        location.UpdatedAt,
	}, nil
}

func toLocationState(row persistencemodels.LocationState) (model.LocationState, error) {
	properties, err := decodeJSON[map[string]any](row.PropertiesJSON)
	if err != nil {
		return model.LocationState{}, err
	}
	return model.LocationState{
		ID:               row.ID,
		ProjectID:        row.ProjectID,
		MapID:            row.MapID,
		AreaID:           row.AreaID,
		RegionID:         row.RegionID,
		ParentLocationID: row.ParentLocationID,
		Name:             row.Name,
		Type:             row.Type,
		Scale:            row.Scale,
		DetailState:      row.DetailState,
		Description:      row.Description,
		X:                row.X,
		Y:                row.Y,
		Radius:           row.Radius,
		Status:           row.Status,
		Properties:       properties,
		CreatedAt:        row.CreatedAt,
		UpdatedAt:        row.UpdatedAt,
	}, nil
}

func factionInfluenceRowFromModel(influence model.FactionInfluence, now time.Time) persistencemodels.FactionInfluence {
	if influence.CreatedAt.IsZero() {
		influence.CreatedAt = now
	}
	if influence.UpdatedAt.IsZero() {
		influence.UpdatedAt = now
	}
	return persistencemodels.FactionInfluence{
		ID:          influence.ID,
		ProjectID:   influence.ProjectID,
		LocationID:  influence.LocationID,
		FactionName: influence.FactionName,
		Influence:   influence.Influence,
		Attitude:    influence.Attitude,
		Description: influence.Description,
		Status:      influence.Status,
		CreatedAt:   influence.CreatedAt,
		UpdatedAt:   influence.UpdatedAt,
	}
}

func toFactionInfluence(row persistencemodels.FactionInfluence) model.FactionInfluence {
	return model.FactionInfluence{
		ID:          row.ID,
		ProjectID:   row.ProjectID,
		LocationID:  row.LocationID,
		FactionName: row.FactionName,
		Influence:   row.Influence,
		Attitude:    row.Attitude,
		Description: row.Description,
		Status:      row.Status,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}
}

func snapshotFromDelta(event model.StoryEvent) model.WorldSnapshot {
	snapshot := model.WorldSnapshot{
		AtEventID:     event.ID,
		StoryTime:     event.StoryTime,
		WorldState:    map[string]model.WorldStateEntry{},
		Characters:    map[string]model.CharacterRuntimeState{},
		Relationships: map[string]model.Relationship{},
	}
	applyDelta(&snapshot, event)
	return snapshot
}

func applyDelta(snapshot *model.WorldSnapshot, event model.StoryEvent) {
	if snapshot.WorldState == nil {
		snapshot.WorldState = map[string]model.WorldStateEntry{}
	}
	if snapshot.Characters == nil {
		snapshot.Characters = map[string]model.CharacterRuntimeState{}
	}
	if snapshot.Relationships == nil {
		snapshot.Relationships = map[string]model.Relationship{}
	}
	for _, update := range event.StateDelta.WorldStateUpdates {
		if update.Key == "" {
			continue
		}
		entry := snapshot.WorldState[update.Key]
		entry.Key = update.Key
		entry.ProjectID = event.ProjectID
		entry.Value = update.Value
		if update.Note != "" {
			entry.Note = update.Note
		}
		if entry.Status == "" {
			entry.Status = "active"
		}
		entry.UpdatedAt = event.StoryTime
		if update.Operation == "delete" {
			delete(snapshot.WorldState, update.Key)
			continue
		}
		snapshot.WorldState[update.Key] = entry
	}
	for _, move := range event.StateDelta.CharacterMoves {
		if move.CharacterID == "" {
			continue
		}
		state := snapshot.Characters[move.CharacterID]
		state.CharacterID = move.CharacterID
		state.LocationKey = move.LocationKey
		state.X = move.X
		state.Y = move.Y
		if state.Tier == "" {
			state.Tier = "main"
		}
		if state.Status == "" {
			state.Status = "active"
		}
		snapshot.Characters[move.CharacterID] = state
	}
	for _, relationship := range event.StateDelta.RelationshipUpdates {
		id := relationship.PairID
		if id == "" && relationship.Pair != nil {
			id = relationship.Pair.ID
		}
		if id == "" {
			continue
		}
		current := snapshot.Relationships[id]
		if relationship.Pair != nil {
			current.Pair = *relationship.Pair
		}
		if relationship.Summary != "" {
			current.Pair.ID = id
			current.Pair.Summary = relationship.Summary
		}
		current.RecentEvents = append(current.RecentEvents, relationship.Events...)
		snapshot.Relationships[id] = current
	}
	snapshot.Factions = append(snapshot.Factions, factionDeltasAsInfluence(event.ProjectID, event.StoryTime, event.StateDelta.FactionDeltas)...)
	for _, delta := range event.StateDelta.LocationDeltas {
		for i := range snapshot.Locations {
			if snapshot.Locations[i].ID == delta.LocationKey || snapshot.Locations[i].Name == delta.LocationKey {
				if delta.Status != "" {
					snapshot.Locations[i].Status = delta.Status
				}
				if delta.Properties != nil {
					snapshot.Locations[i].Properties = delta.Properties
				}
			}
		}
	}
	snapshot.AtEventID = event.ID
	snapshot.StoryTime = event.StoryTime
}

func applyActionRuntime(snapshot *model.WorldSnapshot, event model.StoryEvent) {
	switch event.Kind {
	case model.EventKindActionScheduled:
		action, ok := actionFromEvent(event)
		if !ok {
			return
		}
		if action.ID == "" {
			action.ID = event.ID
		}
		state := snapshot.Characters[action.CharacterID]
		state.CharacterID = action.CharacterID
		if action.ArriveAt.IsZero() || action.StartAt.IsZero() || !action.ArriveAt.After(action.StartAt) {
			state.LocationKey = action.TargetLocationKey
		}
		if state.Status == "" {
			state.Status = "active"
		}
		state.OngoingAction = &action
		snapshot.Characters[action.CharacterID] = state
	case model.EventKindActionCompleted, model.EventKindActionVoided, model.EventKindActionSuperseded, model.EventKindSceneResolved:
		for _, actorID := range event.ActorIDs {
			state := snapshot.Characters[actorID]
			state.CharacterID = actorID
			state.OngoingAction = nil
			if event.LocationKey != "" {
				state.LocationKey = event.LocationKey
			}
			if state.Status == "" {
				state.Status = "active"
			}
			snapshot.Characters[actorID] = state
		}
	}
}

func factionDeltasAsInfluence(projectID string, at time.Time, deltas []model.FactionDelta) []model.FactionInfluence {
	out := make([]model.FactionInfluence, 0, len(deltas))
	for _, delta := range deltas {
		out = append(out, model.FactionInfluence{
			ProjectID:   projectID,
			LocationID:  delta.LocationID,
			FactionName: delta.FactionName,
			Influence:   delta.Influence,
			Attitude:    delta.Attitude,
			Description: delta.Description,
			Status:      "active",
			CreatedAt:   at,
			UpdatedAt:   at,
		})
	}
	return out
}

func genesisDelta(snapshot model.WorldSnapshot) model.EventStateDelta {
	delta := model.EventStateDelta{}
	for _, entry := range snapshot.WorldState {
		delta.WorldStateUpdates = append(delta.WorldStateUpdates, model.WorldStateUpdate{
			Key:       entry.Key,
			Operation: "set",
			Value:     entry.Value,
			Note:      entry.Note,
		})
	}
	for _, state := range snapshot.Characters {
		delta.CharacterMoves = append(delta.CharacterMoves, model.CharacterMove{
			CharacterID: state.CharacterID,
			LocationKey: state.LocationKey,
			X:           state.X,
			Y:           state.Y,
		})
	}
	return delta
}

func actionFromEvent(event model.StoryEvent) (model.OngoingAction, bool) {
	if event.Payload == nil {
		return model.OngoingAction{}, false
	}
	raw, ok := event.Payload["action"]
	if !ok {
		return model.OngoingAction{}, false
	}
	bytes, err := encodeJSON(raw)
	if err != nil {
		return model.OngoingAction{}, false
	}
	action, err := decodeJSON[model.OngoingAction](bytes)
	if err != nil {
		return model.OngoingAction{}, false
	}
	return action, action.CharacterID != ""
}
