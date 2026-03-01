package claude

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
)

// HistoryEntry represents a single entry in history.jsonl
type HistoryEntry struct {
	Display   string `json:"display"`
	Timestamp int64  `json:"timestamp"`
	Project   string `json:"project"`
	SessionID string `json:"sessionId"`
}

// ParseHistory reads a JSONL history file and returns entries that have a sessionId.
func ParseHistory(path string) ([]HistoryEntry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening history file: %w", err)
	}
	defer f.Close()

	var entries []HistoryEntry
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		var entry HistoryEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			// Claude history may contain corrupted or incompatible lines; skip gracefully
			continue
		}
		if entry.SessionID == "" {
			continue // skip entries without sessionId
		}
		entries = append(entries, entry)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading history file: %w", err)
	}
	return entries, nil
}
