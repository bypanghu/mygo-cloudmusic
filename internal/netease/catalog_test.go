package netease

import (
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

type catalogTransport func(*http.Request) string

func (f catalogTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(f(r))), Request: r}, nil
}
func mockCatalog(f catalogTransport) func() {
	old := httpClient
	httpClient = &http.Client{Transport: f}
	return func() { httpClient = old }
}

// Search sends the selected category and pagination and decodes its counts and entities.
func TestSearchPageCategories(t *testing.T) {
	tests := []struct {
		kind  SearchType
		body  string
		total int
		more  bool
	}{
		{SearchSongs, `"songs":[{"id":1}],"songCount":8`, 8, true},
		{SearchAlbums, `"albums":[{"id":2,"artist":{"id":9}}],"albumCount":4`, 4, false},
		{SearchArtists, `"artists":[{"id":3,"img1v1Url":"portrait"}],"artistCount":8,"hasMore":false`, 8, false},
		{SearchUsers, `"userprofiles":[{"userId":4,"nickname":"listener"}],"userprofileCount":4,"hasMore":true`, 4, true},
	}
	for _, tt := range tests {
		t.Run(strconv.Itoa(int(tt.kind)), func(t *testing.T) {
			defer mockCatalog(func(r *http.Request) string {
				if r.URL.Path == "/api/v3/song/detail" {
					return `{"code":200,"songs":[{"id":1}],"privileges":[{"id":1,"pl":320000}]}`
				}
				q := r.URL.Query()
				if r.Method != "GET" || r.URL.Path != "/api/cloudsearch/pc" || q.Get("s") != "a & b" || q.Get("type") != strconv.Itoa(int(tt.kind)) || q.Get("limit") != "1" || q.Get("offset") != "3" {
					t.Fatalf("request: %s", r.URL)
				}
				return `{"code":200,"result":{` + tt.body + `}}`
			})()
			got, err := SearchPage("a & b", tt.kind, 1, 3)
			if err != nil || got.Total != tt.total || got.More != tt.more {
				t.Fatalf("result %#v, %v", got, err)
			}
			switch tt.kind {
			case SearchSongs:
				if len(got.Songs) != 1 || !got.Songs[0].Playable() {
					t.Fatal(got.Songs)
				}
			case SearchAlbums:
				if len(got.Albums) != 1 || got.Albums[0].Artist.ID != 9 {
					t.Fatal(got.Albums)
				}
			case SearchArtists:
				if len(got.Artists) != 1 || got.Artists[0].Img1v1URL != "portrait" {
					t.Fatal(got.Artists)
				}
			case SearchUsers:
				if len(got.Users) != 1 || got.Users[0].UserID != 4 {
					t.Fatal(got.Users)
				}
			}
		})
	}
}

// Song details override incorrect search privileges by ID rather than response order.
func TestSearchPrivilegeHydration(t *testing.T) {
	defer mockCatalog(func(r *http.Request) string {
		if r.URL.Path == "/api/cloudsearch/pc" {
			if r.URL.Query().Get("offset") != "0" {
				t.Fatal("compatibility search must start at zero")
			}
			return `{"code":200,"result":{"songs":[{"id":1,"fee":1,"privilege":{"id":1,"st":-200}},{"id":2},{"id":3,"fee":1}]}}`
		}
		if r.URL.Path != "/api/v3/song/detail" || r.Method != "POST" {
			t.Fatalf("unexpected request %s", r.URL)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		if r.Form.Get("c") != `[{"id":1},{"id":2},{"id":3}]` {
			t.Fatal(r.Form)
		}
		return `{"code":200,"songs":[{"id":2},{"id":1}],"privileges":[{"id":1,"pl":320000},{"id":2,"pl":0}]}`
	})()
	songs, err := Search("query", 3)
	if err != nil || len(songs) != 3 {
		t.Fatalf("%v %v", songs, err)
	}
	if !songs[0].Playable() || songs[1].Playable() || songs[2].Playable() || songs[2].Privilege != nil {
		t.Fatalf("incorrect privileges: %#v", songs)
	}
}

// Legacy album songs and modern song details retain metadata, with modern fields taking precedence.
func TestTrackLegacyDecoding(t *testing.T) {
	for _, fields := range []string{
		`"artists":[{"id":2}],"album":{"id":3},"duration":42`,
		`"ar":[{"id":2}],"al":{"id":3},"dt":42`,
		`"ar":[{"id":2}],"al":{"id":3},"dt":42,"artists":[{"id":99}],"album":{"id":99},"duration":99`,
	} {
		var got Track
		if err := json.Unmarshal([]byte(`{"id":1,"name":"song","fee":1,"alia":["alias"],"privilege":{"id":1,"pl":320000},`+fields+`}`), &got); err != nil {
			t.Fatal(err)
		}
		if got.ID != 1 || got.Name != "song" || got.Fee != 1 || !reflect.DeepEqual(got.Alias, []string{"alias"}) || len(got.Artists) != 1 || got.Artists[0].ID != 2 || got.Album.ID != 3 || got.Duration != 42 || !got.Playable() {
			t.Fatalf("%#v", got)
		}
	}
}

// Native detail endpoints decode entity metadata and hydrate playable album and artist songs.
func TestCatalogDetails(t *testing.T) {
	artist := `{"id":9,"name":"artist","picUrl":"picture","img1v1Url":"square","alias":["alias"],"briefDesc":"bio","musicSize":12,"albumSize":2}`
	album := `{"id":3,"name":"album","picUrl":"cover","artist":` + artist + `,"artists":[` + artist + `],"publishTime":123,"size":1,"description":"notes"}`
	defer mockCatalog(func(r *http.Request) string {
		switch r.URL.Path {
		case "/api/artist/9":
			return `{"code":200,"artist":` + artist + `,"hotSongs":[{"id":1,"ar":[` + artist + `]}]}`
		case "/api/artist/albums/9":
			if r.URL.Query().Get("limit") != "2" || r.URL.Query().Get("offset") != "4" {
				t.Fatal(r.URL)
			}
			return `{"code":200,"hotAlbums":[` + album + `]}`
		case "/api/album/3":
			return `{"code":200,"album":` + album + `,"songs":[{"id":1,"artists":[` + artist + `],"album":` + album + `,"duration":42,"privilege":{"id":1,"st":-200}}]}`
		case "/api/v1/user/detail/7":
			return `{"code":200,"profile":{"userId":7,"nickname":"listener","avatarUrl":"avatar","signature":"hello","playlistCount":3,"followeds":4,"follows":5}}`
		case "/api/v3/song/detail":
			return `{"code":200,"songs":[{"id":1}],"privileges":[{"id":1,"pl":320000}]}`
		default:
			t.Fatalf("unexpected request %s", r.URL)
			return ""
		}
	})()
	a, err := ArtistDetail(9)
	if err != nil {
		t.Fatal(err)
	}
	want := Artist{ID: 9, Name: "artist", PicURL: "picture", Img1v1URL: "square", Alias: []string{"alias"}, BriefDesc: "bio", MusicSize: 12, AlbumSize: 2}
	if !reflect.DeepEqual(a.Artist, want) || len(a.Songs) != 1 || !a.Songs[0].Playable() {
		t.Fatalf("%#v", a)
	}
	albums, err := ArtistAlbums(9, 2, 4)
	if err != nil || len(albums) != 1 {
		t.Fatalf("%v %v", albums, err)
	}
	b, err := AlbumDetail(3)
	if err != nil {
		t.Fatal(err)
	}
	wantAlbum := Album{ID: 3, Name: "album", PicURL: "cover", Artist: want, Artists: []Artist{want}, PublishTime: 123, Size: 1, Description: "notes"}
	if !reflect.DeepEqual(b.Album, wantAlbum) || !reflect.DeepEqual(albums[0], wantAlbum) || len(b.Songs) != 1 || !b.Songs[0].Playable() || b.Songs[0].Duration != 42 {
		t.Fatalf("%#v", b)
	}
	u, err := UserDetail(7)
	if err != nil || *u != (User{UserID: 7, Nickname: "listener", AvatarURL: "avatar", Signature: "hello", PlaylistCount: 3, Followeds: 4, Follows: 5}) {
		t.Fatalf("%#v %v", u, err)
	}
	var p Playlist
	if err := json.Unmarshal([]byte(`{"creator":{"userId":7,"nickname":"listener"}}`), &p); err != nil || p.Creator.UserID != 7 {
		t.Fatalf("%#v %v", p.Creator, err)
	}
}

// Legacy album responses nest songs under album; discarding them used to leave a playable album empty.
func TestAlbumNestedSongs(t *testing.T) {
	defer mockCatalog(func(r *http.Request) string {
		if r.URL.Path == "/api/album/3" {
			return `{"code":200,"album":{"id":3,"name":"album","songs":[{"id":1,"name":"song","artists":[{"id":9,"name":"artist"}],"duration":42000}]}}`
		}
		return `{"code":200,"songs":[{"id":1}],"privileges":[{"id":1,"pl":320000}]}`
	})()
	album, err := AlbumDetail(3)
	if err != nil || album.Album.ID != 3 || len(album.Songs) != 1 || album.Songs[0].Artists[0].ID != 9 || album.Songs[0].Duration != 42000 || !album.Songs[0].Playable() {
		t.Fatalf("nested album songs lost: %+v / %v", album, err)
	}
}

// API errors propagate, while a failed privilege lookup preserves the search song's fallback.
func TestCatalogErrors(t *testing.T) {
	defer mockCatalog(func(r *http.Request) string {
		if r.URL.Path == "/api/cloudsearch/pc" {
			return `{"code":200,"result":{"songs":[{"id":1,"fee":1}]}}`
		}
		return `{"code":500,"message":"unavailable"}`
	})()
	if _, err := ArtistDetail(1); err == nil {
		t.Fatal("artist error lost")
	}
	if _, err := ArtistAlbums(1, 1, 0); err == nil {
		t.Fatal("albums error lost")
	}
	if _, err := AlbumDetail(1); err == nil {
		t.Fatal("album error lost")
	}
	if _, err := UserDetail(1); err == nil {
		t.Fatal("user error lost")
	}
	if _, err := SearchPage("q", SearchType(99), 1, 0); err == nil {
		t.Fatal("invalid category accepted")
	}
	songs, err := Search("q", 1)
	if err != nil || len(songs) != 1 || songs[0].Playable() {
		t.Fatalf("%v %v", songs, err)
	}
}
