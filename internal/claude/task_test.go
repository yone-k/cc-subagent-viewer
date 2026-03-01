package claude

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadTasks_ValidTasks(t *testing.T) {
	dir := filepath.Join("testdata", "tasks")
	tasks, err := LoadTasks(dir)
	if err != nil {
		t.Fatalf("LoadTasks() error = %v", err)
	}
	if len(tasks) != 2 {
		t.Fatalf("LoadTasks() returned %d tasks, want 2", len(tasks))
	}
	// Should be sorted by numeric ID
	if tasks[0].ID != "1" {
		t.Errorf("tasks[0].ID = %q, want \"1\"", tasks[0].ID)
	}
	if tasks[1].ID != "2" {
		t.Errorf("tasks[1].ID = %q, want \"2\"", tasks[1].ID)
	}
}

func TestLoadTasks_SkipsLockAndHighwatermark(t *testing.T) {
	dir := t.TempDir()
	// Create .lock, .highwatermark, and a valid .json file explicitly
	if err := os.WriteFile(filepath.Join(dir, ".lock"), []byte{}, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".highwatermark"), []byte("2"), 0644); err != nil {
		t.Fatal(err)
	}
	validJSON := `{"id":"1","subject":"Test","status":"pending","blocks":[],"blockedBy":[]}`
	if err := os.WriteFile(filepath.Join(dir, "1.json"), []byte(validJSON), 0644); err != nil {
		t.Fatal(err)
	}

	tasks, err := LoadTasks(dir)
	if err != nil {
		t.Fatalf("LoadTasks() error = %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("LoadTasks() returned %d tasks, want 1", len(tasks))
	}
	if tasks[0].ID != "1" {
		t.Errorf("tasks[0].ID = %q, want \"1\"", tasks[0].ID)
	}
}

func TestLoadTasks_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	tasks, err := LoadTasks(dir)
	if err != nil {
		t.Fatalf("LoadTasks() error = %v", err)
	}
	if len(tasks) != 0 {
		t.Errorf("LoadTasks() returned %d tasks for empty dir, want 0", len(tasks))
	}
}

func TestLoadTasks_NonExistentDir(t *testing.T) {
	_, err := LoadTasks("/nonexistent/path")
	if err == nil {
		t.Error("LoadTasks() expected error for non-existent dir, got nil")
	}
}

func TestLoadTask_Single(t *testing.T) {
	path := filepath.Join("testdata", "tasks", "1.json")
	task, err := LoadTask(path)
	if err != nil {
		t.Fatalf("LoadTask() error = %v", err)
	}
	if task.ID != "1" {
		t.Errorf("task.ID = %q, want \"1\"", task.ID)
	}
	if task.Subject != "プロジェクト分析" {
		t.Errorf("task.Subject = %q, want \"プロジェクト分析\"", task.Subject)
	}
	if task.Description != "プロジェクトのコードベースを分析し、テスト・リント設定を確認する" {
		t.Errorf("task.Description = %q", task.Description)
	}
	if task.ActiveForm != "分析中" {
		t.Errorf("task.ActiveForm = %q, want \"分析中\"", task.ActiveForm)
	}
	if task.Status != "completed" {
		t.Errorf("task.Status = %q, want \"completed\"", task.Status)
	}
	if len(task.Blocks) != 2 || task.Blocks[0] != "2" || task.Blocks[1] != "3" {
		t.Errorf("task.Blocks = %v, want [\"2\", \"3\"]", task.Blocks)
	}
	if len(task.BlockedBy) != 0 {
		t.Errorf("task.BlockedBy = %v, want []", task.BlockedBy)
	}
}

func TestLoadTasks_NonNumericIDs(t *testing.T) {
	dir := t.TempDir()
	// Mix of numeric and non-numeric IDs
	for _, id := range []string{"3", "abc", "1", "zzz"} {
		content := `{"id":"` + id + `","subject":"Task ` + id + `","status":"pending","blocks":[],"blockedBy":[]}`
		if err := os.WriteFile(filepath.Join(dir, id+".json"), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	tasks, err := LoadTasks(dir)
	if err != nil {
		t.Fatalf("LoadTasks() error = %v", err)
	}
	if len(tasks) != 4 {
		t.Fatalf("LoadTasks() returned %d tasks, want 4", len(tasks))
	}
	// When non-numeric IDs are present, fallback to lexicographic order
	// Expected order: "1", "3", "abc", "zzz"
	expectedOrder := []string{"1", "3", "abc", "zzz"}
	for i, want := range expectedOrder {
		if tasks[i].ID != want {
			t.Errorf("tasks[%d].ID = %q, want %q", i, tasks[i].ID, want)
		}
	}
}

func TestLoadTask_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(path, []byte("{invalid json content"), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := LoadTask(path)
	if err == nil {
		t.Error("LoadTask() expected error for invalid JSON, got nil")
	}
}

func TestLoadTasks_NonSequentialIDs(t *testing.T) {
	dir := t.TempDir()
	// Create non-sequential task files
	for _, id := range []string{"5", "9", "7"} {
		content := `{"id":"` + id + `","subject":"Task ` + id + `","status":"pending","blocks":[],"blockedBy":[]}`
		if err := os.WriteFile(filepath.Join(dir, id+".json"), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	tasks, err := LoadTasks(dir)
	if err != nil {
		t.Fatalf("LoadTasks() error = %v", err)
	}
	if len(tasks) != 3 {
		t.Fatalf("LoadTasks() returned %d tasks, want 3", len(tasks))
	}
	// Should be sorted numerically: 5, 7, 9
	if tasks[0].ID != "5" {
		t.Errorf("tasks[0].ID = %q, want \"5\"", tasks[0].ID)
	}
	if tasks[1].ID != "7" {
		t.Errorf("tasks[1].ID = %q, want \"7\"", tasks[1].ID)
	}
	if tasks[2].ID != "9" {
		t.Errorf("tasks[2].ID = %q, want \"9\"", tasks[2].ID)
	}
}
