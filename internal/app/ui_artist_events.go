package app

import (
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"netease-music-mygo/internal/netease"
)

// artistEvents is the body below artistHeader, inside the existing entityScroll.
func (a *app) artistEvents(c *ui.Context, id int64) {
	artist := a.catalog.artists[id]
	if artist == nil {
		return
	}
	s := &artist.feed
	ui.Column(c.Key("artist-events")).Children(func() {
		ui.Row(c).Justify(ui.End).Margin(0, 0, 8, 0).Children(func() {
			iconButton(c, iconReload, 16, a.pal.muted, "刷新动态").Disabled(s.loading).
				OnClick(func() { a.refreshArtistEvents(id) })
		})
		for i := range s.items {
			a.artistEvent(c.Key(s.items[i].ID), &s.items[i])
		}
		ui.Column(c.Key("feed-status")).Gap(12).Padding(20, 0).Children(func() {
			switch {
			case s.loading:
				ui.Text(c, "正在加载动态…").FontSize(14).TextColor(a.pal.muted)
			case s.err != "":
				ui.Text(c, s.err).FontSize(14).TextColor(a.pal.muted)
				ui.Button(c, "重试加载动态").OnClick(func() { a.loadArtistEvents(id) })
			case !s.loaded:
				ui.Button(c, "加载动态").OnClick(func() { a.loadArtistEvents(id) })
			case len(s.items) == 0:
				text := "暂无公开动态"
				if s.userID <= 0 {
					text = "这位歌手尚未绑定公开账号，暂无动态可查看"
				}
				ui.Text(c, text).FontSize(14).TextColor(a.pal.muted)
			case s.more:
				ui.Button(c, "加载更多动态").OnClick(func() { a.loadArtistEvents(id) })
			default:
				ui.Text(c, "已显示全部动态").FontSize(13).TextColor(a.pal.muted)
			}
		})
	})
}

func (a *app) artistEvent(c *ui.Context, event *netease.ArtistEvent) {
	ui.Column(c).Gap(12).Padding(20, 0).BorderWidth(0, 0, 1, 0).BorderColor(a.pal.hairline).Children(func() {
		ui.Row(c).Gap(12).AlignItems(ui.Center).Children(func() {
			a.cover(c, event.User.AvatarURL, 120, 40, 20)
			ui.Column(c).Grow(1).Shrink(1).MinWidth(0).Gap(4).Children(func() {
				name := event.User.Nickname
				if name == "" {
					name = "歌手动态"
				}
				ui.Text(c, name).FontSize(14).FontWeight(500).SingleLine()
				if event.Time > 0 {
					ui.Text(c, time.UnixMilli(event.Time).Format("2006-01-02 15:04")).FontSize(13).TextColor(a.pal.muted)
				}
			})
		})
		content := &event.Content
		if text := strings.TrimSpace(content.Msg); text != "" {
			ui.Text(c, text).FontSize(14).LineHeight(1.6)
		}
		if content.ForwardedText != "" {
			ui.Box(c).Padding(12).Radius(8).Background(a.pal.hover).Children(func() {
				ui.Text(c, "转发 · "+content.ForwardedText).FontSize(13).LineHeight(1.5).TextColor(a.pal.muted)
			})
		}
		pictures := event.Pictures
		if len(pictures) > 0 {
			ui.Grid(c).Columns(3).Gap(8).Children(func() {
				for i, picture := range pictures {
					image := picture.PCSquareURL
					if image == "" {
						image = picture.SquareURL
					}
					if image == "" {
						image = picture.OriginURL
					}
					if image == "" {
						continue
					}
					ui.Box(c.Key(i)).AspectRatio(1).Children(func() { a.coverFill(c, image, 400, 8) })
				}
			})
		}
		if song := content.Song; song != nil {
			// There is no song detail route. Open its album, or a song-only search,
			// never the player: reading a shared song must not change the queue.
			target := route{page: pageSearch, query: strings.TrimSpace(song.Name + " " + song.ArtistNames()), kind: netease.SearchSongs}
			meta := "歌曲 · " + song.ArtistNames()
			if song.Album.ID > 0 {
				target = route{page: pageAlbum, entity: song.Album.ID}
				meta += " · 查看专辑"
			}
			a.artistEventShare(c.Key("song"), song.Name, meta, song.Album.PicURL, target)
		}
		if album := content.Album; album != nil {
			a.artistEventShare(c.Key("album"), album.Name, "专辑 · "+album.Artist.Name, album.PicURL, route{page: pageAlbum, entity: album.ID})
		}
		if playlist := content.Playlist; playlist != nil {
			a.artistEventShare(c.Key("playlist"), playlist.Name, "歌单", playlist.Cover(), route{page: pagePlaylist, playlist: playlist.ID})
		}
		if artist := content.Artist; artist != nil {
			a.artistEventShare(c.Key("artist"), artist.Name, "歌手", artistPicture(artist), route{page: pageArtist, entity: artist.ID})
		}
		if len(content.Video) > 0 && string(content.Video) != "null" || len(content.MV) > 0 && string(content.MV) != "null" {
			ui.Text(c, "分享了视频 · 暂不支持在应用内查看").FontSize(13).TextColor(a.pal.muted)
		}
		ui.Textf(c, "%d 赞 · %d 评论 · %d 转发", event.Info.LikedCount, event.Info.CommentCount, event.Info.ShareCount).
			FontSize(13).TextColor(a.pal.muted)
	})
}

func (a *app) artistEventShare(c *ui.Context, name, meta, image string, target route) {
	card := ui.ButtonBase(c).Gap(12).Padding(10).Radius(8).AlignItems(ui.Center).Label(name + "，" + meta).
		OnClick(func() { a.open(target) })
	if card.Hovered() {
		card.Background(a.pal.hover)
	}
	card.Children(func() {
		a.cover(c, image, 120, 48, 6)
		ui.Column(c).Grow(1).Shrink(1).MinWidth(0).Gap(4).Children(func() {
			ui.Text(c, name).FontSize(14).FontWeight(500).MaxLines(2)
			ui.Text(c, meta).FontSize(13).TextColor(a.pal.muted).SingleLine()
		})
	})
}
