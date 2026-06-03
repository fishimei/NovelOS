package chapterseq

import (
	"context"

	"github.com/fishimei/NovelOS/internal/application/model"
	"github.com/fishimei/NovelOS/internal/application/port"
)

const chapterPageSize = 1000

// NextChapterNumber returns the next chapter number after the highest existing
// number, keeping numbering stable even when chapters have gaps.
func NextChapterNumber(ctx context.Context, chapters port.ChapterRepository, projectID string) (int, error) {
	page := 1
	seen := 0
	maxNumber := 0

	for {
		result, err := chapters.ListByProjectID(ctx, projectID, model.PageInput{Page: page, PageSize: chapterPageSize})
		if err != nil {
			return 0, err
		}
		for _, chapter := range result.Items {
			if chapter.ChapterNumber > maxNumber {
				maxNumber = chapter.ChapterNumber
			}
		}
		seen += len(result.Items)
		if len(result.Items) == 0 || seen >= result.Total {
			return maxNumber + 1, nil
		}
		page++
	}
}
