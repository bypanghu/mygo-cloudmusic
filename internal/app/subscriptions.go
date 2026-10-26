package app

import "netease-music-mygo/internal/netease"

var (
	subscribePodcast = netease.SubscribePodcast
	followArtist     = netease.FollowArtist
)

func (a *app) togglePodcastSubscription(id int64) {
	s := a.catalog.podcasts[id]
	if id <= 0 || s == nil || s.data == nil || s.subscribing {
		return
	}
	if a.profile == nil {
		a.showMain()
		a.openLogin()
		return
	}
	podcast := s.data
	subscribe, previousCount := !podcast.Subed, podcast.SubCount
	podcast.Subed, s.subscribing = subscribe, true
	if subscribe {
		podcast.SubCount++
	} else {
		podcast.SubCount = max(0, podcast.SubCount-1)
	}
	profile, send := a.profile, subscribePodcast
	go func() {
		err := send(id, subscribe)
		a.post(func() { a.completePodcastSubscription(id, s, profile, subscribe, previousCount, err) })
	}()
}

func (a *app) completePodcastSubscription(id int64, s *catalogPodcast, profile *netease.Profile, subscribe bool, previousCount int64, err error) {
	// An old account's request must not change the new account's cache or toast.
	if a.profile != profile || a.catalog.podcasts[id] != s {
		return
	}
	s.subscribing = false
	if err != nil {
		s.data.Subed, s.data.SubCount = !subscribe, previousCount
		a.toast = "订阅操作失败，请稍后再试"
		return
	}
	if subscribe {
		a.toast = "已订阅播客"
	} else {
		a.toast = "已取消订阅播客"
	}
}

func (a *app) toggleArtistFollow(id int64) {
	s := a.catalog.artists[id]
	if id <= 0 || s == nil || s.data == nil || s.following {
		return
	}
	if a.profile == nil {
		a.showMain()
		a.openLogin()
		return
	}
	follow := !s.data.Artist.Followed
	s.data.Artist.Followed, s.following = follow, true
	profile, send := a.profile, followArtist
	go func() {
		err := send(id, follow)
		a.post(func() { a.completeArtistFollow(id, s, profile, follow, err) })
	}()
}

func (a *app) completeArtistFollow(id int64, s *catalogArtist, profile *netease.Profile, follow bool, err error) {
	if a.profile != profile || a.catalog.artists[id] != s {
		return
	}
	s.following = false
	if err != nil {
		s.data.Artist.Followed = !follow
		a.toast = "关注操作失败，请稍后再试"
		return
	}
	if follow {
		a.toast = "已关注音乐人"
	} else {
		a.toast = "已取消关注音乐人"
	}
}
