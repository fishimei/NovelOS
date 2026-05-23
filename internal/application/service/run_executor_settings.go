package service

import "time"

const (
	defaultRunExecutorPollInterval = 2 * time.Second
	defaultRunExecutorStaleAfter   = 10 * time.Minute
	defaultRunExecutorRunTimeout   = 10 * time.Minute
	defaultRunExecutorBatchSize    = 10
)

// RunExecutorSettings controls the lightweight DB scanner loop.
type RunExecutorSettings struct {
	Enabled             bool `mapstructure:"enabled"`
	PollIntervalSeconds int  `mapstructure:"poll_interval_seconds"`
	StaleAfterSeconds   int  `mapstructure:"stale_after_seconds"`
	BatchSize           int  `mapstructure:"batch_size"`
	RunTimeoutSeconds   int  `mapstructure:"run_timeout_seconds"`
}

func (s RunExecutorSettings) normalized() RunExecutorSettings {
	if s.PollIntervalSeconds <= 0 {
		s.PollIntervalSeconds = int(defaultRunExecutorPollInterval / time.Second)
	}
	if s.StaleAfterSeconds <= 0 {
		s.StaleAfterSeconds = int(defaultRunExecutorStaleAfter / time.Second)
	}
	if s.BatchSize <= 0 {
		s.BatchSize = defaultRunExecutorBatchSize
	}
	if s.RunTimeoutSeconds <= 0 {
		s.RunTimeoutSeconds = int(defaultRunExecutorRunTimeout / time.Second)
	}
	return s
}

func (s RunExecutorSettings) pollInterval() time.Duration {
	return time.Duration(s.normalized().PollIntervalSeconds) * time.Second
}

func (s RunExecutorSettings) staleAfter() time.Duration {
	return time.Duration(s.normalized().StaleAfterSeconds) * time.Second
}

func (s RunExecutorSettings) runTimeout() time.Duration {
	return time.Duration(s.normalized().RunTimeoutSeconds) * time.Second
}

func (s RunExecutorSettings) batchSize() int {
	return s.normalized().BatchSize
}
