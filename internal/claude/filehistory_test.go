package claude

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadFileHistory_ValidDir(t *testing.T) {
	dir := filepath.Join("testdata", "file-history")
	groups, err := LoadFileHistory(dir)
	if err != nil {
		t.Fatalf("LoadFileHistory() error = %v", err)
	}
	if len(groups) != 2 {
		t.Errorf("LoadFileHistory() returned %d groups, want 2", len(groups))
	}
}

func TestLoadFileHistory_GroupByHash(t *testing.T) {
	dir := filepath.Join("testdata", "file-history")
	groups, err := LoadFileHistory(dir)
	if err != nil {
		t.Fatalf("LoadFileHistory() error = %v", err)
	}
	found := false
	for _, g := range groups {
		if g.Hash == "abcd1234abcd1234" {
			found = true
			if len(g.Versions) != 2 {
				t.Fatalf("abcd1234abcd1234 has %d versions, want 2", len(g.Versions))
			}
			if g.Versions[0].Version != 1 {
				t.Errorf("Versions[0].Version = %d, want 1", g.Versions[0].Version)
			}
			if g.Versions[1].Version != 2 {
				t.Errorf("Versions[1].Version = %d, want 2", g.Versions[1].Version)
			}
			if !strings.Contains(g.Versions[0].Path, "@v1") {
				t.Errorf("Versions[0].Path = %q, want to contain @v1", g.Versions[0].Path)
			}
			if !strings.Contains(g.Versions[1].Path, "@v2") {
				t.Errorf("Versions[1].Path = %q, want to contain @v2", g.Versions[1].Path)
			}
		}
	}
	if !found {
		t.Error("LoadFileHistory() missing group for abcd1234abcd1234")
	}
}

func TestLoadFileHistory_VersionSort(t *testing.T) {
	dir := filepath.Join("testdata", "file-history")
	groups, err := LoadFileHistory(dir)
	if err != nil {
		t.Fatalf("LoadFileHistory() error = %v", err)
	}
	for _, g := range groups {
		if g.Hash == "abcd1234abcd1234" {
			if len(g.Versions) < 2 {
				t.Fatal("expected at least 2 versions")
			}
			if g.Versions[0].Version >= g.Versions[1].Version {
				t.Errorf("versions not sorted: v%d >= v%d", g.Versions[0].Version, g.Versions[1].Version)
			}
		}
	}
}

func TestLoadFileHistory_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	groups, err := LoadFileHistory(dir)
	if err != nil {
		t.Fatalf("LoadFileHistory() error = %v", err)
	}
	if len(groups) != 0 {
		t.Errorf("LoadFileHistory() returned %d groups, want 0", len(groups))
	}
}

func TestLoadFileHistory_InvalidFileName(t *testing.T) {
	dir := t.TempDir()
	// Create files that don't match the pattern
	os.WriteFile(filepath.Join(dir, "README.md"), []byte("test"), 0644)
	os.WriteFile(filepath.Join(dir, "no-at-sign"), []byte("test"), 0644)

	groups, err := LoadFileHistory(dir)
	if err != nil {
		t.Fatalf("LoadFileHistory() error = %v", err)
	}
	if len(groups) != 0 {
		t.Errorf("LoadFileHistory() returned %d groups, want 0 (invalid files should be skipped)", len(groups))
	}
}
