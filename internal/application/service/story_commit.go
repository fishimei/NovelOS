package service

import (
	"context"

	"github.com/fishimei/NovelOS/internal/application/model"
	"github.com/fishimei/NovelOS/internal/application/port"
	"github.com/fishimei/NovelOS/internal/pkgerr"
)

// StoryRunCommitter 负责将已审阅的故事草稿转换为已提交的故事状态。
// 包括创建章节、应用记忆补丁、更新世界状态和关系。
// 这是一个跨多个仓库的复杂业务操作，需要在事务中执行以保证一致性。
type StoryRunCommitter struct {
	sessions      port.StorySessionRepository
	chapters      port.ChapterRepository
	memories      port.MemoryRepository
	worldState    port.WorldStateRepository
	relationships port.RelationshipRepository
	audit         port.AuditRepository
	tx            port.TxManager
	clock         port.Clock
	ids           port.IDGenerator
}

func NewStoryRunCommitter(
	sessions port.StorySessionRepository,
	chapters port.ChapterRepository,
	memories port.MemoryRepository,
	worldState port.WorldStateRepository,
	relationships port.RelationshipRepository,
	audit port.AuditRepository,
	tx port.TxManager,
	clock port.Clock,
	ids port.IDGenerator,
) *StoryRunCommitter {
	return &StoryRunCommitter{
		sessions:      sessions,
		chapters:      chapters,
		memories:      memories,
		worldState:    worldState,
		relationships: relationships,
		audit:         audit,
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
	err = c.tx.WithinTransaction(ctx, func(txCtx context.Context) error {
		chapter, err = c.createCommittedChapter(txCtx, run, result, input)
		if err != nil {
			return err
		}
		memories, err := c.createMemories(txCtx, chapter.ID, result.MemoryPatch.CharacterMemoryUpdates)
		if err != nil {
			return err
		}
		worldEntries, err := c.upsertWorldState(txCtx, run.ProjectID, result.MemoryPatch.WorldStateUpdates)
		if err != nil {
			return err
		}
		if err := c.applyRelationshipUpdates(txCtx, run.ProjectID, result.MemoryPatch.RelationshipUpdates); err != nil {
			return err
		}
		if err := c.writeCommitRevisions(txCtx, run.ProjectID, runID, chapter, memories, worldEntries); err != nil {
			return err
		}
		if err := c.appendCommittedEvent(txCtx, runID, chapter, result.MemoryPatch.ID, input.AuthorNote); err != nil {
			return err
		}
		return c.sessions.MarkCommitted(txCtx, runID)
	})
	if err != nil {
		return model.CommitStoryRunResult{}, err
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
	entries := make([]model.WorldStateEntry, 0, len(updates))
	for _, update := range updates {
		entries = append(entries, model.WorldStateEntry{
			ID:        generatedID(c.ids, c.clock, "world"),
			ProjectID: projectID,
			Key:       update.Key,
			Value:     update.Value,
			Note:      update.Note,
			Status:    "active",
			UpdatedAt: currentTime(c.clock),
		})
	}
	return entries, c.worldState.UpsertEntries(ctx, projectID, entries)
}

func (c *StoryRunCommitter) applyRelationshipUpdates(ctx context.Context, projectID string, updates []model.RelationshipUpdate) error {
	for _, update := range updates {
		if update.Pair == nil {
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
			return err
		}
		views := make([]model.RelationshipView, 0, len(update.Views))
		for _, viewUpdate := range update.Views {
			views = append(views, c.storyRelationshipView(projectID, savedPair.ID, viewUpdate))
		}
		if err := c.relationships.UpsertViews(ctx, savedPair.ID, views); err != nil {
			return err
		}
		for _, event := range update.Events {
			if err := c.addRelationshipEvent(ctx, projectID, savedPair.ID, event); err != nil {
				return err
			}
		}
	}
	return nil
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

func (c *StoryRunCommitter) appendCommittedEvent(ctx context.Context, runID string, chapter model.Chapter, memoryPatchID string, authorNote string) error {
	_, err := c.audit.AppendRunEvent(ctx, model.RunEvent{
		ID:        generatedID(c.ids, c.clock, "event"),
		RunKind:   "story",
		RunID:     runID,
		EventName: "story_run_committed",
		Sequence:  1,
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
