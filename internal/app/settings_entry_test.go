package app

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"netease-music-mygo/internal/netease"
)

// The top-right settings entry opens settings from both a page and the full-screen player.
func TestSettingsEntry(t *testing.T) {
	old := showSettings
	defer func() { showSettings = old }()
	for _, playing := range []bool{false, true} {
		a := newTestApp()
		a.settings = defaultSettings()
		if playing {
			a.pl.Cue([]netease.Track{{ID: 1, Name: "春日影"}})
			a.pl.SetLyrics(nil)
			a.comments = commentState{track: 1, loaded: true}
			a.nowPlaying = true
		}
		opened := 0
		showSettings = func(got *app) {
			if got != a {
				t.Error("settings opened for another app")
			}
			opened++
		}
		tt := ui.NewTester(a.view, 1200, 800)
		tt.Advance(npSettle + time.Second)
		if err := tt.Click("设置"); err != nil {
			t.Fatal(err)
		}
		if opened != 1 {
			t.Fatalf("Now Playing %v: opened settings %d times", playing, opened)
		}
	}
}
