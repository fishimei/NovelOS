package service

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/fishimei/NovelOS/internal/application/model"
	"github.com/fishimei/NovelOS/internal/application/port"
)

// RunExecutor scans durable run state and dispatches claimed setup/story/dialogue runs.
type RunExecutor struct {
	repo             port.RunExecutionRepository
	setupAdvancer    *SetupSessionAdvancer
	storyAdvancer    *StorySessionAdvancer
	dialogueAdvancer *DialogueSessionAdvancer
	settings         RunExecutorSettings
	clock            port.Clock
	sem              chan struct{}
}

func NewRunExecutor(repo port.RunExecutionRepository, setupAdvancer *SetupSessionAdvancer, storyAdvancer *StorySessionAdvancer, dialogueAdvancer *DialogueSessionAdvancer, settings RunExecutorSettings, clock port.Clock) *RunExecutor {
	settings = settings.normalized()
	return &RunExecutor{repo: repo, setupAdvancer: setupAdvancer, storyAdvancer: storyAdvancer, dialogueAdvancer: dialogueAdvancer, settings: settings, clock: clock, sem: make(chan struct{}, settings.batchSize())}
}

func (e *RunExecutor) Start(ctx context.Context) {
	if e == nil || !e.settings.Enabled || e.repo == nil {
		return
	}
	go e.loop(ctx)
}

func (e *RunExecutor) loop(ctx context.Context) {
	ticker := time.NewTicker(e.settings.pollInterval())
	defer ticker.Stop()
	e.scan(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			e.scan(ctx)
		}
	}
}

func (e *RunExecutor) scan(ctx context.Context) {
	staleBefore := currentTime(e.clock).Add(-e.settings.staleAfter())
	works, err := e.repo.ListRunnableRuns(ctx, staleBefore, e.settings.batchSize())
	if err != nil {
		log.Printf("run executor scan failed: %v", err)
		return
	}
	var wg sync.WaitGroup
	for _, work := range works {
		select {
		case <-ctx.Done():
			return
		case e.sem <- struct{}{}:
		}
		wg.Add(1)
		go func(work model.RunExecutionWork) {
			defer wg.Done()
			defer func() { <-e.sem }()
			e.handle(ctx, work, staleBefore)
		}(work)
	}
	wg.Wait()
}

func (e *RunExecutor) handle(parent context.Context, work model.RunExecutionWork, staleBefore time.Time) {
	claimed, err := e.repo.ClaimRun(parent, work, staleBefore)
	if err != nil {
		log.Printf("run executor claim %s %s failed: %v", work.RunKind, work.RunID, err)
		return
	}
	if !claimed {
		return
	}
	ctx, cancel := context.WithTimeout(parent, e.settings.runTimeout())
	defer cancel()
	switch work.RunKind {
	case port.RunKindSetup:
		if e.setupAdvancer != nil {
			e.setupAdvancer.Generate(ctx, work.RunID)
		}
	case port.RunKindStory:
		if e.storyAdvancer != nil {
			e.storyAdvancer.Generate(ctx, work.RunID)
		}
	case port.RunKindDialogue:
		if e.dialogueAdvancer != nil {
			e.dialogueAdvancer.Generate(ctx, work.RunID)
		}
	default:
		log.Printf("run executor ignored unknown run kind %q for %s", work.RunKind, work.RunID)
	}
}
