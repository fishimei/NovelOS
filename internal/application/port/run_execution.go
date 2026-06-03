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
	ClaimRun(ctx context.Context, work model.RunExecutionWork, lease RunLease, staleBefore time.Time) (bool, error)
}

type RunLease struct {
	Owner    string
	Duration time.Duration
}

func (l RunLease) Valid() bool {
	return l.Owner != "" && l.Duration > 0
}

func (l RunLease) ExpiresAt(now time.Time) time.Time {
	duration := l.Duration
	if duration <= 0 {
		duration = 10 * time.Minute
	}
	return now.Add(duration)
}

type runLeaseContextKey struct{}

func ContextWithRunLease(ctx context.Context, lease RunLease) context.Context {
	if !lease.Valid() {
		return ctx
	}
	return context.WithValue(ctx, runLeaseContextKey{}, lease)
}

func RunLeaseFromContext(ctx context.Context) (RunLease, bool) {
	lease, ok := ctx.Value(runLeaseContextKey{}).(RunLease)
	return lease, ok && lease.Valid()
}
