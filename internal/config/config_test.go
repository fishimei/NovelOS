package config

import (
	"os"
	"path/filepath"
	"testing"
)

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

func TestLoadAgentLimitDefaults(t *testing.T) {
	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.AI.StoryAgent.MaxTurns != DefaultStoryAgentMaxTurns {
		t.Fatalf("StoryAgent.MaxTurns = %d, want %d", cfg.AI.StoryAgent.MaxTurns, DefaultStoryAgentMaxTurns)
	}
	if cfg.AI.StoryAgent.MaxReactSteps != DefaultStoryAgentMaxReactSteps {
		t.Fatalf("StoryAgent.MaxReactSteps = %d, want %d", cfg.AI.StoryAgent.MaxReactSteps, DefaultStoryAgentMaxReactSteps)
	}
	if cfg.AI.StoryAgent.MaxChapterTokens != DefaultStoryAgentMaxChapterTokens {
		t.Fatalf("StoryAgent.MaxChapterTokens = %d, want %d", cfg.AI.StoryAgent.MaxChapterTokens, DefaultStoryAgentMaxChapterTokens)
	}
	if cfg.AI.StoryAgent.MaxTurnTokens != DefaultStoryAgentMaxTurnTokens {
		t.Fatalf("StoryAgent.MaxTurnTokens = %d, want %d", cfg.AI.StoryAgent.MaxTurnTokens, DefaultStoryAgentMaxTurnTokens)
	}
	if cfg.AI.StoryAgent.MaxAssemblerTokens != DefaultStoryAgentMaxAssemblerTokens {
		t.Fatalf("StoryAgent.MaxAssemblerTokens = %d, want %d", cfg.AI.StoryAgent.MaxAssemblerTokens, DefaultStoryAgentMaxAssemblerTokens)
	}
	if cfg.AI.DialogueAgent.MaxSteps != DefaultDialogueAgentMaxSteps {
		t.Fatalf("DialogueAgent.MaxSteps = %d, want %d", cfg.AI.DialogueAgent.MaxSteps, DefaultDialogueAgentMaxSteps)
	}
	if cfg.AI.DialogueAgent.AutoPilot {
		t.Fatal("DialogueAgent.AutoPilot = true, want false")
	}
}

func TestLoadAgentLimitOverrides(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	content := []byte(`
ai:
  story_agent:
    max_turns: 40
    max_react_steps: 140
    max_chapter_tokens: 6000
    max_turn_tokens: 1500
    max_assembler_tokens: 5000
  dialogue_agent:
    max_steps: 48
    auto_pilot: true
`)
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.AI.StoryAgent.MaxTurns != 40 {
		t.Fatalf("StoryAgent.MaxTurns = %d, want 40", cfg.AI.StoryAgent.MaxTurns)
	}
	if cfg.AI.StoryAgent.MaxReactSteps != 140 {
		t.Fatalf("StoryAgent.MaxReactSteps = %d, want 140", cfg.AI.StoryAgent.MaxReactSteps)
	}
	if cfg.AI.StoryAgent.MaxChapterTokens != 6000 {
		t.Fatalf("StoryAgent.MaxChapterTokens = %d, want 6000", cfg.AI.StoryAgent.MaxChapterTokens)
	}
	if cfg.AI.StoryAgent.MaxTurnTokens != 1500 {
		t.Fatalf("StoryAgent.MaxTurnTokens = %d, want 1500", cfg.AI.StoryAgent.MaxTurnTokens)
	}
	if cfg.AI.StoryAgent.MaxAssemblerTokens != 5000 {
		t.Fatalf("StoryAgent.MaxAssemblerTokens = %d, want 5000", cfg.AI.StoryAgent.MaxAssemblerTokens)
	}
	if cfg.AI.DialogueAgent.MaxSteps != 48 {
		t.Fatalf("DialogueAgent.MaxSteps = %d, want 48", cfg.AI.DialogueAgent.MaxSteps)
	}
	if !cfg.AI.DialogueAgent.AutoPilot {
		t.Fatal("DialogueAgent.AutoPilot = false, want true")
	}
}
