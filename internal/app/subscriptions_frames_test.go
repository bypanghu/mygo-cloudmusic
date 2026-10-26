package app

import (
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"netease-music-mygo/internal/netease"
)

// With NM_FRAMES set, saves subscription controls at the minimum and normal window widths in both appearances.
func TestSubscriptionFrames(t *testing.T) {
	dir := os.Getenv("NM_FRAMES")
	if dir == "" {
		t.Skip()
	}
	for _, width := range []int{880, 1180} {
		for _, dark := range []bool{false, true} {
			for _, page := range []pageKind{pagePodcast, pageArtist} {
				for _, subscribed := range []bool{false, true} {
					a := newTestApp()
					a.settings = defaultSettings()
					a.profile = &netease.Profile{UserID: 1, Nickname: "小明"}
					a.catalog.podcasts = map[int64]*catalogPodcast{7: {
						data: &netease.Podcast{ID: 7, Name: "在云端", DJ: netease.User{UserID: 8, Nickname: "音乐主播"}, ProgramCount: 403, SubCount: 89986, Subed: subscribed, Description: "在音乐与故事之间，与你分享每一天的声音。"},
						programs: []netease.PodcastProgram{
							{ID: 11, Name: "把故事说给你听", Duration: 1800000, MainSong: netease.Track{ID: 101}},
							{ID: 12, Name: "今天想听的那首歌", Duration: 1500000, MainSong: netease.Track{ID: 102}},
						},
						programsLoaded: true,
					}}
					a.catalog.artists = map[int64]*catalogArtist{7: {
						data: &netease.ArtistPage{Artist: netease.Artist{ID: 7, Name: "MyGO!!!!!", Alias: []string{"マイゴ"}, MusicSize: 42, AlbumSize: 7, Followed: subscribed}},
					}}
					a.open(route{page: page, entity: 7})
					tt := ui.NewTester(a.view, width, 700)
					tt.SetDark(dark)
					tt.Advance(time.Second)
					tt.Frame()
					name := fmt.Sprintf("subscription-%d-%d-dark%t-saved%t.png", page, width, dark, subscribed)
					f, err := os.Create(filepath.Join(dir, name))
					if err != nil {
						t.Fatal(err)
					}
					err = png.Encode(f, tt.Image())
					f.Close()
					if err != nil {
						t.Fatal(err)
					}
				}
			}
		}
	}
}
