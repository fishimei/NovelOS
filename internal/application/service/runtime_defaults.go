package service

import (
	"context"
	"sync"
	"time"

	"github.com/fishimei/NovelOS/internal/application/port"
)

type systemClockImpl struct{}

func (systemClockImpl) Now() time.Time {
	return time.Now().UTC()
}

type inMemoryEventStream struct {
	mu   sync.Mutex
	subs map[string][]chan port.GenerationEvent
}

func NewInMemoryEventStream() port.GenerationEventStream {
	return &inMemoryEventStream{subs: make(map[string][]chan port.GenerationEvent)}
}

func (s *inMemoryEventStream) Publish(_ context.Context, runID string, event port.GenerationEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, ch := range s.subs[runID] {
		select {
		case ch <- event:
		default:
		}
	}
	return nil
}

func (s *inMemoryEventStream) Subscribe(ctx context.Context, runID string) (<-chan port.GenerationEvent, func(), error) {
	return s.SubscribeAfter(ctx, runID, port.GenerationEventCursor{})
}

func (s *inMemoryEventStream) SubscribeAfter(_ context.Context, runID string, _ port.GenerationEventCursor) (<-chan port.GenerationEvent, func(), error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ch := make(chan port.GenerationEvent, 16)
	s.subs[runID] = append(s.subs[runID], ch)
	cancel := func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		subs := s.subs[runID]
		filtered := make([]chan port.GenerationEvent, 0, len(subs))
		for _, sub := range subs {
			if sub != ch {
				filtered = append(filtered, sub)
			}
		}
		s.subs[runID] = filtered
		close(ch)
	}
	return ch, cancel, nil
}
