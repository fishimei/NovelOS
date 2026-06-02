package port

import (
	"context"
	"time"

	"github.com/fishimei/NovelOS/internal/application/model"
)

const (
	RunKindSetup    = "setup"
	RunKindStory    = "story"
	RunKindDialogue = "dialogue"
)

// RunExecutionRepository exposes durable setup/story run work discovery and claim operations.
type RunExecutionRepository interface {
	ListRunnableRuns(ctx context.Context, staleBefore time.Time, limit int) ([]model.RunExecutionWork, error)
	ClaimRun(ctx context.Context, work model.RunExecutionWork, staleBefore time.Time) (bool, error)
}
