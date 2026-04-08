package claude

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"
)

const bashToolName = "Bash"
const persistedOutputReadLimit = 8 * 1024

type bashToolInput struct {
	Command     string `json:"command"`
	Description string `json:"description"`
}

// BashCommand represents a Bash tool_use entry in the parent conversation.
type BashCommand struct {
	ToolUseID   string
	Command     string
	Description string
	Timestamp   time.Time
	Result      *BashCommandResult
}

// BashCommandResult stores the parsed toolUseResult variants.
type BashCommandResult struct {
	Stdout              string
	Stderr              string
	Error               string
	PersistedOutputPath string
	PersistedOutputSize int64
	PersistedOutput     string
	PersistedTruncated  bool
	PersistedReadError  string
}

// BashCommandCache holds accumulated state for incremental Bash parsing.
type BashCommandCache struct {
	Commands       []BashCommand
	commandIndex   map[string]int
	pendingResults map[string]*BashCommandResult
	modTime        time.Time
}

// BashIncrementalParseResult holds the result of incremental Bash parsing.
type BashIncrementalParseResult struct {
	Cache   *BashCommandCache
	Offset  int64
	ModTime time.Time
}

// ExtractBashCommandsIncremental extracts Bash commands from the parent conversation incrementally.
func ExtractBashCommandsIncremental(parentPath string, offset int64, cache *BashCommandCache) (BashIncrementalParseResult, error) {
	f, err := os.Open(parentPath)
	if err != nil {
		return BashIncrementalParseResult{Cache: cache, Offset: offset}, fmt.Errorf("opening parent conversation: %w", err)
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return BashIncrementalParseResult{Cache: cache, Offset: offset}, fmt.Errorf("stat parent conversation: %w", err)
	}
	fileSize := fi.Size()
	fileMtime := fi.ModTime()

	if cache != nil && fileSize == offset && !fi.ModTime().Equal(cache.modTime) {
		cache = nil
		offset = 0
	}

	if fileSize < offset {
		cache = nil
		offset = 0
	}

	if offset > 0 {
		if _, err := f.Seek(offset, 0); err != nil {
			return BashIncrementalParseResult{Cache: cache, Offset: offset, ModTime: fileMtime}, fmt.Errorf("seeking parent conversation: %w", err)
		}
	}

	if cache == nil {
		cache = &BashCommandCache{
			commandIndex:   make(map[string]int),
			pendingResults: make(map[string]*BashCommandResult),
		}
	}

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)

	var parsedBytes int64
	for scanner.Scan() {
		lineBytes := scanner.Bytes()
		line := string(lineBytes)
		if line == "" {
			parsedBytes += int64(len(lineBytes)) + 1
			continue
		}

		var rl rawLine
		if err := json.Unmarshal([]byte(line), &rl); err != nil {
			continue
		}
		parsedBytes += int64(len(lineBytes)) + 1

		switch rl.Type {
		case "assistant":
			if err := cache.parseToolUse(rl); err != nil {
				continue
			}
		case "user":
			if err := cache.parseToolResult(rl); err != nil {
				continue
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return BashIncrementalParseResult{Cache: cache, Offset: offset, ModTime: fileMtime}, fmt.Errorf("reading parent conversation: %w", err)
	}

	newOffset := offset + parsedBytes
	if newOffset > fileSize {
		newOffset = fileSize
	}
	cache.modTime = fileMtime

	return BashIncrementalParseResult{Cache: cache, Offset: newOffset, ModTime: fileMtime}, nil
}

func (c *BashCommandCache) parseToolUse(rl rawLine) error {
	var msg rawMessage
	if err := json.Unmarshal(rl.Message, &msg); err != nil {
		return err
	}

	var rawBlocks []rawContentBlock
	if err := json.Unmarshal(msg.Content, &rawBlocks); err != nil {
		return err
	}

	for _, rb := range rawBlocks {
		if rb.Type != "tool_use" || rb.Name != bashToolName || rb.ID == "" || rb.Input == nil {
			continue
		}

		var input bashToolInput
		if err := json.Unmarshal(rb.Input, &input); err != nil {
			continue
		}
		if input.Command == "" {
			continue
		}

		command := BashCommand{
			ToolUseID:   rb.ID,
			Command:     input.Command,
			Description: input.Description,
			Timestamp:   parseBashTimestamp(rl.Timestamp),
		}
		if pending, ok := c.pendingResults[rb.ID]; ok {
			command.Result = pending
			delete(c.pendingResults, rb.ID)
		}

		c.commandIndex[rb.ID] = len(c.Commands)
		c.Commands = append(c.Commands, command)
	}

	return nil
}

func (c *BashCommandCache) parseToolResult(rl rawLine) error {
	var msg rawMessage
	if err := json.Unmarshal(rl.Message, &msg); err != nil {
		return err
	}

	var rawBlocks []rawContentBlock
	if err := json.Unmarshal(msg.Content, &rawBlocks); err != nil {
		return err
	}

	result, err := parseBashCommandResult(rl.ToolUseResult)
	if err != nil {
		return err
	}

	for _, rb := range rawBlocks {
		if rb.Type != "tool_result" || rb.ToolUseID == "" {
			continue
		}
		if idx, ok := c.commandIndex[rb.ToolUseID]; ok {
			c.Commands[idx].Result = result
			continue
		}
		c.pendingResults[rb.ToolUseID] = result
	}

	return nil
}

func parseBashCommandResult(raw json.RawMessage) (*BashCommandResult, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}

	var object struct {
		Stdout              string `json:"stdout"`
		Stderr              string `json:"stderr"`
		PersistedOutputPath string `json:"persistedOutputPath"`
		PersistedOutputSize int64  `json:"persistedOutputSize"`
	}
	if err := json.Unmarshal(raw, &object); err == nil {
		persistedOutput := ""
		persistedTruncated := false
		persistedReadError := ""
		if object.PersistedOutputPath != "" {
			output, truncated, err := readPersistedOutput(object.PersistedOutputPath, persistedOutputReadLimit)
			if err != nil {
				persistedReadError = err.Error()
			} else {
				persistedOutput = output
				persistedTruncated = truncated
			}
		}
		return &BashCommandResult{
			Stdout:              object.Stdout,
			Stderr:              object.Stderr,
			PersistedOutputPath: object.PersistedOutputPath,
			PersistedOutputSize: object.PersistedOutputSize,
			PersistedOutput:     persistedOutput,
			PersistedTruncated:  persistedTruncated,
			PersistedReadError:  persistedReadError,
		}, nil
	}

	var result string
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, err
	}
	return &BashCommandResult{Error: result}, nil
}

func parseBashTimestamp(value string) time.Time {
	if value == "" {
		return time.Time{}
	}
	timestamp, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}
	}
	return timestamp
}

func readPersistedOutput(path string, limit int64) (string, bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", false, fmt.Errorf("opening persisted output %q: %w", path, err)
	}
	defer f.Close()

	content, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil {
		return "", false, fmt.Errorf("reading persisted output %q: %w", path, err)
	}

	if int64(len(content)) > limit {
		return string(content[:limit]), true, nil
	}
	return string(content), false, nil
}
