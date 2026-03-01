package claude

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadProjectStats_ValidConfig(t *testing.T) {
	configPath := filepath.Join("testdata", "claude.json")
	stats, err := LoadProjectStats(configPath, "/test/project")
	if err != nil {
		t.Fatalf("LoadProjectStats() error = %v", err)
	}
	if stats == nil {
		t.Fatal("LoadProjectStats() returned nil")
	}
	if stats.LastCost != 1.178 {
		t.Errorf("LastCost = %f, want 1.178", stats.LastCost)
	}
	if stats.LastDuration != 1212000 {
		t.Errorf("LastDuration = %d, want 1212000", stats.LastDuration)
	}
	if stats.LastTotalInputTokens != 150000 {
		t.Errorf("LastTotalInputTokens = %d, want 150000", stats.LastTotalInputTokens)
	}
	if stats.LastTotalOutputTokens != 25000 {
		t.Errorf("LastTotalOutputTokens = %d, want 25000", stats.LastTotalOutputTokens)
	}
}

func TestLoadProjectStats_LastSessionID(t *testing.T) {
	configPath := filepath.Join("testdata", "claude.json")
	stats, err := LoadProjectStats(configPath, "/test/project")
	if err != nil {
		t.Fatalf("LoadProjectStats() error = %v", err)
	}
	if stats.LastSessionID != "7ba50137-65c8-4349-b420-cdce14c38d2a" {
		t.Errorf("LastSessionID = %q, want \"7ba50137-65c8-4349-b420-cdce14c38d2a\"", stats.LastSessionID)
	}
}

func TestLoadProjectStats_ModelUsage(t *testing.T) {
	configPath := filepath.Join("testdata", "claude.json")
	stats, err := LoadProjectStats(configPath, "/test/project")
	if err != nil {
		t.Fatalf("LoadProjectStats() error = %v", err)
	}
	if len(stats.LastModelUsage) != 2 {
		t.Fatalf("LastModelUsage has %d models, want 2", len(stats.LastModelUsage))
	}
	sonnet, ok := stats.LastModelUsage["claude-sonnet-4-20250514"]
	if !ok {
		t.Fatal("LastModelUsage missing claude-sonnet-4-20250514")
	}
	if sonnet.InputTokens != 120000 {
		t.Errorf("sonnet.InputTokens = %d, want 120000", sonnet.InputTokens)
	}
	if sonnet.OutputTokens != 20000 {
		t.Errorf("sonnet.OutputTokens = %d, want 20000", sonnet.OutputTokens)
	}
	if sonnet.CacheCreationInputTokens != 50000 {
		t.Errorf("sonnet.CacheCreationInputTokens = %d, want 50000", sonnet.CacheCreationInputTokens)
	}
	if sonnet.CacheReadInputTokens != 80000 {
		t.Errorf("sonnet.CacheReadInputTokens = %d, want 80000", sonnet.CacheReadInputTokens)
	}
}

func TestLoadProjectStats_MissingProject(t *testing.T) {
	configPath := filepath.Join("testdata", "claude.json")
	stats, err := LoadProjectStats(configPath, "/nonexistent/project")
	if err != nil {
		t.Fatalf("LoadProjectStats() error = %v", err)
	}
	if stats != nil {
		t.Error("LoadProjectStats() expected nil for missing project")
	}
}

func TestLoadProjectStats_MissingFile(t *testing.T) {
	_, err := LoadProjectStats("/nonexistent/claude.json", "/test")
	if err == nil {
		t.Error("LoadProjectStats() expected error for missing file, got nil")
	}
}

func TestLoadProjectStats_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "invalid.json")
	os.WriteFile(path, []byte(`{invalid json`), 0644)

	_, err := LoadProjectStats(path, "/test/project")
	if err == nil {
		t.Error("LoadProjectStats() expected error for invalid JSON, got nil")
	}
}
