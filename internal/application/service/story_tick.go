package service

import (
	"context"
	"errors"
	"math"
	"sort"
	"time"

	"github.com/fishimei/NovelOS/internal/application/model"
	"github.com/fishimei/NovelOS/internal/application/port"
	"github.com/fishimei/NovelOS/internal/domain"
	"github.com/fishimei/NovelOS/internal/pkgerr"
)

type StoryTickAdvancer struct {
	simulation   port.SimulationRepository
	characters   port.CharacterRepository
	decider      port.CharacterActionDecider
	nearbyRadius int
	tx           port.TxManager
	clock        port.Clock
	ids          port.IDGenerator
}

func NewStoryTickAdvancer(
	simulation port.SimulationRepository,
	characters port.CharacterRepository,
	decider port.CharacterActionDecider,
	tx port.TxManager,
	clock port.Clock,
	ids port.IDGenerator,
	nearbyRadius int,
) *StoryTickAdvancer {
	if nearbyRadius <= 0 {
		nearbyRadius = 25
	}
	return &StoryTickAdvancer{simulation: simulation, characters: characters, decider: decider, nearbyRadius: nearbyRadius, tx: tx, clock: clock, ids: ids}
}

func (a *StoryTickAdvancer) Advance(ctx context.Context, projectID string, input model.AdvanceStoryTickInput) (model.AdvanceStoryTickResult, error) {
	if a.tx == nil {
		return model.AdvanceStoryTickResult{}, pkgerr.Internal("tx manager is required", nil)
	}
	if a.simulation == nil {
		return model.AdvanceStoryTickResult{}, pkgerr.Internal("simulation repository is required", nil)
	}
	if a.characters == nil {
		return model.AdvanceStoryTickResult{}, pkgerr.Internal("character repository is required", nil)
	}
	if a.decider == nil {
		return model.AdvanceStoryTickResult{}, pkgerr.Internal("character action decider is required", nil)
	}
	if input.TickHours <= 0 {
		input.TickHours = 1
	}

	var result model.AdvanceStoryTickResult
	err := a.tx.WithinTransaction(ctx, func(txCtx context.Context) error {
		timeline, err := a.loadOrCreateTimeline(txCtx, projectID)
		if err != nil {
			return err
		}
		fromTime := timeline.CurrentTime
		toTime := fromTime.Add(time.Duration(input.TickHours) * time.Hour)
		run, err := a.simulation.CreateTickRun(txCtx, model.StoryTickRun{
			ID:          generatedID(a.ids, a.clock, "tick"),
			ProjectID:   projectID,
			Tick:        timeline.Tick + 1,
			FromTime:    fromTime,
			ToTime:      toTime,
			Status:      domain.StoryTickStatusRunning,
			CurrentStep: domain.EventStoryTickStarted,
			CreatedAt:   currentTime(a.clock),
			UpdatedAt:   currentTime(a.clock),
		})
		if err != nil {
			return err
		}

		events := make([]model.SimulationEvent, 0)
		event, err := a.appendEvent(txCtx, run, domain.EventStoryTickStarted, "时间线开始推进", "", "", map[string]any{"from_time": fromTime, "to_time": toTime})
		if err != nil {
			return err
		}
		events = append(events, event)

		worldMap, err := a.loadWorldMap(txCtx, projectID)
		if err != nil {
			return err
		}
		tiles, err := a.simulation.ListMapTilesByProjectID(txCtx, projectID)
		if err != nil {
			return err
		}
		locations, err := a.loadOrCreateLocations(txCtx, projectID)
		if err != nil {
			return err
		}
		factions, err := a.simulation.ListFactionInfluencesByProjectID(txCtx, projectID)
		if err != nil {
			return err
		}
		characterList, err := a.listCharacters(txCtx, projectID)
		if err != nil {
			return err
		}
		states, err := a.loadOrCreateCharacterStates(txCtx, projectID, characterList, locations[0])
		if err != nil {
			return err
		}

		locationByID := make(map[string]model.LocationState, len(locations))
		for _, location := range locations {
			locationByID[location.ID] = location
		}
		stateByCharacterID := make(map[string]model.CharacterSimulationState, len(states))
		for _, state := range states {
			stateByCharacterID[state.CharacterID] = state
		}

		updatedStates := make([]model.CharacterSimulationState, 0, len(characterList))
		for _, character := range characterList {
			state := stateByCharacterID[character.ID]
			if state.ID == "" {
				state = a.newCharacterState(projectID, character.ID, locations[0])
			}
			if state.LocationID == "" {
				state.LocationID = locations[0].ID
				state.X = locations[0].X
				state.Y = locations[0].Y
			}

			if state.OngoingAction != nil && !state.OngoingAction.EndsAt.After(fromTime) {
				completed := *state.OngoingAction
				completed.Status = domain.CharacterActionStatusCompleted
				event, err := a.appendEvent(txCtx, run, domain.EventCharacterActionCompleted, completed.Description, character.ID, state.LocationID, map[string]any{"action": completed})
				if err != nil {
					return err
				}
				events = append(events, event)
				state.OngoingAction = nil
			}

			if state.OngoingAction != nil && state.OngoingAction.EndsAt.After(fromTime) {
				event, err := a.appendEvent(txCtx, run, domain.EventCharacterSkippedOngoingAction, state.OngoingAction.Description, character.ID, state.LocationID, map[string]any{"action": state.OngoingAction})
				if err != nil {
					return err
				}
				events = append(events, event)
				updatedStates = append(updatedStates, state)
				continue
			}

			location := locationByID[state.LocationID]
			if location.ID == "" {
				location = locations[0]
				state.LocationID = location.ID
				state.X = location.X
				state.Y = location.Y
			}
			decision, err := a.decider.Decide(txCtx, model.CharacterActionDecisionInput{
				Timeline:          timeline,
				Character:         character,
				CharacterState:    state,
				Location:          location,
				FactionInfluences: influencesForLocation(factions, location.ID),
				NearbyLocations:   nearbyLocations(state, locations, factions, a.nearbyRadius),
			})
			if err != nil {
				return err
			}
			if decision.DurationHours <= 0 {
				return pkgerr.Validation("character action duration_hours must be greater than zero")
			}
			state.OngoingAction = &model.CharacterOngoingAction{
				ActionType:  firstNonEmpty(decision.ActionType, "act"),
				Description: decision.Description,
				StartedAt:   fromTime,
				EndsAt:      fromTime.Add(time.Duration(decision.DurationHours) * time.Hour),
				Status:      domain.CharacterActionStatusOngoing,
				Rationale:   decision.Rationale,
			}
			event, err := a.appendEvent(txCtx, run, domain.EventCharacterActionStarted, state.OngoingAction.Description, character.ID, state.LocationID, map[string]any{"action": state.OngoingAction})
			if err != nil {
				return err
			}
			events = append(events, event)
			updatedStates = append(updatedStates, state)
		}

		if err := a.simulation.UpsertCharacterStates(txCtx, projectID, updatedStates); err != nil {
			return err
		}
		timeline.CurrentTime = toTime
		timeline.Tick++
		timeline.UpdatedAt = currentTime(a.clock)
		timeline, err = a.simulation.UpsertTimeline(txCtx, timeline)
		if err != nil {
			return err
		}
		run.Status = domain.StoryTickStatusCompleted
		run.CurrentStep = domain.EventStoryTickCompleted
		run.UpdatedAt = currentTime(a.clock)
		run, err = a.simulation.UpdateTickRun(txCtx, run)
		if err != nil {
			return err
		}
		event, err = a.appendEvent(txCtx, run, domain.EventStoryTickCompleted, "时间线推进完成", "", "", map[string]any{"tick": timeline.Tick, "current_time": timeline.CurrentTime})
		if err != nil {
			return err
		}
		events = append(events, event)

		state := model.StorySimulationState{
			Timeline:        timeline,
			Map:             worldMap,
			Tiles:           tiles,
			Locations:       locations,
			Factions:        factions,
			CharacterStates: updatedStates,
			Characters:      characterList,
			LatestEvents:    events,
		}
		snapshot, err := a.simulation.CreateSnapshot(txCtx, model.SimulationSnapshot{
			ID:        generatedID(a.ids, a.clock, "snapshot"),
			ProjectID: projectID,
			TickRunID: run.ID,
			Tick:      timeline.Tick,
			Snapshot:  state,
			CreatedAt: currentTime(a.clock),
		})
		if err != nil {
			return err
		}
		result = model.AdvanceStoryTickResult{Run: run, Events: events, Snapshot: snapshot, State: state}
		return nil
	})
	if err != nil {
		return model.AdvanceStoryTickResult{}, err
	}
	return result, nil
}

func (a *StoryTickAdvancer) CurrentState(ctx context.Context, projectID string) (model.StorySimulationState, error) {
	timeline, err := a.loadOrCreateTimeline(ctx, projectID)
	if err != nil {
		return model.StorySimulationState{}, err
	}
	worldMap, err := a.loadWorldMap(ctx, projectID)
	if err != nil {
		return model.StorySimulationState{}, err
	}
	tiles, err := a.simulation.ListMapTilesByProjectID(ctx, projectID)
	if err != nil {
		return model.StorySimulationState{}, err
	}
	locations, err := a.loadOrCreateLocations(ctx, projectID)
	if err != nil {
		return model.StorySimulationState{}, err
	}
	factions, err := a.simulation.ListFactionInfluencesByProjectID(ctx, projectID)
	if err != nil {
		return model.StorySimulationState{}, err
	}
	characters, err := a.listCharacters(ctx, projectID)
	if err != nil {
		return model.StorySimulationState{}, err
	}
	states, err := a.loadOrCreateCharacterStates(ctx, projectID, characters, locations[0])
	if err != nil {
		return model.StorySimulationState{}, err
	}
	return model.StorySimulationState{Timeline: timeline, Map: worldMap, Tiles: tiles, Locations: locations, Factions: factions, CharacterStates: states, Characters: characters}, nil
}

func (a *StoryTickAdvancer) loadOrCreateTimeline(ctx context.Context, projectID string) (model.StoryTimeline, error) {
	timeline, err := a.simulation.GetTimelineByProjectID(ctx, projectID)
	if err == nil {
		return timeline, nil
	}
	if !isNotFound(err) {
		return model.StoryTimeline{}, err
	}
	now := currentTime(a.clock)
	return a.simulation.UpsertTimeline(ctx, model.StoryTimeline{
		ID:          generatedID(a.ids, a.clock, "timeline"),
		ProjectID:   projectID,
		CurrentTime: now,
		Tick:        0,
		CreatedAt:   now,
		UpdatedAt:   now,
	})
}

func (a *StoryTickAdvancer) loadWorldMap(ctx context.Context, projectID string) (*model.WorldMap, error) {
	worldMap, err := a.simulation.GetWorldMapByProjectID(ctx, projectID)
	if err == nil {
		return &worldMap, nil
	}
	if isNotFound(err) {
		return nil, nil
	}
	return nil, err
}

func (a *StoryTickAdvancer) loadOrCreateLocations(ctx context.Context, projectID string) ([]model.LocationState, error) {
	locations, err := a.simulation.ListLocationsByProjectID(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if len(locations) > 0 {
		return locations, nil
	}
	now := currentTime(a.clock)
	location := model.LocationState{
		ID:          generatedID(a.ids, a.clock, "location"),
		ProjectID:   projectID,
		Name:        "初始地点",
		Type:        "origin",
		Description: "故事模拟的默认起点",
		Status:      "active",
		Properties:  map[string]any{},
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := a.simulation.UpsertLocations(ctx, projectID, []model.LocationState{location}); err != nil {
		return nil, err
	}
	return []model.LocationState{location}, nil
}

func (a *StoryTickAdvancer) listCharacters(ctx context.Context, projectID string) ([]model.Character, error) {
	result, err := a.characters.ListByProjectID(ctx, projectID, model.PageInput{Page: 1, PageSize: 100})
	if err != nil {
		return nil, err
	}
	characters := result.Items
	sort.SliceStable(characters, func(i, j int) bool { return characters[i].CreatedAt.Before(characters[j].CreatedAt) })
	return characters, nil
}

func (a *StoryTickAdvancer) loadOrCreateCharacterStates(ctx context.Context, projectID string, characters []model.Character, location model.LocationState) ([]model.CharacterSimulationState, error) {
	states, err := a.simulation.ListCharacterStatesByProjectID(ctx, projectID)
	if err != nil {
		return nil, err
	}
	byCharacterID := make(map[string]model.CharacterSimulationState, len(states))
	for _, state := range states {
		byCharacterID[state.CharacterID] = state
	}
	created := make([]model.CharacterSimulationState, 0)
	for _, character := range characters {
		if byCharacterID[character.ID].ID != "" {
			continue
		}
		state := a.newCharacterState(projectID, character.ID, location)
		byCharacterID[character.ID] = state
		created = append(created, state)
	}
	if err := a.simulation.UpsertCharacterStates(ctx, projectID, created); err != nil {
		return nil, err
	}
	ordered := make([]model.CharacterSimulationState, 0, len(characters))
	for _, character := range characters {
		ordered = append(ordered, byCharacterID[character.ID])
	}
	return ordered, nil
}

func (a *StoryTickAdvancer) newCharacterState(projectID string, characterID string, location model.LocationState) model.CharacterSimulationState {
	now := currentTime(a.clock)
	return model.CharacterSimulationState{
		ID:          generatedID(a.ids, a.clock, "cstate"),
		ProjectID:   projectID,
		CharacterID: characterID,
		LocationID:  location.ID,
		X:           location.X,
		Y:           location.Y,
		Status:      "active",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

func (a *StoryTickAdvancer) appendEvent(ctx context.Context, run model.StoryTickRun, name string, summary string, characterID string, locationID string, payload map[string]any) (model.SimulationEvent, error) {
	return a.simulation.AppendEvent(ctx, model.SimulationEvent{
		ID:          generatedID(a.ids, a.clock, "sevent"),
		ProjectID:   run.ProjectID,
		TickRunID:   run.ID,
		EventName:   name,
		CharacterID: characterID,
		LocationID:  locationID,
		Summary:     summary,
		Payload:     payload,
		OccurredAt:  currentTime(a.clock),
		CreatedAt:   currentTime(a.clock),
	})
}

func nearbyLocations(state model.CharacterSimulationState, locations []model.LocationState, influences []model.FactionInfluence, radius int) []model.NearbyLocationContext {
	nearby := make([]model.NearbyLocationContext, 0)
	for _, location := range locations {
		if location.ID == state.LocationID {
			continue
		}
		distance := math.Hypot(float64(location.X-state.X), float64(location.Y-state.Y))
		if radius > 0 && distance > float64(radius) {
			continue
		}
		nearby = append(nearby, model.NearbyLocationContext{
			Location:          location,
			Distance:          distance,
			FactionInfluences: influencesForLocation(influences, location.ID),
		})
	}
	sort.SliceStable(nearby, func(i, j int) bool { return nearby[i].Distance < nearby[j].Distance })
	return nearby
}

func influencesForLocation(influences []model.FactionInfluence, locationID string) []model.FactionInfluence {
	filtered := make([]model.FactionInfluence, 0)
	for _, influence := range influences {
		if influence.LocationID == locationID {
			filtered = append(filtered, influence)
		}
	}
	return filtered
}

func isNotFound(err error) bool {
	var appErr *pkgerr.Error
	return errors.As(err, &appErr) && appErr.Code == pkgerr.CodeNotFound
}

func (a *StoryTickAdvancer) Events(ctx context.Context, tickRunID string) ([]model.SimulationEvent, error) {
	return a.simulation.ListEventsByTickRunID(ctx, tickRunID)
}

func (a *StoryTickAdvancer) Snapshot(ctx context.Context, tickRunID string) (model.SimulationSnapshot, error) {
	return a.simulation.GetSnapshotByTickRunID(ctx, tickRunID)
}

func (a *StoryTickAdvancer) TickRun(ctx context.Context, tickRunID string) (model.StoryTickRun, error) {
	return a.simulation.GetTickRunByID(ctx, tickRunID)
}
