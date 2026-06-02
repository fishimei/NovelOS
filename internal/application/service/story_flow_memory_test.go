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
