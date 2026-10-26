package app

import (
	"fmt"
	"strings"
	"time"

	"netease-music-mygo/internal/netease"
)

var suggestSearch = netease.SearchSuggest

type searchSuggestion struct {
	key, name, detail, category string
	route                       route
}

type suggestionState struct {
	query      string
	generation uint64
	timer      *time.Timer
	loading    bool
	failed     bool
	items      []searchSuggestion
}

func (a *app) dismissSuggestions() {
	s := &a.searches.suggestions
	if s.timer != nil {
		s.timer.Stop()
		s.timer = nil
	}
	s.generation++
	s.query, s.items, s.loading, s.failed = "", nil, false, false
}

// Wait for a pause in typing, then ignore responses for replaced or dismissed input.
func (a *app) suggest(q string) {
	q = strings.TrimSpace(q)
	s := &a.searches.suggestions
	if q == s.query {
		return
	}
	a.dismissSuggestions()
	if q == "" {
		return
	}
	s.query, s.loading = q, true
	generation, fetch := s.generation, suggestSearch
	s.timer = time.AfterFunc(220*time.Millisecond, func() {
		result, err := fetch(q)
		a.post(func() {
			if generation != s.generation || q != s.query {
				return
			}
			s.timer, s.loading, s.failed = nil, false, err != nil
			if err == nil {
				s.items = suggestionItems(result)
			}
		})
	})
}

func suggestionItems(result netease.SearchSuggestions) []searchSuggestion {
	groups := make(map[string][]searchSuggestion)
	for _, tr := range result.Songs {
		groups["songs"] = append(groups["songs"], searchSuggestion{
			key: fmt.Sprintf("song:%d", tr.ID), name: tr.Name, detail: tr.ArtistNames(), category: "单曲",
			route: route{page: pageSearch, query: tr.Name, kind: netease.SearchSongs},
		})
	}
	for _, artist := range result.Artists {
		groups["artists"] = append(groups["artists"], searchSuggestion{
			key: fmt.Sprintf("artist:%d", artist.ID), name: artist.Name, category: "歌手",
			route: route{page: pageArtist, entity: artist.ID},
		})
	}
	for _, album := range result.Albums {
		groups["albums"] = append(groups["albums"], searchSuggestion{
			key: fmt.Sprintf("album:%d", album.ID), name: album.Name, detail: album.Artist.Name, category: "专辑",
			route: route{page: pageAlbum, entity: album.ID},
		})
	}
	for _, playlist := range result.Playlists {
		groups["playlists"] = append(groups["playlists"], searchSuggestion{
			key: fmt.Sprintf("playlist:%d", playlist.ID), name: playlist.Name, detail: playlist.Creator.Nickname, category: "歌单",
			route: route{page: pagePlaylist, playlist: playlist.ID},
		})
	}
	order := append(append([]string{}, result.Order...), "songs", "artists", "albums", "playlists")
	seen := make(map[string]bool)
	var items []searchSuggestion
	for _, group := range order {
		for _, item := range groups[group] {
			if item.name != "" && !seen[item.key] {
				items = append(items, item)
				seen[item.key] = true
				if len(items) == 10 {
					return items
				}
			}
		}
	}
	return items
}

func (a *app) chooseSuggestion(item searchSuggestion) {
	a.query = item.name
	a.rememberSearch(item.name)
	a.dismissSuggestions()
	a.nowPlaying = false
	if a.route == item.route && item.route.page == pageSearch {
		a.loadSearch(item.route, true, false)
		return
	}
	a.open(item.route)
}
