package config

import (
	"os"
	"path/filepath"
	"strings"
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
	if cfg.AI.StoryAgent.MaxSceneTokens != DefaultStoryAgentMaxSceneTokens {
		t.Fatalf("StoryAgent.MaxSceneTokens = %d, want %d", cfg.AI.StoryAgent.MaxSceneTokens, DefaultStoryAgentMaxSceneTokens)
	}
	if cfg.AI.StoryAgent.MaxReflectTokens != DefaultStoryAgentMaxReflectTokens {
		t.Fatalf("StoryAgent.MaxReflectTokens = %d, want %d", cfg.AI.StoryAgent.MaxReflectTokens, DefaultStoryAgentMaxReflectTokens)
	}
	if cfg.AI.StoryAgent.ScenePrompt == "" {
		t.Fatal("StoryAgent.ScenePrompt is empty")
	}
	if cfg.AI.StoryAgent.ReflectPrompt == "" {
		t.Fatal("StoryAgent.ReflectPrompt is empty")
	}
	if !strings.Contains(cfg.AI.StoryAgent.SimulationPrompt, "participant_ids") || !strings.Contains(cfg.AI.StoryAgent.SimulationPrompt, "target_location_key") {
		t.Fatalf("StoryAgent.SimulationPrompt must request intent fields, got %q", cfg.AI.StoryAgent.SimulationPrompt)
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
    max_scene_tokens: 9000
    max_reflect_tokens: 3500
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
	if cfg.AI.StoryAgent.MaxSceneTokens != 9000 {
		t.Fatalf("StoryAgent.MaxSceneTokens = %d, want 9000", cfg.AI.StoryAgent.MaxSceneTokens)
	}
	if cfg.AI.StoryAgent.MaxReflectTokens != 3500 {
		t.Fatalf("StoryAgent.MaxReflectTokens = %d, want 3500", cfg.AI.StoryAgent.MaxReflectTokens)
	}
	if cfg.AI.DialogueAgent.MaxSteps != 48 {
		t.Fatalf("DialogueAgent.MaxSteps = %d, want 48", cfg.AI.DialogueAgent.MaxSteps)
	}
	if !cfg.AI.DialogueAgent.AutoPilot {
		t.Fatal("DialogueAgent.AutoPilot = false, want true")
	}
}
