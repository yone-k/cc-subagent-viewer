package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/yone/subagent-viewer/internal/claude"
)

// SessionSelectedMsg is sent when a session is selected.
// (This type will be defined in selector.go)

func TestSelectorModel_Init(t *testing.T) {
	sessions := []claude.SessionInfo{
		{SessionID: "s1", Project: "/test/proj", Timestamp: 1000, FirstInput: "hello"},
		{SessionID: "s2", Project: "/test/proj2", Timestamp: 2000, FirstInput: "world"},
	}
	m := NewSelectorModel(sessions)
	if len(m.sessions) != 2 {
		t.Errorf("expected 2 sessions loaded, got %d", len(m.sessions))
	}
}

func TestSelectorModel_SelectSession(t *testing.T) {
	sessions := []claude.SessionInfo{
		{SessionID: "s1", Project: "/test/proj", Timestamp: 1000, FirstInput: "hello"},
	}
	m := NewSelectorModel(sessions)
	m.SetSize(80, 24)

	// Simulate Enter key
	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = newModel.(SelectorModel)

	if cmd == nil {
		t.Fatal("expected a command from Enter press")
	}
	// Execute the command to get the message
	msg := cmd()
	if _, ok := msg.(SessionSelectedMsg); !ok {
		t.Errorf("expected SessionSelectedMsg, got %T", msg)
	}
}

func TestSelectorModel_EmptyList(t *testing.T) {
	m := NewSelectorModel(nil)
	m.SetSize(80, 24)
	view := m.View()
	if !strings.Contains(view, "セッションが見つかりません") {
		t.Errorf("empty state message not found in view: %s", view)
	}
}

func TestSessionItem_LongFirstInput(t *testing.T) {
	longInput := strings.Repeat("a", 61) // 61 characters, exceeds 60 limit
	item := sessionItem{session: claude.SessionInfo{
		SessionID:  "s1",
		Project:    "/test",
		Timestamp:  1000,
		FirstInput: longInput,
	}}
	desc := item.Description()
	if !strings.Contains(desc, "...") {
		t.Errorf("description should contain '...' for long input, got %q", desc)
	}
	// The truncated input should be 57 chars + "..." = 60 chars
	if strings.Contains(desc, longInput) {
		t.Errorf("description should not contain full long input, got %q", desc)
	}
}

func TestSelectorModel_DisplayFormat(t *testing.T) {
	sessions := []claude.SessionInfo{
		{
			SessionID:      "s1",
			Project:        "/test/project",
			Timestamp:      1772326237190,
			FirstInput:     "プロジェクトを分析して",
			HasTasks:       true,
			HasDebugLog:    true,
			HasFileHistory: false,
		},
	}
	m := NewSelectorModel(sessions)
	m.SetSize(120, 40)

	// Get the item description
	items := m.list.Items()
	if len(items) == 0 {
		t.Fatal("no items in list")
	}
	item := items[0].(sessionItem)
	title := item.Title()
	desc := item.Description()

	// Should contain project path
	if !strings.Contains(title, "/test/project") {
		t.Errorf("title should contain project path, got %q", title)
	}

	// Should contain first input
	if !strings.Contains(desc, "プロジェクトを分析して") {
		t.Errorf("description should contain first input, got %q", desc)
	}
}
