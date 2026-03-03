package tui

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/yone-k/cc-subagent-viewer/internal/claude"
	"github.com/yone-k/cc-subagent-viewer/internal/watcher"
)

// TestAgentView_EmptyState verifies that View() contains "サブエージェントなし" when no agents exist.
func TestAgentView_EmptyState(t *testing.T) {
	m := NewAgentViewModel()
	m.SetSize(80, 24)

	got := m.View()
	if !strings.Contains(got, "サブエージェントなし") {
		t.Errorf("expected View() to contain 'サブエージェントなし', got:\n%s", got)
	}
}

// TestAgentView_ModeTransition verifies mode transitions between list and conversation.
func TestAgentView_ModeTransition(t *testing.T) {
	agents := []claude.SubagentInfo{
		{AgentID: "a1", Slug: "test-agent", Prompt: "test prompt", EntryCount: 3},
	}

	t.Run("enter_key_transitions_to_conversation", func(t *testing.T) {
		m := NewAgentViewModel()
		m.SetSize(80, 24)

		// Inject agents via SubagentsDiscoveredMsg
		updated, _ := m.Update(watcher.SubagentsDiscoveredMsg{Agents: agents})
		m = updated.(AgentViewModel)

		if m.Mode() != AgentViewModeList {
			t.Fatalf("expected initial mode to be AgentViewModeList, got %d", m.Mode())
		}

		// Press enter to select first agent
		enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
		updated, _ = m.Update(enterMsg)
		m = updated.(AgentViewModel)

		if m.Mode() != AgentViewModeConversation {
			t.Errorf("expected mode to be AgentViewModeConversation after enter, got %d", m.Mode())
		}
	})

	t.Run("esc_key_returns_to_list", func(t *testing.T) {
		m := NewAgentViewModel()
		m.SetSize(80, 24)

		// Inject agents and transition to conversation mode
		updated, _ := m.Update(watcher.SubagentsDiscoveredMsg{Agents: agents})
		m = updated.(AgentViewModel)

		enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
		updated, _ = m.Update(enterMsg)
		m = updated.(AgentViewModel)

		if m.Mode() != AgentViewModeConversation {
			t.Fatalf("expected mode to be AgentViewModeConversation, got %d", m.Mode())
		}

		// Press esc to return to list
		escMsg := tea.KeyMsg{Type: tea.KeyEscape}
		updated, _ = m.Update(escMsg)
		m = updated.(AgentViewModel)

		if m.Mode() != AgentViewModeList {
			t.Errorf("expected mode to be AgentViewModeList after esc, got %d", m.Mode())
		}
	})
}

// TestAgentViewModel_ViewStatus verifies that status badges are displayed for running and closed agents.
func TestAgentViewModel_ViewStatus(t *testing.T) {
	agents := []claude.SubagentInfo{
		{
			AgentID: "running-agent",
			Prompt:  "running test",
			Status:  claude.SubagentRunning,
		},
		{
			AgentID: "closed-agent",
			Prompt:  "closed test",
			Status:  claude.SubagentClosed,
		},
	}

	m := NewAgentViewModel()
	m.SetSize(80, 24)

	// Inject agents via SubagentsDiscoveredMsg
	updated, _ := m.Update(watcher.SubagentsDiscoveredMsg{Agents: agents})
	m = updated.(AgentViewModel)

	view := m.View()

	// Verify Running badge (yellow ●)
	if !strings.Contains(view, "●") {
		t.Error("expected running indicator ● in view")
	}

	// Verify Closed badge (green ✓)
	if !strings.Contains(view, "✓") {
		t.Error("expected completed indicator ✓ in view")
	}
}

// TestAgentView_ConversationUpdatedMsg verifies that ConversationUpdatedMsg updates stored data.
func TestAgentView_ConversationUpdatedMsg(t *testing.T) {
	m := NewAgentViewModel()
	m.SetSize(80, 24)

	agents := []claude.SubagentInfo{
		{AgentID: "a1", Slug: "test-agent", Prompt: "test prompt", EntryCount: 3},
	}

	entries := []claude.ConversationEntry{
		{Type: claude.EntryTypeUser, Content: []claude.ContentBlock{{Type: "text", Text: "Hello"}}},
	}

	info := &claude.SubagentInfo{AgentID: "a1", Slug: "test-agent", EntryCount: 1}

	// Inject agents first
	updated, _ := m.Update(watcher.SubagentsDiscoveredMsg{Agents: agents})
	m = updated.(AgentViewModel)

	// Send ConversationUpdatedMsg
	updated, _ = m.Update(watcher.ConversationUpdatedMsg{
		AgentID: "a1",
		Entries: entries,
		Info:    info,
	})
	m = updated.(AgentViewModel)

	// Verify data is stored by entering conversation mode and checking the view
	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	updated, _ = m.Update(enterMsg)
	m = updated.(AgentViewModel)

	if m.Mode() != AgentViewModeConversation {
		t.Fatalf("expected AgentViewModeConversation, got %d", m.Mode())
	}

	got := m.View()
	if !strings.Contains(got, "Hello") {
		t.Errorf("expected conversation view to contain 'Hello', got:\n%s", got)
	}
}

func TestAgentView_ScrollFollowsSelection(t *testing.T) {
	m := NewAgentViewModel()
	m.SetSize(80, 11) // height=11: viewHeight = 11-2 = 9, visibleItems = 9/3 = 3

	// Create 10 agents
	agents := make([]claude.SubagentInfo, 10)
	for i := range agents {
		agents[i] = claude.SubagentInfo{
			AgentID: fmt.Sprintf("agent-%d", i+1),
			Slug:    fmt.Sprintf("agent-%d", i+1),
			Prompt:  fmt.Sprintf("Prompt for agent %d", i+1),
		}
	}
	updated, _ := m.Update(watcher.SubagentsDiscoveredMsg{Agents: agents})
	m = updated.(AgentViewModel)

	if m.scrollOffset != 0 {
		t.Errorf("initial scrollOffset = %d, want 0", m.scrollOffset)
	}

	// Move down to item 3 (index 2) — still visible
	for i := 0; i < 2; i++ {
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
		m = updated.(AgentViewModel)
	}
	if m.agentSelected != 2 {
		t.Errorf("agentSelected = %d, want 2", m.agentSelected)
	}
	if m.scrollOffset != 0 {
		t.Errorf("scrollOffset = %d, want 0", m.scrollOffset)
	}

	// Move down to item 4 (index 3) — should start scrolling
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	m = updated.(AgentViewModel)
	if m.agentSelected != 3 {
		t.Errorf("agentSelected = %d, want 3", m.agentSelected)
	}
	if m.scrollOffset != 1 {
		t.Errorf("scrollOffset = %d, want 1", m.scrollOffset)
	}

	// Move to last item (index 9)
	for i := 0; i < 6; i++ {
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
		m = updated.(AgentViewModel)
	}
	if m.agentSelected != 9 {
		t.Errorf("agentSelected = %d, want 9", m.agentSelected)
	}
	if m.scrollOffset != 7 {
		t.Errorf("scrollOffset = %d, want 7", m.scrollOffset)
	}

	// Move all the way back up
	for i := 0; i < 9; i++ {
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
		m = updated.(AgentViewModel)
	}
	if m.agentSelected != 0 {
		t.Errorf("agentSelected = %d, want 0", m.agentSelected)
	}
	if m.scrollOffset != 0 {
		t.Errorf("scrollOffset = %d, want 0", m.scrollOffset)
	}
}

// pgUpKey, pgDownKey are declared in conversationview_test.go (same package)

func TestAgentView_PageDown(t *testing.T) {
	m := NewAgentViewModel()
	m.SetSize(80, 11) // viewHeight = 11-2 = 9, pageSize = 9/3 = 3

	// Given: 10 agents, agentSelected at 0
	agents := make([]claude.SubagentInfo, 10)
	for i := range agents {
		agents[i] = claude.SubagentInfo{
			AgentID: fmt.Sprintf("agent-%d", i+1),
			Slug:    fmt.Sprintf("agent-%d", i+1),
			Prompt:  fmt.Sprintf("Prompt for agent %d", i+1),
		}
	}
	updated, _ := m.Update(watcher.SubagentsDiscoveredMsg{Agents: agents})
	m = updated.(AgentViewModel)

	// When: pgdown is pressed
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyPgDown})
	m = updated.(AgentViewModel)

	// Then: agentSelected moves by pageSize (3)
	if m.agentSelected != 3 {
		t.Errorf("after pgdown, agentSelected = %d, want 3", m.agentSelected)
	}
}

func TestAgentView_PageUp(t *testing.T) {
	m := NewAgentViewModel()
	m.SetSize(80, 11) // viewHeight = 11-2 = 9, pageSize = 9/3 = 3

	// Given: 10 agents, agentSelected at index 6
	agents := make([]claude.SubagentInfo, 10)
	for i := range agents {
		agents[i] = claude.SubagentInfo{
			AgentID: fmt.Sprintf("agent-%d", i+1),
			Slug:    fmt.Sprintf("agent-%d", i+1),
			Prompt:  fmt.Sprintf("Prompt for agent %d", i+1),
		}
	}
	updated, _ := m.Update(watcher.SubagentsDiscoveredMsg{Agents: agents})
	m = updated.(AgentViewModel)

	// Move to index 6
	for i := 0; i < 6; i++ {
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
		m = updated.(AgentViewModel)
	}
	if m.agentSelected != 6 {
		t.Fatalf("setup: agentSelected = %d, want 6", m.agentSelected)
	}

	// When: pgup is pressed
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyPgUp})
	m = updated.(AgentViewModel)

	// Then: agentSelected moves up by pageSize (3)
	if m.agentSelected != 3 {
		t.Errorf("after pgup, agentSelected = %d, want 3", m.agentSelected)
	}
}

func TestAgentView_PageDown_ClampsAtEnd(t *testing.T) {
	m := NewAgentViewModel()
	m.SetSize(80, 11) // viewHeight = 11-2 = 9, pageSize = 9/3 = 3

	// Given: 5 agents, agentSelected at index 3
	agents := make([]claude.SubagentInfo, 5)
	for i := range agents {
		agents[i] = claude.SubagentInfo{
			AgentID: fmt.Sprintf("agent-%d", i+1),
			Slug:    fmt.Sprintf("agent-%d", i+1),
			Prompt:  fmt.Sprintf("Prompt for agent %d", i+1),
		}
	}
	updated, _ := m.Update(watcher.SubagentsDiscoveredMsg{Agents: agents})
	m = updated.(AgentViewModel)

	// Move to index 3
	for i := 0; i < 3; i++ {
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
		m = updated.(AgentViewModel)
	}

	// When: pgdown is pressed (3 + 3 = 6, but max is 4)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyPgDown})
	m = updated.(AgentViewModel)

	// Then: agentSelected clamped to last item (index 4)
	if m.agentSelected != 4 {
		t.Errorf("after pgdown near end, agentSelected = %d, want 4", m.agentSelected)
	}
}

func TestAgentView_PageUp_ClampsAtStart(t *testing.T) {
	m := NewAgentViewModel()
	m.SetSize(80, 11) // viewHeight = 11-2 = 9, pageSize = 9/3 = 3

	// Given: 10 agents, agentSelected at index 2
	agents := make([]claude.SubagentInfo, 10)
	for i := range agents {
		agents[i] = claude.SubagentInfo{
			AgentID: fmt.Sprintf("agent-%d", i+1),
			Slug:    fmt.Sprintf("agent-%d", i+1),
			Prompt:  fmt.Sprintf("Prompt for agent %d", i+1),
		}
	}
	updated, _ := m.Update(watcher.SubagentsDiscoveredMsg{Agents: agents})
	m = updated.(AgentViewModel)

	// Move to index 2
	for i := 0; i < 2; i++ {
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
		m = updated.(AgentViewModel)
	}

	// When: pgup is pressed (2 - 3 = -1, should clamp to 0)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyPgUp})
	m = updated.(AgentViewModel)

	// Then: agentSelected clamped to 0
	if m.agentSelected != 0 {
		t.Errorf("after pgup near start, agentSelected = %d, want 0", m.agentSelected)
	}
}

func TestAgentView_PageScroll_FewerAgentsThanPageSize(t *testing.T) {
	m := NewAgentViewModel()
	m.SetSize(80, 20) // viewHeight = 20-2 = 18, pageSize = 18/3 = 6

	// Given: only 2 agents (fewer than pageSize of 6)
	agents := make([]claude.SubagentInfo, 2)
	for i := range agents {
		agents[i] = claude.SubagentInfo{
			AgentID: fmt.Sprintf("agent-%d", i+1),
			Slug:    fmt.Sprintf("agent-%d", i+1),
			Prompt:  fmt.Sprintf("Prompt for agent %d", i+1),
		}
	}
	updated, _ := m.Update(watcher.SubagentsDiscoveredMsg{Agents: agents})
	m = updated.(AgentViewModel)

	// When: pgdown is pressed
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyPgDown})
	m = updated.(AgentViewModel)

	// Then: agentSelected moves to last item (index 1)
	if m.agentSelected != 1 {
		t.Errorf("pgdown with few agents, agentSelected = %d, want 1", m.agentSelected)
	}

	// When: pgup is pressed
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyPgUp})
	m = updated.(AgentViewModel)

	// Then: agentSelected moves back to 0
	if m.agentSelected != 0 {
		t.Errorf("pgup with few agents, agentSelected = %d, want 0", m.agentSelected)
	}
}

func TestAgentView_PageScroll_OnlyInListMode(t *testing.T) {
	m := NewAgentViewModel()
	m.SetSize(80, 11)

	// Given: agents and in conversation mode
	agents := []claude.SubagentInfo{
		{AgentID: "a1", Slug: "test-agent", Prompt: "test prompt", EntryCount: 3},
	}
	updated, _ := m.Update(watcher.SubagentsDiscoveredMsg{Agents: agents})
	m = updated.(AgentViewModel)

	// Enter conversation mode
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(AgentViewModel)
	if m.Mode() != AgentViewModeConversation {
		t.Fatalf("expected AgentViewModeConversation, got %d", m.Mode())
	}

	// When: pgdown is pressed in conversation mode
	// Then: it should not affect agentSelected (it's delegated to conversation view)
	prevSelected := m.agentSelected
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyPgDown})
	m = updated.(AgentViewModel)

	if m.agentSelected != prevSelected {
		t.Errorf("pgdown in conversation mode should not change agentSelected, was %d, got %d", prevSelected, m.agentSelected)
	}
}
