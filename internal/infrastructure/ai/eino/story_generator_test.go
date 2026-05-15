package eino

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/fishimei/NovelOS/internal/application/model"
	"github.com/fishimei/NovelOS/internal/application/port"
	"github.com/fishimei/NovelOS/internal/config"
)

type fakeChapterRepository struct{}

func (fakeChapterRepository) ListByProjectID(context.Context, string, model.PageInput) (model.ListResult[model.Chapter], error) {
	return model.ListResult[model.Chapter]{Items: []model.Chapter{{ChapterNumber: 2}}}, nil
}

func (fakeChapterRepository) GetByID(context.Context, string) (model.Chapter, error) {
	return model.Chapter{}, nil
}

func (fakeChapterRepository) Create(context.Context, model.Chapter) (model.Chapter, error) {
	return model.Chapter{}, nil
}

type fakeClock struct{}

func (fakeClock) Now() time.Time {
	return time.Unix(100, 0).UTC()
}

type fakeIDGenerator struct{}

func (fakeIDGenerator) New(prefix string) string {
	return prefix + "_id"
}

func TestBuildResultUsesTurnPlan(t *testing.T) {
	generator := &StoryRunGenerator{
		cfg:   config.AIConfig{},
		deps:  storyGeneratorDeps{chapters: fakeChapterRepository{}},
		clock: fakeClock{},
		ids:   fakeIDGenerator{},
	}
	result, err := generator.buildResult(context.Background(), port.StoryRunGenerationInput{
		Run: model.StoryRun{RunID: "run_1", SessionID: "story_1", ProjectID: "project_1"},
		Session: model.StorySession{
			Title:             "第一章",
			OpeningSituation:  "密信被截获",
			LastAuthorMessage: "让角色围绕密信行动",
		},
	}, StoryPlanResult{
		Summary:    "密信引发怀疑",
		StopReason: "怀疑已经形成",
		Turns: []StoryTurnPlan{
			{TurnIndex: 1, ActorID: "character_1", ActorName: "林澈", ActionType: "speak", Intent: "试探对方"},
		},
	})
	if err != nil {
		t.Fatalf("buildResult returned error: %v", err)
	}
	if result.Status != "review_required" {
		t.Fatalf("unexpected status %q", result.Status)
	}
	if result.Draft.ChapterNumber != 3 {
		t.Fatalf("expected chapter number 3, got %d", result.Draft.ChapterNumber)
	}
	if !strings.Contains(result.Draft.Content, "林澈") {
		t.Fatalf("expected draft content to include actor name, got %q", result.Draft.Content)
	}
	if result.MemoryPatch.ID == "" {
		t.Fatal("expected memory patch id")
	}
	if len(result.PlotVariable.RelatedCharacterIDs) != 1 || result.PlotVariable.RelatedCharacterIDs[0] != "character_1" {
		t.Fatalf("unexpected related character ids: %#v", result.PlotVariable.RelatedCharacterIDs)
	}
}
