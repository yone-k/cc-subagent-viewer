package watcher

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/yone/subagent-viewer/internal/claude"
)

const conversationPollInterval = 1 * time.Second

// ConversationWatcher polls subagent conversation files for changes.
type ConversationWatcher struct {
	dir         string
	program     *tea.Program
	sessionID   string
	findDirFunc func(string) (string, error)
	offsets     map[string]int64
	entries     map[string][]claude.ConversationEntry
	infos       map[string]*claude.SubagentInfo
}

// NewConversationWatcher creates a new ConversationWatcher.
// dir may be empty; in that case, findDirFunc is called each poll to discover it.
func NewConversationWatcher(dir string, sessionID string, program *tea.Program, findDirFunc func(string) (string, error)) *ConversationWatcher {
	return &ConversationWatcher{
		dir:         dir,
		program:     program,
		sessionID:   sessionID,
		findDirFunc: findDirFunc,
		offsets:     make(map[string]int64),
		entries:     make(map[string][]claude.ConversationEntry),
		infos:       make(map[string]*claude.SubagentInfo),
	}
}

// Start begins polling for conversation file changes.
func (cw *ConversationWatcher) Start(ctx context.Context) {
	// Try to discover dir if empty
	cw.tryDiscoverDir()

	// Initial scan
	cw.scan()

	ticker := time.NewTicker(conversationPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// If dir is still empty, try to discover it
			if cw.dir == "" {
				cw.tryDiscoverDir()
			}
			cw.poll()
		}
	}
}

func (cw *ConversationWatcher) tryDiscoverDir() {
	if cw.dir != "" || cw.findDirFunc == nil || cw.sessionID == "" {
		return
	}
	dir, err := cw.findDirFunc(cw.sessionID)
	if err == nil && dir != "" {
		cw.dir = dir
	}
}

// scan does the initial full scan of all agent files.
func (cw *ConversationWatcher) scan() {
	if cw.dir == "" {
		// Send empty discovery message
		cw.program.Send(SubagentsDiscoveredMsg{})
		return
	}

	agents, err := claude.DiscoverSubagents(cw.dir)
	if err != nil {
		cw.program.Send(SubagentsDiscoveredMsg{})
		return
	}

	cw.program.Send(SubagentsDiscoveredMsg{Agents: agents})

	// Load all conversations and track offsets
	for _, agent := range agents {
		entries, info, err := claude.ParseConversationFile(agent.FilePath)
		if err != nil {
			continue
		}

		// Track file offset (file size)
		fi, err := os.Stat(agent.FilePath)
		if err == nil {
			cw.offsets[agent.FilePath] = fi.Size()
		}

		cw.entries[agent.FilePath] = entries
		cw.infos[agent.FilePath] = info

		if len(entries) > 0 {
			cw.program.Send(ConversationUpdatedMsg{
				AgentID: agent.AgentID,
				Entries: entries,
				Info:    info,
			})
		}
	}
}

// poll checks for new files and new entries in existing files.
func (cw *ConversationWatcher) poll() {
	if cw.dir == "" {
		return
	}

	// Check for new agent files
	matches, err := filepath.Glob(filepath.Join(cw.dir, "agent-*.jsonl"))
	if err != nil {
		return
	}

	newFileFound := false
	for _, path := range matches {
		if _, exists := cw.offsets[path]; !exists {
			// New file found
			newFileFound = true
			entries, info, err := claude.ParseConversationFile(path)
			if err != nil {
				continue
			}

			fi, err := os.Stat(path)
			if err == nil {
				cw.offsets[path] = fi.Size()
			}

			cw.entries[path] = entries
			cw.infos[path] = info

			if info != nil && len(entries) > 0 {
				cw.program.Send(ConversationUpdatedMsg{
					AgentID: info.AgentID,
					Entries: entries,
					Info:    info,
				})
			}
		}
	}

	if newFileFound {
		// Re-discover agents to update the list
		agents, err := claude.DiscoverSubagents(cw.dir)
		if err == nil {
			cw.program.Send(SubagentsDiscoveredMsg{Agents: agents})
		}
	}

	// Check existing files for new content
	for path, prevOffset := range cw.offsets {
		fi, err := os.Stat(path)
		if err != nil {
			continue
		}
		currentSize := fi.Size()
		if currentSize <= prevOffset {
			continue
		}

		// Read new lines from offset
		newEntries, info := cw.readNewEntries(path, prevOffset)
		if len(newEntries) == 0 {
			// Update offset even if no valid entries parsed
			cw.offsets[path] = currentSize
			continue
		}

		cw.offsets[path] = currentSize

		// Append to accumulated entries
		cw.entries[path] = append(cw.entries[path], newEntries...)
		if info != nil {
			if existing := cw.infos[path]; existing != nil {
				existing.EntryCount = len(cw.entries[path])
			} else {
				cw.infos[path] = info
			}
		}

		agentID := ""
		agentInfo := cw.infos[path]
		if agentInfo != nil {
			agentID = agentInfo.AgentID
			agentInfo.EntryCount = len(cw.entries[path])
		}

		// Send full snapshot
		cw.program.Send(ConversationUpdatedMsg{
			AgentID: agentID,
			Entries: cw.entries[path],
			Info:    agentInfo,
		})
	}
}

// readNewEntries reads new JSONL lines from the given offset.
func (cw *ConversationWatcher) readNewEntries(path string, offset int64) ([]claude.ConversationEntry, *claude.SubagentInfo) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil
	}
	defer f.Close()

	if _, err := f.Seek(offset, 0); err != nil {
		return nil, nil
	}

	var entries []claude.ConversationEntry
	var info *claude.SubagentInfo

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var rl struct {
			Type    string          `json:"type"`
			Message json.RawMessage `json:"message"`
			AgentID string          `json:"agentId"`
			Slug    string          `json:"slug"`
		}
		if err := json.Unmarshal([]byte(line), &rl); err != nil {
			continue
		}

		if rl.Type != "user" && rl.Type != "assistant" {
			continue
		}

		var msg struct {
			Content json.RawMessage `json:"content"`
		}
		if err := json.Unmarshal(rl.Message, &msg); err != nil {
			continue
		}

		blocks := claude.ParseContentBlocks(msg.Content)

		entry := claude.ConversationEntry{
			Type:    claude.ConversationEntryType(rl.Type),
			Content: blocks,
		}
		entries = append(entries, entry)

		if info == nil {
			info = &claude.SubagentInfo{
				AgentID: rl.AgentID,
				Slug:    rl.Slug,
			}
		}
	}

	return entries, info
}


