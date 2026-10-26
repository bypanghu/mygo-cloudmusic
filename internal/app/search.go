package app

import (
	"strings"

	"netease-music-mygo/internal/netease"
)

const searchLimit = 60

var searchCatalog = netease.SearchPage

type searchState struct {
	last        route
	entries     map[route]*searchEntry
	serial      uint64
	suggestions suggestionState
}

type searchEntry struct {
	result  netease.SearchResult
	loading bool
	err     string
	request uint64
}

func searchKind(kind netease.SearchType) netease.SearchType {
	switch kind {
	case netease.SearchArtists, netease.SearchAlbums, netease.SearchUsers, netease.SearchPodcasts:
		return kind
	default:
		return netease.SearchSongs
	}
}

func (a *app) search(q string) {
	q = strings.TrimSpace(q)
	if q == "" {
		return
	}
	a.rememberSearch(q)
	a.dismissSuggestions()
	r := route{page: pageSearch, query: q, kind: searchKind(a.searches.last.kind)}
	if r == a.route {
		a.nowPlaying = false
		a.loadSearch(r, true, false)
		return
	}
	a.open(r)
}

func (a *app) chooseSearch(kind netease.SearchType) {
	if a.route.page != pageSearch || a.route.kind == kind {
		return
	}
	r := a.route
	r.kind = kind
	a.show(r)
}

func (a *app) searchEntry() *searchEntry {
	return a.searches.entries[a.route]
}

func searchCount(r netease.SearchResult, kind netease.SearchType) int {
	switch kind {
	case netease.SearchArtists:
		return len(r.Artists)
	case netease.SearchAlbums:
		return len(r.Albums)
	case netease.SearchUsers:
		return len(r.Users)
	case netease.SearchPodcasts:
		return len(r.Podcasts)
	default:
		return len(r.Songs)
	}
}

func (a *app) loadSearch(r route, refresh, more bool) {
	if a.searches.entries == nil {
		a.searches.entries = make(map[route]*searchEntry)
	}
	e := a.searches.entries[r]
	if e != nil && !refresh && (!more || e.loading || !e.result.More) {
		return
	}
	if e == nil || refresh {
		e = &searchEntry{}
		a.searches.entries[r] = e
	}
	offset := 0
	if more {
		offset = searchCount(e.result, r.kind)
	}
	a.searches.serial++
	e.request, e.loading, e.err = a.searches.serial, true, ""
	request, fetch := e.request, searchCatalog
	go func() {
		res, err := fetch(r.query, r.kind, searchLimit, offset)
		a.post(func() { a.finishSearch(r, request, offset, res, err) })
	}()
}

// finishSearch rejects an older request, even if its query and category match.
func (a *app) finishSearch(r route, request uint64, offset int, res netease.SearchResult, err error) {
	e := a.searches.entries[r]
	if e == nil || e.request != request {
		return
	}
	e.loading = false
	if err != nil {
		e.err = "搜索失败，请检查网络后重试"
		return
	}
	if offset > 0 {
		res.Songs = append(e.result.Songs, res.Songs...)
		res.Artists = append(e.result.Artists, res.Artists...)
		res.Albums = append(e.result.Albums, res.Albums...)
		res.Users = append(e.result.Users, res.Users...)
		res.Podcasts = append(e.result.Podcasts, res.Podcasts...)
		// A server can claim more while returning an empty page.
		if searchCount(res, r.kind) == offset {
			res.More = false
		}
	}
	e.result, e.err = res, ""
}

// Signing in changes song privileges, including results kept in history.
func (a *app) refreshSearch() {
	a.searches.entries = nil
	if a.searches.last.query != "" {
		a.loadSearch(a.searches.last, true, false)
	}
}
