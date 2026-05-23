package service

import (
	"context"
	"log"

	"github.com/fishimei/NovelOS/internal/application/model"
	"github.com/fishimei/NovelOS/internal/application/port"
	"github.com/fishimei/NovelOS/internal/domain"
	"github.com/fishimei/NovelOS/internal/pkgerr"
)

// StoryRunCommitter 负责将已审阅的故事草稿转换为已提交的故事状态。
// 包括创建章节、应用记忆补丁、更新世界状态和关系。
// 这是一个跨多个仓库的复杂业务操作，需要在事务中执行以保证一致性。
type StoryRunCommitter struct {
	sessions      port.StorySessionRepository
	timeline      port.StoryTimelineRepository
	chapters      port.ChapterRepository
	memories      port.MemoryRepository
	worldState    port.WorldStateRepository
	relationships port.RelationshipRepository
	audit         port.AuditRepository
	memoryService port.CharacterMemoryService
	tx            port.TxManager
	clock         port.Clock
	ids           port.IDGenerator
}

func NewStoryRunCommitter(
	sessions port.StorySessionRepository,
	timeline port.StoryTimelineRepository,
	chapters port.ChapterRepository,
	memories port.MemoryRepository,
	worldState port.WorldStateRepository,
	relationships port.RelationshipRepository,
	audit port.AuditRepository,
	memoryService port.CharacterMemoryService,
	tx port.TxManager,
	clock port.Clock,
	ids port.IDGenerator,
) *StoryRunCommitter {
	return &StoryRunCommitter{
		sessions:      sessions,
		timeline:      timeline,
		chapters:      chapters,
		memories:      memories,
		worldState:    worldState,
		relationships: relationships,
		audit:         audit,
		memoryService: memoryService,
		tx:            tx,
		clock:         clock,
		ids:           ids,
	}
}

func (c *StoryRunCommitter) Commit(ctx context.Context, runID string, input model.CommitStoryRunInput) (model.CommitStoryRunResult, error) {
	if c.tx == nil {
		return model.CommitStoryRunResult{}, pkgerr.Internal("tx manager is required", nil)
	}
	run, err := c.sessions.GetRunByID(ctx, runID)
	if err != nil {
		return model.CommitStoryRunResult{}, err
	}
	// 已提交的 run 已经进入正史；重复提交会再次创建章节和记忆，必须在事务前拦截。
	if run.Status == "committed" || run.CommittedAt != nil {
		return model.CommitStoryRunResult{}, pkgerr.Conflict(pkgerr.CodeRunAlreadyCommitted, "story run already committed")
	}
	result, err := c.sessions.GetRunResultByID(ctx, runID)
	if err != nil {
		return model.CommitStoryRunResult{}, err
	}
	if input.DraftID != "" && input.DraftID != result.Draft.ID {
		return model.CommitStoryRunResult{}, pkgerr.Validation("draft id mismatch")
	}
	if input.MemoryPatchID != "" && input.MemoryPatchID != result.MemoryPatch.ID {
		return model.CommitStoryRunResult{}, pkgerr.Validation("memory patch id mismatch")
	}

	var chapter model.Chapter
	var committedMemories []model.Memory
	var committedRelationships []model.Relationship
	err = c.tx.WithinTransaction(ctx, func(txCtx context.Context) error {
		branch, err := c.commitBranch(txCtx, run, result)
		if err != nil {
			return err
		}
		chapter, err = c.createCommittedChapter(txCtx, run, result, input)
		if err != nil {
			return err
		}
		memories, err := c.createMemories(txCtx, chapter.ID, result.MemoryPatch.CharacterMemoryUpdates)
		if err != nil {
			return err
		}
		committedMemories = memories
		worldEntries, err := c.upsertWorldState(txCtx, run.ProjectID, result.MemoryPatch.WorldStateUpdates)
		if err != nil {
			return err
		}
		relationships, err := c.applyRelationshipUpdates(txCtx, run.ProjectID, result.MemoryPatch.RelationshipUpdates)
		if err != nil {
			return err
		}
		committedRelationships = relationships
		if err := c.writeCommitRevisions(txCtx, run.ProjectID, runID, chapter, memories, worldEntries); err != nil {
			return err
		}
		if err := c.appendCommittedEvent(txCtx, runID, chapter, result.MemoryPatch.ID, input.AuthorNote); err != nil {
			return err
		}
		commitTick, err := c.appendCommitTick(txCtx, run, branch, chapter, memories, worldEntries, committedRelationships, result.MemoryPatch.ID, input.AuthorNote)
		if err != nil {
			return err
		}
		if commitTick.ID != "" {
			if err := c.sessions.UpdateRunTimeline(txCtx, runID, commitTick.ID); err != nil {
				return err
			}
		}
		return c.sessions.MarkCommitted(txCtx, runID)
	})
	if err != nil {
		return model.CommitStoryRunResult{}, err
	}
	if err := c.commitExternalMemories(ctx, run, chapter, committedMemories); err != nil {
		log.Printf("story run %s external memory commit failed: %v", runID, err)
		c.appendExternalMemoryCommitFailedEvent(ctx, runID, err)
	}

	updatedRun, err := c.sessions.GetRunByID(ctx, runID)
	if err != nil {
		return model.CommitStoryRunResult{}, err
	}
	return model.CommitStoryRunResult{
		Chapter:  chapter,
		Patch:    result.MemoryPatch,
		StoryRun: updatedRun,
	}, nil
}

func (c *StoryRunCommitter) commitBranch(ctx context.Context, run model.StoryRun, result model.StoryRunResult) (model.StoryBranch, error) {
	if c.timeline == nil || run.BranchID == "" {
		return model.StoryBranch{}, nil
	}
	branch, err := c.timeline.GetBranchByID(ctx, run.BranchID)
	if err != nil {
		return model.StoryBranch{}, err
	}
	if branch.SessionID != run.SessionID {
		return model.StoryBranch{}, pkgerr.Validation("branch does not belong to story run")
	}
	if result.HeadTickID != "" && branch.HeadTickID != "" && result.HeadTickID != branch.HeadTickID {
		return model.StoryBranch{}, pkgerr.Conflict(pkgerr.CodeConflict, "story branch has advanced since run result was generated")
	}
	return branch, nil
}

func (c *StoryRunCommitter) commitExternalMemories(ctx context.Context, run model.StoryRun, chapter model.Chapter, memories []model.Memory) error {
	if c.memoryService == nil || len(memories) == 0 {
		return nil
	}
	return c.memoryService.Commit(ctx, port.CharacterMemoryCommitInput{
		ProjectID: run.ProjectID,
		RunID:     run.RunID,
		Chapter:   chapter,
		Memories:  memories,
	})
}

func (c *StoryRunCommitter) appendExternalMemoryCommitFailedEvent(ctx context.Context, runID string, err error) {
	if c.audit == nil {
		return
	}
	if _, appendErr := c.audit.AppendRunEvent(ctx, model.RunEvent{
		RunKind:   "story",
		RunID:     runID,
		EventName: domain.EventGenerationStep,
		Payload: map[string]any{
			"step":  "external_memory_commit_failed",
			"error": err.Error(),
		},
	}); appendErr != nil {
		log.Printf("append story run %s external memory failure event failed: %v", runID, appendErr)
	}
}

func (c *StoryRunCommitter) createCommittedChapter(ctx context.Context, run model.StoryRun, result model.StoryRunResult, input model.CommitStoryRunInput) (model.Chapter, error) {
	chapter := model.Chapter{
		ID:            generatedID(c.ids, c.clock, "chapter"),
		ProjectID:     run.ProjectID,
		ChapterNumber: result.Draft.ChapterNumber,
		Title:         result.Draft.Title,
		Summary:       result.Draft.Summary,
		Content:       result.Draft.Content,
		AuthorNote:    input.AuthorNote,
		Status:        "committed",
		WordCount:     result.Draft.WordCount,
		CommittedAt:   currentTime(c.clock),
	}
	return c.chapters.Create(ctx, chapter)
}

func (c *StoryRunCommitter) createMemories(ctx context.Context, chapterID string, updates []model.CharacterMemoryUpdate) ([]model.Memory, error) {
	memories := make([]model.Memory, 0, len(updates))
	for _, update := range updates {
		memories = append(memories, model.Memory{
			ID:              generatedID(c.ids, c.clock, "memory"),
			CharacterID:     update.CharacterID,
			Content:         update.Content,
			SourceChapterID: chapterID,
			Importance:      update.Importance,
			Status:          "active",
			CreatedAt:       currentTime(c.clock),
		})
	}
	return memories, c.memories.CreateBatch(ctx, memories)
}

func (c *StoryRunCommitter) upsertWorldState(ctx context.Context, projectID string, updates []model.WorldStateUpdate) ([]model.WorldStateEntry, error) {
	if len(updates) == 0 {
		return nil, nil
	}
	currentEntries, err := c.worldState.ListByProjectID(ctx, projectID)
	if err != nil {
		return nil, err
	}
	currentByKey := make(map[string]model.WorldStateEntry, len(currentEntries))
	for _, entry := range currentEntries {
		currentByKey[entry.Key] = entry
	}
	entries := make([]model.WorldStateEntry, 0, len(updates))
	for _, update := range updates {
		entry := currentByKey[update.Key]
		if entry.ID == "" {
			entry.ID = generatedID(c.ids, c.clock, "world")
		}
		entry.ProjectID = projectID
		entry.Key = update.Key
		entry.Value = update.Value
		entry.Note = firstNonEmpty(update.Note, entry.Note)
		entry.Status = "active"
		entry.UpdatedAt = currentTime(c.clock)
		entries = append(entries, entry)
	}
	return entries, c.worldState.UpsertEntries(ctx, projectID, entries)
}

func (c *StoryRunCommitter) applyRelationshipUpdates(ctx context.Context, projectID string, updates []model.RelationshipUpdate) ([]model.Relationship, error) {
	relationships := make([]model.Relationship, 0, len(updates))
	for _, update := range updates {
		if update.Pair == nil {
			if update.PairID == "" || (update.Summary == "" && update.TensionDelta == "" && len(update.Events) == 0) {
				continue
			}
			current, err := c.relationships.GetByID(ctx, update.PairID)
			if err != nil {
				return nil, err
			}
			if update.Summary != "" {
				current.Pair.Summary = update.Summary
				current.Pair.UpdatedAt = currentTime(c.clock)
				if _, err := c.relationships.UpsertPair(ctx, current.Pair); err != nil {
					return nil, err
				}
			}
			if update.TensionDelta != "" {
				event := model.RelationshipEvent{EventType: "tension_delta", Summary: update.TensionDelta}
				if err := c.addRelationshipEvent(ctx, projectID, current.Pair.ID, event); err != nil {
					return nil, err
				}
			}
			for _, event := range update.Events {
				if err := c.addRelationshipEvent(ctx, projectID, current.Pair.ID, event); err != nil {
					return nil, err
				}
			}
			current, err = c.relationships.GetByID(ctx, current.Pair.ID)
			if err != nil {
				return nil, err
			}
			relationships = append(relationships, current)
			continue
		}
		pair := *update.Pair
		if pair.ID == "" {
			pair.ID = generatedID(c.ids, c.clock, "pair")
		}
		pair.ProjectID = projectID
		if pair.Status == "" {
			pair.Status = "active"
		}
		pair.UpdatedAt = currentTime(c.clock)
		savedPair, err := c.relationships.UpsertPair(ctx, pair)
		if err != nil {
			return nil, err
		}
		views := make([]model.RelationshipView, 0, len(update.Views))
		for _, viewUpdate := range update.Views {
			views = append(views, c.storyRelationshipView(projectID, savedPair.ID, viewUpdate))
		}
		if err := c.relationships.UpsertViews(ctx, savedPair.ID, views); err != nil {
			return nil, err
		}
		for _, event := range update.Events {
			if err := c.addRelationshipEvent(ctx, projectID, savedPair.ID, event); err != nil {
				return nil, err
			}
		}
		relationship, err := c.relationships.GetByID(ctx, savedPair.ID)
		if err != nil {
			return nil, err
		}
		relationships = append(relationships, relationship)
	}
	return relationships, nil
}

func (c *StoryRunCommitter) storyRelationshipView(projectID string, pairID string, update model.RelationshipViewUpdate) model.RelationshipView {
	now := currentTime(c.clock)
	return model.RelationshipView{
		ID:                     firstNonEmpty(update.ViewID, generatedID(c.ids, c.clock, "view")),
		ProjectID:              projectID,
		PairID:                 pairID,
		SourceCharacterID:      update.SourceCharacterID,
		TargetCharacterID:      update.TargetCharacterID,
		PublicAttitude:         update.PublicAttitude,
		PrivateAttitude:        update.PrivateAttitude,
		BelievedTargetAttitude: update.BelievedTargetAttitude,
		MaskingStrategy:        update.MaskingStrategy,
		Status:                 "active",
		CreatedAt:              now,
		UpdatedAt:              now,
	}
}

func (c *StoryRunCommitter) addRelationshipEvent(ctx context.Context, projectID string, pairID string, event model.RelationshipEvent) error {
	if event.ID == "" {
		event.ID = generatedID(c.ids, c.clock, "revent")
	}
	event.ProjectID = projectID
	event.PairID = pairID
	event.CreatedAt = currentTime(c.clock)
	_, err := c.relationships.AddEvent(ctx, event)
	return err
}

func (c *StoryRunCommitter) writeCommitRevisions(ctx context.Context, projectID string, runID string, chapter model.Chapter, memories []model.Memory, worldEntries []model.WorldStateEntry) error {
	if err := c.writeRevision(ctx, projectID, "chapter", chapter.ID, runID, chapter); err != nil {
		return err
	}
	for _, memory := range memories {
		if err := c.writeRevision(ctx, projectID, "character_memory", memory.ID, runID, memory); err != nil {
			return err
		}
	}
	for _, entry := range worldEntries {
		if err := c.writeRevision(ctx, projectID, "world_state_entry", entry.ID, runID, entry); err != nil {
			return err
		}
	}
	return nil
}

func (c *StoryRunCommitter) appendCommitTick(ctx context.Context, run model.StoryRun, branch model.StoryBranch, chapter model.Chapter, memories []model.Memory, worldEntries []model.WorldStateEntry, relationships []model.Relationship, memoryPatchID string, authorNote string) (model.StoryTick, error) {
	if c.timeline == nil || branch.ID == "" {
		return model.StoryTick{}, nil
	}
	parentTickID := branch.HeadTickID
	sequence, err := c.commitTickSequence(ctx, branch.ID, parentTickID)
	if err != nil {
		return model.StoryTick{}, err
	}
	tickID := generatedID(c.ids, c.clock, "tick")
	versions, refs := c.commitStateRefs(run, tickID, chapter, memories, worldEntries, relationships)
	tick, err := c.timeline.AppendTick(ctx, model.StoryTick{
		ID:           tickID,
		ProjectID:    run.ProjectID,
		SessionID:    run.SessionID,
		BranchID:     branch.ID,
		ParentTickID: parentTickID,
		SourceRunID:  run.RunID,
		Sequence:     sequence,
		Kind:         "commit",
		Summary:      chapter.Summary,
		Payload: map[string]any{
			"chapter_id":      chapter.ID,
			"memory_patch_id": memoryPatchID,
			"author_note":     authorNote,
		},
		CreatedAt: currentTime(c.clock),
	}, refs, versions)
	if err != nil {
		return model.StoryTick{}, err
	}
	return tick, c.timeline.UpdateBranchHead(ctx, branch.ID, tick.ID)
}

func (c *StoryRunCommitter) commitTickSequence(ctx context.Context, branchID string, parentTickID string) (int, error) {
	if parentTickID != "" {
		parent, err := c.timeline.GetTickByID(ctx, parentTickID)
		if err != nil {
			return 0, err
		}
		return parent.Sequence + 1, nil
	}
	ticks, err := c.timeline.ListTicksByBranchID(ctx, branchID)
	if err != nil {
		return 0, err
	}
	if len(ticks) == 0 {
		return 1, nil
	}
	return ticks[len(ticks)-1].Sequence + 1, nil
}

func (c *StoryRunCommitter) commitStateRefs(run model.StoryRun, tickID string, chapter model.Chapter, memories []model.Memory, worldEntries []model.WorldStateEntry, relationships []model.Relationship) ([]model.StoryStateVersion, []model.StoryTickStateRef) {
	versions := []model.StoryStateVersion{}
	refs := []model.StoryTickStateRef{}
	add := func(entityType string, entityID string, snapshot map[string]any) {
		if entityID == "" {
			return
		}
		versionID := generatedID(c.ids, c.clock, "sversion")
		versions = append(versions, model.StoryStateVersion{
			ID:           versionID,
			ProjectID:    run.ProjectID,
			EntityType:   entityType,
			EntityID:     entityID,
			SourceTickID: tickID,
			SourceRunID:  run.RunID,
			Snapshot:     snapshot,
			CreatedAt:    currentTime(c.clock),
		})
		refs = append(refs, model.StoryTickStateRef{
			TickID:     tickID,
			ProjectID:  run.ProjectID,
			EntityType: entityType,
			EntityID:   entityID,
			VersionID:  versionID,
		})
	}
	add("chapter", chapter.ID, map[string]any{"entity": chapter})
	for _, memory := range memories {
		add("character_memory", memory.ID, map[string]any{"entity": memory})
	}
	for _, entry := range worldEntries {
		add("world_state", entry.Key, map[string]any{"entity": entry})
	}
	for _, relationship := range relationships {
		add("relationship", relationship.Pair.ID, map[string]any{"entity": relationship})
	}
	return versions, refs
}

func (c *StoryRunCommitter) appendCommittedEvent(ctx context.Context, runID string, chapter model.Chapter, memoryPatchID string, authorNote string) error {
	_, err := c.audit.AppendRunEvent(ctx, model.RunEvent{
		ID:        generatedID(c.ids, c.clock, "event"),
		RunKind:   "story",
		RunID:     runID,
		EventName: "story_run_committed",
		Payload: map[string]any{
			"chapter_id":      chapter.ID,
			"chapter_number":  chapter.ChapterNumber,
			"memory_patch_id": memoryPatchID,
			"author_note":     authorNote,
		},
		CreatedAt: currentTime(c.clock),
	})
	return err
}

func (c *StoryRunCommitter) writeRevision(ctx context.Context, projectID, entityType, entityID, runID string, snapshot any) error {
	_, err := c.audit.CreateRevision(ctx, model.StateRevision{
		ID:          generatedID(c.ids, c.clock, "revision"),
		ProjectID:   projectID,
		EntityType:  entityType,
		EntityID:    entityID,
		SourceRunID: runID,
		Snapshot: map[string]any{
			"entity": snapshot,
		},
		CreatedAt: currentTime(c.clock),
	})
	return err
}
