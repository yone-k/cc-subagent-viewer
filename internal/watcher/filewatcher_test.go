package watcher

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFileWatcher_InitialLoad(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "abcd1234abcd1234@v1"), []byte("test"), 0644)

	collector := newMsgCollector()
	p := newTestProgram(collector)
	defer p.Kill()

	fw := NewFileWatcher(dir, p)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go fw.Start(ctx)

	msg, ok := collector.waitForMsg(2 * time.Second)
	if !ok {
		t.Fatal("timeout waiting for FileHistoryUpdatedMsg")
	}
	fhMsg, ok := msg.(FileHistoryUpdatedMsg)
	if !ok {
		t.Fatalf("expected FileHistoryUpdatedMsg, got %T", msg)
	}
	if len(fhMsg.Groups) != 1 {
		t.Errorf("expected 1 group, got %d", len(fhMsg.Groups))
	}
}

func TestFileWatcher_DetectsNewVersion(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "abcd1234abcd1234@v1"), []byte("v1"), 0644)

	collector := newMsgCollector()
	p := newTestProgram(collector)
	defer p.Kill()

	fw := NewFileWatcher(dir, p)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go fw.Start(ctx)

	// Wait for initial load
	_, _ = collector.waitForMsg(2 * time.Second)

	// Add new version
	time.Sleep(100 * time.Millisecond)
	os.WriteFile(filepath.Join(dir, "abcd1234abcd1234@v2"), []byte("v2"), 0644)

	msg, ok := collector.waitForMsg(2 * time.Second)
	if !ok {
		t.Fatal("timeout waiting for FileHistoryUpdatedMsg after new version")
	}
	fhMsg, ok := msg.(FileHistoryUpdatedMsg)
	if !ok {
		t.Fatalf("expected FileHistoryUpdatedMsg, got %T", msg)
	}
	found := false
	for _, g := range fhMsg.Groups {
		if g.Hash == "abcd1234abcd1234" && len(g.Versions) == 2 {
			found = true
		}
	}
	if !found {
		t.Error("expected group with 2 versions after adding v2")
	}
}

func TestFileWatcher_Debounce(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "abcd1234abcd1234@v1"), []byte("v1"), 0644)

	collector := newMsgCollector()
	p := newTestProgram(collector)
	defer p.Kill()

	fw := NewFileWatcher(dir, p)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go fw.Start(ctx)

	// Wait for initial load
	_, _ = collector.waitForMsg(2 * time.Second)

	// Rapidly create multiple files
	time.Sleep(100 * time.Millisecond)
	os.WriteFile(filepath.Join(dir, "abcd1234abcd1234@v2"), []byte("v2"), 0644)
	os.WriteFile(filepath.Join(dir, "abcd1234abcd1234@v3"), []byte("v3"), 0644)
	os.WriteFile(filepath.Join(dir, "ef567890ef567890@v1"), []byte("ef1"), 0644)

	// Should receive only one debounced update (or very few)
	msg, ok := collector.waitForMsg(2 * time.Second)
	if !ok {
		t.Fatal("timeout waiting for debounced FileHistoryUpdatedMsg")
	}
	_, ok = msg.(FileHistoryUpdatedMsg)
	if !ok {
		t.Fatalf("expected FileHistoryUpdatedMsg, got %T", msg)
	}
}
