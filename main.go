package main

import (
	"fmt"
	"os"
	"regexp"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/yone/subagent-viewer/internal/claude"
	"github.com/yone/subagent-viewer/internal/tui"
)

type runMode int

const (
	modeSelector runMode = iota
	modeViewer
	modeError
)

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

func parseArgs(args []string) (runMode, string) {
	if len(args) < 2 {
		return modeSelector, ""
	}

	sessionID := args[1]
	if !uuidPattern.MatchString(sessionID) {
		return modeError, sessionID
	}

	return modeViewer, sessionID
}

func main() {
	mode, sessionID := parseArgs(os.Args)

	switch mode {
	case modeError:
		fmt.Fprintf(os.Stderr, "無効なセッションID: %s\nUUID形式で指定してください（例: 7ba50137-65c8-4349-b420-cdce14c38d2a）\n", sessionID)
		os.Exit(1)

	case modeViewer:
		// Direct session mode
		session := claude.BuildSessionInfo(claude.ClaudeDir(), claude.GlobalConfigPath(), sessionID)

		model := tui.NewAppModelWithSession(session)
		p := tea.NewProgram(&model, tea.WithAltScreen())
		model.SetProgram(p)
		defer model.Cleanup()

		if _, err := p.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "エラー: %v\n", err)
			os.Exit(1)
		}

	case modeSelector:
		// Selector mode - discover all sessions
		sessions, err := claude.DiscoverSessions(claude.ClaudeDir(), claude.GlobalConfigPath())
		if err != nil {
			fmt.Fprintf(os.Stderr, "セッション一覧の取得に失敗: %v\n", err)
			os.Exit(1)
		}

		model := tui.NewAppModel(sessions)
		p := tea.NewProgram(&model, tea.WithAltScreen())
		model.SetProgram(p)
		defer model.Cleanup()

		if _, err := p.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "エラー: %v\n", err)
			os.Exit(1)
		}
	}
}
