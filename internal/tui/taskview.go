package tui

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/yone/subagent-viewer/internal/claude"
	"github.com/yone/subagent-viewer/internal/watcher"
)

const maxProgressBarWidth = 40

// TaskViewMode represents the current view mode within the Tasks tab.
type TaskViewMode int

const (
	TaskViewModeTasks        TaskViewMode = iota // Task list (default)
	TaskViewModeAgents                           // Subagent list
	TaskViewModeConversation                     // Conversation view
)

// TaskViewModel manages the Tasks tab view.
type TaskViewModel struct {
	tasks      []claude.Task
	selected   int
	showDetail bool
	width      int
	height     int

	// View mode
	mode TaskViewMode

	// Agent list
	agents        []claude.SubagentInfo
	agentSelected int

	// Conversation view
	conversations    map[string][]claude.ConversationEntry
	conversationInfo map[string]*claude.SubagentInfo
	conversationScroll int
	currentAgentID   string
}

// NewTaskViewModel creates a new TaskViewModel.
func NewTaskViewModel() TaskViewModel {
	return TaskViewModel{
		conversations:    make(map[string][]claude.ConversationEntry),
		conversationInfo: make(map[string]*claude.SubagentInfo),
	}
}

// SetSize uses a pointer receiver because app.go calls it through a pointer to AppModel's field.
func (m *TaskViewModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// Mode returns the current view mode.
func (m TaskViewModel) Mode() TaskViewMode {
	return m.mode
}

// Init initializes the model.
func (m TaskViewModel) Init() tea.Cmd {
	return nil
}

// Update handles messages.
func (m TaskViewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case watcher.TasksUpdatedMsg:
		m.tasks = msg.Tasks
		if m.selected >= len(m.tasks) {
			m.selected = 0
		}
	case watcher.TaskChangedMsg:
		found := false
		for i, task := range m.tasks {
			if task.ID == msg.Task.ID {
				m.tasks[i] = msg.Task
				found = true
				break
			}
		}
		if !found {
			m.tasks = append(m.tasks, msg.Task)
			sort.Slice(m.tasks, func(i, j int) bool {
				ni, _ := strconv.Atoi(m.tasks[i].ID)
				nj, _ := strconv.Atoi(m.tasks[j].ID)
				return ni < nj
			})
		}
	case watcher.SubagentsDiscoveredMsg:
		m.agents = msg.Agents
		if m.agentSelected >= len(m.agents) {
			m.agentSelected = 0
		}
	case watcher.ConversationUpdatedMsg:
		m.conversations[msg.AgentID] = msg.Entries
		if msg.Info != nil {
			m.conversationInfo[msg.AgentID] = msg.Info
		}
	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m TaskViewModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.mode {
	case TaskViewModeTasks:
		switch {
		case key.Matches(msg, TaskKeys.ShowAgents):
			m.mode = TaskViewModeAgents
			return m, nil
		}
		// Default task list key handling
		switch msg.String() {
		case "up", "k":
			if m.selected > 0 {
				m.selected--
			}
		case "down", "j":
			if m.selected < len(m.tasks)-1 {
				m.selected++
			}
		case "enter":
			m.showDetail = !m.showDetail
		}

	case TaskViewModeAgents:
		switch {
		case key.Matches(msg, TaskKeys.Escape):
			m.mode = TaskViewModeTasks
			return m, nil
		}
		switch msg.String() {
		case "up", "k":
			if m.agentSelected > 0 {
				m.agentSelected--
			}
		case "down", "j":
			if m.agentSelected < len(m.agents)-1 {
				m.agentSelected++
			}
		case "enter":
			if len(m.agents) > 0 && m.agentSelected < len(m.agents) {
				m.currentAgentID = m.agents[m.agentSelected].AgentID
				m.conversationScroll = 0
				m.mode = TaskViewModeConversation
			}
		}

	case TaskViewModeConversation:
		switch {
		case key.Matches(msg, TaskKeys.Escape):
			m.mode = TaskViewModeAgents
			return m, nil
		}
		entries := m.conversations[m.currentAgentID]
		switch msg.String() {
		case "up", "k":
			if m.conversationScroll > 0 {
				m.conversationScroll--
			}
		case "down", "j":
			if m.conversationScroll < len(entries)-1 {
				m.conversationScroll++
			}
		}
	}
	return m, nil
}

// View renders the task list.
func (m TaskViewModel) View() string {
	switch m.mode {
	case TaskViewModeAgents:
		return m.viewAgents()
	case TaskViewModeConversation:
		return m.viewConversation()
	default:
		return m.viewTasks()
	}
}

func (m TaskViewModel) viewTasks() string {
	if len(m.tasks) == 0 {
		return EmptyStateStyle.Render("サブエージェントのタスクなし")
	}

	var b strings.Builder

	// Progress bar
	completed := 0
	for _, t := range m.tasks {
		if t.Status == "completed" {
			completed++
		}
	}
	total := len(m.tasks)
	b.WriteString(renderProgressBar(completed, total, m.width-4))
	b.WriteString(fmt.Sprintf("  %d/%d\n\n", completed, total))

	// Task list
	for i, task := range m.tasks {
		icon := statusIcon(task)
		prefix := "  "
		if i == m.selected {
			prefix = "> "
		}

		line := fmt.Sprintf("%s%s %s", prefix, icon, task.Subject)

		if len(task.BlockedBy) > 0 {
			refs := make([]string, len(task.BlockedBy))
			for j, id := range task.BlockedBy {
				refs[j] = "#" + id
			}
			line += DimStyle.Render(fmt.Sprintf(" (blocked by %s)", strings.Join(refs, ", ")))
		}

		if task.Status == "in_progress" && task.ActiveForm != "" {
			line += DimStyle.Render(fmt.Sprintf(" — %s", task.ActiveForm))
		}

		b.WriteString(line + "\n")

		// Show detail for selected task
		if i == m.selected && m.showDetail && task.Description != "" {
			b.WriteString(BorderStyle.Render(task.Description))
			b.WriteString("\n")
		}
	}

	return b.String()
}

func (m TaskViewModel) viewAgents() string {
	if len(m.agents) == 0 {
		return EmptyStateStyle.Render("サブエージェントなし")
	}

	var b strings.Builder
	b.WriteString(TitleStyle.Render("サブエージェント一覧"))
	b.WriteString("\n\n")

	for i, agent := range m.agents {
		prefix := "  "
		if i == m.agentSelected {
			prefix = "> "
		}

		slug := agent.Slug
		if slug == "" {
			slug = agent.AgentID
		}

		prompt := agent.Prompt
		if len([]rune(prompt)) > 60 {
			prompt = string([]rune(prompt)[:60]) + "..."
		}

		line := fmt.Sprintf("%s%s", prefix, ConversationAssistantStyle.Render(slug))
		line += DimStyle.Render(fmt.Sprintf("  %s", prompt))
		line += HelpStyle.Render(fmt.Sprintf("  (%d entries)", agent.EntryCount))

		b.WriteString(line + "\n")
	}

	return b.String()
}

func (m TaskViewModel) viewConversation() string {
	entries := m.conversations[m.currentAgentID]
	if len(entries) == 0 {
		return EmptyStateStyle.Render("会話データなし")
	}

	var b strings.Builder

	// Header with agent info
	info := m.conversationInfo[m.currentAgentID]
	if info != nil {
		slug := info.Slug
		if slug == "" {
			slug = info.AgentID
		}
		b.WriteString(TitleStyle.Render(fmt.Sprintf("会話: %s", slug)))
	} else {
		b.WriteString(TitleStyle.Render(fmt.Sprintf("会話: %s", m.currentAgentID)))
	}
	b.WriteString(HelpStyle.Render(fmt.Sprintf("  %d/%d", m.conversationScroll+1, len(entries))))
	b.WriteString("\n\n")

	// Calculate visible range
	visibleLines := m.height - 6
	if visibleLines < 1 {
		visibleLines = 10
	}

	start := m.conversationScroll
	if start >= len(entries) {
		start = len(entries) - 1
	}
	if start < 0 {
		start = 0
	}

	// Render entries from scroll position
	linesRendered := 0
	for i := start; i < len(entries) && linesRendered < visibleLines; i++ {
		entry := entries[i]
		rendered := m.renderEntry(entry)
		lines := strings.Count(rendered, "\n") + 1
		b.WriteString(rendered)
		b.WriteString("\n")
		linesRendered += lines
	}

	return b.String()
}

func (m TaskViewModel) renderEntry(entry claude.ConversationEntry) string {
	var b strings.Builder

	switch entry.Type {
	case claude.EntryTypeUser:
		b.WriteString(ConversationUserStyle.Render("[USER]"))
		b.WriteString(" ")
		for _, block := range entry.Content {
			switch block.Type {
			case "text":
				b.WriteString(block.Text)
			case "tool_result":
				text := block.Text
				if len([]rune(text)) > 100 {
					text = string([]rune(text)[:100]) + "..."
				}
				b.WriteString(ConversationToolStyle.Render(fmt.Sprintf("[TOOL_RESULT] %s", text)))
			}
		}

	case claude.EntryTypeAssistant:
		b.WriteString(ConversationAssistantStyle.Render("[ASSISTANT]"))
		b.WriteString(" ")
		for _, block := range entry.Content {
			switch block.Type {
			case "text":
				b.WriteString(block.Text)
			case "tool_use":
				input := block.ToolInput
				if len([]rune(input)) > 60 {
					input = string([]rune(input)[:60]) + "..."
				}
				b.WriteString(ConversationToolStyle.Render(fmt.Sprintf("[TOOL] %s %s", block.ToolName, input)))
			case "thinking":
				text := block.Text
				if len([]rune(text)) > 100 {
					text = string([]rune(text)[:100]) + "..."
				}
				b.WriteString(ConversationThinkingStyle.Render(fmt.Sprintf("[thinking] %s", text)))
			}
		}
	}

	return b.String()
}

func statusIcon(task claude.Task) string {
	if len(task.BlockedBy) > 0 && task.Status != "completed" {
		return StatusBlocked.String()
	}
	switch task.Status {
	case "completed":
		return StatusCompleted.String()
	case "in_progress":
		return StatusInProgress.String()
	default:
		return StatusPending.String()
	}
}

func renderProgressBar(completed, total, width int) string {
	if total == 0 || width <= 0 {
		return ""
	}
	barWidth := width
	if barWidth > maxProgressBarWidth {
		barWidth = maxProgressBarWidth
	}
	filled := barWidth * completed / total
	empty := barWidth - filled

	bar := ProgressBarFilled.Render(strings.Repeat("█", filled))
	bar += ProgressBarEmpty.Render(strings.Repeat("░", empty))
	return bar
}
