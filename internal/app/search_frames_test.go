package app

import (
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"netease-music-mygo/internal/lyrics"
	"netease-music-mygo/internal/netease"
)

// Search and catalog pages can be rendered in both appearances for visual inspection, without network access.
func TestSearchFrames(t *testing.T) {
	dir := os.Getenv("NM_FRAMES")
	if dir == "" {
		t.Skip()
	}
	for _, width := range []int{1200, 880} {
		for _, dark := range []bool{false, true} {
			a := newTestApp()
			a.settings = defaultSettings()
			artist := netease.Artist{ID: 7, Name: "MyGO!!!!!", Alias: []string{"マイゴ"}, MusicSize: 38, AlbumSize: 9, BriefDesc: "迷いながらでも、前へ。"}
			album := netease.Album{ID: 8, Name: "迷跡波", Artist: artist, PublishTime: 1698192000000}
			user := netease.User{UserID: 9, Nickname: "听歌的人", Signature: "在音乐里，寻找同路的人。", Follows: 12, Followeds: 24}
			var tracks []netease.Track
			for i, name := range []string{"春日影", "迷星叫", "壱雫空", "碧天伴走", "音一会", "栞"} {
				tracks = append(tracks, netease.Track{ID: int64(i + 1), Name: name, Artists: []netease.Artist{artist}, Album: album, Duration: int64(210000 + 16000*i)})
			}
			var artists []netease.Artist
			for i, name := range []string{"MyGO!!!!!", "Ave Mujica", "Roselia", "Poppin’Party", "RAISE A SUILEN", "Afterglow"} {
				ar := artist
				ar.ID, ar.Name = int64(i+10), name
				artists = append(artists, ar)
			}
			var albums []netease.Album
			for i, name := range []string{"迷跡波", "跡暖空", "壱雫空", "音一会", "迷星叫", "焚音打"} {
				al := album
				al.ID, al.Name = int64(i+20), name
				albums = append(albums, al)
			}
			users := []netease.User{user, {UserID: 10, Nickname: "夜航电台", Signature: "分享每一次单曲循环。"}, {UserID: 11, Nickname: "夏日来信", Signature: "今天也有好好听歌。"}}
			a.catalog.artists = map[int64]*catalogArtist{7: {data: &netease.ArtistPage{Artist: artist, Songs: tracks}, albums: albums, albumsLoaded: true}}
			a.catalog.albums = map[int64]*catalogAlbum{8: {data: &netease.AlbumPage{Album: album, Songs: tracks}}}
			a.catalog.users = map[int64]*catalogUser{9: {data: &user, playlistsLoaded: true}}
			for _, tc := range []struct {
				name string
				kind netease.SearchType
				res  netease.SearchResult
			}{
				{"songs", netease.SearchSongs, netease.SearchResult{Songs: tracks, Total: 38, More: true}},
				{"artists", netease.SearchArtists, netease.SearchResult{Artists: artists, Total: 6}},
				{"albums", netease.SearchAlbums, netease.SearchResult{Albums: albums, Total: 6}},
				{"users", netease.SearchUsers, netease.SearchResult{Users: users, Total: 3}},
			} {
				a.open(cachedSearch(a, "MyGO!!!!!", tc.kind, tc.res))
				saveSearchFrame(t, a, dir, tc.name, width, dark)
			}
			for _, tc := range []struct {
				name string
				page pageKind
				id   int64
			}{{"artist", pageArtist, 7}, {"album", pageAlbum, 8}, {"user", pageUser, 9}} {
				a.open(route{page: tc.page, entity: tc.id})
				saveSearchFrame(t, a, dir, tc.name, width, dark)
			}
		}
	}
}

// Search history is photographed with long keywords and a full list in both appearances and window sizes.
func TestSearchHistoryFrames(t *testing.T) {
	dir := os.Getenv("NM_FRAMES")
	if dir == "" {
		t.Skip()
	}
	for _, size := range [][2]int{{1200, 780}, {880, 580}} {
		for _, dark := range []bool{false, true} {
			a := newTestApp()
			a.settings.SearchHistory = []string{
				"春日影", "MyGO!!!!!", "迷星叫", "Ave Mujica",
				"夜晚听的歌：一个很长很长的搜索关键词，用来检查文字截断和布局",
			}
			for i := len(a.settings.SearchHistory); i < searchHistoryLimit; i++ {
				a.settings.SearchHistory = append(a.settings.SearchHistory, fmt.Sprintf("最近听过的歌 %d", i))
			}
			tt := ui.NewTester(a.view, size[0], size[1])
			tt.SetDark(dark)
			if err := tt.Click("搜索"); err != nil {
				t.Fatal(err)
			}
			tt.Advance(time.Second)
			tt.Frame()
			path := filepath.Join(dir, fmt.Sprintf("search-history-%d-%t.png", size[0], dark))
			f, err := os.Create(path)
			if err != nil {
				t.Fatal(err)
			}
			if err := png.Encode(f, tt.Image()); err != nil {
				f.Close()
				t.Fatal(err)
			}
			f.Close()
		}
	}
}

// Full-screen playback is photographed at native window sizes instead of judged by scaled screenshots.
func TestNowPlayingLayoutFrames(t *testing.T) {
	dir := os.Getenv("NM_FRAMES")
	if dir == "" {
		t.Skip()
	}
	for _, size := range [][2]int{{1180, 780}, {1920, 1080}, {2560, 1440}, {880, 580}} {
		a := newTestApp()
		a.settings = defaultSettings()
		tr := netease.Track{ID: 1, Name: "春日影", Artists: []netease.Artist{{ID: 7, Name: "MyGO!!!!!"}}, Album: netease.Album{ID: 8, Name: "迷跡波"}, Duration: 220000}
		a.pl.Cue([]netease.Track{tr})
		a.pl.SetLyrics(lyrics.Build("[00:00.00]乾いた木洩れ日\n[00:10.00]どうしてこんなにも\n[00:20.00]もう離さないで\n[00:30.00]君の隣にいたい", "", ""))
		a.comments = commentState{track: tr.ID, loaded: true}
		a.nowPlaying = true
		tt := ui.NewTester(a.view, size[0], size[1])
		tt.SetPreferences(ui.Preferences{ReduceMotion: true})
		tt.Move(float32(size[0])/2, 100)
		tt.Advance(npSettle + time.Second)
		tt.Frame()
		path := filepath.Join(dir, fmt.Sprintf("nowplaying-%d-%d.png", size[0], size[1]))
		f, err := os.Create(path)
		if err != nil {
			t.Fatal(err)
		}
		if err := png.Encode(f, tt.Image()); err != nil {
			t.Fatal(err)
		}
		f.Close()
	}
}

func saveSearchFrame(t *testing.T, a *app, dir, name string, width int, dark bool) {
	t.Helper()
	tt := ui.NewTester(a.view, width, 780)
	tt.SetDark(dark)
	tt.Advance(time.Second)
	tt.Frame()
	path := filepath.Join(dir, fmt.Sprintf("search-%s-%d-%t.png", name, width, dark))
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, tt.Image()); err != nil {
		t.Fatal(err)
	}
}
