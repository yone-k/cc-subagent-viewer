package claude

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestParseLogLine_AllLevels(t *testing.T) {
	tests := []struct {
		line  string
		level LogLevel
	}{
		{"2026-03-01T00:39:12.103Z [DEBUG] test debug message", LevelDEBUG},
		{"2026-03-01T00:39:12.103Z [ERROR] test error message", LevelERROR},
		{"2026-03-01T00:39:12.103Z [WARN] test warn message", LevelWARN},
		{"2026-03-01T00:39:12.103Z [MCP] test mcp message", LevelMCP},
		{"2026-03-01T00:39:12.103Z [STARTUP] test startup message", LevelSTARTUP},
		{"2026-03-01T00:39:12.103Z [META] test meta message", LevelMETA},
		{"2026-03-01T00:39:12.103Z [ATTACHMENT] test attachment message", LevelATTACHMENT},
	}
	for _, tt := range tests {
		t.Run(string(tt.level), func(t *testing.T) {
			entry, err := ParseLogLine(tt.line)
			if err != nil {
				t.Fatalf("ParseLogLine() error = %v", err)
			}
			if entry.Level != tt.level {
				t.Errorf("Level = %q, want %q", entry.Level, tt.level)
			}
		})
	}
}

func TestParseLogLine_InvalidFormat(t *testing.T) {
	_, err := ParseLogLine("this is not a log line")
	if err == nil {
		t.Error("ParseLogLine() expected error for invalid format, got nil")
	}
}

func TestParseLogLine_TimestampParsing(t *testing.T) {
	entry, err := ParseLogLine("2026-03-01T00:39:12.103Z [DEBUG] test message")
	if err != nil {
		t.Fatalf("ParseLogLine() error = %v", err)
	}
	expected := time.Date(2026, 3, 1, 0, 39, 12, 103000000, time.UTC)
	if !entry.Timestamp.Equal(expected) {
		t.Errorf("Timestamp = %v, want %v", entry.Timestamp, expected)
	}
}

func TestReadLogTail_LastNLines(t *testing.T) {
	path := filepath.Join("testdata", "debug.txt")
	entries, offset, err := ReadLogTail(path, 10)
	if err != nil {
		t.Fatalf("ReadLogTail() error = %v", err)
	}
	if len(entries) != 10 {
		t.Errorf("ReadLogTail() returned %d entries, want 10", len(entries))
	}
	if offset <= 0 {
		t.Errorf("ReadLogTail() offset = %d, want > 0", offset)
	}
}

func TestReadLogTail_LessThanNLines(t *testing.T) {
	// Create a small temp file with 3 log lines
	dir := t.TempDir()
	path := filepath.Join(dir, "small.txt")
	content := "2026-03-01T00:00:00.000Z [DEBUG] line 1\n2026-03-01T00:00:01.000Z [DEBUG] line 2\n2026-03-01T00:00:02.000Z [DEBUG] line 3\n"
	os.WriteFile(path, []byte(content), 0644)

	entries, _, err := ReadLogTail(path, 100)
	if err != nil {
		t.Fatalf("ReadLogTail() error = %v", err)
	}
	if len(entries) != 3 {
		t.Errorf("ReadLogTail() returned %d entries, want 3", len(entries))
	}
}

func TestReadLogTail_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.txt")
	os.WriteFile(path, []byte{}, 0644)

	entries, _, err := ReadLogTail(path, 10)
	if err != nil {
		t.Fatalf("ReadLogTail() error = %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("ReadLogTail() returned %d entries, want 0", len(entries))
	}
}

func TestReadLogFrom_Offset(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	line1 := "2026-03-01T00:00:00.000Z [DEBUG] first line\n"
	line2 := "2026-03-01T00:00:01.000Z [DEBUG] second line\n"
	os.WriteFile(path, []byte(line1+line2), 0644)

	// Read from offset after first line
	offset := int64(len(line1))
	entries, newOffset, err := ReadLogFrom(path, offset)
	if err != nil {
		t.Fatalf("ReadLogFrom() error = %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("ReadLogFrom() returned %d entries, want 1", len(entries))
	}
	if !strings.Contains(entries[0].Message, "second line") {
		t.Errorf("entry.Message = %q, want to contain \"second line\"", entries[0].Message)
	}
	if newOffset <= offset {
		t.Errorf("newOffset = %d, want > %d", newOffset, offset)
	}
}

func TestReadLogFrom_ContinuationLines(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	content := "2026-03-01T00:00:00.000Z [DEBUG] main message\n  continuation line 1\n  continuation line 2\n2026-03-01T00:00:01.000Z [DEBUG] next message\n"
	os.WriteFile(path, []byte(content), 0644)

	entries, _, err := ReadLogFrom(path, 0)
	if err != nil {
		t.Fatalf("ReadLogFrom() error = %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("ReadLogFrom() returned %d entries, want 2", len(entries))
	}
	if !strings.Contains(entries[0].Message, "continuation line 1") {
		t.Errorf("first entry message should contain continuation lines, got %q", entries[0].Message)
	}
	if !strings.Contains(entries[0].Message, "continuation line 2") {
		t.Errorf("first entry message should contain continuation line 2, got %q", entries[0].Message)
	}
}

func TestReadLogFrom_Truncation(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	content := "2026-03-01T00:00:00.000Z [DEBUG] after truncation\n"
	os.WriteFile(path, []byte(content), 0644)

	// Offset larger than file size - should reset to 0
	entries, _, err := ReadLogFrom(path, 99999)
	if err != nil {
		t.Fatalf("ReadLogFrom() error = %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("ReadLogFrom() returned %d entries, want 1", len(entries))
	}
	if !strings.Contains(entries[0].Message, "after truncation") {
		t.Errorf("entry.Message = %q, want to contain \"after truncation\"", entries[0].Message)
	}
}

func TestReadLogFrom_FileNotExist(t *testing.T) {
	_, _, err := ReadLogFrom("/nonexistent/path/debug.txt", 0)
	if err == nil {
		t.Error("ReadLogFrom() expected error for non-existent file, got nil")
	}
}

func TestReadLogFrom_LongLine(t *testing.T) {
	// Create a log file with a single line exceeding 64KiB (the default bufio.Scanner limit).
	dir := t.TempDir()
	path := filepath.Join(dir, "longline.txt")

	longMsg := strings.Repeat("x", 70*1024) // 70KiB message
	line := fmt.Sprintf("2026-03-01T00:00:00.000Z [DEBUG] %s\n", longMsg)
	if err := os.WriteFile(path, []byte(line), 0644); err != nil {
		t.Fatal(err)
	}

	entries, _, err := ReadLogFrom(path, 0)
	if err != nil {
		t.Fatalf("ReadLogFrom() error = %v (should handle lines > 64KiB)", err)
	}
	if len(entries) != 1 {
		t.Fatalf("ReadLogFrom() returned %d entries, want 1", len(entries))
	}
	if !strings.Contains(entries[0].Message, "xxxx") {
		t.Errorf("entry.Message should contain the long payload")
	}
}

func TestReadLogTail_SeekChunkBoundary(t *testing.T) {
	// Build a log file around seekChunkSize (8192) bytes to exercise boundary behavior.
	dir := t.TempDir()
	path := filepath.Join(dir, "boundary.txt")

	var b strings.Builder
	lineCount := 0
	for b.Len() < seekChunkSize+1024 {
		fmt.Fprintf(&b, "2026-03-01T00:00:%02d.000Z [DEBUG] line %d padding-data-to-fill-space\n", lineCount%60, lineCount)
		lineCount++
	}
	if err := os.WriteFile(path, []byte(b.String()), 0644); err != nil {
		t.Fatal(err)
	}

	// Request fewer entries than available — should get exactly 5.
	entries, offset, err := ReadLogTail(path, 5)
	if err != nil {
		t.Fatalf("ReadLogTail() error = %v", err)
	}
	if len(entries) != 5 {
		t.Errorf("ReadLogTail() returned %d entries, want 5", len(entries))
	}
	if offset != int64(b.Len()) {
		t.Errorf("offset = %d, want %d", offset, b.Len())
	}

	// The last entry should be the last line written.
	lastEntry := entries[len(entries)-1]
	expectedMsg := fmt.Sprintf("line %d padding-data-to-fill-space", lineCount-1)
	if !strings.Contains(lastEntry.Message, expectedMsg) {
		t.Errorf("last entry message = %q, want to contain %q", lastEntry.Message, expectedMsg)
	}
}

func TestParseLogLine_TimestampLocalTimezone(t *testing.T) {
	line := "2026-03-01T00:39:12.103Z [DEBUG] test message"
	entry, err := ParseLogLine(line)
	if err != nil {
		t.Fatalf("ParseLogLine() error = %v", err)
	}

	if entry.Timestamp.Location() != time.Local {
		t.Errorf("Timestamp.Location() = %v, want %v (time.Local)", entry.Timestamp.Location(), time.Local)
	}

	expectedUTC := time.Date(2026, 3, 1, 0, 39, 12, 103000000, time.UTC)
	if !entry.Timestamp.Equal(expectedUTC) {
		t.Errorf("Timestamp instant = %v, want equal to %v", entry.Timestamp, expectedUTC)
	}
}

func TestParseLogLine_UnknownLevel(t *testing.T) {
	// Unknown levels like "INFO" should still parse successfully.
	line := "2026-03-01T12:00:00.000Z [INFO] an informational message"
	entry, err := ParseLogLine(line)
	if err != nil {
		t.Fatalf("ParseLogLine() error = %v; unknown levels should be accepted", err)
	}
	if entry.Level != LogLevel("INFO") {
		t.Errorf("Level = %q, want %q", entry.Level, "INFO")
	}
	if entry.Message != "an informational message" {
		t.Errorf("Message = %q, want %q", entry.Message, "an informational message")
	}
}
