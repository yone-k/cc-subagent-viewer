package claude

import (
	"log"
	"os"
	"path/filepath"
	"sync"
)

var (
	homeDir     string
	homeDirOnce sync.Once
)

func init() {
	homeDirOnce.Do(func() {
		var err error
		homeDir, err = os.UserHomeDir()
		if err != nil {
			log.Fatalf("failed to determine user home directory: %v", err)
		}
	})
}

// ClaudeDir returns the path to ~/.claude
func ClaudeDir() string {
	return filepath.Join(homeDir, ".claude")
}

// HistoryPath returns the path to ~/.claude/history.jsonl
func HistoryPath() string {
	return filepath.Join(ClaudeDir(), "history.jsonl")
}

// TasksDir returns the path to ~/.claude/tasks/{sessionID}
func TasksDir(sessionID string) string {
	return filepath.Join(ClaudeDir(), "tasks", sessionID)
}

// DebugLogPath returns the path to ~/.claude/debug/{sessionID}.txt
func DebugLogPath(sessionID string) string {
	return filepath.Join(ClaudeDir(), "debug", sessionID+".txt")
}

// FileHistoryDir returns the path to ~/.claude/file-history/{sessionID}
func FileHistoryDir(sessionID string) string {
	return filepath.Join(ClaudeDir(), "file-history", sessionID)
}

// GlobalConfigPath returns the path to ~/.claude.json
func GlobalConfigPath() string {
	return filepath.Join(homeDir, ".claude.json")
}
