package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/fishimei/NovelOS/internal/application/model"
	"github.com/fishimei/NovelOS/internal/application/port"
	"github.com/fishimei/NovelOS/internal/domain"
	"github.com/fishimei/NovelOS/internal/pkgerr"
)

type StoryChapterCutter struct {
	sessions      port.StorySessionRepository
	store         port.StoryEventStore
	chapters      port.ChapterRepository
	audit         port.AuditRepository
	memoryService port.CharacterMemoryService
	tx            port.TxManager
	clock         port.Clock
	ids           port.IDGenerator
}

func NewStoryChapterCutter(
	sessions port.StorySessionRepository,
	store port.StoryEventStore,
	chapters port.ChapterRepository,
	audit port.AuditRepository,
	memoryService port.CharacterMemoryService,
	tx port.TxManager,
	clock port.Clock,
	ids port.IDGenerator,
) *StoryChapterCutter {
	return &StoryChapterCutter{
		sessions:      sessions,
		store:         store,
		chapters:      chapters,
		audit:         audit,
		memoryService: memoryService,
		tx:            tx,
		clock:         clock,
		ids:           ids,
	}
}

func (c *StoryChapterCutter) CutChapter(ctx context.Context, runID string, input model.CutChapterInput) (model.CutChapterResult, error) {
	if c.tx == nil {
		return model.CutChapterResult{}, pkgerr.Internal("tx manager is required", nil)
	}
	run, err := c.sessions.GetRunByID(ctx, runID)
	if err != nil {
		return model.CutChapterResult{}, err
	}
	if input.BranchID == "" {
		input.BranchID = run.BranchID
	}
	if input.FromEventID == "" {
		input.FromEventID = run.BaseEventID
	}
	if input.ToEventID == "" {
		input.ToEventID = run.HeadEventID
	}
	if input.BranchID == "" || input.FromEventID == "" || input.ToEventID == "" {
		return model.CutChapterResult{}, pkgerr.Validation("branch_id, from_event_id, and to_event_id are required")
	}

	var chapter model.Chapter
	var span model.ChapterEventSpan
	var memories []model.Memory
	err = c.tx.WithinTransaction(ctx, func(txCtx context.Context) error {
		branch, events, err := c.eventSpan(txCtx, run, input)
		if err != nil {
			return err
		}
		if _, err := c.store.GetChapterSpanByRange(txCtx, input.BranchID, input.FromEventID, input.ToEventID); err == nil {
			return pkgerr.Conflict(pkgerr.CodeConflict, "event span is already cut into a chapter")
		} else if !isNotFound(err) {
			return err
		}
		chapter, memories, err = c.createChapterFromEvents(txCtx, run, input, events)
		if err != nil {
			return err
		}
		span, err = c.store.CreateChapterSpan(txCtx, model.ChapterEventSpan{
			ProjectID:   run.ProjectID,
			ChapterID:   chapter.ID,
			BranchID:    branch.ID,
			FromEventID: input.FromEventID,
			ToEventID:   input.ToEventID,
			CreatedAt:   currentTime(c.clock),
		})
		if err != nil {
			return err
		}
		if err := c.store.SetPublishedFrontier(txCtx, branch.ID, input.ToEventID); err != nil {
			return err
		}
		if err := c.appendCutEvent(txCtx, runID, chapter, span, input.AuthorNote); err != nil {
			return err
		}
		return c.sessions.MarkCut(txCtx, runID)
	})
	if err != nil {
		return model.CutChapterResult{}, err
	}
	if err := c.flushExternalMemories(ctx, run, chapter, memories); err != nil {
		log.Printf("story run %s external memory flush failed: %v", runID, err)
		c.appendExternalMemoryFlushFailedEvent(ctx, runID, err)
	} else if c.memoryService != nil && len(memories) > 0 {
		c.appendExternalMemoryCommittedEvent(ctx, runID, chapter, len(memories))
	}
	updatedRun, err := c.sessions.GetRunByID(ctx, runID)
	if err != nil {
		return model.CutChapterResult{}, err
	}
	return model.CutChapterResult{Chapter: chapter, Span: span, StoryRun: updatedRun}, nil
}

func (c *StoryChapterCutter) CutLatestCompletedSpan(ctx context.Context, runID string, input model.CutChapterInput) (model.CutChapterResult, error) {
	run, err := c.sessions.GetRunByID(ctx, runID)
	if err != nil {
		return model.CutChapterResult{}, err
	}
	if run.Status == domain.RunStatusCut || run.CutAt != nil {
		return model.CutChapterResult{}, pkgerr.Conflict(pkgerr.CodeConflict, "story run is already cut")
	}
	if run.Status != domain.RunStatusCompleted {
		return model.CutChapterResult{}, pkgerr.Validation("latest completed span requires a completed story run")
	}
	if run.BranchID == "" || run.BaseEventID == "" || run.HeadEventID == "" {
		return model.CutChapterResult{}, pkgerr.Validation("completed story run has no resolvable event span")
	}
	input.BranchID = run.BranchID
	input.FromEventID = run.BaseEventID
	input.ToEventID = run.HeadEventID
	return c.CutChapter(ctx, runID, input)
}

func (c *StoryChapterCutter) eventSpan(ctx context.Context, run model.StoryRun, input model.CutChapterInput) (model.Branch, []model.StoryEvent, error) {
	branch, err := c.store.GetBranch(ctx, input.BranchID)
	if err != nil {
		return model.Branch{}, nil, err
	}
	if branch.SessionID != run.SessionID {
		return model.Branch{}, nil, pkgerr.Validation("branch does not belong to story run")
	}
	events, err := c.store.ListEventsByBranch(ctx, branch.ID)
	if err != nil {
		return model.Branch{}, nil, err
	}
	out := make([]model.StoryEvent, 0)
	inRange := false
	for _, event := range events {
		if event.ID == input.FromEventID {
			inRange = true
		}
		if inRange {
			out = append(out, event)
		}
		if event.ID == input.ToEventID {
			if !inRange {
				return model.Branch{}, nil, pkgerr.Validation("from_event_id must be before to_event_id on the same branch")
			}
			return branch, out, nil
		}
	}
	return model.Branch{}, nil, pkgerr.Validation("event span is not present on branch")
}

func (c *StoryChapterCutter) createChapterFromEvents(ctx context.Context, run model.StoryRun, input model.CutChapterInput, events []model.StoryEvent) (model.Chapter, []model.Memory, error) {
	title := strings.TrimSpace(input.Title)
	summaries := make([]string, 0)
	contents := make([]string, 0)
	memories := make([]model.Memory, 0)
	chapterNumber := 0
	for _, event := range events {
		if event.Kind != model.EventKindSceneResolved {
			continue
		}
		draft := draftFromEvent(event)
		if title == "" {
			title = draft.Title
		}
		if draft.ChapterNumber > 0 && chapterNumber == 0 {
			chapterNumber = draft.ChapterNumber
		}
		if draft.Summary != "" {
			summaries = append(summaries, draft.Summary)
		} else if event.Summary != "" {
			summaries = append(summaries, event.Summary)
		}
		if draft.Content != "" {
			contents = append(contents, draft.Content)
		}
		for _, update := range event.StateDelta.CharacterMemoryUpdates {
			if update.CharacterID == "" || update.Content == "" {
				continue
			}
			memories = append(memories, model.Memory{
				ID:          generatedID(c.ids, c.clock, "memory"),
				CharacterID: update.CharacterID,
				Content:     update.Content,
				Importance:  update.Importance,
				Note:        domain.MemoryScopeExternalCommitted + ":" + domain.MemoryCommitTriggerChapterCut,
				Status:      "active",
				CreatedAt:   currentTime(c.clock),
			})
		}
	}
	if len(contents) == 0 {
		return model.Chapter{}, nil, pkgerr.Validation("event span contains no rendered scene content")
	}
	if chapterNumber == 0 {
		next, err := c.nextChapterNumber(ctx, run.ProjectID)
		if err != nil {
			return model.Chapter{}, nil, err
		}
		chapterNumber = next
	}
	if title == "" {
		title = fmt.Sprintf("Chapter %d", chapterNumber)
	}
	content := strings.Join(contents, "\n\n")
	chapter := model.Chapter{
		ID:            generatedID(c.ids, c.clock, "chapter"),
		ProjectID:     run.ProjectID,
		ChapterNumber: chapterNumber,
		Title:         title,
		Summary:       strings.Join(summaries, "\n"),
		Content:       content,
		AuthorNote:    input.AuthorNote,
		Status:        "published",
		WordCount:     len([]rune(content)),
		CommittedAt:   currentTime(c.clock),
	}
	saved, err := c.chapters.Create(ctx, chapter)
	if err != nil {
		return model.Chapter{}, nil, err
	}
	for i := range memories {
		memories[i].SourceChapterID = saved.ID
	}
	return saved, memories, nil
}

func (c *StoryChapterCutter) nextChapterNumber(ctx context.Context, projectID string) (int, error) {
	result, err := c.chapters.ListByProjectID(ctx, projectID, model.PageInput{Page: 1, PageSize: 1})
	if err != nil {
		return 0, err
	}
	return result.Total + 1, nil
}

func draftFromEvent(event model.StoryEvent) model.Draft {
	raw, ok := event.Payload["draft"]
	if !ok {
		return model.Draft{}
	}
	bytes, err := json.Marshal(raw)
	if err != nil {
		return model.Draft{}
	}
	var draft model.Draft
	if err := json.Unmarshal(bytes, &draft); err != nil {
		return model.Draft{}
	}
	return draft
}

func (c *StoryChapterCutter) flushExternalMemories(ctx context.Context, run model.StoryRun, chapter model.Chapter, memories []model.Memory) error {
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

func (c *StoryChapterCutter) appendCutEvent(ctx context.Context, runID string, chapter model.Chapter, span model.ChapterEventSpan, authorNote string) error {
	if c.audit == nil {
		return nil
	}
	_, err := c.audit.AppendRunEvent(ctx, model.RunEvent{
		ID:        generatedID(c.ids, c.clock, "event"),
		RunKind:   "story",
		RunID:     runID,
		EventName: "story_chapter_cut",
		Payload: map[string]any{
			"chapter_id":     chapter.ID,
			"chapter_number": chapter.ChapterNumber,
			"span_id":        span.ID,
			"from_event_id":  span.FromEventID,
			"to_event_id":    span.ToEventID,
			"author_note":    authorNote,
			"memory_scope":   domain.MemoryScopeExternalCommitted,
			"memory_trigger": domain.MemoryCommitTriggerChapterCut,
		},
		CreatedAt: currentTime(c.clock),
	})
	return err
}

func (c *StoryChapterCutter) appendExternalMemoryCommittedEvent(ctx context.Context, runID string, chapter model.Chapter, memoryCount int) {
	if c.audit == nil {
		return
	}
	_, _ = c.audit.AppendRunEvent(ctx, model.RunEvent{
		RunKind:   "story",
		RunID:     runID,
		EventName: "external_memory_committed",
		Payload: map[string]any{
			"chapter_id":   chapter.ID,
			"memory_count": memoryCount,
			"scope":        domain.MemoryScopeExternalCommitted,
			"trigger":      domain.MemoryCommitTriggerChapterCut,
		},
		CreatedAt: currentTime(c.clock),
	})
}

func (c *StoryChapterCutter) appendExternalMemoryFlushFailedEvent(ctx context.Context, runID string, err error) {
	if c.audit == nil {
		return
	}
	if _, appendErr := c.audit.AppendRunEvent(ctx, model.RunEvent{
		RunKind:   "story",
		RunID:     runID,
		EventName: domain.EventGenerationStep,
		Payload: map[string]any{
			"step":  "external_memory_flush_failed",
			"error": err.Error(),
		},
	}); appendErr != nil {
		log.Printf("append story run %s external memory failure event failed: %v", runID, appendErr)
	}
}
