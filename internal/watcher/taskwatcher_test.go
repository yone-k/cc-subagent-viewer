package watcher

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestTaskWatcher_InitialLoad(t *testing.T) {
	dir := t.TempDir()
	// Create task files
	os.WriteFile(filepath.Join(dir, "1.json"),
		[]byte(`{"id":"1","subject":"Test","status":"pending","blocks":[],"blockedBy":[]}`), 0644)

	collector := newMsgCollector()
	p := newTestProgram(collector)
	defer p.Kill()

	tw := NewTaskWatcher(dir, p)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go tw.Start(ctx)

	msg, ok := collector.waitForMsg(2 * time.Second)
	if !ok {
		t.Fatal("timeout waiting for TasksUpdatedMsg")
	}
	tasksMsg, ok := msg.(TasksUpdatedMsg)
	if !ok {
		t.Fatalf("expected TasksUpdatedMsg, got %T", msg)
	}
	if len(tasksMsg.Tasks) != 1 {
		t.Errorf("expected 1 task, got %d", len(tasksMsg.Tasks))
	}
}

func TestTaskWatcher_DetectsNewFile(t *testing.T) {
	dir := t.TempDir()
	// Create initial task
	os.WriteFile(filepath.Join(dir, "1.json"),
		[]byte(`{"id":"1","subject":"Test","status":"pending","blocks":[],"blockedBy":[]}`), 0644)

	collector := newMsgCollector()
	p := newTestProgram(collector)
	defer p.Kill()

	tw := NewTaskWatcher(dir, p)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go tw.Start(ctx)

	// Wait for initial load
	_, ok := collector.waitForMsg(2 * time.Second)
	if !ok {
		t.Fatal("timeout waiting for initial load")
	}

	// Create a new file
	time.Sleep(100 * time.Millisecond)
	os.WriteFile(filepath.Join(dir, "2.json"),
		[]byte(`{"id":"2","subject":"New Task","status":"in_progress","blocks":[],"blockedBy":[]}`), 0644)

	msg, ok := collector.waitForMsg(2 * time.Second)
	if !ok {
		t.Fatal("timeout waiting for TaskChangedMsg")
	}
	changedMsg, ok := msg.(TaskChangedMsg)
	if !ok {
		t.Fatalf("expected TaskChangedMsg, got %T", msg)
	}
	if changedMsg.Task.ID != "2" {
		t.Errorf("expected task ID 2, got %s", changedMsg.Task.ID)
	}
}

func TestTaskWatcher_IgnoresLockFile(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "1.json"),
		[]byte(`{"id":"1","subject":"Test","status":"pending","blocks":[],"blockedBy":[]}`), 0644)

	collector := newMsgCollector()
	p := newTestProgram(collector)
	defer p.Kill()

	tw := NewTaskWatcher(dir, p)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go tw.Start(ctx)

	// Wait for initial load
	_, _ = collector.waitForMsg(2 * time.Second)

	// Create a .lock file - should be ignored
	time.Sleep(100 * time.Millisecond)
	os.WriteFile(filepath.Join(dir, ".lock"), []byte{}, 0644)

	// Should not receive any message
	_, ok := collector.waitForMsg(500 * time.Millisecond)
	if ok {
		t.Error("should not receive message for .lock file")
	}
}

func TestTaskWatcher_DirNotExist(t *testing.T) {
	collector := newMsgCollector()
	p := newTestProgram(collector)
	defer p.Kill()

	tw := NewTaskWatcher("/nonexistent/dir", p)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go tw.Start(ctx)

	msg, ok := collector.waitForMsg(2 * time.Second)
	if !ok {
		t.Fatal("timeout waiting for WatcherErrorMsg")
	}
	_, ok = msg.(WatcherErrorMsg)
	if !ok {
		t.Fatalf("expected WatcherErrorMsg, got %T", msg)
	}
}
