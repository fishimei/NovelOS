package repository

import (
	"context"
	"errors"
	"time"

	"github.com/fishimei/NovelOS/internal/application/model"
	persistencemodels "github.com/fishimei/NovelOS/internal/infrastructure/persistence/gorm/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type simulationRepository struct {
	*container
}

func (r *simulationRepository) GetTimelineByProjectID(ctx context.Context, projectID string) (model.StoryTimeline, error) {
	var row persistencemodels.StoryTimeline
	if err := r.dbFor(ctx).First(&row, "project_id = ?", projectID).Error; err != nil {
		return model.StoryTimeline{}, mapDBError(err, "story timeline not found")
	}
	return toStoryTimeline(row), nil
}

func (r *simulationRepository) UpsertTimeline(ctx context.Context, timeline model.StoryTimeline) (model.StoryTimeline, error) {
	now := r.now()
	if timeline.ID == "" {
		timeline.ID = r.nextID("timeline")
	}
	if timeline.CreatedAt.IsZero() {
		timeline.CreatedAt = now
	}
	if timeline.UpdatedAt.IsZero() {
		timeline.UpdatedAt = now
	}
	row := persistencemodels.StoryTimeline{
		ID:          timeline.ID,
		ProjectID:   timeline.ProjectID,
		CurrentTime: timeline.CurrentTime,
		Tick:        timeline.Tick,
		CreatedAt:   timeline.CreatedAt,
		UpdatedAt:   timeline.UpdatedAt,
	}
	if err := r.dbFor(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "project_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"current_time", "tick", "updated_at"}),
	}).Create(&row).Error; err != nil {
		return model.StoryTimeline{}, mapDBError(err, "story timeline not found")
	}
	return r.GetTimelineByProjectID(ctx, timeline.ProjectID)
}

func (r *simulationRepository) GetWorldMapByProjectID(ctx context.Context, projectID string) (model.WorldMap, error) {
	var row persistencemodels.WorldMap
	if err := r.dbFor(ctx).First(&row, "project_id = ?", projectID).Error; err != nil {
		return model.WorldMap{}, mapDBError(err, "world map not found")
	}
	return toWorldMap(row)
}

func (r *simulationRepository) UpsertWorldMap(ctx context.Context, worldMap model.WorldMap) (model.WorldMap, error) {
	now := r.now()
	if worldMap.ID == "" {
		worldMap.ID = r.nextID("map")
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
	if err := r.dbFor(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "project_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"name", "seed", "width", "height", "status", "properties_json", "updated_at"}),
	}).Create(&row).Error; err != nil {
		return model.WorldMap{}, mapDBError(err, "world map not found")
	}
	return r.GetWorldMapByProjectID(ctx, worldMap.ProjectID)
}

func (r *simulationRepository) ListMapTilesByProjectID(ctx context.Context, projectID string) ([]model.MapTile, error) {
	var rows []persistencemodels.MapTile
	if err := r.dbFor(ctx).Where("project_id = ?", projectID).Order("y asc, x asc").Find(&rows).Error; err != nil {
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

func (r *simulationRepository) UpsertMapTiles(ctx context.Context, projectID string, tiles []model.MapTile) error {
	if len(tiles) == 0 {
		return nil
	}
	now := r.now()
	rows := make([]persistencemodels.MapTile, 0, len(tiles))
	for _, tile := range tiles {
		if tile.ID == "" {
			tile.ID = r.nextID("tile")
		}
		tile.ProjectID = projectID
		row, err := mapTileRowFromModel(tile, now)
		if err != nil {
			return payloadError("map tile", err)
		}
		rows = append(rows, row)
	}
	return mapDBError(r.dbFor(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "map_id"}, {Name: "x"}, {Name: "y"}},
		DoUpdates: clause.AssignmentColumns([]string{"project_id", "altitude", "temperature", "humidity", "is_ocean", "terrain", "properties_json", "updated_at"}),
	}).Create(&rows).Error, "map tile not found")
}

func (r *simulationRepository) CreateTickRun(ctx context.Context, run model.StoryTickRun) (model.StoryTickRun, error) {
	now := r.now()
	if run.ID == "" {
		run.ID = r.nextID("tick")
	}
	if run.CreatedAt.IsZero() {
		run.CreatedAt = now
	}
	if run.UpdatedAt.IsZero() {
		run.UpdatedAt = now
	}
	row := storyTickRunRow(run)
	if err := r.dbFor(ctx).Create(&row).Error; err != nil {
		return model.StoryTickRun{}, mapDBError(err, "story tick run not found")
	}
	return toStoryTickRun(row), nil
}

func (r *simulationRepository) UpdateTickRun(ctx context.Context, run model.StoryTickRun) (model.StoryTickRun, error) {
	run.UpdatedAt = r.now()
	updates := map[string]any{
		"tick":         run.Tick,
		"from_time":    run.FromTime,
		"to_time":      run.ToTime,
		"status":       run.Status,
		"current_step": run.CurrentStep,
		"error":        run.Error,
		"updated_at":   run.UpdatedAt,
	}
	if err := r.dbFor(ctx).Model(&persistencemodels.StoryTickRun{}).Where("id = ?", run.ID).Updates(updates).Error; err != nil {
		return model.StoryTickRun{}, mapDBError(err, "story tick run not found")
	}
	return r.GetTickRunByID(ctx, run.ID)
}

func (r *simulationRepository) GetTickRunByID(ctx context.Context, tickRunID string) (model.StoryTickRun, error) {
	var row persistencemodels.StoryTickRun
	if err := r.dbFor(ctx).First(&row, "id = ?", tickRunID).Error; err != nil {
		return model.StoryTickRun{}, mapDBError(err, "story tick run not found")
	}
	return toStoryTickRun(row), nil
}

func (r *simulationRepository) ListLocationsByProjectID(ctx context.Context, projectID string) ([]model.LocationState, error) {
	var rows []persistencemodels.LocationState
	if err := r.dbFor(ctx).Where("project_id = ?", projectID).Order("created_at asc").Find(&rows).Error; err != nil {
		return nil, mapDBError(err, "location state not found")
	}
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

func (r *simulationRepository) UpsertLocations(ctx context.Context, projectID string, locations []model.LocationState) error {
	if len(locations) == 0 {
		return nil
	}
	now := r.now()
	rows := make([]persistencemodels.LocationState, 0, len(locations))
	for _, location := range locations {
		if location.ID == "" {
			location.ID = r.nextID("location")
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
	return mapDBError(r.dbFor(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"project_id", "map_id", "region_id", "name", "type", "description", "x", "y", "radius", "status", "properties_json", "updated_at"}),
	}).Create(&rows).Error, "location state not found")
}

func (r *simulationRepository) ListFactionInfluencesByProjectID(ctx context.Context, projectID string) ([]model.FactionInfluence, error) {
	var rows []persistencemodels.FactionInfluence
	if err := r.dbFor(ctx).Where("project_id = ?", projectID).Order("created_at asc").Find(&rows).Error; err != nil {
		return nil, mapDBError(err, "faction influence not found")
	}
	influences := make([]model.FactionInfluence, 0, len(rows))
	for _, row := range rows {
		influences = append(influences, toFactionInfluence(row))
	}
	return influences, nil
}

func (r *simulationRepository) UpsertFactionInfluences(ctx context.Context, projectID string, influences []model.FactionInfluence) error {
	if len(influences) == 0 {
		return nil
	}
	now := r.now()
	rows := make([]persistencemodels.FactionInfluence, 0, len(influences))
	for _, influence := range influences {
		if influence.ID == "" {
			influence.ID = r.nextID("faction")
		}
		influence.ProjectID = projectID
		if influence.Status == "" {
			influence.Status = "active"
		}
		rows = append(rows, factionInfluenceRowFromModel(influence, now))
	}
	return mapDBError(r.dbFor(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"project_id", "location_id", "faction_name", "influence", "attitude", "description", "status", "updated_at"}),
	}).Create(&rows).Error, "faction influence not found")
}

func (r *simulationRepository) ListCharacterStatesByProjectID(ctx context.Context, projectID string) ([]model.CharacterSimulationState, error) {
	var rows []persistencemodels.CharacterSimulationState
	if err := r.dbFor(ctx).Where("project_id = ?", projectID).Order("created_at asc").Find(&rows).Error; err != nil {
		return nil, mapDBError(err, "character simulation state not found")
	}
	states := make([]model.CharacterSimulationState, 0, len(rows))
	for _, row := range rows {
		state, err := toCharacterSimulationState(row)
		if err != nil {
			return nil, payloadError("character simulation state", err)
		}
		states = append(states, state)
	}
	return states, nil
}

func (r *simulationRepository) UpsertCharacterStates(ctx context.Context, projectID string, states []model.CharacterSimulationState) error {
	if len(states) == 0 {
		return nil
	}
	now := r.now()
	rows := make([]persistencemodels.CharacterSimulationState, 0, len(states))
	for _, state := range states {
		if state.ID == "" {
			state.ID = r.nextID("cstate")
		}
		state.ProjectID = projectID
		if state.Status == "" {
			state.Status = "active"
		}
		row, err := characterSimulationStateRowFromModel(state, now)
		if err != nil {
			return payloadError("character simulation state", err)
		}
		rows = append(rows, row)
	}
	return mapDBError(r.dbFor(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "project_id"}, {Name: "character_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"location_id", "x", "y", "status", "ongoing_action_json", "updated_at"}),
	}).Create(&rows).Error, "character simulation state not found")
}

func (r *simulationRepository) AppendEvent(ctx context.Context, event model.SimulationEvent) (model.SimulationEvent, error) {
	if event.ID == "" {
		event.ID = r.nextID("sevent")
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = r.now()
	}
	if event.OccurredAt.IsZero() {
		event.OccurredAt = event.CreatedAt
	}
	if event.Sequence == 0 {
		sequence, err := currentSimulationSequence(ctx, r.dbFor(ctx), event.TickRunID)
		if err != nil {
			return model.SimulationEvent{}, mapDBError(err, "simulation event not found")
		}
		event.Sequence = sequence + 1
	}
	row, err := simulationEventRowFromModel(event, r.now())
	if err != nil {
		return model.SimulationEvent{}, payloadError("simulation event", err)
	}
	if err := r.dbFor(ctx).Create(&row).Error; err != nil {
		return model.SimulationEvent{}, mapDBError(err, "simulation event not found")
	}
	return toSimulationEvent(row)
}

func (r *simulationRepository) ListEventsByTickRunID(ctx context.Context, tickRunID string) ([]model.SimulationEvent, error) {
	var rows []persistencemodels.SimulationEvent
	if err := r.dbFor(ctx).Where("tick_run_id = ?", tickRunID).Order("sequence asc").Find(&rows).Error; err != nil {
		return nil, mapDBError(err, "simulation event not found")
	}
	events := make([]model.SimulationEvent, 0, len(rows))
	for _, row := range rows {
		event, err := toSimulationEvent(row)
		if err != nil {
			return nil, payloadError("simulation event", err)
		}
		events = append(events, event)
	}
	return events, nil
}

func (r *simulationRepository) CreateSnapshot(ctx context.Context, snapshot model.SimulationSnapshot) (model.SimulationSnapshot, error) {
	if snapshot.ID == "" {
		snapshot.ID = r.nextID("snapshot")
	}
	if snapshot.CreatedAt.IsZero() {
		snapshot.CreatedAt = r.now()
	}
	row, err := simulationSnapshotRowFromModel(snapshot, r.now())
	if err != nil {
		return model.SimulationSnapshot{}, payloadError("simulation snapshot", err)
	}
	if err := r.dbFor(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "tick_run_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"project_id", "tick", "snapshot_json", "created_at"}),
	}).Create(&row).Error; err != nil {
		return model.SimulationSnapshot{}, mapDBError(err, "simulation snapshot not found")
	}
	return r.GetSnapshotByTickRunID(ctx, snapshot.TickRunID)
}

func (r *simulationRepository) GetSnapshotByTickRunID(ctx context.Context, tickRunID string) (model.SimulationSnapshot, error) {
	var row persistencemodels.SimulationSnapshot
	if err := r.dbFor(ctx).First(&row, "tick_run_id = ?", tickRunID).Error; err != nil {
		return model.SimulationSnapshot{}, mapDBError(err, "simulation snapshot not found")
	}
	return toSimulationSnapshot(row)
}

func currentSimulationSequence(ctx context.Context, db *gorm.DB, tickRunID string) (int, error) {
	var row persistencemodels.SimulationEvent
	if err := db.WithContext(ctx).Where("tick_run_id = ?", tickRunID).Order("sequence desc").Take(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, nil
		}
		return 0, err
	}
	return row.Sequence, nil
}

func toStoryTimeline(row persistencemodels.StoryTimeline) model.StoryTimeline {
	return model.StoryTimeline{
		ID:          row.ID,
		ProjectID:   row.ProjectID,
		CurrentTime: row.CurrentTime,
		Tick:        row.Tick,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
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

func storyTickRunRow(run model.StoryTickRun) persistencemodels.StoryTickRun {
	return persistencemodels.StoryTickRun{
		ID:          run.ID,
		ProjectID:   run.ProjectID,
		Tick:        run.Tick,
		FromTime:    run.FromTime,
		ToTime:      run.ToTime,
		Status:      run.Status,
		CurrentStep: run.CurrentStep,
		Error:       run.Error,
		CreatedAt:   run.CreatedAt,
		UpdatedAt:   run.UpdatedAt,
	}
}

func toStoryTickRun(row persistencemodels.StoryTickRun) model.StoryTickRun {
	return model.StoryTickRun{
		ID:          row.ID,
		ProjectID:   row.ProjectID,
		Tick:        row.Tick,
		FromTime:    row.FromTime,
		ToTime:      row.ToTime,
		Status:      row.Status,
		CurrentStep: row.CurrentStep,
		Error:       row.Error,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}
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
		ID:             location.ID,
		ProjectID:      location.ProjectID,
		MapID:          location.MapID,
		RegionID:       location.RegionID,
		Name:           location.Name,
		Type:           location.Type,
		Description:    location.Description,
		X:              location.X,
		Y:              location.Y,
		Radius:         location.Radius,
		Status:         location.Status,
		PropertiesJSON: propertiesJSON,
		CreatedAt:      location.CreatedAt,
		UpdatedAt:      location.UpdatedAt,
	}, nil
}

func toLocationState(row persistencemodels.LocationState) (model.LocationState, error) {
	properties, err := decodeJSON[map[string]any](row.PropertiesJSON)
	if err != nil {
		return model.LocationState{}, err
	}
	return model.LocationState{
		ID:          row.ID,
		ProjectID:   row.ProjectID,
		MapID:       row.MapID,
		RegionID:    row.RegionID,
		Name:        row.Name,
		Type:        row.Type,
		Description: row.Description,
		X:           row.X,
		Y:           row.Y,
		Radius:      row.Radius,
		Status:      row.Status,
		Properties:  properties,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
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

func characterSimulationStateRowFromModel(state model.CharacterSimulationState, now time.Time) (persistencemodels.CharacterSimulationState, error) {
	actionJSON, err := encodeJSON(state.OngoingAction)
	if err != nil {
		return persistencemodels.CharacterSimulationState{}, err
	}
	if state.CreatedAt.IsZero() {
		state.CreatedAt = now
	}
	if state.UpdatedAt.IsZero() {
		state.UpdatedAt = now
	}
	return persistencemodels.CharacterSimulationState{
		ID:                state.ID,
		ProjectID:         state.ProjectID,
		CharacterID:       state.CharacterID,
		LocationID:        state.LocationID,
		X:                 state.X,
		Y:                 state.Y,
		Status:            state.Status,
		OngoingActionJSON: actionJSON,
		CreatedAt:         state.CreatedAt,
		UpdatedAt:         state.UpdatedAt,
	}, nil
}

func toCharacterSimulationState(row persistencemodels.CharacterSimulationState) (model.CharacterSimulationState, error) {
	action, err := decodeJSON[*model.CharacterOngoingAction](row.OngoingActionJSON)
	if err != nil {
		return model.CharacterSimulationState{}, err
	}
	return model.CharacterSimulationState{
		ID:            row.ID,
		ProjectID:     row.ProjectID,
		CharacterID:   row.CharacterID,
		LocationID:    row.LocationID,
		X:             row.X,
		Y:             row.Y,
		Status:        row.Status,
		OngoingAction: action,
		CreatedAt:     row.CreatedAt,
		UpdatedAt:     row.UpdatedAt,
	}, nil
}

func simulationEventRowFromModel(event model.SimulationEvent, now time.Time) (persistencemodels.SimulationEvent, error) {
	payloadJSON, err := encodeJSON(event.Payload)
	if err != nil {
		return persistencemodels.SimulationEvent{}, err
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = now
	}
	if event.OccurredAt.IsZero() {
		event.OccurredAt = event.CreatedAt
	}
	return persistencemodels.SimulationEvent{
		ID:          event.ID,
		ProjectID:   event.ProjectID,
		TickRunID:   event.TickRunID,
		EventName:   event.EventName,
		Sequence:    event.Sequence,
		CharacterID: event.CharacterID,
		LocationID:  event.LocationID,
		Summary:     event.Summary,
		PayloadJSON: payloadJSON,
		OccurredAt:  event.OccurredAt,
		CreatedAt:   event.CreatedAt,
	}, nil
}

func toSimulationEvent(row persistencemodels.SimulationEvent) (model.SimulationEvent, error) {
	payload, err := decodeJSON[map[string]any](row.PayloadJSON)
	if err != nil {
		return model.SimulationEvent{}, err
	}
	return model.SimulationEvent{
		ID:          row.ID,
		ProjectID:   row.ProjectID,
		TickRunID:   row.TickRunID,
		EventName:   row.EventName,
		Sequence:    row.Sequence,
		CharacterID: row.CharacterID,
		LocationID:  row.LocationID,
		Summary:     row.Summary,
		Payload:     payload,
		OccurredAt:  row.OccurredAt,
		CreatedAt:   row.CreatedAt,
	}, nil
}

func simulationSnapshotRowFromModel(snapshot model.SimulationSnapshot, now time.Time) (persistencemodels.SimulationSnapshot, error) {
	snapshotJSON, err := encodeJSON(snapshot.Snapshot)
	if err != nil {
		return persistencemodels.SimulationSnapshot{}, err
	}
	if snapshot.CreatedAt.IsZero() {
		snapshot.CreatedAt = now
	}
	return persistencemodels.SimulationSnapshot{
		ID:           snapshot.ID,
		ProjectID:    snapshot.ProjectID,
		TickRunID:    snapshot.TickRunID,
		Tick:         snapshot.Tick,
		SnapshotJSON: snapshotJSON,
		CreatedAt:    snapshot.CreatedAt,
	}, nil
}

func toSimulationSnapshot(row persistencemodels.SimulationSnapshot) (model.SimulationSnapshot, error) {
	snapshot, err := decodeJSON[model.StorySimulationState](row.SnapshotJSON)
	if err != nil {
		return model.SimulationSnapshot{}, err
	}
	return model.SimulationSnapshot{
		ID:        row.ID,
		ProjectID: row.ProjectID,
		TickRunID: row.TickRunID,
		Tick:      row.Tick,
		Snapshot:  snapshot,
		CreatedAt: row.CreatedAt,
	}, nil
}
