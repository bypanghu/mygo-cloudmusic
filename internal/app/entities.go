package app

import "netease-music-mygo/internal/netease"

// Endpoint variables keep detail loading replaceable without a live service.
var (
	fetchArtistDetail  = netease.ArtistDetail
	fetchArtistAlbums  = netease.ArtistAlbums
	fetchAlbumDetail   = netease.AlbumDetail
	fetchUserDetail    = netease.UserDetail
	fetchUserPlaylists = netease.UserPlaylists
)

const artistAlbumLimit = 60

type catalogState struct {
	active        route
	tab           int
	artists       map[int64]*catalogArtist
	albums        map[int64]*catalogAlbum
	users         map[int64]*catalogUser
	podcasts      map[int64]*catalogPodcast
	programSerial uint64
}

type catalogArtist struct {
	data                                    *netease.ArtistPage
	loading                                 bool
	err                                     string
	albums                                  []netease.Album
	albumsLoaded, albumsLoading, albumsMore bool
	albumsErr                               string
	albumsOffset                            int
	feed                                    artistFeedState
	following                               bool
}

type catalogAlbum struct {
	data    *netease.AlbumPage
	loading bool
	err     string
}

type catalogUser struct {
	data                              *netease.User
	loading                           bool
	err                               string
	playlists                         []netease.Playlist
	playlistsLoaded, playlistsLoading bool
	playlistsErr                      string
}

// loadEntity populates ID-keyed caches: finishing an old route never replaces
// the current route's data. All cache writes, including completion, are on UI.
func (a *app) loadEntity(r route) {
	if a.catalog.active != r {
		a.catalog.active, a.catalog.tab = r, 0
	}
	if r.entity <= 0 {
		return
	}
	switch r.page {
	case pageArtist:
		if a.catalog.artists == nil {
			a.catalog.artists = make(map[int64]*catalogArtist)
		}
		s := a.catalog.artists[r.entity]
		if s == nil {
			s = &catalogArtist{}
			a.catalog.artists[r.entity] = s
		}
		if s.data != nil || s.loading {
			return
		}
		s.loading, s.err = true, ""
		fetch := fetchArtistDetail
		go func() {
			data, err := fetch(r.entity)
			a.post(func() { s.complete(data, err) })
		}()
	case pageAlbum:
		if a.catalog.albums == nil {
			a.catalog.albums = make(map[int64]*catalogAlbum)
		}
		s := a.catalog.albums[r.entity]
		if s == nil {
			s = &catalogAlbum{}
			a.catalog.albums[r.entity] = s
		}
		if s.data != nil || s.loading {
			return
		}
		s.loading, s.err = true, ""
		fetch := fetchAlbumDetail
		go func() {
			data, err := fetch(r.entity)
			a.post(func() {
				s.loading = false
				if err != nil || data == nil {
					s.err = "专辑加载失败"
					return
				}
				s.data, s.err = data, ""
			})
		}()
	case pageUser:
		if a.catalog.users == nil {
			a.catalog.users = make(map[int64]*catalogUser)
		}
		s := a.catalog.users[r.entity]
		if s == nil {
			s = &catalogUser{}
			a.catalog.users[r.entity] = s
		}
		a.loadUserPlaylists(r.entity)
		if s.data != nil || s.loading {
			return
		}
		s.loading, s.err = true, ""
		fetch := fetchUserDetail
		go func() {
			data, err := fetch(r.entity)
			a.post(func() {
				s.loading = false
				if err != nil || data == nil {
					s.err = "用户资料加载失败"
					return
				}
				s.data, s.err = data, ""
			})
		}()
	}
}

func (s *catalogArtist) complete(data *netease.ArtistPage, err error) {
	s.loading = false
	if err != nil || data == nil {
		s.err = "歌手资料加载失败"
		return
	}
	s.data, s.err = data, ""
}

func (a *app) loadArtistAlbums(id int64) {
	s := a.catalog.artists[id]
	if s == nil || s.albumsLoading || (s.albumsLoaded && !s.albumsMore) {
		return
	}
	s.albumsLoading, s.albumsErr = true, ""
	offset, fetch := s.albumsOffset, fetchArtistAlbums
	go func() {
		albums, err := fetch(id, artistAlbumLimit, offset)
		a.post(func() { s.completeAlbums(albums, err) })
	}()
}

// A failed page keeps its offset and existing rows so retry requests that page,
// not the first one; the raw count advances the offset even if IDs overlap.
func (s *catalogArtist) completeAlbums(albums []netease.Album, err error) {
	s.albumsLoading = false
	if err != nil {
		s.albumsErr = "专辑列表加载失败"
		return
	}
	s.albumsLoaded, s.albumsErr = true, ""
	s.albumsMore = len(albums) == artistAlbumLimit
	s.albumsOffset += len(albums)
	seen := make(map[int64]bool, len(s.albums))
	for _, album := range s.albums {
		seen[album.ID] = true
	}
	for _, album := range albums {
		if !seen[album.ID] {
			s.albums = append(s.albums, album)
			seen[album.ID] = true
		}
	}
}

func (a *app) loadUserPlaylists(id int64) {
	s := a.catalog.users[id]
	if s == nil || s.playlistsLoading || s.playlistsLoaded {
		return
	}
	s.playlistsLoading, s.playlistsErr = true, ""
	fetch := fetchUserPlaylists
	go func() {
		made, saved, err := fetch(id)
		a.post(func() { s.completePlaylists(id, made, saved, err) })
	}()
}

func (s *catalogUser) completePlaylists(id int64, made, saved []netease.Playlist, err error) {
	s.playlistsLoading = false
	if err != nil {
		s.playlistsErr = "公开歌单加载失败"
		return
	}
	s.playlists, s.playlistsLoaded, s.playlistsErr = nil, true, ""
	// The API's made/saved split describes subscription, not ownership.
	for _, lists := range [][]netease.Playlist{made, saved} {
		for _, pl := range lists {
			if pl.Creator.UserID == id && pl.Privacy == 0 {
				s.playlists = append(s.playlists, pl)
			}
		}
	}
}
