package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/yone/subagent-viewer/internal/claude"
	"github.com/yone/subagent-viewer/internal/watcher"
)

func TestFileView_UpdateWithGroups(t *testing.T) {
	m := NewFileViewModel()
	m.SetSize(80, 24)

	groups := []claude.FileGroup{
		{Hash: "abcd1234", Versions: []claude.FileVersion{
			{Hash: "abcd1234", Version: 1, Path: "/tmp/abcd1234@v1", Size: 100},
			{Hash: "abcd1234", Version: 2, Path: "/tmp/abcd1234@v2", Size: 150},
		}},
	}
	newModel, _ := m.Update(watcher.FileHistoryUpdatedMsg{Groups: groups})
	m = newModel.(FileViewModel)

	if len(m.groups) != 1 {
		t.Errorf("expected 1 group, got %d", len(m.groups))
	}
}

func TestFileView_ExpandCollapse(t *testing.T) {
	m := NewFileViewModel()
	m.SetSize(80, 24)

	groups := []claude.FileGroup{
		{Hash: "abcd1234", Versions: []claude.FileVersion{
			{Hash: "abcd1234", Version: 1, Path: "/tmp/abcd1234@v1", Size: 100},
		}},
	}
	newModel, _ := m.Update(watcher.FileHistoryUpdatedMsg{Groups: groups})
	m = newModel.(FileViewModel)

	// Initially collapsed
	if m.expanded["abcd1234"] {
		t.Error("groups should be collapsed initially")
	}

	// Toggle expand with Enter
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = newModel.(FileViewModel)

	if !m.expanded["abcd1234"] {
		t.Error("group should be expanded after Enter")
	}

	// Toggle collapse with Enter
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = newModel.(FileViewModel)

	if m.expanded["abcd1234"] {
		t.Error("group should be collapsed after second Enter")
	}
}

func TestFileView_VersionDisplay(t *testing.T) {
	m := NewFileViewModel()
	m.SetSize(80, 24)

	groups := []claude.FileGroup{
		{Hash: "abcd1234", Versions: []claude.FileVersion{
			{Hash: "abcd1234", Version: 1, Path: "/tmp/abcd1234@v1", Size: 1024},
			{Hash: "abcd1234", Version: 2, Path: "/tmp/abcd1234@v2", Size: 2048},
		}},
	}
	newModel, _ := m.Update(watcher.FileHistoryUpdatedMsg{Groups: groups})
	m = newModel.(FileViewModel)

	// Expand group
	m.expanded["abcd1234"] = true

	view := m.View()
	// Should show version info with size
	if !strings.Contains(view, "v1") {
		t.Error("view should contain version v1")
	}
	if !strings.Contains(view, "v2") {
		t.Error("view should contain version v2")
	}
}

func TestFileView_EmptyState(t *testing.T) {
	m := NewFileViewModel()
	m.SetSize(80, 24)

	view := m.View()
	if !strings.Contains(view, "ファイル変更なし") {
		t.Errorf("empty state not shown: %s", view)
	}
}

func TestFileView_SelectedClampOnGroupsReduced(t *testing.T) {
	m := NewFileViewModel()
	m.SetSize(80, 24)

	// Set up 3 groups
	groups := []claude.FileGroup{
		{Hash: "aaa", Versions: []claude.FileVersion{{Hash: "aaa", Version: 1, Path: "/tmp/aaa@v1", Size: 10}}},
		{Hash: "bbb", Versions: []claude.FileVersion{{Hash: "bbb", Version: 1, Path: "/tmp/bbb@v1", Size: 20}}},
		{Hash: "ccc", Versions: []claude.FileVersion{{Hash: "ccc", Version: 1, Path: "/tmp/ccc@v1", Size: 30}}},
	}
	newModel, _ := m.Update(watcher.FileHistoryUpdatedMsg{Groups: groups})
	m = newModel.(FileViewModel)

	// Move selected to the last item (index 2)
	m.selected = 2

	// Now reduce groups to only 1 item
	reducedGroups := []claude.FileGroup{
		{Hash: "aaa", Versions: []claude.FileVersion{{Hash: "aaa", Version: 1, Path: "/tmp/aaa@v1", Size: 10}}},
	}
	newModel, _ = m.Update(watcher.FileHistoryUpdatedMsg{Groups: reducedGroups})
	m = newModel.(FileViewModel)

	if m.selected >= len(m.groups) {
		t.Errorf("selected = %d, but only %d groups exist", m.selected, len(m.groups))
	}
	if m.selected != 0 {
		t.Errorf("selected = %d, want 0", m.selected)
	}
}
