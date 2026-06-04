package service

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/fishimei/NovelOS/internal/application/model"
	"github.com/fishimei/NovelOS/internal/domain"
)

const (
	StoryAutoStatusRunning  = "running"
	StoryAutoStatusStopping = "stopping"
	StoryAutoStatusStopped  = "stopped"
	StoryAutoStatusFailed   = "failed"
)

// StoryAutoRunner 持续按分支事件头推进故事世界。
// 它维护“开始一次后持续运行，停止一次后暂停”的会话级自动演绎状态；单轮演绎仍由 StorySessionAdvancer 和 RunExecutor 处理。
type StoryAutoRunner struct {
	advancer *StorySessionAdvancer
	sessions map[string]*storyAutoSession
	mu       sync.Mutex
}

type storyAutoSession struct {
	SessionID        string
	BranchID         string
	BaseEventID      string
	CurrentRunID     string
	Status           string
	Iterations       int
	LastError        string
	TickDelay        time.Duration
	StopRequested    bool
	stop             chan struct{}
	done             chan struct{}
	CreatedAt        time.Time
	UpdatedAt        time.Time
	LastCompletedAt  *time.Time
	LastRunStartedAt *time.Time
}

type StoryAutoRunHandle struct {
	SessionID        string     `json:"session_id"`
	BranchID         string     `json:"branch_id,omitempty"`
	BaseEventID      string     `json:"base_event_id,omitempty"`
	CurrentRunID     string     `json:"current_run_id,omitempty"`
	Status           string     `json:"status"`
	Iterations       int        `json:"iterations"`
	LastError        string     `json:"last_error,omitempty"`
	TickDelaySeconds int        `json:"tick_delay_seconds"`
	StopRequested    bool       `json:"stop_requested"`
	CreatedAt        time.Time  `json:"created_at,omitempty"`
	UpdatedAt        time.Time  `json:"updated_at,omitempty"`
	LastCompletedAt  *time.Time `json:"last_completed_at,omitempty"`
	LastRunStartedAt *time.Time `json:"last_run_started_at,omitempty"`
}

func NewStoryAutoRunner(advancer *StorySessionAdvancer) *StoryAutoRunner {
	return &StoryAutoRunner{advancer: advancer, sessions: map[string]*storyAutoSession{}}
}

func (r *StoryAutoRunner) Start(_ context.Context, sessionID string, input model.AdvanceStorySessionInput) (StoryAutoRunHandle, error) {
	if r == nil || r.advancer == nil {
		return StoryAutoRunHandle{SessionID: sessionID, Status: StoryAutoStatusStopped}, nil
	}
	now := time.Now().UTC()
	r.mu.Lock()
	if existing, ok := r.sessions[sessionID]; ok && existing.Status == StoryAutoStatusRunning {
		handle := existing.handleLocked()
		r.mu.Unlock()
		return handle, nil
	}
	state := &storyAutoSession{
		SessionID:     sessionID,
		BranchID:      input.BranchID,
		BaseEventID:   input.BaseEventID,
		Status:        StoryAutoStatusRunning,
		TickDelay:     normalizeStoryAutoTickDelay(input),
		stop:          make(chan struct{}),
		done:          make(chan struct{}),
		CreatedAt:     now,
		UpdatedAt:     now,
		StopRequested: false,
	}
	r.sessions[sessionID] = state
	handle := state.handleLocked()
	r.mu.Unlock()
	go r.loop(context.Background(), state, input)
	return handle, nil
}

func (r *StoryAutoRunner) Stop(sessionID string) StoryAutoRunHandle {
	if r == nil {
		return StoryAutoRunHandle{SessionID: sessionID, Status: StoryAutoStatusStopped}
	}
	r.mu.Lock()
	state, ok := r.sessions[sessionID]
	if !ok {
		r.mu.Unlock()
		return StoryAutoRunHandle{SessionID: sessionID, Status: StoryAutoStatusStopped}
	}
	if !state.StopRequested {
		state.StopRequested = true
		state.Status = StoryAutoStatusStopping
		state.UpdatedAt = time.Now().UTC()
		close(state.stop)
	}
	handle := state.handleLocked()
	r.mu.Unlock()
	return handle
}

func (r *StoryAutoRunner) Status(sessionID string) StoryAutoRunHandle {
	if r == nil {
		return StoryAutoRunHandle{SessionID: sessionID, Status: StoryAutoStatusStopped}
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if state, ok := r.sessions[sessionID]; ok {
		return state.handleLocked()
	}
	return StoryAutoRunHandle{SessionID: sessionID, Status: StoryAutoStatusStopped}
}

func (r *StoryAutoRunner) loop(ctx context.Context, state *storyAutoSession, input model.AdvanceStorySessionInput) {
	defer close(state.done)
	current := input
	current.AuthorMessage = ""
	current.AdvanceMode = "auto"
	for {
		if r.shouldStop(state) {
			r.finishStopped(state)
			return
		}
		run, err := r.advancer.CreateAutoRun(ctx, state.SessionID, current)
		if err != nil {
			r.finishFailed(state, err.Error())
			log.Printf("story auto runner create run for %s failed: %v", state.SessionID, err)
			return
		}
		r.markRunStarted(state, run)
		if err := r.waitRunTerminal(ctx, state, run.RunID); err != nil {
			if r.shouldStop(state) {
				r.finishStopped(state)
				return
			}
			r.finishFailed(state, err.Error())
			log.Printf("story auto runner wait run %s failed: %v", run.RunID, err)
			return
		}
		latest, err := r.advancer.sessions.GetRunByID(ctx, run.RunID)
		if err != nil {
			r.finishFailed(state, err.Error())
			log.Printf("story auto runner reload run %s failed: %v", run.RunID, err)
			return
		}
		if latest.Status != domain.RunStatusCompleted {
			r.finishFailed(state, "auto run ended with status "+latest.Status)
			return
		}
		r.markRunCompleted(state, latest)
		current.BranchID = latest.BranchID
		current.BaseEventID = latest.HeadEventID
		if !r.sleepBetweenTicks(ctx, state) {
			r.finishStopped(state)
			return
		}
	}
}

func (r *StoryAutoRunner) waitRunTerminal(ctx context.Context, state *storyAutoSession, runID string) error {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-state.stop:
			_, _ = r.advancer.RequestStop(context.Background(), runID)
			return context.Canceled
		case <-ticker.C:
			run, err := r.advancer.sessions.GetRunByID(ctx, runID)
			if err != nil {
				return err
			}
			switch run.Status {
			case domain.RunStatusCompleted, domain.RunStatusFailed, domain.RunStatusCancelled, domain.RunStatusCut:
				return nil
			}
		}
	}
}

func (r *StoryAutoRunner) sleepBetweenTicks(ctx context.Context, state *storyAutoSession) bool {
	if state.TickDelay <= 0 {
		return true
	}
	timer := time.NewTimer(state.TickDelay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-state.stop:
		return false
	case <-timer.C:
		return true
	}
}

func (r *StoryAutoRunner) shouldStop(state *storyAutoSession) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return state.StopRequested || state.Status == StoryAutoStatusStopping || state.Status == StoryAutoStatusStopped
}

func (r *StoryAutoRunner) markRunStarted(state *storyAutoSession, run model.StoryRun) {
	now := time.Now().UTC()
	r.mu.Lock()
	defer r.mu.Unlock()
	state.CurrentRunID = run.RunID
	state.BranchID = run.BranchID
	state.BaseEventID = run.BaseEventID
	state.Status = StoryAutoStatusRunning
	state.LastError = ""
	state.LastRunStartedAt = &now
	state.UpdatedAt = now
}

func (r *StoryAutoRunner) markRunCompleted(state *storyAutoSession, run model.StoryRun) {
	now := time.Now().UTC()
	r.mu.Lock()
	defer r.mu.Unlock()
	state.Iterations++
	state.BranchID = run.BranchID
	state.BaseEventID = run.HeadEventID
	state.CurrentRunID = ""
	state.LastCompletedAt = &now
	state.UpdatedAt = now
}

func (r *StoryAutoRunner) finishStopped(state *storyAutoSession) {
	r.mu.Lock()
	defer r.mu.Unlock()
	state.Status = StoryAutoStatusStopped
	state.CurrentRunID = ""
	state.StopRequested = false
	state.UpdatedAt = time.Now().UTC()
}

func (r *StoryAutoRunner) finishFailed(state *storyAutoSession, err string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	state.Status = StoryAutoStatusFailed
	state.LastError = err
	state.CurrentRunID = ""
	state.UpdatedAt = time.Now().UTC()
}

func (s *storyAutoSession) handleLocked() StoryAutoRunHandle {
	return StoryAutoRunHandle{
		SessionID:        s.SessionID,
		BranchID:         s.BranchID,
		BaseEventID:      s.BaseEventID,
		CurrentRunID:     s.CurrentRunID,
		Status:           s.Status,
		Iterations:       s.Iterations,
		LastError:        s.LastError,
		TickDelaySeconds: int(s.TickDelay / time.Second),
		StopRequested:    s.StopRequested,
		CreatedAt:        s.CreatedAt,
		UpdatedAt:        s.UpdatedAt,
		LastCompletedAt:  s.LastCompletedAt,
		LastRunStartedAt: s.LastRunStartedAt,
	}
}

func normalizeStoryAutoTickDelay(input model.AdvanceStorySessionInput) time.Duration {
	if input.TickDelaySeconds < 0 {
		return 0
	}
	if input.TickDelaySeconds > 0 {
		return time.Duration(input.TickDelaySeconds) * time.Second
	}
	return time.Second
}
