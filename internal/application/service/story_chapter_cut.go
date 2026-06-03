package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/fishimei/NovelOS/internal/application/chapterseq"
	"github.com/fishimei/NovelOS/internal/application/model"
	"github.com/fishimei/NovelOS/internal/application/port"
	"github.com/fishimei/NovelOS/internal/domain"
	"github.com/fishimei/NovelOS/internal/pkgerr"
)

type StoryChapterCutter struct {
	sessions port.StorySessionRepository
	store    port.StoryEventStore
	chapters port.ChapterRepository
	audit    port.AuditRepository
	memory   port.CharacterMemoryService
	tx       port.TxManager
	clock    port.Clock
	ids      port.IDGenerator
}

func NewStoryChapterCutter(
	sessions port.StorySessionRepository,
	store port.StoryEventStore,
	chapters port.ChapterRepository,
	audit port.AuditRepository,
	memory port.CharacterMemoryService,
	tx port.TxManager,
	clock port.Clock,
	ids port.IDGenerator,
) *StoryChapterCutter {
	return &StoryChapterCutter{
		sessions: sessions,
		store:    store,
		chapters: chapters,
		audit:    audit,
		memory:   memory,
		tx:       tx,
		clock:    clock,
		ids:      ids,
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
	var spanEvents []model.StoryEvent
	for attempt := 0; attempt < 2; attempt++ {
		err = c.tx.WithinTransaction(ctx, func(txCtx context.Context) error {
			branch, events, err := c.eventSpan(txCtx, run, input)
			if err != nil {
				return err
			}
			spanEvents = events
			if _, err := c.store.GetChapterSpanByRange(txCtx, input.BranchID, input.FromEventID, input.ToEventID); err == nil {
				return pkgerr.Conflict(pkgerr.CodeConflict, "event span is already cut into a chapter")
			} else if !isNotFound(err) {
				return err
			}
			chapter, err = c.createChapterFromEvents(txCtx, run, input, events)
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
		if err == nil || !isChapterNumberConflict(err) || attempt == 1 {
			break
		}
	}
	if err != nil {
		return model.CutChapterResult{}, err
	}
	c.commitCharacterMemories(ctx, run, chapter, span, spanEvents)
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
	if branch.SessionID != run.SessionID || branch.ProjectID != run.ProjectID {
		return model.Branch{}, nil, pkgerr.Validation("branch does not belong to story run")
	}
	events, err := c.reachableEventSpan(ctx, branch, input.FromEventID, input.ToEventID)
	if err != nil {
		return model.Branch{}, nil, err
	}
	return branch, events, nil
}

func (c *StoryChapterCutter) reachableEventSpan(ctx context.Context, branch model.Branch, fromEventID, toEventID string) ([]model.StoryEvent, error) {
	chain := make([]model.StoryEvent, 0)
	currentID := toEventID
	for currentID != "" {
		event, err := c.store.GetEvent(ctx, currentID)
		if err != nil {
			return nil, err
		}
		if event.ProjectID != branch.ProjectID {
			return nil, pkgerr.Validation("event span crosses project boundary")
		}
		if event.SessionID != branch.SessionID && event.ID != branch.BaseEventID && event.ID != fromEventID {
			return nil, pkgerr.Validation("event span crosses story session boundary")
		}
		if event.ID == fromEventID {
			if len(chain) == 0 {
				return nil, pkgerr.Validation("event span contains no events after from_event_id")
			}
			for i, j := 0, len(chain)-1; i < j; i, j = i+1, j-1 {
				chain[i], chain[j] = chain[j], chain[i]
			}
			return chain, nil
		}
		if event.BranchID != branch.ID {
			return nil, pkgerr.Validation("event span crosses branch boundary")
		}
		chain = append(chain, event)
		currentID = event.ParentEventID
	}
	return nil, pkgerr.Validation("event span is not present on branch")
}

func (c *StoryChapterCutter) createChapterFromEvents(ctx context.Context, run model.StoryRun, input model.CutChapterInput, events []model.StoryEvent) (model.Chapter, error) {
	title := strings.TrimSpace(input.Title)
	summaries := make([]string, 0)
	contents := make([]string, 0)
	chapterNumber := 0
	for _, event := range events {
		if event.Kind != model.EventKindSceneResolved {
			continue
		}
		draft := draftFromEvent(event)
		sceneSummary := summaryFromEvent(event)
		if title == "" {
			title = draft.Title
		}
		if draft.ChapterNumber > 0 && chapterNumber == 0 {
			chapterNumber = draft.ChapterNumber
		}
		if sceneSummary != "" {
			summaries = append(summaries, sceneSummary)
		} else if draft.Summary != "" {
			summaries = append(summaries, draft.Summary)
		} else if event.Summary != "" {
			summaries = append(summaries, event.Summary)
		}
		if draft.Content != "" {
			contents = append(contents, draft.Content)
		}
	}
	if chapterNumber == 0 {
		next, err := c.nextChapterNumber(ctx, run.ProjectID)
		if err != nil {
			return model.Chapter{}, err
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
		return model.Chapter{}, err
	}
	return saved, nil
}

func (c *StoryChapterCutter) nextChapterNumber(ctx context.Context, projectID string) (int, error) {
	return chapterseq.NextChapterNumber(ctx, c.chapters, projectID)
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

func summaryFromEvent(event model.StoryEvent) string {
	if event.Payload == nil {
		return event.Summary
	}
	value, ok := event.Payload["summary"]
	if !ok {
		return event.Summary
	}
	summary, ok := value.(string)
	if !ok {
		return event.Summary
	}
	return strings.TrimSpace(summary)
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
		},
		CreatedAt: currentTime(c.clock),
	})
	return err
}

func (c *StoryChapterCutter) commitCharacterMemories(ctx context.Context, run model.StoryRun, chapter model.Chapter, span model.ChapterEventSpan, events []model.StoryEvent) {
	if c.memory == nil {
		return
	}
	memories := make([]model.Memory, 0)
	for _, event := range events {
		if event.Kind != model.EventKindSceneResolved {
			continue
		}
		for _, update := range event.StateDelta.MemoryPatch.CharacterMemoryUpdates {
			if update.CharacterID == "" || update.Content == "" {
				continue
			}
			memories = append(memories, model.Memory{
				ID:              generatedID(c.ids, c.clock, "memory"),
				CharacterID:     update.CharacterID,
				Content:         update.Content,
				SourceChapterID: chapter.ID,
				SourceRunID:     run.RunID,
				BranchID:        span.BranchID,
				SourceEventID:   event.ID,
				Importance:      update.Importance,
				Note:            domain.MemoryScopeExternalCommitted + ":" + domain.MemoryCommitTriggerChapterCut,
				Status:          "active",
				CreatedAt:       currentTime(c.clock),
			})
		}
	}
	if len(memories) == 0 {
		return
	}
	err := c.memory.Commit(ctx, port.CharacterMemoryCommitInput{
		ProjectID: run.ProjectID,
		RunID:     run.RunID,
		Chapter:   chapter,
		Memories:  memories,
	})
	if err != nil {
		c.appendMemoryAuditEvent(ctx, run.RunID, "external_memory_flush_failed", map[string]any{
			"error":      err.Error(),
			"chapter_id": chapter.ID,
			"span_id":    span.ID,
			"trigger":    domain.MemoryCommitTriggerChapterCut,
		})
		return
	}
	c.appendMemoryAuditEvent(ctx, run.RunID, "external_memory_committed", map[string]any{
		"memory_count": len(memories),
		"chapter_id":   chapter.ID,
		"span_id":      span.ID,
		"scope":        domain.MemoryScopeExternalCommitted,
		"trigger":      domain.MemoryCommitTriggerChapterCut,
	})
}

func (c *StoryChapterCutter) appendMemoryAuditEvent(ctx context.Context, runID, eventName string, payload map[string]any) {
	if c.audit == nil {
		return
	}
	if _, err := c.audit.AppendRunEvent(ctx, model.RunEvent{
		RunKind:   "story",
		RunID:     runID,
		EventName: eventName,
		Payload:   payload,
		CreatedAt: currentTime(c.clock),
	}); err != nil {
		return
	}
}
