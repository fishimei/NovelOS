package service

import (
	"context"
	"testing"
	"time"

	"github.com/fishimei/NovelOS/internal/application/model"
	"github.com/fishimei/NovelOS/internal/application/port"
	"github.com/fishimei/NovelOS/internal/domain"
)

func TestStorySessionAdvancerCommitCharacterMemoriesTagsRunCompletion(t *testing.T) {
	memory := &recordingCharacterMemoryService{}
	advancer := &StorySessionAdvancer{
		memory: memory,
		clock:  fixedServiceClock{now: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)},
		ids:    fixedServiceIDs{},
	}
	run := model.StoryRun{
		RunID:     "run_1",
		ProjectID: "project_1",
		BranchID:  "branch_1",
	}
	result := model.StoryRunResult{
		HeadEventID: "scene_event_1",
		MemoryPatch: model.MemoryPatch{CharacterMemoryUpdates: []model.CharacterMemoryUpdate{
			{CharacterID: "character_1", Content: "Lin denied carrying the letter.", Importance: 4},
		}},
	}

	advancer.commitCharacterMemories(context.Background(), run, result)

	if memory.input.ProjectID != "project_1" || memory.input.RunID != "run_1" {
		t.Fatalf("unexpected commit input: %#v", memory.input)
	}
	if len(memory.input.Memories) != 1 {
		t.Fatalf("expected one committed memory, got %#v", memory.input.Memories)
	}
	committed := memory.input.Memories[0]
	if committed.SourceRunID != "run_1" || committed.BranchID != "branch_1" || committed.SourceEventID != "scene_event_1" {
		t.Fatalf("memory source tags not set: %#v", committed)
	}
	if committed.Note != domain.MemoryScopeExternalCommitted+":"+domain.MemoryCommitTriggerRunCompletion {
		t.Fatalf("memory note = %q", committed.Note)
	}
}

func TestStorySessionAdvancerSkipsSceneResolvedForCompletionOnlyResult(t *testing.T) {
	start := time.Date(2026, 6, 1, 11, 0, 0, 0, time.UTC)
	store := &completionOnlyStoryEventStore{}
	sessions := &completionOnlyStorySessions{}
	advancer := &StorySessionAdvancer{
		store:    store,
		sessions: sessions,
		clock:    fixedServiceClock{now: start.Add(3 * time.Hour)},
	}
	run := model.StoryRun{
		RunID:     "run_1",
		ProjectID: "project_1",
		SessionID: "story_1",
		BranchID:  "branch_1",
		CreatedAt: start,
	}
	result := model.StoryRunResult{
		SceneSummary: "advance split watches",
		EventPlan: []model.StoryEventPlan{{
			ID:            "planned_action_1",
			TimeIndex:     1,
			DurationHours: 1,
			CharacterID:   "character_1",
			LocationKey:   "tower",
			ActionType:    "observe",
			Summary:       "Lin watches the tower",
		}},
	}

	persisted, err := advancer.persistResultEvents(context.Background(), run, result)
	if err != nil {
		t.Fatalf("persistResultEvents() error = %v", err)
	}
	if len(store.appended) != 2 {
		t.Fatalf("events = %#v, want scheduled plus completed", store.appended)
	}
	if store.appended[0].Kind != model.EventKindActionScheduled || store.appended[1].Kind != model.EventKindActionCompleted {
		t.Fatalf("unexpected event kinds: %#v", store.appended)
	}
	if !store.appended[0].StoryTime.Equal(start.Add(time.Hour)) || !store.appended[1].StoryTime.Equal(start.Add(2*time.Hour)) {
		t.Fatalf("unexpected story times: %#v", store.appended)
	}
	for _, event := range store.appended {
		if event.Kind == model.EventKindSceneResolved {
			t.Fatalf("completion-only result should not append scene_resolved: %#v", store.appended)
		}
	}
	if persisted.HeadEventID != "completed_1" || len(persisted.Events) != 2 {
		t.Fatalf("persisted head/events = (%q, %#v)", persisted.HeadEventID, persisted.Events)
	}
	if store.branchID != "branch_1" || store.branchHeadID != "completed_1" {
		t.Fatalf("branch head not advanced to completion: %#v", store)
	}
	if sessions.runID != "run_1" || sessions.headEventID != "completed_1" {
		t.Fatalf("run head not advanced to completion: %#v", sessions)
	}
}

type recordingCharacterMemoryService struct {
	input port.CharacterMemoryCommitInput
}

func (s *recordingCharacterMemoryService) Recall(context.Context, port.CharacterMemoryRecallInput) ([]model.Memory, error) {
	return nil, nil
}

func (s *recordingCharacterMemoryService) Commit(_ context.Context, input port.CharacterMemoryCommitInput) error {
	s.input = input
	return nil
}

type fixedServiceClock struct {
	now time.Time
}

func (c fixedServiceClock) Now() time.Time {
	return c.now
}

type fixedServiceIDs struct{}

func (fixedServiceIDs) New(prefix string) string {
	return prefix + "_1"
}

type completionOnlyStoryEventStore struct {
	forkActionStoryEventStore
	appended     []model.StoryEvent
	branchID     string
	branchHeadID string
}

func (s *completionOnlyStoryEventStore) AppendEvent(_ context.Context, event model.StoryEvent) (model.StoryEvent, error) {
	switch event.Kind {
	case model.EventKindActionScheduled:
		event.ID = "scheduled_1"
	case model.EventKindActionCompleted:
		event.ID = "completed_1"
	default:
		event.ID = "unexpected_1"
	}
	s.appended = append(s.appended, event)
	return event, nil
}

func (s *completionOnlyStoryEventStore) UpdateBranchHead(_ context.Context, branchID string, headEventID string) error {
	s.branchID = branchID
	s.branchHeadID = headEventID
	return nil
}

type completionOnlyStorySessions struct {
	latestSpanStorySessions
	runID       string
	headEventID string
}

func (s *completionOnlyStorySessions) UpdateRunHead(_ context.Context, runID string, headEventID string) error {
	s.runID = runID
	s.headEventID = headEventID
	return nil
}
