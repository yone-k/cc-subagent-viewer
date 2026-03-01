package watcher

import (
	"github.com/yone/subagent-viewer/internal/claude"
)

// TasksUpdatedMsg is sent when all tasks are loaded/reloaded.
type TasksUpdatedMsg struct {
	Tasks []claude.Task
}

// TaskChangedMsg is sent when a single task file changes.
type TaskChangedMsg struct {
	Task claude.Task
}

// LogEntriesMsg is sent when new log entries are available.
type LogEntriesMsg struct {
	Entries []claude.LogEntry
	Initial bool // true for initial tail load
}

// FileHistoryUpdatedMsg is sent when file history is updated.
type FileHistoryUpdatedMsg struct {
	Groups []claude.FileGroup
}

// WatcherErrorMsg is sent when a watcher encounters an error.
type WatcherErrorMsg struct {
	Source string
	Err    error
}
