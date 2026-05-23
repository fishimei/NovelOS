package service

import (
	"testing"
	"time"
)

func TestRunExecutorSettingsNormalizeDefaults(t *testing.T) {
	settings := RunExecutorSettings{}.normalized()

	if settings.PollIntervalSeconds != 2 {
		t.Fatalf("PollIntervalSeconds = %d, want 2", settings.PollIntervalSeconds)
	}
	if settings.StaleAfterSeconds != 600 {
		t.Fatalf("StaleAfterSeconds = %d, want 600", settings.StaleAfterSeconds)
	}
	if settings.BatchSize != 10 {
		t.Fatalf("BatchSize = %d, want 10", settings.BatchSize)
	}
	if settings.RunTimeoutSeconds != 600 {
		t.Fatalf("RunTimeoutSeconds = %d, want 600", settings.RunTimeoutSeconds)
	}
	if settings.pollInterval() != 2*time.Second {
		t.Fatalf("pollInterval = %s, want 2s", settings.pollInterval())
	}
	if settings.staleAfter() != 10*time.Minute {
		t.Fatalf("staleAfter = %s, want 10m", settings.staleAfter())
	}
	if settings.runTimeout() != 10*time.Minute {
		t.Fatalf("runTimeout = %s, want 10m", settings.runTimeout())
	}
}

func TestRunExecutorSettingsNormalizeKeepsValidValues(t *testing.T) {
	settings := RunExecutorSettings{
		Enabled:             true,
		PollIntervalSeconds: 5,
		StaleAfterSeconds:   30,
		BatchSize:           3,
		RunTimeoutSeconds:   40,
	}.normalized()

	if !settings.Enabled {
		t.Fatalf("Enabled = false, want true")
	}
	if settings.PollIntervalSeconds != 5 {
		t.Fatalf("PollIntervalSeconds = %d, want 5", settings.PollIntervalSeconds)
	}
	if settings.StaleAfterSeconds != 30 {
		t.Fatalf("StaleAfterSeconds = %d, want 30", settings.StaleAfterSeconds)
	}
	if settings.BatchSize != 3 {
		t.Fatalf("BatchSize = %d, want 3", settings.BatchSize)
	}
	if settings.RunTimeoutSeconds != 40 {
		t.Fatalf("RunTimeoutSeconds = %d, want 40", settings.RunTimeoutSeconds)
	}
}
