package claude

import (
	"os"
	"path/filepath"
	"testing"
)

func TestClaudeDir(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	got := ClaudeDir()
	want := filepath.Join(home, ".claude")
	if got != want {
		t.Errorf("ClaudeDir() = %q, want %q", got, want)
	}
}

func TestTasksDir(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	sessionID := "7ba50137-65c8-4349-b420-cdce14c38d2a"
	got := TasksDir(sessionID)
	want := filepath.Join(home, ".claude", "tasks", sessionID)
	if got != want {
		t.Errorf("TasksDir(%q) = %q, want %q", sessionID, got, want)
	}
}

func TestDebugLogPath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	sessionID := "7ba50137-65c8-4349-b420-cdce14c38d2a"
	got := DebugLogPath(sessionID)
	want := filepath.Join(home, ".claude", "debug", sessionID+".txt")
	if got != want {
		t.Errorf("DebugLogPath(%q) = %q, want %q", sessionID, got, want)
	}
}

func TestHistoryPath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	got := HistoryPath()
	want := filepath.Join(home, ".claude", "history.jsonl")
	if got != want {
		t.Errorf("HistoryPath() = %q, want %q", got, want)
	}
}

func TestFileHistoryDir(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	sessionID := "test-session"
	got := FileHistoryDir(sessionID)
	want := filepath.Join(home, ".claude", "file-history", sessionID)
	if got != want {
		t.Errorf("FileHistoryDir(%q) = %q, want %q", sessionID, got, want)
	}
}

func TestGlobalConfigPath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	got := GlobalConfigPath()
	want := filepath.Join(home, ".claude.json")
	if got != want {
		t.Errorf("GlobalConfigPath() = %q, want %q", got, want)
	}
}
