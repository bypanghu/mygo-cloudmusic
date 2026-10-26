package app

import (
	"errors"
	"testing"

	"github.com/egoist/mygo/ui"

	"netease-music-mygo/internal/netease"
)

// Failed artist-album pages retain their offset, and overlapping pages never duplicate albums.
func TestArtistAlbumsPagination(t *testing.T) {
	s := &catalogArtist{albums: []netease.Album{{ID: 1}}, albumsOffset: 60, albumsLoaded: true, albumsMore: true}
	s.completeAlbums(nil, errors.New("offline"))
	if s.albumsOffset != 60 || len(s.albums) != 1 || s.albumsErr == "" {
		t.Fatalf("failed page changed data: %+v", s)
	}
	s.completeAlbums([]netease.Album{{ID: 1}, {ID: 2}}, nil)
	if s.albumsOffset != 62 || len(s.albums) != 2 || s.albums[1].ID != 2 || s.albumsMore || s.albumsErr != "" {
		t.Fatalf("overlap or retry lost: %+v", s)
	}
}

// Public profiles exclude foreign and private playlists even when the API puts them in the made list.
func TestUserPublicPlaylists(t *testing.T) {
	lists := make([]netease.Playlist, 3)
	for i := range lists {
		lists[i].ID = int64(i + 1)
		lists[i].Creator.UserID = 7
	}
	lists[1].Creator.UserID = 8
	lists[2].Privacy = 10
	s := &catalogUser{}
	s.completePlaylists(7, lists, nil, nil)
	if !s.playlistsLoaded || len(s.playlists) != 1 || s.playlists[0].ID != 1 {
		t.Fatalf("profile leaked playlists: %+v", s.playlists)
	}
	// A viewer's subscription must not hide the playlist from its owner's profile.
	s.completePlaylists(7, lists[1:], lists[:1], nil)
	if len(s.playlists) != 1 || s.playlists[0].ID != 1 {
		t.Fatalf("subscribed owner playlist disappeared: %+v", s.playlists)
	}
}

// Artist tabs fetch albums on demand, retain cached albums, and reset for a different artist.
func TestArtistTabs(t *testing.T) {
	old := fetchArtistAlbums
	defer func() { fetchArtistAlbums = old }()
	calls := make(chan int64, 2)
	fetchArtistAlbums = func(id int64, limit, offset int) ([]netease.Album, error) {
		calls <- id
		return nil, nil
	}
	a := newTestApp()
	a.settings = defaultSettings()
	a.catalog.artists = map[int64]*catalogArtist{
		7: {data: &netease.ArtistPage{Artist: netease.Artist{ID: 7, Name: "MyGO!!!!!"}}},
		8: {data: &netease.ArtistPage{Artist: netease.Artist{ID: 8, Name: "Ave Mujica"}}},
	}
	a.open(route{page: pageArtist, entity: 7})
	tt := ui.NewTester(a.view, 1200, 800)
	if err := tt.Click("专辑"); err != nil {
		t.Fatal(err)
	}
	if id := <-calls; id != 7 || a.catalog.tab != 1 {
		t.Fatalf("album tab: %d / %d", id, a.catalog.tab)
	}
	a.catalog.artists[7].completeAlbums([]netease.Album{{ID: 9, Name: "迷跡波"}}, nil)
	tt.Frame()
	if err := tt.Click("关于"); err != nil {
		t.Fatal(err)
	}
	if a.catalog.tab != 2 {
		t.Fatal("about did not switch")
	}
	a.open(route{page: pageArtist, entity: 8})
	tt.Frame()
	if a.catalog.tab != 0 || a.catalog.artists[7].albums[0].ID != 9 {
		t.Fatal("new artist kept old tab or discarded cached albums")
	}
}
