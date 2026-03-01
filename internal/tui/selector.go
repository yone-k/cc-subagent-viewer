package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/yone/subagent-viewer/internal/claude"
)

// SessionSelectedMsg is sent when a session is selected.
type SessionSelectedMsg struct {
	Session claude.SessionInfo
}

// sessionItem adapts SessionInfo to list.Item
type sessionItem struct {
	session claude.SessionInfo
}

func (i sessionItem) Title() string {
	return fmt.Sprintf("%s  %s", i.session.Project, relativeTime(i.session.Timestamp))
}

func (i sessionItem) Description() string {
	var parts []string
	if i.session.FirstInput != "" {
		input := i.session.FirstInput
		if len(input) > 60 {
			input = input[:57] + "..."
		}
		parts = append(parts, input)
	}

	var indicators []string
	if i.session.HasTasks {
		indicators = append(indicators, "Tasks")
	}
	if i.session.HasDebugLog {
		indicators = append(indicators, "Logs")
	}
	if i.session.HasFileHistory {
		indicators = append(indicators, "Files")
	}
	if len(indicators) > 0 {
		parts = append(parts, "["+strings.Join(indicators, "|")+"]")
	}

	return strings.Join(parts, "  ")
}

func (i sessionItem) FilterValue() string {
	return i.session.Project + " " + i.session.FirstInput + " " + i.session.SessionID
}

func relativeTime(ts int64) string {
	// Negative diff (future timestamps) falls into diff < time.Minute, showing "たった今" which is acceptable for clock skew.
	t := time.UnixMilli(ts)
	diff := time.Since(t)
	switch {
	case diff < time.Minute:
		return "たった今"
	case diff < time.Hour:
		return fmt.Sprintf("%d分前", int(diff.Minutes()))
	case diff < 24*time.Hour:
		return fmt.Sprintf("%d時間前", int(diff.Hours()))
	default:
		return fmt.Sprintf("%d日前", int(diff.Hours()/24))
	}
}

// SelectorModel manages the session selection screen.
type SelectorModel struct {
	list     list.Model
	sessions []claude.SessionInfo
	width    int
	height   int
}

// NewSelectorModel creates a new SelectorModel.
func NewSelectorModel(sessions []claude.SessionInfo) SelectorModel {
	items := make([]list.Item, len(sessions))
	for i, s := range sessions {
		items[i] = sessionItem{session: s}
	}

	l := list.New(items, list.NewDefaultDelegate(), 80, 20)
	l.Title = "Claude Code Sessions"
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.SetShowHelp(true)
	if len(sessions) == 0 {
		l.SetShowStatusBar(false)
	}

	return SelectorModel{
		list:     l,
		sessions: sessions,
		width:    80,
		height:   20,
	}
}

// SetSize updates the model dimensions.
func (m *SelectorModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.list.SetSize(width, height-2)
}

// Init initializes the model.
func (m SelectorModel) Init() tea.Cmd {
	return nil
}

// Update handles messages.
func (m SelectorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.Type == tea.KeyEnter {
			if item, ok := m.list.SelectedItem().(sessionItem); ok {
				return m, func() tea.Msg {
					return SessionSelectedMsg{Session: item.session}
				}
			}
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

// View renders the selector.
func (m SelectorModel) View() string {
	if len(m.sessions) == 0 {
		return EmptyStateStyle.Render("セッションが見つかりません")
	}
	return m.list.View()
}
