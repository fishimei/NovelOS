package repository

import (
	"fmt"

	"github.com/fishimei/NovelOS/internal/application/port"
	"gorm.io/gorm"
)

type Repositories struct {
	port.Repositories
}

func New(db *gorm.DB, ids port.IDGenerator, clock port.Clock) Repositories {
	base := &container{db: db, ids: ids, clock: clock}

	return Repositories{
		Repositories: port.Repositories{
			Projects:         &projectRepository{container: base},
			AuthorBibles:     &authorBibleRepository{container: base},
			WorldState:       &worldStateRepository{container: base},
			Characters:       &characterRepository{container: base},
			Relationships:    &relationshipRepository{container: base},
			SetupSessions:    &setupSessionRepository{container: base},
			DialogueSessions: &dialogueSessionRepository{container: base},
			StorySessions:    &storySessionRepository{container: base},
			StoryTimeline:    &storyTimelineRepository{container: base},
			Chapters:         &chapterRepository{container: base},
			Memories:         &memoryRepository{container: base},
			Audit:            &auditRepository{container: base},
		},
	}
}

func pageOffset(pageInput any) (limit int, offset int) {
	switch p := pageInput.(type) {
	case interface {
		GetPage() int
		GetPageSize() int
	}:
		page := p.GetPage()
		pageSize := p.GetPageSize()
		if page <= 0 {
			page = 1
		}
		if pageSize <= 0 {
			pageSize = 20
		}
		return pageSize, (page - 1) * pageSize
	default:
		return 20, 0
	}
}

func countQueryError(entity string, err error) error {
	return fmt.Errorf("count %s: %w", entity, err)
}
