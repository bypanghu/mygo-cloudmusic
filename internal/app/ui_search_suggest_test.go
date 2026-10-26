package app

import (
	"path/filepath"
	"slices"
	"testing"

	"github.com/egoist/mygo/ui"

	"netease-music-mygo/internal/netease"
)

// Clicking or choosing a previous keyword with the arrows searches it again; rekeying a built combobox item lost its click.
func TestSearchHistoryChoice(t *testing.T) {
	for _, method := range []string{"click", "keyboard"} {
		t.Run(method, func(t *testing.T) {
			a := newTestApp()
			a.settings.SearchHistory = []string{"春日影", "迷星叫"}
			a.searches.last.kind = netease.SearchArtists
			want := cachedSearch(a, "迷星叫", netease.SearchArtists, netease.SearchResult{})
			tt := ui.NewTester(a.view, 1200, 800)
			if err := tt.Click("搜索"); err != nil {
				t.Fatal(err)
			}
			if method == "click" {
				if err := tt.Click("搜索历史：迷星叫"); err != nil {
					t.Fatal(err)
				}
			} else {
				tt.Key(0, ui.KeyDown)
				tt.Key(0, ui.KeyDown)
				tt.Key(0, ui.KeyEnter)
			}
			for range 3 {
				tt.Frame()
			}
			if a.route != want || a.query != want.query {
				t.Fatalf("history choice lost keyword or category: %+v / %q", a.route, a.query)
			}
			if !slices.Equal(a.settings.SearchHistory, []string{"迷星叫", "春日影"}) {
				t.Fatalf("reused keyword was not moved to the front: %q", a.settings.SearchHistory)
			}
		})
	}
}

// Escape keeps history dismissed across rebuilds, while Down can reopen it without making a network request.
func TestSearchHistoryDismiss(t *testing.T) {
	a := newTestApp()
	a.settings.SearchHistory = []string{"春日影"}
	want := cachedSearch(a, "春日影", netease.SearchSongs, netease.SearchResult{})
	tt := ui.NewTester(a.view, 1200, 800)
	if err := tt.Click("搜索"); err != nil {
		t.Fatal(err)
	}
	tt.Key(0, ui.KeyEscape)
	for range 3 {
		tt.Frame()
	}
	tt.Key(0, ui.KeyEnter)
	if a.route.page != pageHome || a.query != "" || a.searches.suggestions.query != "" {
		t.Fatalf("dismissed history submitted or fetched suggestions: %+v / %q", a.route, a.query)
	}
	tt.Key(0, ui.KeyDown)
	tt.Key(0, ui.KeyEnter)
	if a.route != want {
		t.Fatalf("Down did not reopen history: %+v", a.route)
	}
}

// Clearing history removes it from disk and future choices without disturbing results, navigation or the queue.
func TestClearSearchHistoryClick(t *testing.T) {
	a := newTestApp()
	a.settings = defaultSettings()
	a.settings.path = filepath.Join(t.TempDir(), "settings.json")
	a.rememberSearch("春日影")
	a.rememberSearch("迷星叫")
	first := cachedSearch(a, "春日影", netease.SearchSongs, netease.SearchResult{Songs: []netease.Track{{ID: 7}}})
	second := cachedSearch(a, "迷星叫", netease.SearchSongs, netease.SearchResult{})
	a.open(first)
	a.open(second)
	a.back()
	a.query = ""
	a.pl.Cue([]netease.Track{{ID: 99, Name: "原来的歌"}})
	history, forward := slices.Clone(a.history), slices.Clone(a.forward)
	entry := a.searchEntry()
	tt := ui.NewTester(a.view, 1200, 800)
	if err := tt.Click("搜索"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("清空搜索历史"); err != nil {
		t.Fatal(err)
	}
	for range 3 {
		tt.Frame()
	}
	tt.Key(0, ui.KeyDown)
	tt.Key(0, ui.KeyEnter)
	if len(a.settings.SearchHistory) != 0 || len(readSearchSettings(t, a.settings.path).SearchHistory) != 0 {
		t.Fatal("cleared history remains in memory or on disk")
	}
	if a.route != first || a.query != "" || a.searchEntry() != entry {
		t.Fatalf("clear changed results or selected stale history: %+v / %q", a.route, a.query)
	}
	if !slices.Equal(a.history, history) || !slices.Equal(a.forward, forward) || a.searches.last != first {
		t.Fatal("clear discarded navigation")
	}
	if queue := a.pl.Queue(); len(queue) != 1 || queue[0].ID != 99 {
		t.Fatalf("clear changed playback: %+v", queue)
	}
	tt.Key(0, ui.KeyForward)
	if a.route != second || len(a.settings.SearchHistory) != 0 {
		t.Fatal("forward failed or restored cleared keywords")
	}
	tt.Key(0, ui.KeyBack)
	if a.route != first || len(a.settings.SearchHistory) != 0 {
		t.Fatal("back failed or restored cleared keywords")
	}
}
