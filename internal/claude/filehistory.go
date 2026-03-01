package claude

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
)

var fileHistoryPattern = regexp.MustCompile(`^([a-f0-9]+)@v(\d+)$`)

// FileVersion represents a single version of a file in file history.
type FileVersion struct {
	Hash    string
	Version int
	Path    string
	Size    int64
}

// FileGroup represents all versions of a file grouped by hash.
type FileGroup struct {
	Hash     string
	Versions []FileVersion
}

// LoadFileHistory loads file history from the given directory,
// grouping files by hash and sorting versions in ascending order.
func LoadFileHistory(dir string) ([]FileGroup, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading file history dir: %w", err)
	}

	groups := make(map[string][]FileVersion)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		matches := fileHistoryPattern.FindStringSubmatch(entry.Name())
		if matches == nil {
			continue
		}
		hash := matches[1]
		// matches[2] is guaranteed to be numeric by regex \d+
		version, _ := strconv.Atoi(matches[2])
		info, err := entry.Info()
		if err != nil {
			// Skip entries where metadata is unavailable (e.g., concurrent deletion)
			continue
		}
		groups[hash] = append(groups[hash], FileVersion{
			Hash:    hash,
			Version: version,
			Path:    filepath.Join(dir, entry.Name()),
			Size:    info.Size(),
		})
	}

	var result []FileGroup
	for hash, versions := range groups {
		sort.Slice(versions, func(i, j int) bool {
			return versions[i].Version < versions[j].Version
		})
		result = append(result, FileGroup{Hash: hash, Versions: versions})
	}
	// Sort groups by hash for deterministic output
	sort.Slice(result, func(i, j int) bool {
		return result[i].Hash < result[j].Hash
	})
	return result, nil
}
