package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/yone/subagent-viewer/internal/claude"
	"github.com/yone/subagent-viewer/internal/watcher"
)

// FileViewModel manages the Files tab view.
type FileViewModel struct {
	groups   []claude.FileGroup
	expanded map[string]bool
	selected int
	width    int
	height   int
}

// NewFileViewModel creates a new FileViewModel.
func NewFileViewModel() FileViewModel {
	return FileViewModel{
		expanded: make(map[string]bool),
	}
}

// SetSize updates the view dimensions.
func (m *FileViewModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// Init initializes the model.
func (m FileViewModel) Init() tea.Cmd {
	return nil
}

// Update handles messages.
func (m FileViewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case watcher.FileHistoryUpdatedMsg:
		m.groups = msg.Groups
		if m.selected >= len(m.groups) && len(m.groups) > 0 {
			m.selected = len(m.groups) - 1
		} else if len(m.groups) == 0 {
			m.selected = 0
		}
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, FileKeys.Enter):
			if m.selected < len(m.groups) {
				hash := m.groups[m.selected].Hash
				m.expanded[hash] = !m.expanded[hash]
			}
		case key.Matches(msg, FileKeys.Escape):
			// Collapse all
			m.expanded = make(map[string]bool)
		default:
			switch msg.String() {
			case "up", "k":
				if m.selected > 0 {
					m.selected--
				}
			case "down", "j":
				if m.selected < len(m.groups)-1 {
					m.selected++
				}
			}
		}
	}
	return m, nil
}

// View renders the file history view.
func (m FileViewModel) View() string {
	if len(m.groups) == 0 {
		return EmptyStateStyle.Render("ファイル変更なし")
	}

	var b strings.Builder
	for i, group := range m.groups {
		prefix := "  "
		if i == m.selected {
			prefix = "> "
		}

		arrow := "▶"
		if m.expanded[group.Hash] {
			arrow = "▼"
		}

		b.WriteString(fmt.Sprintf("%s%s %s (%d versions)\n", prefix, arrow, group.Hash, len(group.Versions)))

		if m.expanded[group.Hash] {
			for _, v := range group.Versions {
				sizeStr := formatSize(v.Size)
				b.WriteString(fmt.Sprintf("    v%d  %s\n", v.Version, DimStyle.Render(sizeStr)))
			}
		}
	}

	return b.String()
}

func formatSize(bytes int64) string {
	switch {
	case bytes >= 1024*1024:
		return fmt.Sprintf("%.1f MB", float64(bytes)/(1024*1024))
	case bytes >= 1024:
		return fmt.Sprintf("%.1f KB", float64(bytes)/1024)
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}
