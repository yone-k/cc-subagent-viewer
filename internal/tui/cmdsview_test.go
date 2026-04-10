package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/yone-k/cc-subagent-viewer/internal/claude"
	"github.com/yone-k/cc-subagent-viewer/internal/watcher"
)

func testBashCommands() []claude.BashCommand {
	return []claude.BashCommand{
		{
			ToolUseID:   "bash-1",
			Command:     "go test ./...",
			Description: "Run tests",
			Timestamp:   time.Date(2026, 4, 8, 13, 29, 37, 0, time.UTC),
			Result: &claude.BashCommandResult{
				Stdout: "ok\n",
			},
		},
		{
			ToolUseID:   "bash-2",
			Command:     "cat large.log",
			Description: "Inspect log",
			Timestamp:   time.Date(2026, 4, 8, 13, 31, 15, 0, time.UTC),
			Result: &claude.BashCommandResult{
				PersistedOutputPath: "/tmp/large.log",
				PersistedOutputSize: 4096,
				PersistedOutput:     "line1\nline2",
			},
		},
	}
}

func TestCommandsView_EmptyState(t *testing.T) {
	m := NewCommandsViewModel()
	m.SetSize(80, 24)

	view := m.View()
	if !strings.Contains(view, "コマンドなし") {
		t.Errorf("View() should contain 'コマンドなし', got:\n%s", view)
	}
}

func TestCommandsView_ListModeShowsCommandSummary(t *testing.T) {
	m := NewCommandsViewModel()
	m.SetSize(100, 24)

	updated, _ := m.Update(watcher.BashCommandsUpdatedMsg{Commands: testBashCommands()})
	m = updated.(CommandsViewModel)

	view := m.View()
	if !strings.Contains(view, "13:29:37") {
		t.Errorf("list view should contain timestamp, got:\n%s", view)
	}
	if !strings.Contains(view, "$ go test ./...") {
		t.Errorf("list view should contain command, got:\n%s", view)
	}
	if !strings.Contains(view, "Run tests") {
		t.Errorf("list view should contain description, got:\n%s", view)
	}
}

func TestCommandsView_ModeTransition(t *testing.T) {
	m := NewCommandsViewModel()
	m.SetSize(100, 24)

	updated, _ := m.Update(watcher.BashCommandsUpdatedMsg{Commands: testBashCommands()})
	m = updated.(CommandsViewModel)

	if m.Mode() != CmdsViewModeList {
		t.Fatalf("initial mode = %d, want %d", m.Mode(), CmdsViewModeList)
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(CommandsViewModel)
	if m.Mode() != CmdsViewModeDetail {
		t.Fatalf("mode after enter = %d, want %d", m.Mode(), CmdsViewModeDetail)
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	m = updated.(CommandsViewModel)
	if m.Mode() != CmdsViewModeList {
		t.Fatalf("mode after escape = %d, want %d", m.Mode(), CmdsViewModeList)
	}
}

func TestCommandsView_DetailModeShowsResultVariants(t *testing.T) {
	m := NewCommandsViewModel()
	m.SetSize(100, 24)

	commands := []claude.BashCommand{
		{
			ToolUseID:   "bash-err",
			Command:     "go test ./internal/...",
			Description: "Run focused tests",
			Timestamp:   time.Date(2026, 4, 8, 13, 30, 1, 0, time.UTC),
			Result: &claude.BashCommandResult{
				Error: "Error: exit status 1\nFAIL\tinternal/...",
			},
		},
		{
			ToolUseID:   "bash-persisted",
			Command:     "cat large.log",
			Description: "Inspect log",
			Timestamp:   time.Date(2026, 4, 8, 13, 31, 15, 0, time.UTC),
			Result: &claude.BashCommandResult{
				PersistedOutputPath: "/tmp/large.log",
				PersistedOutputSize: 4096,
				PersistedOutput:     "persisted stdout",
			},
		},
	}

	updated, _ := m.Update(watcher.BashCommandsUpdatedMsg{Commands: commands})
	m = updated.(CommandsViewModel)

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(CommandsViewModel)
	view := m.View()
	if !strings.Contains(view, "Error: exit status 1") {
		t.Errorf("detail view should contain string result, got:\n%s", view)
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	m = updated.(CommandsViewModel)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	m = updated.(CommandsViewModel)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(CommandsViewModel)
	view = m.View()
	if !strings.Contains(view, "persisted stdout") {
		t.Errorf("detail view should contain persisted output content, got:\n%s", view)
	}
	if strings.Contains(view, "/tmp/large.log") {
		t.Errorf("detail view should render persisted output content instead of only the path, got:\n%s", view)
	}
}

func TestCommandsView_DetailModeScrollsLongOutput(t *testing.T) {
	m := NewCommandsViewModel()
	m.SetSize(80, 12)

	commands := []claude.BashCommand{
		{
			ToolUseID:   "bash-long",
			Command:     "cat build.log",
			Description: "Inspect build log",
			Timestamp:   time.Date(2026, 4, 8, 13, 45, 0, 0, time.UTC),
			Result: &claude.BashCommandResult{
				Stdout: "line1\nline2\nline3\nline4\nline5\nline6\nline7\nline8\nline9\nline10",
			},
		},
	}

	updated, _ := m.Update(watcher.BashCommandsUpdatedMsg{Commands: commands})
	m = updated.(CommandsViewModel)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(CommandsViewModel)

	view := m.View()
	if !strings.Contains(view, "line1") {
		t.Fatalf("detail view should show the start of output before scrolling, got:\n%s", view)
	}
	if strings.Contains(view, "line10") {
		t.Fatalf("detail view should not show the end of output before scrolling, got:\n%s", view)
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyPgDown})
	m = updated.(CommandsViewModel)

	view = m.View()
	if !strings.Contains(view, "line10") {
		t.Errorf("detail view should show the end of output after scrolling, got:\n%s", view)
	}
	if strings.Contains(view, "Bashコマンド詳細") {
		t.Errorf("detail view should move past the header after scrolling, got:\n%s", view)
	}
}
