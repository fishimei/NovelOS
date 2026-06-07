package service

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/fishimei/NovelOS/internal/application/model"
	"github.com/fishimei/NovelOS/internal/application/port"
	"github.com/fishimei/NovelOS/internal/domain"
)

const (
	StoryAutoStatusRunning  = "running"
	StoryAutoStatusStopping = "stopping"
	StoryAutoStatusStopped  = "stopped"
	StoryAutoStatusFailed   = "failed"
)

// StoryAutoRunner 持续按分支事件头推进故事世界。
// 自动运行状态落库保存；进程重启后由 Resume 恢复 running/stopping 状态。
type StoryAutoRunner struct {
	advancer *StorySessionAdvancer
	states   port.StoryAutoRunRepository
	sessions map[string]*storyAutoSession
	mu       sync.Mutex
}

type storyAutoSession struct {
	ID               string
	ProjectID        string
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

func NewStoryAutoRunner(advancer *StorySessionAdvancer, states port.StoryAutoRunRepository) *StoryAutoRunner {
	return &StoryAutoRunner{advancer: advancer, states: states, sessions: map[string]*storyAutoSession{}}
}

func (r *StoryAutoRunner) Resume(ctx context.Context) error {
	if r == nil || r.advancer == nil || r.states == nil {
		return nil
	}
	states, err := r.states.ListResumable(ctx)
	if err != nil {
		return err
	}
	for _, persisted := range states {
		state := storyAutoSessionFromModel(persisted)
		state.stop = make(chan struct{})
		state.done = make(chan struct{})
		if state.Status == StoryAutoStatusStopping {
			state.StopRequested = true
			close(state.stop)
		}
		r.mu.Lock()
		if existing, ok := r.sessions[state.SessionID]; ok && existing.Status == StoryAutoStatusRunning {
			r.mu.Unlock()
			continue
		}
		r.sessions[state.SessionID] = state
		r.mu.Unlock()
		go r.loop(context.Background(), state, model.AdvanceStorySessionInput{BranchID: state.BranchID, BaseEventID: state.BaseEventID, TickDelaySeconds: int(state.TickDelay / time.Second)})
	}
	return nil
}

func (r *StoryAutoRunner) Start(ctx context.Context, sessionID string, input model.AdvanceStorySessionInput) (StoryAutoRunHandle, error) {
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
	r.mu.Unlock()

	session, err := r.advancer.sessions.GetSessionByID(ctx, sessionID)
	if err != nil {
		return StoryAutoRunHandle{}, err
	}
	persisted := model.StoryAutoRunState{
		ProjectID:        session.ProjectID,
		SessionID:        sessionID,
		BranchID:         input.BranchID,
		BaseEventID:      input.BaseEventID,
		Status:           StoryAutoStatusRunning,
		TickDelaySeconds: int(normalizeStoryAutoTickDelay(input) / time.Second),
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if r.states != nil {
		persisted, err = r.states.Upsert(ctx, persisted)
		if err != nil {
			return StoryAutoRunHandle{}, err
		}
	}
	state := storyAutoSessionFromModel(persisted)
	state.stop = make(chan struct{})
	state.done = make(chan struct{})
	if state.TickDelay == 0 {
		state.TickDelay = normalizeStoryAutoTickDelay(input)
	}

	r.mu.Lock()
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
		if r.states != nil {
			persisted, err := r.states.GetBySessionID(context.Background(), sessionID)
			if err == nil {
				persisted.Status = StoryAutoStatusStopped
				persisted.StopRequested = false
				persisted.CurrentRunID = ""
				updated, updateErr := r.states.Update(context.Background(), persisted)
				if updateErr == nil {
					return storyAutoSessionFromModel(updated).handleLocked()
				}
			}
		}
		return StoryAutoRunHandle{SessionID: sessionID, Status: StoryAutoStatusStopped}
	}
	if !state.StopRequested {
		state.StopRequested = true
		state.Status = StoryAutoStatusStopping
		state.UpdatedAt = time.Now().UTC()
		close(state.stop)
		r.persistStateLocked(context.Background(), state)
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
	if state, ok := r.sessions[sessionID]; ok {
		handle := state.handleLocked()
		r.mu.Unlock()
		return handle
	}
	r.mu.Unlock()
	if r.states != nil {
		persisted, err := r.states.GetBySessionID(context.Background(), sessionID)
		if err == nil {
			return storyAutoSessionFromModel(persisted).handleLocked()
		}
	}
	return StoryAutoRunHandle{SessionID: sessionID, Status: StoryAutoStatusStopped}
}

func (r *StoryAutoRunner) loop(ctx context.Context, state *storyAutoSession, input model.AdvanceStorySessionInput) {
	defer close(state.done)
	current := input
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
	r.persistStateLocked(context.Background(), state)
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
	r.persistStateLocked(context.Background(), state)
}

func (r *StoryAutoRunner) finishStopped(state *storyAutoSession) {
	r.mu.Lock()
	defer r.mu.Unlock()
	state.Status = StoryAutoStatusStopped
	state.CurrentRunID = ""
	state.StopRequested = false
	state.UpdatedAt = time.Now().UTC()
	r.persistStateLocked(context.Background(), state)
}

func (r *StoryAutoRunner) finishFailed(state *storyAutoSession, err string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	state.Status = StoryAutoStatusFailed
	state.LastError = err
	state.CurrentRunID = ""
	state.UpdatedAt = time.Now().UTC()
	r.persistStateLocked(context.Background(), state)
}

func (r *StoryAutoRunner) persistStateLocked(ctx context.Context, state *storyAutoSession) {
	if r.states == nil || state.ID == "" {
		return
	}
	updated, err := r.states.Update(ctx, state.model())
	if err != nil {
		log.Printf("persist story auto state %s failed: %v", state.SessionID, err)
		return
	}
	state.ID = updated.ID
	state.UpdatedAt = updated.UpdatedAt
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

func (s *storyAutoSession) model() model.StoryAutoRunState {
	return model.StoryAutoRunState{
		ID:               s.ID,
		ProjectID:        s.ProjectID,
		SessionID:        s.SessionID,
		BranchID:         s.BranchID,
		BaseEventID:      s.BaseEventID,
		CurrentRunID:     s.CurrentRunID,
		Status:           s.Status,
		StopRequested:    s.StopRequested,
		Iterations:       s.Iterations,
		LastError:        s.LastError,
		TickDelaySeconds: int(s.TickDelay / time.Second),
		CreatedAt:        s.CreatedAt,
		UpdatedAt:        s.UpdatedAt,
		LastRunStartedAt: s.LastRunStartedAt,
		LastCompletedAt:  s.LastCompletedAt,
	}
}

func storyAutoSessionFromModel(state model.StoryAutoRunState) *storyAutoSession {
	return &storyAutoSession{
		ID:               state.ID,
		ProjectID:        state.ProjectID,
		SessionID:        state.SessionID,
		BranchID:         state.BranchID,
		BaseEventID:      state.BaseEventID,
		CurrentRunID:     state.CurrentRunID,
		Status:           state.Status,
		Iterations:       state.Iterations,
		LastError:        state.LastError,
		TickDelay:        time.Duration(state.TickDelaySeconds) * time.Second,
		StopRequested:    state.StopRequested,
		CreatedAt:        state.CreatedAt,
		UpdatedAt:        state.UpdatedAt,
		LastCompletedAt:  state.LastCompletedAt,
		LastRunStartedAt: state.LastRunStartedAt,
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
