package app

import "netease-music-mygo/internal/netease"

var fetchArtistEvents = netease.ArtistEvents

const artistEventLimit = 20

type artistFeedState struct {
	items           []netease.ArtistEvent
	loading, loaded bool
	err             string
	cursor          int64
	more            bool
	userID          int64
}

func (a *app) refreshArtistEvents(id int64) {
	artist := a.catalog.artists[id]
	if artist == nil || artist.feed.loading {
		return
	}
	artist.feed = artistFeedState{}
	a.loadArtistEvents(id)
}

func (a *app) loadArtistEvents(id int64) {
	artist := a.catalog.artists[id]
	if artist == nil || id <= 0 {
		return
	}
	s := &artist.feed
	if s.loading || (s.loaded && !s.more) {
		return
	}
	cursor := s.cursor
	if !s.loaded {
		cursor = -1
	}
	s.loading, s.err = true, ""
	fetch := fetchArtistEvents
	go func() {
		page, err := fetch(id, artistEventLimit, cursor)
		// Publish to the captured cache, not whichever artist is currently visible.
		a.post(func() { s.complete(page, cursor, err) })
	}()
}

func (s *artistFeedState) complete(page netease.ArtistEventPage, requested int64, err error) {
	s.loading = false
	if err != nil {
		s.err = "动态加载失败，请重试"
		return
	}
	s.loaded, s.err, s.userID = true, "", page.UserID
	seen := make(map[int64]bool, len(s.items))
	for _, item := range s.items {
		seen[item.ID] = true
	}
	for _, item := range page.Events {
		if !seen[item.ID] {
			s.items = append(s.items, item)
			seen[item.ID] = true
		}
	}
	// Use the server's cursor rather than the last visible (possibly duplicate)
	// row. A stationary or backwards paging cursor must not create an endless loop.
	advances := page.LastTime > 0 && (requested == -1 || page.LastTime < requested)
	s.cursor = page.LastTime
	s.more = page.More && len(page.Events) > 0 && advances
}
