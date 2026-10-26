package netease

import (
	"fmt"
	"net/url"
	"strconv"
)

type SearchType int

const (
	SearchSongs    SearchType = 1
	SearchAlbums   SearchType = 10
	SearchArtists  SearchType = 100
	SearchUsers    SearchType = 1002
	SearchPodcasts SearchType = 1009
)

type SearchResult struct {
	Songs    []Track
	Albums   []Album
	Artists  []Artist
	Users    []User
	Podcasts []Podcast
	Total    int
	More     bool
}

// SearchPage returns one page of the requested kind of search result.
func SearchPage(query string, kind SearchType, limit, offset int) (SearchResult, error) {
	switch kind {
	case SearchSongs, SearchAlbums, SearchArtists, SearchUsers, SearchPodcasts:
	default:
		return SearchResult{}, fmt.Errorf("netease: unsupported search type %d", kind)
	}
	var res struct {
		Result struct {
			Songs        []Track   `json:"songs"`
			Albums       []Album   `json:"albums"`
			Artists      []Artist  `json:"artists"`
			Users        []User    `json:"userprofiles"`
			Podcasts     []Podcast `json:"djRadios"`
			PodcastCount int       `json:"djRadiosCount"`
			SongCount    int       `json:"songCount"`
			AlbumCount   int       `json:"albumCount"`
			ArtistCount  int       `json:"artistCount"`
			UserCount    int       `json:"userprofileCount"`
			More         *bool     `json:"hasMore"`
		} `json:"result"`
	}
	q := url.Values{"s": {query}, "type": {strconv.Itoa(int(kind))}, "limit": {strconv.Itoa(limit)}, "offset": {strconv.Itoa(offset)}, "total": {"true"}}
	if err := get("/api/cloudsearch/pc", q, &res); err != nil {
		return SearchResult{}, err
	}
	r := res.Result
	out := SearchResult{Songs: r.Songs, Albums: r.Albums, Artists: r.Artists, Users: r.Users, Podcasts: r.Podcasts}
	n := 0
	switch kind {
	case SearchSongs:
		out.Total, n = r.SongCount, len(r.Songs)
		// Search sometimes marks playable catalogues as taken down; song details are authoritative.
		detailPrivileges(out.Songs)
	case SearchAlbums:
		out.Total, n = r.AlbumCount, len(r.Albums)
	case SearchArtists:
		out.Total, n = r.ArtistCount, len(r.Artists)
	case SearchUsers:
		out.Total, n = r.UserCount, len(r.Users)
	case SearchPodcasts:
		out.Total, n = r.PodcastCount, len(r.Podcasts)
	}
	out.More = offset+n < out.Total
	if r.More != nil {
		out.More = *r.More
	}
	return out, nil
}

type ArtistPage struct {
	Artist Artist  `json:"artist"`
	Songs  []Track `json:"hotSongs"`
}

type AlbumPage struct {
	Album Album   `json:"album"`
	Songs []Track `json:"songs"`
}

// ArtistDetail returns an artist and their popular songs with listener privileges.
func ArtistDetail(id int64) (*ArtistPage, error) {
	var res ArtistPage
	if err := get("/api/artist/"+strconv.FormatInt(id, 10), nil, &res); err != nil {
		return nil, err
	}
	detailPrivileges(res.Songs)
	return &res, nil
}

// ArtistAlbums returns one page of an artist's albums.
func ArtistAlbums(id int64, limit, offset int) ([]Album, error) {
	var res struct {
		Albums []Album `json:"hotAlbums"`
	}
	q := url.Values{"limit": {strconv.Itoa(limit)}, "offset": {strconv.Itoa(offset)}}
	err := get("/api/artist/albums/"+strconv.FormatInt(id, 10), q, &res)
	return res.Albums, err
}

// AlbumDetail returns an album and its songs with listener privileges.
func AlbumDetail(id int64) (*AlbumPage, error) {
	var res struct {
		Album struct {
			Album
			Songs []Track `json:"songs"`
		} `json:"album"`
		Songs      []Track     `json:"songs"`
		Privileges []Privilege `json:"privileges"`
	}
	if err := get("/api/album/"+strconv.FormatInt(id, 10), nil, &res); err != nil {
		return nil, err
	}
	if res.Songs == nil {
		res.Songs = res.Album.Songs
	}
	withPrivileges(res.Songs, res.Privileges)
	detailPrivileges(res.Songs)
	return &AlbumPage{Album: res.Album.Album, Songs: res.Songs}, nil
}

// UserDetail returns a user's public profile.
func UserDetail(id int64) (*User, error) {
	var res struct {
		Profile User `json:"profile"`
	}
	if err := get("/api/v1/user/detail/"+strconv.FormatInt(id, 10), nil, &res); err != nil {
		return nil, err
	}
	return &res.Profile, nil
}
