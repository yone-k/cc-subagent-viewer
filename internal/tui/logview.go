package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/yone/subagent-viewer/internal/claude"
	"github.com/yone/subagent-viewer/internal/watcher"
)

const maxLogEntries = 10000

// LogViewModel manages the Logs tab view.
type LogViewModel struct {
	entries       []claude.LogEntry
	filterLevels  map[claude.LogLevel]bool
	searchQuery   string
	searchInput   textinput.Model
	searching     bool
	autoScroll    bool
	scrollOffset  int
	width         int
	height        int
	filteredCache []claude.LogEntry
	filteredDirty bool
}

// NewLogViewModel creates a new LogViewModel.
func NewLogViewModel() LogViewModel {
	ti := textinput.New()
	ti.Placeholder = "検索..."
	ti.CharLimit = 100

	return LogViewModel{
		filterLevels: map[claude.LogLevel]bool{
			claude.LevelDEBUG:      true,
			claude.LevelERROR:      true,
			claude.LevelWARN:       true,
			claude.LevelMCP:        true,
			claude.LevelSTARTUP:    true,
			claude.LevelMETA:       true,
			claude.LevelATTACHMENT: true,
		},
		searchInput:   ti,
		autoScroll:    true,
		filteredDirty: true,
	}
}

// SetSize updates the view dimensions.
func (m *LogViewModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// EntryCount returns the number of stored entries.
func (m LogViewModel) EntryCount() int {
	return len(m.entries)
}

// Init initializes the model.
func (m LogViewModel) Init() tea.Cmd {
	return nil
}

// Update handles messages.
func (m LogViewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case watcher.LogEntriesMsg:
		if msg.Initial {
			m.entries = msg.Entries
		} else {
			m.entries = append(m.entries, msg.Entries...)
		}
		// Ring buffer: trim oldest entries
		if len(m.entries) > maxLogEntries {
			m.entries = m.entries[len(m.entries)-maxLogEntries:]
		}
		m.filteredDirty = true
		if m.autoScroll {
			filtered := m.filteredEntries()
			m.scrollOffset = len(filtered)
		}
		return m, nil

	case tea.KeyMsg:
		if m.searching {
			switch msg.Type {
			case tea.KeyEscape:
				m.searching = false
				m.searchInput.Blur()
				return m, nil
			case tea.KeyEnter:
				m.searchQuery = m.searchInput.Value()
				m.searching = false
				m.searchInput.Blur()
				m.filteredDirty = true
				filtered := m.filteredEntries()
				if m.scrollOffset > len(filtered) {
					m.scrollOffset = len(filtered)
				}
				return m, nil
			default:
				var cmd tea.Cmd
				m.searchInput, cmd = m.searchInput.Update(msg)
				m.searchQuery = m.searchInput.Value()
				m.filteredDirty = true
				return m, cmd
			}
		}

		switch {
		case key.Matches(msg, LogKeys.FilterDEBUG):
			m.filterLevels[claude.LevelDEBUG] = !m.filterLevels[claude.LevelDEBUG]
			m.filteredDirty = true
			filtered := m.filteredEntries()
			if m.scrollOffset > len(filtered) {
				m.scrollOffset = len(filtered)
			}
		case key.Matches(msg, LogKeys.FilterERROR):
			m.filterLevels[claude.LevelERROR] = !m.filterLevels[claude.LevelERROR]
			m.filteredDirty = true
			filtered := m.filteredEntries()
			if m.scrollOffset > len(filtered) {
				m.scrollOffset = len(filtered)
			}
		case key.Matches(msg, LogKeys.FilterWARN):
			m.filterLevels[claude.LevelWARN] = !m.filterLevels[claude.LevelWARN]
			m.filteredDirty = true
			filtered := m.filteredEntries()
			if m.scrollOffset > len(filtered) {
				m.scrollOffset = len(filtered)
			}
		case key.Matches(msg, LogKeys.FilterMCP):
			m.filterLevels[claude.LevelMCP] = !m.filterLevels[claude.LevelMCP]
			m.filteredDirty = true
			filtered := m.filteredEntries()
			if m.scrollOffset > len(filtered) {
				m.scrollOffset = len(filtered)
			}
		case key.Matches(msg, LogKeys.FilterSTARTUP):
			m.filterLevels[claude.LevelSTARTUP] = !m.filterLevels[claude.LevelSTARTUP]
			m.filteredDirty = true
			filtered := m.filteredEntries()
			if m.scrollOffset > len(filtered) {
				m.scrollOffset = len(filtered)
			}
		case key.Matches(msg, LogKeys.FilterMETA):
			m.filterLevels[claude.LevelMETA] = !m.filterLevels[claude.LevelMETA]
			m.filteredDirty = true
			filtered := m.filteredEntries()
			if m.scrollOffset > len(filtered) {
				m.scrollOffset = len(filtered)
			}
		case key.Matches(msg, LogKeys.FilterATTACHMENT):
			m.filterLevels[claude.LevelATTACHMENT] = !m.filterLevels[claude.LevelATTACHMENT]
			m.filteredDirty = true
			filtered := m.filteredEntries()
			if m.scrollOffset > len(filtered) {
				m.scrollOffset = len(filtered)
			}
		case key.Matches(msg, LogKeys.Search):
			m.searching = true
			m.searchInput.Focus()
			return m, m.searchInput.Cursor.BlinkCmd()
		case key.Matches(msg, LogKeys.AutoScroll):
			m.autoScroll = !m.autoScroll
		default:
			switch msg.String() {
			case "up", "k":
				if m.scrollOffset > 0 {
					m.scrollOffset--
					m.autoScroll = false
				}
			case "down", "j":
				m.scrollOffset++
				filtered := m.filteredEntries()
				if m.scrollOffset >= len(filtered) {
					m.scrollOffset = len(filtered)
				}
			}
		}
	}
	return m, nil
}

func (m *LogViewModel) filteredEntries() []claude.LogEntry {
	if !m.filteredDirty && m.filteredCache != nil {
		return m.filteredCache
	}
	var filtered []claude.LogEntry
	for _, entry := range m.entries {
		if !m.filterLevels[entry.Level] {
			continue
		}
		if m.searchQuery != "" && !strings.Contains(entry.Message, m.searchQuery) && !strings.Contains(entry.Raw, m.searchQuery) {
			continue
		}
		filtered = append(filtered, entry)
	}
	m.filteredCache = filtered
	m.filteredDirty = false
	return filtered
}

// View renders the log viewer.
func (m LogViewModel) View() string {
	if len(m.entries) == 0 {
		return EmptyStateStyle.Render("デバッグログなし")
	}

	var b strings.Builder

	// Filter bar
	b.WriteString(m.renderFilterBar())
	b.WriteString("\n\n")

	// Filtered entries
	filtered := m.filteredEntries()
	viewHeight := m.height - 4 // Reserve space for filter bar and status
	if viewHeight < 1 {
		viewHeight = 10
	}

	// Calculate visible range
	start := 0
	if len(filtered) > viewHeight {
		if m.autoScroll {
			start = len(filtered) - viewHeight
		} else if m.scrollOffset > viewHeight {
			start = m.scrollOffset - viewHeight
		}
	}
	end := start + viewHeight
	if end > len(filtered) {
		end = len(filtered)
	}

	for i := start; i < end; i++ {
		entry := filtered[i]
		levelStyle := logLevelStyle(entry.Level)
		ts := entry.Timestamp.Format("15:04:05.000")
		levelStr := levelStyle.Render(fmt.Sprintf("[%-10s]", entry.Level))
		b.WriteString(fmt.Sprintf("%s %s %s\n", DimStyle.Render(ts), levelStr, entry.Message))
	}

	// Status bar
	scrollStatus := "AUTO"
	if !m.autoScroll {
		scrollStatus = "MANUAL"
	}
	b.WriteString(HelpStyle.Render(fmt.Sprintf("\n%d entries | Scroll: %s", len(filtered), scrollStatus)))

	return b.String()
}

func (m LogViewModel) renderFilterBar() string {
	type filterDef struct {
		key   string
		label string
		level claude.LogLevel
	}
	filters := []filterDef{
		{"D", "Debug", claude.LevelDEBUG},
		{"E", "Error", claude.LevelERROR},
		{"W", "Warn", claude.LevelWARN},
		{"M", "MCP", claude.LevelMCP},
		{"S", "Startup", claude.LevelSTARTUP},
		{"T", "Meta", claude.LevelMETA},
		{"A", "Attach", claude.LevelATTACHMENT},
	}

	var parts []string
	for _, f := range filters {
		label := formatFilterLabel(f.key, f.label)
		if m.filterLevels[f.level] {
			parts = append(parts, FilterActiveStyle.Render(label))
		} else {
			parts = append(parts, FilterInactiveStyle.Render(label))
		}
	}

	filterBar := "Filter: " + strings.Join(parts, " ")

	if m.searching {
		filterBar += "  Search: " + m.searchInput.View()
	} else if m.searchQuery != "" {
		filterBar += "  Search: " + m.searchQuery
	}

	return filterBar
}

func formatFilterLabel(key, label string) string {
	keyLower := strings.ToLower(key)
	labelLower := strings.ToLower(label)
	idx := strings.Index(labelLower, keyLower)
	if idx >= 0 {
		return label[:idx] + "[" + string(label[idx]) + "]" + label[idx+1:]
	}
	return "[" + key + "] " + label
}

func logLevelStyle(level claude.LogLevel) lipgloss.Style {
	switch level {
	case claude.LevelDEBUG:
		return LogLevelDEBUG
	case claude.LevelERROR:
		return LogLevelERROR
	case claude.LevelWARN:
		return LogLevelWARN
	case claude.LevelMCP:
		return LogLevelMCP
	case claude.LevelSTARTUP:
		return LogLevelSTARTUP
	case claude.LevelMETA:
		return LogLevelMETA
	case claude.LevelATTACHMENT:
		return LogLevelATTACHMENT
	default:
		return LogLevelDEBUG
	}
}
