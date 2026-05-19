package memory

import (
	"context"
	"strings"

	"github.com/fishimei/NovelOS/internal/application/model"
	"github.com/fishimei/NovelOS/internal/application/port"
)

type FanoutService struct {
	primary port.CharacterMemoryService
	writers []port.CharacterMemoryService
}

func NewFanoutService(primary port.CharacterMemoryService, writers ...port.CharacterMemoryService) *FanoutService {
	return &FanoutService{primary: primary, writers: writers}
}

func (s *FanoutService) Recall(ctx context.Context, input port.CharacterMemoryRecallInput) ([]model.Memory, error) {
	if s == nil || s.primary == nil {
		return nil, nil
	}
	return s.primary.Recall(ctx, input)
}

func (s *FanoutService) Commit(ctx context.Context, input port.CharacterMemoryCommitInput) error {
	if s == nil {
		return nil
	}
	var firstErr error
	for _, writer := range s.writers {
		if writer == nil {
			continue
		}
		if err := writer.Commit(ctx, input); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func ProviderEnabled(provider string, target string) bool {
	for _, part := range strings.Split(provider, "+") {
		if strings.EqualFold(strings.TrimSpace(part), target) {
			return true
		}
	}
	return false
}
