package app

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"netease-music-mygo/internal/netease"
)

// Now Playing artist links leave the overlay for the selected artist without changing playback.
func TestNowPlayingArtistLink(t *testing.T) {
	for _, pane := range []int{paneLyrics, paneStage} {
		a := newTestApp()
		a.settings = defaultSettings()
		tr := netease.Track{ID: 1, Name: "春日影", Artists: []netease.Artist{{ID: 7, Name: "MyGO!!!!!"}}, Album: netease.Album{ID: 8, Name: "迷跡波"}}
		a.pl.Cue([]netease.Track{tr})
		a.pl.SetLyrics(nil)
		a.comments = commentState{track: tr.ID, loaded: true}
		a.catalog.artists = map[int64]*catalogArtist{7: {data: &netease.ArtistPage{Artist: tr.Artists[0]}}}
		a.nowPlaying, a.npPane = true, pane
		tt := ui.NewTester(a.view, 1200, 800)
		tt.Advance(npSettle + time.Second)
		if err := tt.Click("查看歌手：MyGO!!!!!"); err != nil {
			t.Fatal(err)
		}
		if a.nowPlaying || a.route != (route{page: pageArtist, entity: 7}) || a.pl.Current().ID != tr.ID {
			t.Fatalf("pane %d did not navigate safely: %+v / open %v", pane, a.route, a.nowPlaying)
		}
	}
}

// Resizing to full-screen bounds keeps the cover near the top and the playback controls reachable.
func TestNowPlayingResize(t *testing.T) {
	a := newTestApp()
	a.settings = defaultSettings()
	tr := netease.Track{ID: 1, Name: "春日影", Artists: []netease.Artist{{ID: 7, Name: "MyGO!!!!!"}}}
	a.pl.Cue([]netease.Track{tr})
	a.pl.SetLyrics(nil)
	a.comments = commentState{track: tr.ID, loaded: true}
	a.nowPlaying = true
	tt := ui.NewTester(a.view, 1180, 780)
	tt.SetPreferences(ui.Preferences{ReduceMotion: true})
	tt.Advance(npSettle + time.Second)
	for _, size := range [][2]int{{1920, 1080}, {2560, 1440}, {880, 580}, {1180, 780}} {
		tt.SetSize(size[0], size[1])
		tt.Advance(time.Second)
		tt.Frame()
		cover, ok := tt.Find("专辑封面")
		if !ok || cover.Y > 110 || cover.W > 442 || cover.Y < 30 {
			t.Fatalf("%v cover scaled or left a blank upper band: %+v", size, cover)
		}
		control, ok := tt.Find("播放/暂停")
		if !ok || control.Y+control.H > float32(size[1])-12 {
			t.Fatalf("%v playback control outside view: %+v", size, control)
		}
	}
}
