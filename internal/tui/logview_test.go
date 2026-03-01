package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/yone/subagent-viewer/internal/claude"
	"github.com/yone/subagent-viewer/internal/watcher"
)

func makeLogEntry(level claude.LogLevel, message string) claude.LogEntry {
	return claude.LogEntry{
		Timestamp: time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		Level:     level,
		Message:   message,
		Raw:       "2026-03-01T00:00:00.000Z [" + string(level) + "] " + message,
	}
}

func TestLogView_UpdateWithEntries(t *testing.T) {
	m := NewLogViewModel()
	m.SetSize(80, 24)

	entries := []claude.LogEntry{
		makeLogEntry(claude.LevelDEBUG, "debug msg"),
		makeLogEntry(claude.LevelERROR, "error msg"),
	}
	newModel, _ := m.Update(watcher.LogEntriesMsg{Entries: entries, Initial: true})
	m = newModel.(LogViewModel)

	if m.EntryCount() != 2 {
		t.Errorf("expected 2 entries, got %d", m.EntryCount())
	}
}

func TestLogView_RingBuffer(t *testing.T) {
	m := NewLogViewModel()
	m.SetSize(80, 24)

	// Add more than maxEntries (10000)
	entries := make([]claude.LogEntry, 10500)
	for i := range entries {
		entries[i] = makeLogEntry(claude.LevelDEBUG, "msg")
	}
	newModel, _ := m.Update(watcher.LogEntriesMsg{Entries: entries, Initial: true})
	m = newModel.(LogViewModel)

	if m.EntryCount() > 10000 {
		t.Errorf("ring buffer should cap at 10000, got %d", m.EntryCount())
	}
}

func TestLogView_FilterByLevel(t *testing.T) {
	m := NewLogViewModel()
	m.SetSize(80, 24)

	entries := []claude.LogEntry{
		makeLogEntry(claude.LevelDEBUG, "debug msg"),
		makeLogEntry(claude.LevelERROR, "error msg"),
		makeLogEntry(claude.LevelWARN, "warn msg"),
	}
	newModel, _ := m.Update(watcher.LogEntriesMsg{Entries: entries, Initial: true})
	m = newModel.(LogViewModel)

	// Disable DEBUG filter (only show ERROR and WARN)
	m.filterLevels[claude.LevelDEBUG] = false
	m.filteredDirty = true

	view := m.View()
	if strings.Contains(view, "debug msg") {
		t.Error("filtered out DEBUG should not appear in view")
	}
	if !strings.Contains(view, "error msg") {
		t.Error("ERROR should appear in view")
	}
}

func TestLogView_FilterToggle_AllLevels(t *testing.T) {
	m := NewLogViewModel()
	m.SetSize(80, 24)

	// All levels should be enabled by default
	levels := map[string]claude.LogLevel{
		"D": claude.LevelDEBUG,
		"E": claude.LevelERROR,
		"W": claude.LevelWARN,
		"M": claude.LevelMCP,
		"S": claude.LevelSTARTUP,
		"T": claude.LevelMETA,
		"A": claude.LevelATTACHMENT,
	}

	for key, level := range levels {
		if !m.filterLevels[level] {
			t.Errorf("level %s should be enabled by default", level)
		}
		// Toggle off
		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})
		m = newModel.(LogViewModel)
		if m.filterLevels[level] {
			t.Errorf("level %s should be disabled after toggle with key %s", level, key)
		}
		// Toggle back on
		newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})
		m = newModel.(LogViewModel)
		if !m.filterLevels[level] {
			t.Errorf("level %s should be re-enabled after second toggle", level)
		}
	}
}

func TestLogView_Search(t *testing.T) {
	m := NewLogViewModel()
	m.SetSize(80, 24)

	entries := []claude.LogEntry{
		makeLogEntry(claude.LevelDEBUG, "hello world"),
		makeLogEntry(claude.LevelDEBUG, "foo bar"),
		makeLogEntry(claude.LevelDEBUG, "hello again"),
	}
	newModel, _ := m.Update(watcher.LogEntriesMsg{Entries: entries, Initial: true})
	m = newModel.(LogViewModel)

	// Set search query directly
	m.searchQuery = "hello"
	m.filteredDirty = true

	view := m.View()
	if !strings.Contains(view, "hello world") {
		t.Error("matching entry should appear")
	}
	if strings.Contains(view, "foo bar") {
		t.Error("non-matching entry should not appear")
	}
}

func TestLogView_SearchMode(t *testing.T) {
	m := NewLogViewModel()
	m.SetSize(80, 24)

	// Enter search mode with /
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	m = newModel.(LogViewModel)
	if !m.searching {
		t.Error("should be in search mode after /")
	}

	// Exit search mode with Esc
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	m = newModel.(LogViewModel)
	if m.searching {
		t.Error("should exit search mode after Esc")
	}
}

func TestLogView_AutoScroll(t *testing.T) {
	m := NewLogViewModel()
	m.SetSize(80, 24)

	// Default should be autoScroll on
	if !m.autoScroll {
		t.Error("autoScroll should be true by default")
	}

	// Toggle with f
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("f")})
	m = newModel.(LogViewModel)
	if m.autoScroll {
		t.Error("autoScroll should be false after toggle")
	}

	// Toggle back
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("f")})
	m = newModel.(LogViewModel)
	if !m.autoScroll {
		t.Error("autoScroll should be true after second toggle")
	}
}

func TestLogView_AutoScroll_NewEntries(t *testing.T) {
	m := NewLogViewModel()
	m.SetSize(80, 24)
	m.autoScroll = true

	// Add many entries to fill viewport
	entries := make([]claude.LogEntry, 50)
	for i := range entries {
		entries[i] = makeLogEntry(claude.LevelDEBUG, "line")
	}
	newModel, _ := m.Update(watcher.LogEntriesMsg{Entries: entries, Initial: true})
	m = newModel.(LogViewModel)

	// Add more entries
	newEntries := []claude.LogEntry{makeLogEntry(claude.LevelERROR, "new entry")}
	newModel, _ = m.Update(watcher.LogEntriesMsg{Entries: newEntries, Initial: false})
	m = newModel.(LogViewModel)

	// The view should contain the new entry (auto-scrolled to bottom)
	view := m.View()
	if !strings.Contains(view, "new entry") {
		t.Error("auto-scroll should show new entry at bottom")
	}
}

func TestLogView_EmptyState(t *testing.T) {
	m := NewLogViewModel()
	m.SetSize(80, 24)

	view := m.View()
	if !strings.Contains(view, "デバッグログなし") {
		t.Errorf("empty state message not found in view: %s", view)
	}
}

func TestLogView_ScrollOffset_ClampOnFilterChange(t *testing.T) {
	m := NewLogViewModel()
	m.SetSize(80, 24)

	entries := []claude.LogEntry{
		makeLogEntry(claude.LevelDEBUG, "debug1"),
		makeLogEntry(claude.LevelDEBUG, "debug2"),
		makeLogEntry(claude.LevelDEBUG, "debug3"),
		makeLogEntry(claude.LevelERROR, "error1"),
	}
	newModel, _ := m.Update(watcher.LogEntriesMsg{Entries: entries, Initial: true})
	m = newModel.(LogViewModel)

	// Turn off autoScroll and set scrollOffset to a large value
	m.autoScroll = false
	m.scrollOffset = 100

	// Toggle off DEBUG filter (key "D") - only 1 ERROR entry should remain
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("D")})
	m = newModel.(LogViewModel)

	// scrollOffset should be clamped to the number of filtered entries (1)
	if m.scrollOffset > 1 {
		t.Errorf("scrollOffset = %d, want <= 1 after filter change", m.scrollOffset)
	}
}
