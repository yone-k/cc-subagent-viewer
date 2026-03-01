package watcher

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLogWatcher_InitialTail(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "debug.txt")
	// Create a log file with a few entries
	content := ""
	for i := 0; i < 5; i++ {
		content += "2026-03-01T00:00:00.000Z [DEBUG] line " + fmt.Sprintf("%d", i) + "\n"
	}
	os.WriteFile(path, []byte(content), 0644)

	collector := newMsgCollector()
	p := newTestProgram(collector)
	defer p.Kill()

	lw := NewLogWatcher(path, p)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go lw.Start(ctx)

	msg, ok := collector.waitForMsg(2 * time.Second)
	if !ok {
		t.Fatal("timeout waiting for LogEntriesMsg")
	}
	logMsg, ok := msg.(LogEntriesMsg)
	if !ok {
		t.Fatalf("expected LogEntriesMsg, got %T", msg)
	}
	if !logMsg.Initial {
		t.Error("expected Initial to be true")
	}
	if len(logMsg.Entries) != 5 {
		t.Errorf("expected 5 entries, got %d", len(logMsg.Entries))
	}
}

func TestLogWatcher_PollsNewLines(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "debug.txt")
	os.WriteFile(path, []byte("2026-03-01T00:00:00.000Z [DEBUG] initial\n"), 0644)

	collector := newMsgCollector()
	p := newTestProgram(collector)
	defer p.Kill()

	lw := NewLogWatcher(path, p)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go lw.Start(ctx)

	// Wait for initial load
	_, _ = collector.waitForMsg(2 * time.Second)

	// Append new line
	f, _ := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0644)
	f.WriteString("2026-03-01T00:00:01.000Z [ERROR] new error\n")
	f.Close()

	// Wait for poll
	msg, ok := collector.waitForMsg(2 * time.Second)
	if !ok {
		t.Fatal("timeout waiting for new log entries")
	}
	logMsg, ok := msg.(LogEntriesMsg)
	if !ok {
		t.Fatalf("expected LogEntriesMsg, got %T", msg)
	}
	if logMsg.Initial {
		t.Error("expected Initial to be false")
	}
	if len(logMsg.Entries) == 0 {
		t.Error("expected at least 1 new entry")
	}
}

func TestLogWatcher_Truncation(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "debug.txt")
	os.WriteFile(path, []byte("2026-03-01T00:00:00.000Z [DEBUG] original content that is long enough\n"), 0644)

	collector := newMsgCollector()
	p := newTestProgram(collector)
	defer p.Kill()

	lw := NewLogWatcher(path, p)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go lw.Start(ctx)

	// Wait for initial load
	_, _ = collector.waitForMsg(2 * time.Second)

	// Truncate and write shorter content
	os.WriteFile(path, []byte("2026-03-01T00:00:01.000Z [DEBUG] after truncation\n"), 0644)

	// Wait for poll to detect the new content
	msg, ok := collector.waitForMsg(2 * time.Second)
	if !ok {
		t.Fatal("timeout waiting for entries after truncation")
	}
	logMsg, ok := msg.(LogEntriesMsg)
	if !ok {
		t.Fatalf("expected LogEntriesMsg, got %T", msg)
	}
	if len(logMsg.Entries) == 0 {
		t.Error("expected entries after truncation")
	}
}

func TestLogWatcher_FileNotExist(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nonexistent.txt")

	collector := newMsgCollector()
	p := newTestProgram(collector)
	defer p.Kill()

	lw := NewLogWatcher(path, p)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go lw.Start(ctx)

	// Should not crash. Wait a bit then create the file
	time.Sleep(600 * time.Millisecond)
	os.WriteFile(path, []byte("2026-03-01T00:00:00.000Z [DEBUG] created later\n"), 0644)

	msg, ok := collector.waitForMsg(2 * time.Second)
	if !ok {
		t.Fatal("timeout waiting for entries after file creation")
	}
	logMsg, ok := msg.(LogEntriesMsg)
	if !ok {
		t.Fatalf("expected LogEntriesMsg, got %T", msg)
	}
	if len(logMsg.Entries) == 0 {
		t.Error("expected entries after file creation")
	}
}

func TestLogWatcher_OffsetAdvancesWithoutEntries(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "debug.txt")
	// Start with a valid log line
	os.WriteFile(path, []byte("2026-03-01T00:00:00.000Z [DEBUG] initial\n"), 0644)

	collector := newMsgCollector()
	p := newTestProgram(collector)
	defer p.Kill()

	lw := NewLogWatcher(path, p)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go lw.Start(ctx)

	// Wait for initial load
	_, _ = collector.waitForMsg(2 * time.Second)

	// Append a line without a timestamp (unparseable as a standalone entry).
	// ReadLogFrom will return entries==nil but newOffset should still advance.
	f, _ := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0644)
	f.WriteString("this line has no timestamp\n")
	f.Close()

	// Wait enough polls for the offset to advance (no LogEntriesMsg expected).
	time.Sleep(2 * time.Second)

	// Append a valid log line after the unparseable one.
	f, _ = os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0644)
	f.WriteString("2026-03-01T00:00:02.000Z [DEBUG] after gap\n")
	f.Close()

	// We should receive only the new valid entry, not the unparseable line again.
	msg, ok := collector.waitForMsg(2 * time.Second)
	if !ok {
		t.Fatal("timeout waiting for LogEntriesMsg after unparseable line")
	}
	logMsg, ok := msg.(LogEntriesMsg)
	if !ok {
		t.Fatalf("expected LogEntriesMsg, got %T", msg)
	}
	if len(logMsg.Entries) != 1 {
		t.Errorf("expected 1 entry, got %d", len(logMsg.Entries))
	}
	if len(logMsg.Entries) > 0 && logMsg.Entries[0].Message != "after gap" {
		t.Errorf("expected message 'after gap', got %q", logMsg.Entries[0].Message)
	}
}
