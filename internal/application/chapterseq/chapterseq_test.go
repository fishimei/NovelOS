package chapterseq

import (
	"context"
	"testing"

	"github.com/fishimei/NovelOS/internal/application/model"
)

type fakeChapterRepository struct {
	pages map[int]model.ListResult[model.Chapter]
}

func (r fakeChapterRepository) ListByProjectID(_ context.Context, _ string, pageInput model.PageInput) (model.ListResult[model.Chapter], error) {
	return r.pages[pageInput.Page], nil
}

func (fakeChapterRepository) GetByID(context.Context, string) (model.Chapter, error) {
	return model.Chapter{}, nil
}

func (fakeChapterRepository) Create(context.Context, model.Chapter) (model.Chapter, error) {
	return model.Chapter{}, nil
}

func TestNextChapterNumberUsesMaxExistingNumber(t *testing.T) {
	repo := fakeChapterRepository{pages: map[int]model.ListResult[model.Chapter]{
		1: {
			Items: []model.Chapter{
				{ChapterNumber: 1},
				{ChapterNumber: 3},
			},
			Total: 2,
		},
	}}

	next, err := NextChapterNumber(context.Background(), repo, "project_1")
	if err != nil {
		t.Fatalf("NextChapterNumber() error = %v", err)
	}
	if next != 4 {
		t.Fatalf("NextChapterNumber() = %d, want 4", next)
	}
}

func TestNextChapterNumberScansAllPages(t *testing.T) {
	repo := fakeChapterRepository{pages: map[int]model.ListResult[model.Chapter]{
		1: {
			Items: []model.Chapter{{ChapterNumber: 1}},
			Total: 2,
		},
		2: {
			Items: []model.Chapter{{ChapterNumber: 1002}},
			Total: 2,
		},
	}}

	next, err := NextChapterNumber(context.Background(), repo, "project_1")
	if err != nil {
		t.Fatalf("NextChapterNumber() error = %v", err)
	}
	if next != 1003 {
		t.Fatalf("NextChapterNumber() = %d, want 1003", next)
	}
}
