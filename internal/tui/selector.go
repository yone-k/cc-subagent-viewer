package tui

import (
	"fmt"
	"io"
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
		parts = append(parts, i.session.FirstInput)
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

// sessionDelegate implements list.ItemDelegate with the same cursor style as other views.
type sessionDelegate struct{}

func (d sessionDelegate) Height() int                               { return 2 }
func (d sessionDelegate) Spacing() int                              { return 1 }
func (d sessionDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd   { return nil }
func (d sessionDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	si, ok := item.(sessionItem)
	if !ok {
		return
	}

	isSelected := index == m.Index()

	prefix := "  "
	if isSelected {
		prefix = "> "
	}

	title := si.Title()
	desc := si.Description()
	// 4 = indent for description line ("    ")
	maxDescWidth := m.Width() - 4
	if maxDescWidth < 10 {
		maxDescWidth = 10
	}
	desc = truncateText(desc, maxDescWidth)

	if isSelected {
		fmt.Fprintf(w, "%s%s\n", prefix, SelectedLabelStyle.Render(title))
		if desc != "" {
			fmt.Fprintf(w, "    %s", SelectedDetailStyle.Render(desc))
		}
	} else {
		fmt.Fprintf(w, "%s%s\n", prefix, title)
		if desc != "" {
			fmt.Fprintf(w, "    %s", DimStyle.Render(desc))
		}
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

	l := list.New(items, sessionDelegate{}, 80, 20)
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
