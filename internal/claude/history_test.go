package claude

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseHistory_ValidEntries(t *testing.T) {
	path := filepath.Join("testdata", "history.jsonl")
	entries, err := ParseHistory(path)
	if err != nil {
		t.Fatalf("ParseHistory() error = %v", err)
	}
	// Should have 5 entries with sessionId (the one without sessionId is skipped)
	if len(entries) != 5 {
		t.Errorf("ParseHistory() returned %d entries, want 5", len(entries))
	}
}

func TestParseHistory_SkipNoSessionID(t *testing.T) {
	path := filepath.Join("testdata", "history.jsonl")
	entries, err := ParseHistory(path)
	if err != nil {
		t.Fatalf("ParseHistory() error = %v", err)
	}
	for _, entry := range entries {
		if entry.SessionID == "" {
			t.Error("ParseHistory() should skip entries without sessionId")
		}
	}
}

func TestParseHistory_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.jsonl")
	os.WriteFile(path, []byte{}, 0644)

	entries, err := ParseHistory(path)
	if err != nil {
		t.Fatalf("ParseHistory() error = %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("ParseHistory() returned %d entries, want 0", len(entries))
	}
}

func TestParseHistory_MalformedLines(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "malformed.jsonl")
	content := `{invalid json line}
{"display":"valid entry","timestamp":1000000,"project":"/test","sessionId":"abc-123"}
not json at all
{"display":"another valid","timestamp":2000000,"project":"/test","sessionId":"def-456"}
`
	os.WriteFile(path, []byte(content), 0644)

	entries, err := ParseHistory(path)
	if err != nil {
		t.Fatalf("ParseHistory() error = %v", err)
	}
	if len(entries) != 2 {
		t.Errorf("ParseHistory() returned %d entries, want 2 (malformed lines should be skipped)", len(entries))
	}
}

func TestParseHistory_FileNotFound(t *testing.T) {
	_, err := ParseHistory("/nonexistent/path/history.jsonl")
	if err == nil {
		t.Error("ParseHistory() expected error for nonexistent file, got nil")
	}
}
