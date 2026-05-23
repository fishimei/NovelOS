package config

import "testing"

func TestLoadRunExecutorDefaults(t *testing.T) {
	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if !cfg.RunExecutor.Enabled {
		t.Fatalf("RunExecutor.Enabled = false, want true")
	}
	if cfg.RunExecutor.PollIntervalSeconds != 2 {
		t.Fatalf("RunExecutor.PollIntervalSeconds = %d, want 2", cfg.RunExecutor.PollIntervalSeconds)
	}
	if cfg.RunExecutor.StaleAfterSeconds != 600 {
		t.Fatalf("RunExecutor.StaleAfterSeconds = %d, want 600", cfg.RunExecutor.StaleAfterSeconds)
	}
	if cfg.RunExecutor.BatchSize != 10 {
		t.Fatalf("RunExecutor.BatchSize = %d, want 10", cfg.RunExecutor.BatchSize)
	}
	if cfg.RunExecutor.RunTimeoutSeconds != 600 {
		t.Fatalf("RunExecutor.RunTimeoutSeconds = %d, want 600", cfg.RunExecutor.RunTimeoutSeconds)
	}
}
