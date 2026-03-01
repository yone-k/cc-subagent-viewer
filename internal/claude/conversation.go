package claude

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// ConversationEntryType represents the type of a conversation entry.
type ConversationEntryType string

const (
	EntryTypeUser      ConversationEntryType = "user"
	EntryTypeAssistant ConversationEntryType = "assistant"
)

// ContentBlock represents a single block within a conversation message.
type ContentBlock struct {
	Type      string // "text", "tool_use", "tool_result", "thinking"
	Text      string
	ToolName  string
	ToolInput string
}

// ConversationEntry represents a single parsed conversation entry.
type ConversationEntry struct {
	Type    ConversationEntryType
	Content []ContentBlock
}

// SubagentInfo holds metadata about a discovered subagent.
type SubagentInfo struct {
	AgentID    string
	Slug       string
	Prompt     string // first user message, truncated
	EntryCount int
	FilePath   string
}

// rawLine represents the top-level JSON structure of a conversation JSONL line.
type rawLine struct {
	Type    string          `json:"type"`
	Message json.RawMessage `json:"message"`
	AgentID string          `json:"agentId"`
	Slug    string          `json:"slug"`
}

// rawMessage represents the message field within a conversation line.
type rawMessage struct {
	Content json.RawMessage `json:"content"`
}

// rawContentBlock represents a single content block within a message content array.
type rawContentBlock struct {
	Type     string          `json:"type"`
	Text     string          `json:"text"`
	Thinking string          `json:"thinking"`
	Name     string          `json:"name"`
	Input    json.RawMessage `json:"input"`
	Content  json.RawMessage `json:"content"`
}

// ParseConversationFile parses a JSONL conversation file and returns entries and subagent info.
func ParseConversationFile(path string) ([]ConversationEntry, *SubagentInfo, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, fmt.Errorf("opening conversation file: %w", err)
	}
	defer f.Close()

	var entries []ConversationEntry
	var info *SubagentInfo
	firstUserPrompt := ""
	firstLineParsed := false

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var rl rawLine
		if err := json.Unmarshal([]byte(line), &rl); err != nil {
			continue
		}

		// Skip non-user/assistant types (e.g. "progress")
		if rl.Type != "user" && rl.Type != "assistant" {
			continue
		}

		// Extract agentId and slug from first valid line
		if !firstLineParsed {
			info = &SubagentInfo{
				AgentID:  rl.AgentID,
				Slug:     rl.Slug,
				FilePath: path,
			}
			firstLineParsed = true
		}

		// Parse the message
		var msg rawMessage
		if err := json.Unmarshal(rl.Message, &msg); err != nil {
			continue
		}

		// Parse content blocks (polymorphic: string or array)
		blocks := ParseContentBlocks(msg.Content)

		// Capture first user prompt
		if firstUserPrompt == "" && rl.Type == "user" && len(blocks) > 0 {
			for _, b := range blocks {
				if b.Text != "" {
					firstUserPrompt = b.Text
					break
				}
			}
		}

		entry := ConversationEntry{
			Type:    ConversationEntryType(rl.Type),
			Content: blocks,
		}
		entries = append(entries, entry)
	}
	if err := scanner.Err(); err != nil {
		return nil, nil, fmt.Errorf("reading conversation file: %w", err)
	}

	// Finalize SubagentInfo
	if info != nil {
		info.EntryCount = len(entries)
		info.Prompt = truncateString(firstUserPrompt, 60)
	}

	return entries, info, nil
}

// ParseContentBlocks parses polymorphic content: either a JSON string or an array of content blocks.
func ParseContentBlocks(raw json.RawMessage) []ContentBlock {
	// Try string first
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return []ContentBlock{{Type: "text", Text: s}}
	}

	// Try array of content blocks
	var rawBlocks []rawContentBlock
	if err := json.Unmarshal(raw, &rawBlocks); err != nil {
		return nil
	}

	var blocks []ContentBlock
	for _, rb := range rawBlocks {
		switch rb.Type {
		case "text":
			blocks = append(blocks, ContentBlock{Type: "text", Text: rb.Text})
		case "tool_use":
			toolInput := ""
			if rb.Input != nil {
				inputBytes, err := json.Marshal(json.RawMessage(rb.Input))
				if err == nil {
					toolInput = string(inputBytes)
				}
			}
			blocks = append(blocks, ContentBlock{
				Type:      "tool_use",
				ToolName:  rb.Name,
				ToolInput: toolInput,
			})
		case "tool_result":
			text := parseToolResultContent(rb.Content)
			blocks = append(blocks, ContentBlock{Type: "tool_result", Text: text})
		case "thinking":
			blocks = append(blocks, ContentBlock{Type: "thinking", Text: rb.Thinking})
		}
	}
	return blocks
}

// parseToolResultContent converts tool_result content to a string.
// Content can be a plain string or an array of objects.
func parseToolResultContent(raw json.RawMessage) string {
	if raw == nil {
		return ""
	}
	// Try string
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	// Fallback: stringify the raw JSON
	return string(raw)
}

// truncateString truncates a string to maxLen characters.
func truncateString(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen])
}

// DiscoverSubagents scans a directory for agent-*.jsonl files and returns info about each.
func DiscoverSubagents(subagentsDir string) ([]SubagentInfo, error) {
	matches, err := filepath.Glob(filepath.Join(subagentsDir, "agent-*.jsonl"))
	if err != nil {
		return nil, fmt.Errorf("globbing subagent files: %w", err)
	}

	var agents []SubagentInfo
	for _, path := range matches {
		_, info, err := ParseConversationFile(path)
		if err != nil {
			continue // skip broken files
		}
		if info != nil {
			agents = append(agents, *info)
		}
	}
	return agents, nil
}
