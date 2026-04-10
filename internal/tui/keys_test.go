package tui

import (
	"testing"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

func TestLogKeys_FilterToggle_BindsEnterAndShiftEnter(t *testing.T) {
	keys := LogKeys.FilterToggle.Keys()
	wantKeys := map[string]bool{
		"enter":       false,
		"shift+enter": false,
	}

	for _, k := range keys {
		if _, ok := wantKeys[k]; ok {
			wantKeys[k] = true
		}
	}

	for k, found := range wantKeys {
		if !found {
			t.Errorf("LogKeys.FilterToggle should contain %q, got keys: %v", k, keys)
		}
	}
}

func TestConversationKeys_FilterToggle_BindsEnterAndShiftEnter(t *testing.T) {
	keys := ConversationKeys.FilterToggle.Keys()
	wantKeys := map[string]bool{
		"enter":       false,
		"shift+enter": false,
	}

	for _, k := range keys {
		if _, ok := wantKeys[k]; ok {
			wantKeys[k] = true
		}
	}

	for k, found := range wantKeys {
		if !found {
			t.Errorf("ConversationKeys.FilterToggle should contain %q, got keys: %v", k, keys)
		}
	}
}

func TestLogKeys_FilterToggle_MatchesEnterKeyMsg(t *testing.T) {
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	if !key.Matches(msg, LogKeys.FilterToggle) {
		t.Error("LogKeys.FilterToggle should match enter KeyMsg")
	}
}

func TestConversationKeys_FilterToggle_MatchesEnterKeyMsg(t *testing.T) {
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	if !key.Matches(msg, ConversationKeys.FilterToggle) {
		t.Error("ConversationKeys.FilterToggle should match enter KeyMsg")
	}
}

func TestLogKeys_FilterToggle_ExistingEnterBindingUnchanged(t *testing.T) {
	// Verify enter is still the first key (no accidental reordering)
	keys := LogKeys.FilterToggle.Keys()
	if len(keys) == 0 {
		t.Fatal("LogKeys.FilterToggle should have at least one key")
	}
	if keys[0] != "enter" {
		t.Errorf("LogKeys.FilterToggle first key should be %q, got %q", "enter", keys[0])
	}
}

func TestConversationKeys_FilterToggle_ExistingEnterBindingUnchanged(t *testing.T) {
	// Verify enter is still the first key (no accidental reordering)
	keys := ConversationKeys.FilterToggle.Keys()
	if len(keys) == 0 {
		t.Fatal("ConversationKeys.FilterToggle should have at least one key")
	}
	if keys[0] != "enter" {
		t.Errorf("ConversationKeys.FilterToggle first key should be %q, got %q", "enter", keys[0])
	}
}

func TestGlobalKeys_Tab5Binding(t *testing.T) {
	keys := GlobalKeys.Tab5.Keys()
	if len(keys) != 1 || keys[0] != "5" {
		t.Fatalf("GlobalKeys.Tab5 should be bound to 5, got %v", keys)
	}

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("5")}
	if !key.Matches(msg, GlobalKeys.Tab5) {
		t.Error("GlobalKeys.Tab5 should match key '5'")
	}
}
