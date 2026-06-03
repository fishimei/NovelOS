package service

import (
	"context"

	"github.com/fishimei/NovelOS/internal/application/model"
	"github.com/fishimei/NovelOS/internal/application/port"
	"github.com/fishimei/NovelOS/internal/pkgerr"
)

// SetupRunApplier 负责将已接受的设置草稿转换为正式项目状态。
// 这是一个跨多个仓库的复杂业务操作，需要在事务中执行以保证一致性。
type SetupRunApplier struct {
	sessions         port.SetupSessionRepository
	authorBibles     port.AuthorBibleRepository
	worldState       port.WorldStateRepository
	characters       port.CharacterRepository
	relationships    port.RelationshipRepository
	audit            port.AuditRepository
	events           port.StoryEventStore
	worldInitializer port.WorldInitializer
	world            WorldInitializationSettings
	tx               port.TxManager
	clock            port.Clock
	ids              port.IDGenerator
}

type WorldInitializationSettings struct {
	Enabled       bool
	Seed          string
	LocationCount int
	MapWidth      int
	MapHeight     int
}

func NewSetupRunApplier(
	sessions port.SetupSessionRepository,
	authorBibles port.AuthorBibleRepository,
	worldState port.WorldStateRepository,
	characters port.CharacterRepository,
	relationships port.RelationshipRepository,
	audit port.AuditRepository,
	tx port.TxManager,
	clock port.Clock,
	ids port.IDGenerator,
	events port.StoryEventStore,
	worldInitializer port.WorldInitializer,
	world WorldInitializationSettings,
) *SetupRunApplier {
	return &SetupRunApplier{
		sessions:         sessions,
		authorBibles:     authorBibles,
		worldState:       worldState,
		characters:       characters,
		relationships:    relationships,
		audit:            audit,
		events:           events,
		worldInitializer: worldInitializer,
		world:            world,
		tx:               tx,
		clock:            clock,
		ids:              ids,
	}
}

func (a *SetupRunApplier) Apply(ctx context.Context, sessionID string, input model.ApplySetupRunInput) (model.ApplySetupRunResult, error) {
	if a.tx == nil {
		return model.ApplySetupRunResult{}, pkgerr.Internal("tx manager is required", nil)
	}
	run, err := a.sessions.GetRunByID(ctx, input.RunID)
	if err != nil {
		return model.ApplySetupRunResult{}, err
	}
	if run.SessionID != sessionID {
		return model.ApplySetupRunResult{}, pkgerr.Validation("run does not belong to session")
	}
	result, err := a.sessions.GetRunResultByID(ctx, input.RunID)
	if err != nil {
		return model.ApplySetupRunResult{}, err
	}

	err = a.tx.WithinTransaction(ctx, func(txCtx context.Context) error {
		if input.AcceptAuthorBible {
			if err := a.applyAuthorBible(txCtx, run, result.SetupDraft.AuthorBible); err != nil {
				return err
			}
		}
		if input.AcceptWorldState {
			if err := a.applyWorldState(txCtx, run, result.SetupDraft.WorldState); err != nil {
				return err
			}
		}
		if input.AcceptCharacters {
			if err := a.applyCharacters(txCtx, run, result.SetupDraft.Characters); err != nil {
				return err
			}
		}
		if input.AcceptRelationships {
			if err := a.applyRelationships(txCtx, run, result.SetupDraft.Relationships); err != nil {
				return err
			}
		}
		if err := a.applyWorldInitialization(txCtx, run, result.SetupDraft); err != nil {
			return err
		}
		if err := a.appendAppliedEvent(txCtx, sessionID, run, input); err != nil {
			return err
		}
		return a.sessions.MarkApplied(txCtx, sessionID, run.RunID)
	})
	if err != nil {
		return model.ApplySetupRunResult{}, err
	}

	return model.ApplySetupRunResult{
		ProjectID: run.ProjectID,
		RunID:     run.RunID,
		Status:    "applied",
	}, nil
}

func (a *SetupRunApplier) applyAuthorBible(ctx context.Context, run model.SetupRun, bible model.AuthorBible) error {
	if bible.ID == "" {
		bible.ID = generatedID(a.ids, a.clock, "bible")
	}
	bible.ProjectID = run.ProjectID
	if bible.Status == "" {
		bible.Status = "active"
	}
	if _, err := a.authorBibles.Upsert(ctx, bible); err != nil {
		return err
	}
	return a.writeRevision(ctx, run.ProjectID, "author_bible", bible.ID, run.RunID, bible)
}

func (a *SetupRunApplier) applyWorldState(ctx context.Context, run model.SetupRun, entries []model.WorldStateEntry) error {
	for i := range entries {
		if entries[i].ID == "" {
			entries[i].ID = generatedID(a.ids, a.clock, "world")
		}
		entries[i].ProjectID = run.ProjectID
		if entries[i].Status == "" {
			entries[i].Status = "active"
		}
	}
	if err := a.worldState.UpsertEntries(ctx, run.ProjectID, entries); err != nil {
		return err
	}
	for _, entry := range entries {
		if err := a.writeRevision(ctx, run.ProjectID, "world_state_entry", entry.ID, run.RunID, entry); err != nil {
			return err
		}
	}
	return nil
}

func (a *SetupRunApplier) applyCharacters(ctx context.Context, run model.SetupRun, characters []model.Character) error {
	for _, character := range characters {
		if character.ID == "" {
			character.ID = generatedID(a.ids, a.clock, "character")
		}
		character.ProjectID = run.ProjectID
		if character.Status == "" {
			character.Status = "active"
		}
		now := currentTime(a.clock)
		character.CreatedAt = now
		character.UpdatedAt = now
		saved, err := a.characters.Upsert(ctx, character)
		if err != nil {
			return err
		}
		if err := a.writeRevision(ctx, run.ProjectID, "character", saved.ID, run.RunID, saved); err != nil {
			return err
		}
	}
	return nil
}

func (a *SetupRunApplier) applyWorldInitialization(ctx context.Context, run model.SetupRun, draft model.SetupDraft) error {
	if !a.world.Enabled || a.events == nil || a.worldInitializer == nil {
		return nil
	}
	if _, err := a.events.GetWorldMapByProjectID(ctx, run.ProjectID); err == nil {
		return nil
	} else if !isNotFound(err) {
		return err
	}
	locations, err := a.events.ListLocationsByProjectID(ctx, run.ProjectID)
	if err != nil {
		return err
	}
	if len(locations) > 0 {
		return nil
	}
	characters, err := a.characters.ListByProjectID(ctx, run.ProjectID, model.PageInput{Page: 1, PageSize: 100})
	if err != nil {
		return err
	}
	now := currentTime(a.clock)
	result, err := a.worldInitializer.Initialize(ctx, port.WorldInitializationInput{
		ProjectID:     run.ProjectID,
		Seed:          a.world.Seed,
		LocationCount: a.world.LocationCount,
		MapWidth:      a.world.MapWidth,
		MapHeight:     a.world.MapHeight,
		SetupRun:      run,
		SetupDraft:    draft,
		Characters:    characters.Items,
		CurrentTime:   now,
	})
	if err != nil {
		return err
	}
	if _, err := a.events.UpsertWorldMap(ctx, result.Map); err != nil {
		return err
	}
	if err := a.events.UpsertMapTiles(ctx, run.ProjectID, result.Tiles); err != nil {
		return err
	}
	if err := a.events.UpsertLocations(ctx, run.ProjectID, result.Locations); err != nil {
		return err
	}
	if err := a.events.UpsertFactionInfluences(ctx, run.ProjectID, result.Factions); err != nil {
		return err
	}
	_, err = a.events.InitGenesis(ctx, run.ProjectID, run.SessionID, result.Snapshot)
	return err
}

func (a *SetupRunApplier) applyRelationships(ctx context.Context, run model.SetupRun, relationships []model.Relationship) error {
	for _, relationship := range relationships {
		pair := relationship.Pair
		if pair.ID == "" {
			pair.ID = generatedID(a.ids, a.clock, "pair")
		}
		pair.ProjectID = run.ProjectID
		if pair.Status == "" {
			pair.Status = "active"
		}
		now := currentTime(a.clock)
		if pair.CreatedAt.IsZero() {
			pair.CreatedAt = now
		}
		pair.UpdatedAt = now
		savedPair, err := a.relationships.UpsertPair(ctx, pair)
		if err != nil {
			return err
		}

		views := make([]model.RelationshipView, 0, len(relationship.Views))
		for _, view := range relationship.Views {
			views = append(views, a.setupRelationshipView(run, savedPair.ID, view))
		}
		if err := a.relationships.UpsertViews(ctx, savedPair.ID, views); err != nil {
			return err
		}
		if err := a.writeRevision(ctx, run.ProjectID, "relationship_pair", savedPair.ID, run.RunID, savedPair); err != nil {
			return err
		}
		for _, view := range views {
			if err := a.writeRevision(ctx, run.ProjectID, "relationship_view", view.ID, run.RunID, view); err != nil {
				return err
			}
		}
	}
	return nil
}

func (a *SetupRunApplier) setupRelationshipView(run model.SetupRun, pairID string, view model.RelationshipView) model.RelationshipView {
	if view.ID == "" {
		view.ID = generatedID(a.ids, a.clock, "view")
	}
	view.PairID = pairID
	view.ProjectID = run.ProjectID
	if view.Status == "" {
		view.Status = "active"
	}
	now := currentTime(a.clock)
	if view.CreatedAt.IsZero() {
		view.CreatedAt = now
	}
	view.UpdatedAt = now
	return view
}

func (a *SetupRunApplier) appendAppliedEvent(ctx context.Context, sessionID string, run model.SetupRun, input model.ApplySetupRunInput) error {
	_, err := a.audit.AppendRunEvent(ctx, model.RunEvent{
		ID:        generatedID(a.ids, a.clock, "event"),
		RunKind:   "setup",
		RunID:     run.RunID,
		EventName: "setup_run_applied",
		Payload: map[string]any{
			"session_id":           sessionID,
			"accept_author_bible":  input.AcceptAuthorBible,
			"accept_characters":    input.AcceptCharacters,
			"accept_relationships": input.AcceptRelationships,
			"accept_world_state":   input.AcceptWorldState,
			"author_note":          input.AuthorNote,
		},
		CreatedAt: currentTime(a.clock),
	})
	return err
}

func (a *SetupRunApplier) writeRevision(ctx context.Context, projectID, entityType, entityID, runID string, snapshot any) error {
	_, err := a.audit.CreateRevision(ctx, model.StateRevision{
		ID:          generatedID(a.ids, a.clock, "revision"),
		ProjectID:   projectID,
		EntityType:  entityType,
		EntityID:    entityID,
		SourceRunID: runID,
		Snapshot: map[string]any{
			"entity": snapshot,
		},
		CreatedAt: currentTime(a.clock),
	})
	return err
}
