package claude

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// SessionInfo represents a discovered Claude Code session.
type SessionInfo struct {
	SessionID      string
	Project        string
	Timestamp      int64
	FirstInput     string
	HasTasks       bool
	HasDebugLog    bool
	HasFileHistory bool
	Stats          *ProjectStats
}

// DiscoverSessions discovers sessions from the given base path (typically ~/.claude).
// configPath is the path to the global config file (typically ~/.claude.json).
// It reads history.jsonl, groups entries by sessionId, detects capabilities,
// and attaches project stats.
func DiscoverSessions(basePath, configPath string) ([]SessionInfo, error) {
	historyPath := filepath.Join(basePath, "history.jsonl")
	entries, err := ParseHistory(historyPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	if len(entries) == 0 {
		return nil, nil
	}

	// Group by sessionId
	type sessionData struct {
		sessionID string
		project   string
		timestamp int64
		inputs    []string
	}
	sessionMap := make(map[string]*sessionData)
	for _, e := range entries {
		sd, ok := sessionMap[e.SessionID]
		if !ok {
			sd = &sessionData{
				sessionID: e.SessionID,
				project:   e.Project,
				timestamp: e.Timestamp,
			}
			sessionMap[e.SessionID] = sd
		}
		// Use the latest timestamp
		if e.Timestamp > sd.timestamp {
			sd.timestamp = e.Timestamp
		}
		sd.inputs = append(sd.inputs, e.Display)
	}
	projectStats := make(map[string]*ProjectStats)
	if data, err := os.ReadFile(configPath); err != nil {
		if !os.IsNotExist(err) {
			log.Printf("warning: failed to read config file %s: %v", configPath, err)
		}
	} else {
		var cfg globalConfig
		if jsonErr := json.Unmarshal(data, &cfg); jsonErr != nil {
			log.Printf("warning: failed to parse config file %s: %v", configPath, jsonErr)
		} else {
			projectStats = cfg.Projects
		}
	}

	// Build SessionInfo list
	var sessions []SessionInfo
	for _, sd := range sessionMap {
		info := SessionInfo{
			SessionID: sd.sessionID,
			Project:   sd.project,
			Timestamp: sd.timestamp,
		}

		// Find first non-command, non-empty input
		for _, input := range sd.inputs {
			if input == "" || strings.HasPrefix(input, "/") {
				continue
			}
			info.FirstInput = input
			break
		}

		// Check capabilities
		info.HasTasks = hasTaskFiles(filepath.Join(basePath, "tasks", sd.sessionID))
		info.HasDebugLog = FileExists(filepath.Join(basePath, "debug", sd.sessionID+".txt"))
		info.HasFileHistory = HasDirFiles(filepath.Join(basePath, "file-history", sd.sessionID))

		// Attach stats for the project
		if stats, ok := projectStats[sd.project]; ok {
			info.Stats = stats
		}

		sessions = append(sessions, info)
	}

	// Sort by timestamp descending
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].Timestamp > sessions[j].Timestamp
	})

	return sessions, nil
}

func hasTaskFiles(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".json") {
			return true
		}
	}
	return false
}

// FileExists returns true if the given path exists.
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// HasDirFiles returns true if the given directory contains at least one entry.
func HasDirFiles(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	return len(entries) > 0
}

// BuildSessionInfo constructs a SessionInfo for a single session without
// scanning history.jsonl. It detects capabilities from the filesystem
// and attaches project stats from configPath (project is left empty).
func BuildSessionInfo(basePath, configPath, sessionID string) SessionInfo {
	info := SessionInfo{
		SessionID:      sessionID,
		HasTasks:       hasTaskFiles(filepath.Join(basePath, "tasks", sessionID)),
		HasDebugLog:    FileExists(filepath.Join(basePath, "debug", sessionID+".txt")),
		HasFileHistory: HasDirFiles(filepath.Join(basePath, "file-history", sessionID)),
	}

	// Try to load stats from config (project unknown, so iterate to find matching session)
	if data, err := os.ReadFile(configPath); err == nil {
		var cfg globalConfig
		if jsonErr := json.Unmarshal(data, &cfg); jsonErr == nil {
			for _, stats := range cfg.Projects {
				if stats.LastSessionID == sessionID {
					info.Stats = stats
					break
				}
			}
		}
	}

	return info
}
