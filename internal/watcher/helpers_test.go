package watcher

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// msgCollector collects tea.Msg values sent via a tea.Program for test assertions.
type msgCollector struct {
	msgs chan tea.Msg
}

func newMsgCollector() *msgCollector {
	return &msgCollector{msgs: make(chan tea.Msg, 100)}
}

func (mc *msgCollector) waitForMsg(timeout time.Duration) (tea.Msg, bool) {
	select {
	case msg := <-mc.msgs:
		return msg, true
	case <-time.After(timeout):
		return nil, false
	}
}

// testModel is a simple bubbletea model that forwards all messages to a msgCollector.
type testModel struct {
	collector *msgCollector
}

func (m testModel) Init() tea.Cmd { return nil }
func (m testModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	m.collector.msgs <- msg
	return m, nil
}
func (m testModel) View() string { return "" }

func newTestProgram(collector *msgCollector) *tea.Program {
	p := tea.NewProgram(
		testModel{collector: collector},
		tea.WithoutRenderer(),
		tea.WithInput(strings.NewReader("")),
	)
	go p.Run()
	time.Sleep(50 * time.Millisecond) // Let program start
	return p
}
