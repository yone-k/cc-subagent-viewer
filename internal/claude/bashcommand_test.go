package claude

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestExtractBashCommandsIncremental_ParsesAllResultVariants(t *testing.T) {
	tmpDir := t.TempDir()
	persistedPath := filepath.Join(tmpDir, "bash-output.txt")
	persistedContent := strings.Repeat("persisted output\n", 700)
	if err := os.WriteFile(persistedPath, []byte(persistedContent), 0o644); err != nil {
		t.Fatal(err)
	}

	fixturePath := filepath.Join("testdata", "parent-conversation-bash.jsonl")
	fixture, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(tmpDir, "parent-conversation-bash.jsonl")
	updatedFixture := strings.ReplaceAll(string(fixture), "/tmp/bash-output.txt", persistedPath)
	if err := os.WriteFile(path, []byte(updatedFixture), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := ExtractBashCommandsIncremental(path, 0, nil)
	if err != nil {
		t.Fatalf("ExtractBashCommandsIncremental() error = %v", err)
	}

	if result.Cache == nil {
		t.Fatal("cache should not be nil")
	}
	if len(result.Cache.Commands) != 4 {
		t.Fatalf("got %d commands, want 4", len(result.Cache.Commands))
	}

	first := result.Cache.Commands[0]
	if first.ToolUseID != "bash_tool_1" {
		t.Errorf("first.ToolUseID = %q, want %q", first.ToolUseID, "bash_tool_1")
	}
	if first.Command != "go test ./..." {
		t.Errorf("first.Command = %q, want %q", first.Command, "go test ./...")
	}
	if first.Description != "Run tests" {
		t.Errorf("first.Description = %q, want %q", first.Description, "Run tests")
	}
	if !first.Timestamp.Equal(time.Date(2026, 4, 8, 13, 29, 37, 69_000_000, time.UTC)) {
		t.Errorf("first.Timestamp = %s, want %s", first.Timestamp.Format(time.RFC3339Nano), "2026-04-08T13:29:37.069Z")
	}
	if first.Result == nil {
		t.Fatal("first.Result should not be nil")
	}
	if first.Result.Stdout != "ok\n" {
		t.Errorf("first.Result.Stdout = %q, want %q", first.Result.Stdout, "ok\n")
	}
	if first.Result.Stderr != "" {
		t.Errorf("first.Result.Stderr = %q, want empty", first.Result.Stderr)
	}

	second := result.Cache.Commands[1]
	if second.Result == nil {
		t.Fatal("second.Result should not be nil")
	}
	if second.Result.Error != "Error: exit status 1\nFAIL\tinternal/..." {
		t.Errorf("second.Result.Error = %q, want string toolUseResult content", second.Result.Error)
	}

	third := result.Cache.Commands[2]
	if third.Result == nil {
		t.Fatal("third.Result should not be nil")
	}
	if third.Result.PersistedOutputPath != persistedPath {
		t.Errorf("third.Result.PersistedOutputPath = %q, want %q", third.Result.PersistedOutputPath, persistedPath)
	}
	if third.Result.PersistedOutputSize != 8192 {
		t.Errorf("third.Result.PersistedOutputSize = %d, want 8192", third.Result.PersistedOutputSize)
	}
	if third.Result.PersistedOutput == "" {
		t.Fatal("third.Result.PersistedOutput should not be empty")
	}
	if len(third.Result.PersistedOutput) != persistedOutputReadLimit {
		t.Errorf("len(third.Result.PersistedOutput) = %d, want %d", len(third.Result.PersistedOutput), persistedOutputReadLimit)
	}
	if !third.Result.PersistedTruncated {
		t.Error("third.Result.PersistedTruncated = false, want true")
	}
	if third.Result.PersistedReadError != "" {
		t.Errorf("third.Result.PersistedReadError = %q, want empty", third.Result.PersistedReadError)
	}

	fourth := result.Cache.Commands[3]
	if fourth.ToolUseID != "bash_tool_4" {
		t.Errorf("fourth.ToolUseID = %q, want %q", fourth.ToolUseID, "bash_tool_4")
	}
	if fourth.Result != nil {
		t.Errorf("fourth.Result = %+v, want nil for in-progress command", fourth.Result)
	}
}

func TestExtractBashCommandsIncremental_PersistedOutputReadErrorIsCaptured(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "conversation.jsonl")

	lines := []string{
		`{"type":"assistant","timestamp":"2026-04-08T13:29:37.069Z","message":{"content":[{"type":"tool_use","id":"bash_tool_missing","name":"Bash","input":{"command":"cat missing.log","description":"Inspect missing log"}}]}}`,
		`{"type":"user","message":{"content":[{"type":"tool_result","tool_use_id":"bash_tool_missing","content":"stored externally"}]},"toolUseResult":{"stdout":"","stderr":"","persistedOutputPath":"` + filepath.Join(tmpDir, "missing.log") + `","persistedOutputSize":128}}`,
	}
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := ExtractBashCommandsIncremental(path, 0, nil)
	if err != nil {
		t.Fatalf("ExtractBashCommandsIncremental() error = %v", err)
	}
	if len(result.Cache.Commands) != 1 {
		t.Fatalf("got %d commands, want 1", len(result.Cache.Commands))
	}

	command := result.Cache.Commands[0]
	if command.Result == nil {
		t.Fatal("command.Result should not be nil")
	}
	if command.Result.PersistedReadError == "" {
		t.Fatal("command.Result.PersistedReadError should not be empty")
	}
}

func TestExtractBashCommandsIncremental_UpdatesExistingCommandWhenResultArrives(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "conversation.jsonl")

	line1 := `{"type":"assistant","timestamp":"2026-04-08T13:29:37.069Z","message":{"content":[{"type":"tool_use","id":"bash_tool_inc","name":"Bash","input":{"command":"go build ./...","description":"Build packages"}}]}}` + "\n"
	if err := os.WriteFile(path, []byte(line1), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := ExtractBashCommandsIncremental(path, 0, nil)
	if err != nil {
		t.Fatalf("phase 1 error = %v", err)
	}
	if len(result.Cache.Commands) != 1 {
		t.Fatalf("phase 1 got %d commands, want 1", len(result.Cache.Commands))
	}
	if result.Cache.Commands[0].Result != nil {
		t.Errorf("phase 1 Result = %+v, want nil", result.Cache.Commands[0].Result)
	}

	line2 := `{"type":"user","message":{"content":[{"type":"tool_result","tool_use_id":"bash_tool_inc","content":"done"}]},"toolUseResult":{"stdout":"build ok\n","stderr":""}}` + "\n"
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(line2); err != nil {
		f.Close()
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	result, err = ExtractBashCommandsIncremental(path, result.Offset, result.Cache)
	if err != nil {
		t.Fatalf("phase 2 error = %v", err)
	}
	if len(result.Cache.Commands) != 1 {
		t.Fatalf("phase 2 got %d commands, want 1", len(result.Cache.Commands))
	}
	if result.Cache.Commands[0].Result == nil {
		t.Fatal("phase 2 Result should not be nil")
	}
	if result.Cache.Commands[0].Result.Stdout != "build ok\n" {
		t.Errorf("phase 2 Stdout = %q, want %q", result.Cache.Commands[0].Result.Stdout, "build ok\n")
	}
}

func TestExtractBashCommandsIncremental_TruncateResetsCache(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "conversation.jsonl")

	oldLine := `{"type":"assistant","timestamp":"2026-04-08T13:29:37.069Z","message":{"content":[{"type":"tool_use","id":"bash_old","name":"Bash","input":{"command":"old","description":"Old command"}}]}}` + "\n"
	if err := os.WriteFile(path, []byte(oldLine), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := ExtractBashCommandsIncremental(path, 0, nil)
	if err != nil {
		t.Fatalf("phase 1 error = %v", err)
	}
	if len(result.Cache.Commands) != 1 {
		t.Fatalf("phase 1 got %d commands, want 1", len(result.Cache.Commands))
	}

	newLine := `{"type":"assistant","timestamp":"2026-04-08T13:40:00.000Z","message":{"content":[{"type":"tool_use","id":"bash_new","name":"Bash","input":{"command":"new","description":"New command"}}]}}` + "\n"
	if err := os.WriteFile(path, []byte(newLine), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err = ExtractBashCommandsIncremental(path, result.Offset, result.Cache)
	if err != nil {
		t.Fatalf("phase 2 error = %v", err)
	}
	if len(result.Cache.Commands) != 1 {
		t.Fatalf("phase 2 got %d commands, want 1", len(result.Cache.Commands))
	}
	if result.Cache.Commands[0].ToolUseID != "bash_new" {
		t.Errorf("phase 2 ToolUseID = %q, want %q", result.Cache.Commands[0].ToolUseID, "bash_new")
	}
}
