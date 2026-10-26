package app

import (
	"fmt"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"netease-music-mygo/internal/netease"
)

func artistPicture(artist *netease.Artist) string {
	if artist.Img1v1URL != "" {
		return artist.Img1v1URL
	}
	return artist.PicURL
}

func (a *app) artistCard(c *ui.Context, artist *netease.Artist) {
	id := artist.ID
	card := ui.ButtonBase(c).Column().Gap(10).AlignItems(ui.Stretch).Padding(8).Radius(10).Label(artist.Name).
		OnClick(func() { a.open(route{page: pageArtist, entity: id}) })
	if card.Hovered() {
		card.Background(a.pal.hover)
	}
	card.Children(func() {
		ui.Box(c).AspectRatio(1).Children(func() { a.coverFill(c, artistPicture(artist), 400, 999) })
		ui.Text(c, artist.Name).FontSize(14).FontWeight(500).SingleLine()
		if len(artist.Alias) > 0 {
			ui.Text(c, strings.Join(artist.Alias, " / ")).FontSize(12).TextColor(a.pal.muted).SingleLine()
		}
	})
}

func (a *app) albumCard(c *ui.Context, album *netease.Album) {
	id := album.ID
	card := ui.ButtonBase(c).Column().Gap(8).AlignItems(ui.Stretch).Padding(8).Radius(10).Label(album.Name).
		OnClick(func() { a.open(route{page: pageAlbum, entity: id}) })
	if card.Hovered() {
		card.Background(a.pal.hover)
	}
	card.Children(func() {
		ui.Box(c).AspectRatio(1).Children(func() { a.coverFill(c, album.PicURL, 400, 8) })
		ui.Text(c, album.Name).FontSize(14).FontWeight(500).MaxLines(2)
		var meta []string
		if album.Artist.Name != "" {
			meta = append(meta, album.Artist.Name)
		}
		if album.PublishTime > 0 {
			meta = append(meta, time.UnixMilli(album.PublishTime).Format("2006"))
		}
		if len(meta) > 0 {
			ui.Text(c, strings.Join(meta, " · ")).FontSize(12).TextColor(a.pal.muted).SingleLine()
		}
	})
}

func (a *app) userRow(c *ui.Context, user *netease.User) {
	id := user.UserID
	row := ui.ButtonBase(c).Padding(12).Gap(14).Radius(10).AlignItems(ui.Center).Label(user.Nickname).
		OnClick(func() { a.open(route{page: pageUser, entity: id}) })
	if row.Hovered() {
		row.Background(a.pal.hover)
	}
	row.Children(func() {
		a.cover(c, user.AvatarURL, 120, 48, 24)
		ui.Column(c).Grow(1).Shrink(1).MinWidth(0).Gap(4).Children(func() {
			ui.Text(c, user.Nickname).FontSize(14).FontWeight(500).SingleLine()
			if user.Signature != "" {
				ui.Text(c, user.Signature).FontSize(12).TextColor(a.pal.muted).SingleLine()
			}
		})
	})
}

func (a *app) artistPage(c *ui.Context) {
	id := a.route.entity
	s := a.catalog.artists[id]
	if s == nil || s.data == nil {
		msg, loading := "歌手资料不可用", false
		if s != nil {
			msg, loading = s.err, s.loading
		}
		a.entityStatus(c, msg, loading, func() { a.loadEntity(route{page: pageArtist, entity: id}) })
		return
	}
	if a.catalog.tab == 0 {
		a.trackRows(c, s.data.Songs, true, func(c *ui.Context) {
			a.artistHeader(c, &s.data.Artist)
			if len(s.data.Songs) == 0 {
				ui.Text(c, "暂无热门歌曲").FontSize(14).TextColor(a.pal.muted)
				a.endMark(c)
			}
		})
		return
	}
	a.entityScroll(c, func() {
		a.artistHeader(c, &s.data.Artist)
		switch a.catalog.tab {
		case 1:
			if len(s.albums) > 0 {
				ui.Grid(c).Columns(a.columns(c, 168)).Gap(20).Children(func() {
					for i := range s.albums {
						a.albumCard(c.Key(s.albums[i].ID), &s.albums[i])
					}
				})
			}
			ui.Column(c).Gap(12).Margin(20, 0, 0, 0).Children(func() {
				switch {
				case s.albumsLoading:
					ui.Text(c, "正在加载专辑…").FontSize(14).TextColor(a.pal.muted)
				case s.albumsErr != "":
					ui.Text(c, s.albumsErr).FontSize(14).TextColor(a.pal.muted)
					ui.Button(c, "重试加载专辑").OnClick(func() { a.loadArtistAlbums(id) })
				case !s.albumsLoaded || s.albumsMore:
					ui.Button(c, "加载更多专辑").OnClick(func() { a.loadArtistAlbums(id) })
				case len(s.albums) == 0:
					ui.Text(c, "暂无专辑").FontSize(14).TextColor(a.pal.muted)
				}
			})
		case 2:
			desc := strings.TrimSpace(s.data.Artist.BriefDesc)
			if desc == "" {
				desc = "暂无歌手简介"
			}
			ui.Text(c, desc).FontSize(14).LineHeight(1.6)
		case 3:
			a.artistEvents(c, id)
		}
	})
}

func (a *app) artistHeader(c *ui.Context, artist *netease.Artist) {
	ui.Row(c).Gap(24).AlignItems(ui.Center).Margin(0, 0, 24, 0).Children(func() {
		a.cover(c, artistPicture(artist), 400, 144, 72)
		ui.Column(c).Grow(1).Shrink(1).MinWidth(0).Gap(8).Children(func() {
			ui.Text(c, artist.Name).FontSize(28).FontWeight(700).MaxLines(2)
			if len(artist.Alias) > 0 {
				ui.Text(c, strings.Join(artist.Alias, " / ")).FontSize(14).TextColor(a.pal.muted).MaxLines(2)
			}
			ui.Textf(c, "%d 首歌曲 · %d 张专辑", artist.MusicSize, artist.AlbumSize).FontSize(13).TextColor(a.pal.muted)
			id := a.route.entity
			label, action, icon := "关注音乐人", "关注音乐人", iconPlus
			if artist.Followed {
				label, action, icon = "已关注", "取消关注音乐人", iconCheck
			}
			s := a.catalog.artists[id]
			pillButton(c.Key("follow-artist"), icon, label, a.pal.hover, a.pal.text).
				Label(action).Disabled(s == nil || s.following).AlignSelf(ui.Start).
				OnClick(func() { a.toggleArtistFollow(id) })
		})
	})
	id := a.route.entity
	ui.Row(c).Gap(8).Margin(0, 0, 20, 0).Children(func() {
		for i, label := range []string{"热门歌曲", "专辑", "关于", "动态"} {
			button := ui.ButtonBase(c.Key(i)).Padding(8, 14).Radius(8).Label(label)
			if i == a.catalog.tab || button.Hovered() {
				button.Background(a.pal.hover)
			}
			button.Children(func() { ui.Text(c, label).FontSize(14).FontWeight(500) })
			button.OnClick(func() {
				a.catalog.tab = i
				a.trackList, a.pageScroll = ui.ListState{}, ui.ScrollState{}
				if i == 1 {
					s := a.catalog.artists[id]
					if s != nil && !s.albumsLoaded {
						a.loadArtistAlbums(id)
					}
				} else if i == 3 {
					s := a.catalog.artists[id]
					if s != nil && !s.feed.loaded {
						a.loadArtistEvents(id)
					}
				}
			})
		}
	})
}

func (a *app) albumPage(c *ui.Context) {
	id := a.route.entity
	s := a.catalog.albums[id]
	if s == nil || s.data == nil {
		msg, loading := "专辑不可用", false
		if s != nil {
			msg, loading = s.err, s.loading
		}
		a.entityStatus(c, msg, loading, func() { a.loadEntity(route{page: pageAlbum, entity: id}) })
		return
	}
	a.trackRows(c, s.data.Songs, false, func(c *ui.Context) {
		album := &s.data.Album
		ui.Row(c).Gap(24).AlignItems(ui.Center).Margin(0, 0, 24, 0).Children(func() {
			a.cover(c, album.PicURL, 500, 168, 8)
			ui.Column(c).Grow(1).Shrink(1).MinWidth(0).Gap(8).Children(func() {
				ui.Text(c, album.Name).FontSize(28).FontWeight(700).MaxLines(2)
				artists := album.Artists
				if len(artists) == 0 && album.Artist.Name != "" {
					artists = []netease.Artist{album.Artist}
				}
				for _, artist := range artists {
					id := artist.ID
					ui.ButtonBase(c.Key(id)).Label(artist.Name).Disabled(id <= 0).
						OnClick(func() { a.open(route{page: pageArtist, entity: id}) }).Children(func() {
						ui.Text(c, artist.Name).FontSize(14).TextColor(a.pal.muted)
					})
				}
				meta := fmt.Sprintf("%d 首歌曲", len(s.data.Songs))
				if album.PublishTime > 0 {
					meta += " · " + time.UnixMilli(album.PublishTime).Format("2006-01-02")
				}
				ui.Text(c, meta).FontSize(13).TextColor(a.pal.muted)
			})
		})
		if desc := strings.TrimSpace(album.Description); desc != "" {
			ui.Text(c, desc).FontSize(14).LineHeight(1.5).Margin(0, 0, 20, 0)
		}
		if len(s.data.Songs) == 0 {
			ui.Text(c, "暂无歌曲").FontSize(14).TextColor(a.pal.muted)
			a.endMark(c)
		}
	})
}

func (a *app) userPage(c *ui.Context) {
	id := a.route.entity
	s := a.catalog.users[id]
	if s == nil || s.data == nil {
		msg, loading := "用户资料不可用", false
		if s != nil {
			msg, loading = s.err, s.loading
		}
		a.entityStatus(c, msg, loading, func() { a.loadEntity(route{page: pageUser, entity: id}) })
		return
	}
	a.entityScroll(c, func() {
		user := s.data
		ui.Row(c).Gap(24).AlignItems(ui.Center).Children(func() {
			a.cover(c, user.AvatarURL, 400, 128, 64)
			ui.Column(c).Grow(1).Shrink(1).MinWidth(0).Gap(8).Children(func() {
				ui.Text(c, user.Nickname).FontSize(28).FontWeight(700).MaxLines(2)
				ui.Textf(c, "%d 关注 · %d 粉丝", user.Follows, user.Followeds).FontSize(13).TextColor(a.pal.muted)
				if user.Signature != "" {
					ui.Text(c, user.Signature).FontSize(14).LineHeight(1.5)
				}
			})
		})
		a.section(c, "创建的公开歌单")
		switch {
		case s.playlistsLoading:
			ui.Text(c, "正在加载歌单…").FontSize(14).TextColor(a.pal.muted)
		case s.playlistsErr != "":
			ui.Text(c, s.playlistsErr).FontSize(14).TextColor(a.pal.muted)
			ui.Button(c, "重试加载歌单").OnClick(func() { a.loadUserPlaylists(id) })
		case len(s.playlists) == 0:
			ui.Text(c, "暂无公开歌单").FontSize(14).TextColor(a.pal.muted)
		default:
			ui.Grid(c).Columns(a.columns(c, 168)).Gap(20).Children(func() {
				for _, playlist := range s.playlists {
					pid := playlist.ID
					card := ui.ButtonBase(c.Key(pid)).Column().Gap(8).Padding(8).Radius(10).AlignItems(ui.Stretch).Label(playlist.Name).
						OnClick(func() { a.open(route{page: pagePlaylist, playlist: pid}) })
					if card.Hovered() {
						card.Background(a.pal.hover)
					}
					card.Children(func() {
						ui.Box(c).AspectRatio(1).Children(func() { a.coverFill(c, playlist.Cover(), 400, 8) })
						ui.Text(c, playlist.Name).FontSize(14).FontWeight(500).MaxLines(2)
						ui.Textf(c, "%d 首", playlist.TrackCount).FontSize(12).TextColor(a.pal.muted)
					})
				}
			})
		}
	})
}

func (a *app) entityScroll(c *ui.Context, content func()) {
	top, bottom := a.insets(c)
	ui.Scroll(c.Key(a.route)).TrackScroll(&a.pageScroll).Grow(1).
		Padding(top+pageGap, pagePad, bottom+48, pagePad).ScrollbarInsets(top, 0, bottom, 0).Children(func() {
		a.startMark(c)
		defer a.endMark(c)
		content()
	})
}

func (a *app) entityStatus(c *ui.Context, message string, loading bool, retry func()) {
	if loading {
		a.centered(c, "正在加载…", nil)
		return
	}
	a.centered(c, message, retry)
}
