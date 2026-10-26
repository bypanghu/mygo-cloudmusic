package app

import (
	"github.com/egoist/mygo/ui"

	"netease-music-mygo/internal/netease"
)

func (a *app) nowPlayingCredits(c *ui.Context, tr *netease.Track, album bool) {
	ui.Column(c).Gap(4).Shrink(1).MinWidth(0).Children(func() {
		if tr.ProgramID > 0 {
			ui.Text(c, tr.ArtistNames()).FontSize(14).TextColor(white.Alpha(0.7)).SingleLine().Shrink(1).MinWidth(0)
			if album && tr.Album.Name != "" {
				ui.Text(c, tr.Album.Name).FontSize(13).TextColor(white.Alpha(0.55)).SingleLine().Shrink(1).MinWidth(0)
			}
			return
		}
		ui.Row(c).Gap(5).Shrink(1).MinWidth(0).Children(func() {
			for i, artist := range tr.Artists {
				if i > 0 {
					ui.Text(c, "/").FontSize(13).TextColor(white.Alpha(0.6))
				}
				id := artist.ID
				b := ui.ButtonBase(c.Key(i)).Shrink(1).MinWidth(0).Justify(ui.Start).
					Label("查看歌手：" + artist.Name).Disabled(id <= 0).
					OnClick(func() {
						a.nowPlaying = false
						a.open(route{page: pageArtist, entity: id})
					})
				fg := white.Alpha(0.7)
				if b.Hovered() {
					fg = white
				}
				b.Children(func() {
					ui.Text(c, artist.Name).FontSize(14).TextColor(fg).SingleLine().Shrink(1).MinWidth(0)
				})
			}
		})
		if album && tr.Album.Name != "" {
			id := tr.Album.ID
			ui.ButtonBase(c.Key("album-credit")).Shrink(1).MinWidth(0).Justify(ui.Start).
				Label("查看专辑：" + tr.Album.Name).Disabled(id <= 0).
				OnClick(func() {
					a.nowPlaying = false
					a.open(route{page: pageAlbum, entity: id})
				}).Children(func() {
				ui.Text(c, tr.Album.Name).FontSize(13).TextColor(white.Alpha(0.55)).SingleLine().Shrink(1).MinWidth(0)
			})
		}
	})
}

// Artist credits navigate independently of the song row's playback action.
func (a *app) artistLinks(c *ui.Context, artists []netease.Artist) {
	const limit = 5
	ui.Row(c).Gap(4).Shrink(1).MinWidth(0).AlignItems(ui.Center).Children(func() {
		for i, artist := range artists[:min(len(artists), limit)] {
			if i > 0 {
				ui.Text(c, "/").FontSize(12).TextColor(a.pal.muted)
			}
			id := artist.ID
			b := ui.ButtonBase(c.Key(i)).Shrink(1).MinWidth(0).Justify(ui.Start).
				Label("查看歌手：" + artist.Name).Disabled(id <= 0).
				OnClick(func() { a.open(route{page: pageArtist, entity: id}) })
			color := a.pal.muted
			if b.Hovered() {
				color = a.pal.accent
			}
			b.Children(func() {
				ui.Text(c, artist.Name).FontSize(12.5).TextColor(color).SingleLine().Shrink(1).MinWidth(0)
			})
		}
		if len(artists) > limit {
			ui.Text(c, "...").FontSize(12.5).TextColor(a.pal.muted).SingleLine()
		}
	})
}
