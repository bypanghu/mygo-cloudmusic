package app

import (
	"errors"
	"slices"
	"testing"

	"github.com/egoist/mygo/ui"

	"netease-music-mygo/internal/netease"
)

func cachedSearch(a *app, q string, kind netease.SearchType, res netease.SearchResult) route {
	r := route{page: pageSearch, query: q, kind: kind}
	if a.searches.entries == nil {
		a.searches.entries = make(map[route]*searchEntry)
	}
	a.searches.entries[r] = &searchEntry{result: res}
	return r
}

// Category clicks and back/forward restore the search rather than showing the last query everywhere.
func TestSearchCategoriesAndHistory(t *testing.T) {
	a := newTestApp()
	a.settings = defaultSettings()
	songs := cachedSearch(a, "春日影", netease.SearchSongs, netease.SearchResult{})
	artists := cachedSearch(a, "春日影", netease.SearchArtists, netease.SearchResult{Artists: []netease.Artist{{ID: 7, Name: "MyGO!!!!!"}}, Total: 1})
	second := cachedSearch(a, "迷星叫", netease.SearchArtists, netease.SearchResult{})
	a.catalog.artists = map[int64]*catalogArtist{7: {data: &netease.ArtistPage{Artist: netease.Artist{ID: 7, Name: "MyGO!!!!!"}}}}
	a.open(songs)
	tt := ui.NewTester(a.view, 1200, 800)
	if err := tt.Click("歌手"); err != nil {
		t.Fatal(err)
	}
	if a.route != artists {
		t.Fatalf("category click: %+v", a.route)
	}
	if err := tt.Click("MyGO!!!!!"); err != nil {
		t.Fatal(err)
	}
	if a.route != (route{page: pageArtist, entity: 7}) {
		t.Fatalf("artist click: %+v", a.route)
	}
	tt.Key(0, ui.KeyBack)
	if a.route != artists || a.query != "春日影" {
		t.Fatalf("back lost search: route %+v, query %q", a.route, a.query)
	}
	if err := tt.Click("搜索"); err != nil {
		t.Fatal(err)
	}
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Type("迷星叫")
	tt.Key(0, ui.KeyEnter)
	if a.route != second {
		t.Fatalf("submit did not keep category: %+v", a.route)
	}
	tt.Key(0, ui.KeyBack)
	if a.route != artists || a.searchEntry().result.Artists[0].ID != 7 {
		t.Fatalf("back did not restore cached result: %+v", a.route)
	}
	tt.Key(0, ui.KeyForward)
	if a.route != second || a.query != "迷星叫" {
		t.Fatalf("forward lost query: %+v / %q", a.route, a.query)
	}
	if !slices.Equal(a.settings.SearchHistory, []string{"迷星叫"}) {
		t.Fatalf("submit was not recorded or navigation added keywords: %q", a.settings.SearchHistory)
	}
}

// Older same-query requests and late page failures cannot overwrite newer search results.
func TestSearchCompletion(t *testing.T) {
	a := newTestApp()
	r := cachedSearch(a, "春日影", netease.SearchAlbums, netease.SearchResult{})
	e := a.searches.entries[r]
	e.request, e.loading = 2, true
	a.finishSearch(r, 1, 0, netease.SearchResult{Albums: []netease.Album{{ID: 1}}}, nil)
	if !e.loading || len(e.result.Albums) != 0 {
		t.Fatal("older request changed the active request")
	}
	a.finishSearch(r, 2, 0, netease.SearchResult{Albums: []netease.Album{{ID: 2}}, Total: 2, More: true}, nil)
	e.request, e.loading = 3, true
	a.finishSearch(r, 3, 1, netease.SearchResult{}, errors.New("offline"))
	if len(e.result.Albums) != 1 || e.result.Albums[0].ID != 2 || e.err == "" {
		t.Fatal("failed page erased the existing result")
	}
	e.request, e.loading = 4, true
	a.finishSearch(r, 4, 1, netease.SearchResult{Albums: []netease.Album{{ID: 3}}, Total: 2}, nil)
	if len(e.result.Albums) != 2 || e.result.Albums[1].ID != 3 || e.loading || e.err != "" {
		t.Fatalf("retry did not append: %+v", e)
	}
	e.request, e.loading = 5, true
	a.finishSearch(r, 5, 2, netease.SearchResult{Total: 3, More: true}, nil)
	if e.result.More {
		t.Fatal("empty page kept requesting more forever")
	}
}

// Load-more and retry buttons request the unfilled page without discarding loaded rows.
func TestSearchPaginationRetry(t *testing.T) {
	old := searchCatalog
	defer func() { searchCatalog = old }()
	type call struct {
		query         string
		kind          netease.SearchType
		limit, offset int
	}
	calls := make(chan call, 4)
	searchCatalog = func(q string, k netease.SearchType, limit, offset int) (netease.SearchResult, error) {
		calls <- call{q, k, limit, offset}
		return netease.SearchResult{}, nil
	}
	a := newTestApp()
	a.settings = defaultSettings()
	r := cachedSearch(a, "春日影", netease.SearchUsers, netease.SearchResult{Users: []netease.User{{UserID: 9, Nickname: "听歌的人"}}, Total: 2, More: true})
	a.open(r)
	tt := ui.NewTester(a.view, 1200, 800)
	if err := tt.Click("加载更多"); err != nil {
		t.Fatal(err)
	}
	got := <-calls
	if got != (call{"春日影", netease.SearchUsers, searchLimit, 1}) {
		t.Fatalf("next page request: %+v", got)
	}
	e := a.searches.entries[r]
	a.finishSearch(r, e.request, 1, netease.SearchResult{}, errors.New("offline"))
	tt.Frame()
	if err := tt.Click("重试加载"); err != nil {
		t.Fatal(err)
	}
	if retry := <-calls; retry != got || len(e.result.Users) != 1 {
		t.Fatalf("retry lost page or data: %+v", retry)
	}
}

// Clicking an artist credit or an album opens details without replacing the playback queue.
func TestTrackCreditsNavigateWithoutPlaying(t *testing.T) {
	for _, tc := range []struct {
		label string
		want  route
	}{
		{"查看歌手：MyGO!!!!!", route{page: pageArtist, entity: 7}},
		{"查看专辑：迷跡波", route{page: pageAlbum, entity: 8}},
	} {
		t.Run(tc.label, func(t *testing.T) {
			a := newTestApp()
			a.settings = defaultSettings()
			tr := netease.Track{ID: 1, Name: "春日影", Artists: []netease.Artist{{ID: 7, Name: "MyGO!!!!!"}}, Album: netease.Album{ID: 8, Name: "迷跡波"}}
			r := cachedSearch(a, tr.Name, netease.SearchSongs, netease.SearchResult{Songs: []netease.Track{tr}, Total: 1})
			a.catalog.artists = map[int64]*catalogArtist{7: {data: &netease.ArtistPage{Artist: tr.Artists[0]}}}
			a.catalog.albums = map[int64]*catalogAlbum{8: {data: &netease.AlbumPage{Album: tr.Album}}}
			a.pl.Cue([]netease.Track{{ID: 99, Name: "原来的歌"}})
			a.open(r)
			tt := ui.NewTester(a.view, 1200, 800)
			if err := tt.Click(tc.label); err != nil {
				t.Fatal(err)
			}
			if a.route != tc.want || len(a.pl.Queue()) != 1 || a.pl.Queue()[0].ID != 99 {
				t.Fatalf("credit triggered playback or wrong route: %+v / %+v", a.route, a.pl.Queue())
			}
		})
	}
}

// Album and user search results navigate to their own entity rather than a playlist with the same ID.
func TestSearchEntityCards(t *testing.T) {
	for _, tc := range []struct {
		kind   netease.SearchType
		label  string
		result netease.SearchResult
		want   route
	}{
		{netease.SearchAlbums, "迷跡波", netease.SearchResult{Albums: []netease.Album{{ID: 8, Name: "迷跡波"}}, Total: 1}, route{page: pageAlbum, entity: 8}},
		{netease.SearchUsers, "听歌的人", netease.SearchResult{Users: []netease.User{{UserID: 9, Nickname: "听歌的人"}}, Total: 1}, route{page: pageUser, entity: 9}},
	} {
		t.Run(tc.label, func(t *testing.T) {
			a := newTestApp()
			a.settings = defaultSettings()
			a.catalog.albums = map[int64]*catalogAlbum{8: {data: &netease.AlbumPage{Album: netease.Album{ID: 8, Name: "迷跡波"}}}}
			a.catalog.users = map[int64]*catalogUser{9: {data: &netease.User{UserID: 9, Nickname: "听歌的人"}, playlistsLoaded: true}}
			a.open(cachedSearch(a, "春日影", tc.kind, tc.result))
			tt := ui.NewTester(a.view, 1200, 800)
			if err := tt.Click(tc.label); err != nil {
				t.Fatal(err)
			}
			if a.route != tc.want {
				t.Fatalf("card route: %+v", a.route)
			}
		})
	}
}
