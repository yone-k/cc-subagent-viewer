package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/yone-k/cc-subagent-viewer/internal/claude"
	"github.com/yone-k/cc-subagent-viewer/internal/watcher"
)

// CmdsViewMode represents the current view mode within the Cmds tab.
type CmdsViewMode int

const (
	CmdsViewModeList CmdsViewMode = iota
	CmdsViewModeDetail
)

// CommandsViewModel manages the Cmds tab.
type CommandsViewModel struct {
	commands      []claude.BashCommand
	selected      int
	scrollOffset  int
	detailScroll  int
	mode          CmdsViewMode
	width, height int
}

func NewCommandsViewModel() CommandsViewModel {
	return CommandsViewModel{}
}

func (m *CommandsViewModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

func (m CommandsViewModel) Mode() CmdsViewMode {
	return m.mode
}

func (m CommandsViewModel) Init() tea.Cmd {
	return nil
}

func (m CommandsViewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case watcher.BashCommandsUpdatedMsg:
		m.commands = msg.Commands
		if len(m.commands) == 0 {
			m.selected = 0
			m.scrollOffset = 0
			m.detailScroll = 0
			m.mode = CmdsViewModeList
			return m, nil
		}
		if m.selected >= len(m.commands) {
			m.selected = len(m.commands) - 1
		}
		m = m.clampScroll()
		m = m.clampDetailScroll()
	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m CommandsViewModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.mode {
	case CmdsViewModeDetail:
		switch msg.String() {
		case "esc":
			m.mode = CmdsViewModeList
		case "up", "k":
			if m.detailScroll > 0 {
				m.detailScroll--
			}
		case "down", "j":
			m.detailScroll++
			m = m.clampDetailScroll()
		case "pgdown":
			m.detailScroll += m.pageSize()
			m = m.clampDetailScroll()
		case "pgup":
			m.detailScroll -= m.pageSize()
			if m.detailScroll < 0 {
				m.detailScroll = 0
			}
		}
		return m, nil
	default:
		switch msg.String() {
		case "up", "k":
			if m.selected > 0 {
				m.selected--
			}
		case "down", "j":
			if m.selected < len(m.commands)-1 {
				m.selected++
			}
		case "pgdown":
			m.selected += m.pageSize()
			if m.selected >= len(m.commands) && len(m.commands) > 0 {
				m.selected = len(m.commands) - 1
			}
		case "pgup":
			m.selected -= m.pageSize()
			if m.selected < 0 {
				m.selected = 0
			}
		case "enter":
			if len(m.commands) > 0 {
				m.mode = CmdsViewModeDetail
				m.detailScroll = 0
			}
		}
		m = m.clampScroll()
		return m, nil
	}
}

func (m CommandsViewModel) View() string {
	if len(m.commands) == 0 {
		return EmptyStateStyle.Render("コマンドなし")
	}
	if m.mode == CmdsViewModeDetail {
		return m.viewDetail()
	}
	return m.viewList()
}

func (m CommandsViewModel) viewList() string {
	var b strings.Builder
	b.WriteString(TitleStyle.Render("Bashコマンド一覧"))
	b.WriteString("\n\n")

	viewHeight := m.viewHeight()
	for i := m.scrollOffset; i < len(m.commands) && i < m.scrollOffset+viewHeight; i++ {
		cmd := m.commands[i]
		label := fmt.Sprintf("%s $ %s", cmd.Timestamp.Format("15:04:05"), cmd.Command)
		if cmd.Description != "" {
			label += fmt.Sprintf(" (%s)", cmd.Description)
		}
		b.WriteString(renderListItem(i == m.selected, truncateText(label, m.width-2)))
		if i < len(m.commands)-1 && i < m.scrollOffset+viewHeight-1 {
			b.WriteString("\n")
		}
	}

	return b.String()
}

func (m CommandsViewModel) viewDetail() string {
	lines := m.detailLines()
	if len(lines) == 0 {
		return ""
	}
	viewHeight := m.viewHeight()
	start := m.detailScroll
	if start < 0 {
		start = 0
	}
	end := start + viewHeight
	if end > len(lines) {
		end = len(lines)
	}
	return strings.Join(lines[start:end], "\n")
}

func (m CommandsViewModel) detailLines() []string {
	cmd := m.commands[m.selected]
	sections := []string{
		TitleStyle.Render("Bashコマンド詳細"),
		fmt.Sprintf("時刻: %s", cmd.Timestamp.Format("2006-01-02 15:04:05")),
		fmt.Sprintf("コマンド: %s", cmd.Command),
		fmt.Sprintf("description: %s", cmd.Description),
		"",
		"結果:",
		m.renderResult(cmd.Result),
	}
	return strings.Split(strings.Join(sections, "\n"), "\n")
}

func (m CommandsViewModel) renderResult(result *claude.BashCommandResult) string {
	if result == nil {
		return "(実行中)"
	}
	if result.Error != "" {
		return wordWrap(result.Error, m.wrapWidth())
	}
	if result.PersistedOutputPath != "" {
		if result.PersistedReadError != "" {
			return wordWrap(fmt.Sprintf("persistedOutputPath: %s\npersistedOutputSize: %d\nread error: %s", result.PersistedOutputPath, result.PersistedOutputSize, result.PersistedReadError), m.wrapWidth())
		}
		output := result.PersistedOutput
		if result.PersistedTruncated {
			output += "\n(truncated)"
		}
		return wordWrap(output, m.wrapWidth())
	}
	parts := make([]string, 0, 2)
	if result.Stdout != "" {
		parts = append(parts, "stdout:\n"+result.Stdout)
	}
	if result.Stderr != "" {
		parts = append(parts, "stderr:\n"+result.Stderr)
	}
	if len(parts) == 0 {
		return "(出力なし)"
	}
	return wordWrap(strings.Join(parts, "\n"), m.wrapWidth())
}

func (m CommandsViewModel) viewHeight() int {
	height := m.height - 2
	if height < 1 {
		return 1
	}
	return height
}

func (m CommandsViewModel) pageSize() int {
	pageSize := m.viewHeight()
	if pageSize < 1 {
		return 1
	}
	return pageSize
}

func (m CommandsViewModel) wrapWidth() int {
	if m.width <= 0 {
		return 80
	}
	return m.width
}

func (m CommandsViewModel) clampScroll() CommandsViewModel {
	viewHeight := m.viewHeight()
	if m.selected < m.scrollOffset {
		m.scrollOffset = m.selected
	}
	if m.selected >= m.scrollOffset+viewHeight {
		m.scrollOffset = m.selected - viewHeight + 1
	}
	maxOffset := len(m.commands) - viewHeight
	if maxOffset < 0 {
		maxOffset = 0
	}
	if m.scrollOffset > maxOffset {
		m.scrollOffset = maxOffset
	}
	if m.scrollOffset < 0 {
		m.scrollOffset = 0
	}
	return m
}

func (m CommandsViewModel) clampDetailScroll() CommandsViewModel {
	maxOffset := len(m.detailLines()) - m.viewHeight()
	if maxOffset < 0 {
		maxOffset = 0
	}
	if m.detailScroll > maxOffset {
		m.detailScroll = maxOffset
	}
	if m.detailScroll < 0 {
		m.detailScroll = 0
	}
	return m
}
