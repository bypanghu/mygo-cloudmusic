package app

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"netease-music-mygo/internal/lyrics"
	"netease-music-mygo/internal/netease"
)

// playingApp is an app playing a track with lyrics, at a position the test
// sets with pl.Seek, and the number of times its view was built.
func playingApp() (*app, *ui.Tester, *int) {
	a := newTestApp()
	a.settings = defaultSettings()
	tr := netease.Track{ID: 7, Name: "宝船", Artists: []netease.Artist{{Name: "DOUDOU"}}, Duration: 200000}
	a.pl.Cue([]netease.Track{tr})
	// A line every 10 seconds, sung over the first 5 of them.
	var lrc string
	for i := range 12 {
		lrc += "[0" + string(rune('0'+i*10/60)) + ":" + string(rune('0'+i*10%60/10)) + "0.00]一二三四五六\n"
	}
	lines := lyrics.Build(lrc, "", "")
	for i := range lines {
		l := &lines[i]
		l.End = l.At + 5*time.Second
		l.Words = lyrics.SpreadWords(l.Text, l.At, l.End)
	}
	a.pl.SetLyrics(lines)
	a.pl.Hold(0)
	builds := 0
	tt := ui.NewTester(func(c *ui.Context) { builds++; a.view(c) }, 1200, 800)
	return a, tt, &builds
}

// framesAt counts the frames the view asks for over a second, at 60 a
// second, with playback held at pos.
func framesAt(a *app, tt *ui.Tester, pos time.Duration) int {
	a.pl.Seek(pos)
	tt.Frame()
	tt.Advance(time.Second)
	tt.Frame()
	asked := 0
	for range 60 {
		tt.Advance(time.Second / 60)
		if tt.FramePending() {
			asked++
			tt.Frame()
		}
	}
	return asked
}

// Now Playing asks for frames only as fast as what it shows changes. The
// Tester runs timers as it settles a frame, so this counts only what does
// not change for a while: a frame log of the app (MYGO_FRAME_STATS) measures
// the rest.
func TestNowPlayingFrameBudget(t *testing.T) {
	a, tt, _ := playingApp()
	if n := framesAt(a, tt, 12*time.Second); n > 3 {
		t.Errorf("a page under the player: %d frames a second", n)
	}
	a.nowPlaying, a.npPane = true, paneStage
	if n := framesAt(a, tt, 18*time.Second); n > 12 {
		t.Errorf("the stage between lines: %d frames a second", n)
	}
}

// Loaded search categories and entity details settle without scheduling idle animation frames.
func TestCatalogFrameBudget(t *testing.T) {
	for _, page := range []pageKind{pageSearch, pageArtist, pageAlbum, pageUser} {
		a := newTestApp()
		a.settings = defaultSettings()
		switch page {
		case pageSearch:
			a.open(cachedSearch(a, "春日影", netease.SearchAlbums, netease.SearchResult{Albums: []netease.Album{{ID: 1, Name: "迷跡波"}}, Total: 1}))
		case pageArtist:
			a.catalog.artists = map[int64]*catalogArtist{1: {data: &netease.ArtistPage{Artist: netease.Artist{ID: 1, Name: "MyGO!!!!!"}}}}
			a.open(route{page: pageArtist, entity: 1})
		case pageAlbum:
			a.catalog.albums = map[int64]*catalogAlbum{1: {data: &netease.AlbumPage{Album: netease.Album{ID: 1, Name: "迷跡波"}}}}
			a.open(route{page: pageAlbum, entity: 1})
		case pageUser:
			a.catalog.users = map[int64]*catalogUser{1: {data: &netease.User{UserID: 1, Nickname: "听歌的人"}, playlistsLoaded: true}}
			a.open(route{page: pageUser, entity: 1})
		}
		tt := ui.NewTester(a.view, 1200, 800)
		tt.Advance(time.Second)
		tt.Frame()
		tt.Advance(time.Second)
		if tt.FramePending() {
			t.Errorf("page %d keeps requesting frames while idle", page)
		}
	}
}

// Another song's backdrop fades in over the last one's, which is then let
// go, so that the backdrop stops asking for frames.
func TestBackdropFadeEnds(t *testing.T) {
	a, tt, _ := coverApp(t)
	a.art.art[netease.Sized("a", 600)+"#backdrop"] = &artwork{tint: ui.RGB(200, 40, 40)}
	a.art.art[netease.Sized("b", 600)+"#backdrop"] = &artwork{tint: ui.RGB(40, 40, 200)}
	q := a.pl.Queue()
	q[1].Album.PicURL, q[2].Album.PicURL = "a", "b"
	tt.Frame()
	a.pl.Next()
	tt.Frame()
	if a.backdropCache.before == nil {
		t.Fatal("the backdrop changed songs without fading")
	}
	tt.Advance(backdropFade + 100*time.Millisecond)
	tt.Frame()
	if a.backdropCache.before != nil {
		t.Error("the last song's backdrop still fades after the fade")
	}
}
