package claude

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// Task represents a Claude Code subagent task.
type Task struct {
	ID          string   `json:"id"`
	Subject     string   `json:"subject"`
	Description string   `json:"description"`
	ActiveForm  string   `json:"activeForm"`
	Status      string   `json:"status"`
	Blocks      []string `json:"blocks"`
	BlockedBy   []string `json:"blockedBy"`
}

// LoadTasks loads all task JSON files from the given directory,
// sorted by numeric ID.
func LoadTasks(dir string) ([]Task, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading tasks dir: %w", err)
	}

	tasks := make([]Task, 0)
	for _, entry := range entries {
		name := entry.Name()
		// Skip non-JSON files
		if !strings.HasSuffix(name, ".json") {
			continue
		}
		task, err := LoadTask(filepath.Join(dir, name))
		if err != nil {
			return nil, fmt.Errorf("loading task %s: %w", name, err)
		}
		tasks = append(tasks, task)
	}

	// Sort by numeric ID. If IDs are not valid integers (e.g., non-numeric strings),
	// fall back to lexicographic comparison so the order is still deterministic.
	sort.Slice(tasks, func(i, j int) bool {
		ni, errI := strconv.Atoi(tasks[i].ID)
		nj, errJ := strconv.Atoi(tasks[j].ID)
		if errI != nil || errJ != nil {
			// Fallback: compare as strings when numeric conversion fails
			return tasks[i].ID < tasks[j].ID
		}
		return ni < nj
	})

	return tasks, nil
}

// LoadTask loads a single task from a JSON file.
func LoadTask(path string) (Task, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Task{}, fmt.Errorf("reading task file: %w", err)
	}
	var task Task
	if err := json.Unmarshal(data, &task); err != nil {
		return Task{}, fmt.Errorf("parsing task JSON: %w", err)
	}
	return task, nil
}
