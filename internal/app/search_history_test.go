package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// Submitted keywords are trimmed, moved to the front on reuse and bounded without keeping blank input.
func TestRememberSearch(t *testing.T) {
	a := newTestApp()
	a.rememberSearch("  春日影\n")
	a.rememberSearch("迷星叫")
	a.rememberSearch("春日影")
	a.rememberSearch(" \t\n")
	if want := []string{"春日影", "迷星叫"}; !slices.Equal(a.settings.SearchHistory, want) {
		t.Fatalf("history = %q, want %q", a.settings.SearchHistory, want)
	}
	for i := range searchHistoryLimit + 3 {
		a.rememberSearch(fmt.Sprintf("keyword-%d", i))
	}
	want := make([]string, searchHistoryLimit)
	for i := range want {
		want[i] = fmt.Sprintf("keyword-%d", searchHistoryLimit+2-i)
	}
	if !slices.Equal(a.settings.SearchHistory, want) {
		t.Fatalf("bounded history = %q, want %q", a.settings.SearchHistory, want)
	}
}

// Saving and clearing history survive reloading settings without changing the listener's other choices.
func TestSearchHistoryPersistence(t *testing.T) {
	a := newTestApp()
	a.settings = defaultSettings()
	a.settings.Quality = "lossless"
	a.settings.path = filepath.Join(t.TempDir(), "settings.json")
	a.rememberSearch("春日影")
	a.rememberSearch("迷星叫")
	stored := readSearchSettings(t, a.settings.path)
	if !slices.Equal(stored.SearchHistory, a.settings.SearchHistory) || stored.Quality != "lossless" {
		t.Fatalf("saved settings lost history or quality: %+v", stored)
	}
	a.clearSearchHistory()
	stored = readSearchSettings(t, a.settings.path)
	if len(stored.SearchHistory) != 0 || stored.Quality != "lossless" {
		t.Fatalf("cleared history came back or changed quality: %+v", stored)
	}
}

// A failed clear keeps history available to retry instead of falsely reporting that it was removed from disk.
func TestClearSearchHistoryFailure(t *testing.T) {
	a := newTestApp()
	a.settings.SearchHistory = []string{"春日影"}
	a.settings.path = t.TempDir() // A directory cannot be overwritten by settings.json.
	a.clearSearchHistory()
	if !slices.Equal(a.settings.SearchHistory, []string{"春日影"}) || a.toast == "" {
		t.Fatalf("failed clear discarded history or hid the failure: %q / %q", a.settings.SearchHistory, a.toast)
	}
}

func readSearchSettings(t *testing.T, path string) settings {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	s := defaultSettings()
	if err := json.Unmarshal(b, &s); err != nil {
		t.Fatal(err)
	}
	return s
}
