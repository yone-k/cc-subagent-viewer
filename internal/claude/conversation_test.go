package claude

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseConversationFile_ValidEntries(t *testing.T) {
	path := filepath.Join("testdata", "subagents", "agent-test1.jsonl")
	entries, info, err := ParseConversationFile(path)
	if err != nil {
		t.Fatalf("ParseConversationFile() error = %v", err)
	}

	if len(entries) != 4 {
		t.Fatalf("got %d entries, want 4", len(entries))
	}

	// First entry: user with string content
	if entries[0].Type != EntryTypeUser {
		t.Errorf("entry[0].Type = %q, want %q", entries[0].Type, EntryTypeUser)
	}
	if len(entries[0].Content) != 1 || entries[0].Content[0].Text != "Implement the feature" {
		t.Errorf("entry[0].Content unexpected: %+v", entries[0].Content)
	}

	// Second entry: assistant with text + tool_use
	if entries[1].Type != EntryTypeAssistant {
		t.Errorf("entry[1].Type = %q, want %q", entries[1].Type, EntryTypeAssistant)
	}
	if len(entries[1].Content) != 2 {
		t.Fatalf("entry[1] got %d content blocks, want 2", len(entries[1].Content))
	}
	if entries[1].Content[0].Type != "text" || entries[1].Content[0].Text != "I'll implement this feature." {
		t.Errorf("entry[1].Content[0] unexpected: %+v", entries[1].Content[0])
	}
	if entries[1].Content[1].Type != "tool_use" || entries[1].Content[1].ToolName != "Read" {
		t.Errorf("entry[1].Content[1] unexpected: %+v", entries[1].Content[1])
	}

	// Third entry: user with tool_result
	if entries[2].Content[0].Type != "tool_result" {
		t.Errorf("entry[2].Content[0].Type = %q, want tool_result", entries[2].Content[0].Type)
	}

	// Fourth entry: assistant with thinking + text
	if entries[3].Content[0].Type != "thinking" {
		t.Errorf("entry[3].Content[0].Type = %q, want thinking", entries[3].Content[0].Type)
	}

	// SubagentInfo
	if info == nil {
		t.Fatal("info should not be nil")
	}
	if info.AgentID != "abc123" {
		t.Errorf("info.AgentID = %q, want %q", info.AgentID, "abc123")
	}
	if info.Slug != "implement-feature" {
		t.Errorf("info.Slug = %q, want %q", info.Slug, "implement-feature")
	}
	if info.Prompt != "Implement the feature" {
		t.Errorf("info.Prompt = %q, want %q", info.Prompt, "Implement the feature")
	}
	if info.EntryCount != 4 {
		t.Errorf("info.EntryCount = %d, want 4", info.EntryCount)
	}
}

func TestParseConversationFile_SkipsProgress(t *testing.T) {
	path := filepath.Join("testdata", "subagents", "agent-test2.jsonl")
	entries, info, err := ParseConversationFile(path)
	if err != nil {
		t.Fatalf("ParseConversationFile() error = %v", err)
	}

	// progress line should be skipped
	if len(entries) != 2 {
		t.Fatalf("got %d entries, want 2 (progress should be skipped)", len(entries))
	}

	if info.AgentID != "def456" {
		t.Errorf("info.AgentID = %q, want %q", info.AgentID, "def456")
	}
}

func TestParseConversationFile_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	emptyFile := filepath.Join(tmpDir, "empty.jsonl")
	if err := os.WriteFile(emptyFile, []byte{}, 0o644); err != nil {
		t.Fatal(err)
	}

	entries, info, err := ParseConversationFile(emptyFile)
	if err != nil {
		t.Fatalf("ParseConversationFile() error = %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("got %d entries, want 0", len(entries))
	}
	if info != nil {
		t.Error("info should be nil for empty file")
	}
}

func TestParseConversationFile_NonExistentFile(t *testing.T) {
	_, _, err := ParseConversationFile("/nonexistent/path.jsonl")
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

func TestDiscoverSubagents(t *testing.T) {
	dir := filepath.Join("testdata", "subagents")
	agents, err := DiscoverSubagents(dir)
	if err != nil {
		t.Fatalf("DiscoverSubagents() error = %v", err)
	}

	if len(agents) != 2 {
		t.Fatalf("got %d agents, want 2", len(agents))
	}

	// Verify both agents were found (order may vary due to glob)
	agentIDs := map[string]bool{}
	for _, a := range agents {
		agentIDs[a.AgentID] = true
	}
	if !agentIDs["abc123"] {
		t.Error("expected agent abc123 to be found")
	}
	if !agentIDs["def456"] {
		t.Error("expected agent def456 to be found")
	}
}
