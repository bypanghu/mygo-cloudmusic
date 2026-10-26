package app

import (
	"fmt"

	"github.com/egoist/mygo/ui"

	"netease-music-mygo/internal/netease"
)

var searchKinds = []struct {
	kind netease.SearchType
	name string
	unit string
}{
	{netease.SearchSongs, "单曲", "首单曲"},
	{netease.SearchArtists, "歌手", "位歌手"},
	{netease.SearchAlbums, "专辑", "张专辑"},
	{netease.SearchUsers, "用户", "位用户"},
	{netease.SearchPodcasts, "播客", "个播客"},
}

func (a *app) searchHeader(c *ui.Context) {
	p, r, e := a.pal, a.route, a.searchEntry()
	ui.Column(c).Gap(8).Margin(0, 0, 20, 0).Children(func() {
		ui.Text(c, "搜索「"+r.query+"」").FontSize(28).FontWeight(700).LetterSpacing(-0.5).SingleLine()
		status := "正在搜索…"
		if e != nil && !e.loading {
			for _, k := range searchKinds {
				if k.kind == r.kind {
					status = fmt.Sprintf("找到 %d %s", e.result.Total, k.unit)
				}
			}
			if e.err != "" {
				status = "暂时无法获取搜索结果"
			}
		}
		ui.Text(c, status).FontSize(13).TextColor(p.muted)
		ui.Row(c).Gap(6).AlignItems(ui.Center).Margin(10, 0, 0, 0).Children(func() {
			for _, k := range searchKinds {
				selected := k.kind == r.kind
				b := ui.ButtonBase(c.Key(k.kind)).Height(36).PaddingX(16).Radius(8).Center().Label(k.name)
				fg, bg := p.muted, ui.Transparent
				if selected {
					fg, bg = p.accent, p.accent.Alpha(0.08)
				} else if b.Hovered() {
					fg, bg = p.text, p.hover
				}
				b.Background(bg).OnClick(func() { a.chooseSearch(k.kind) }).Children(func() {
					ui.Text(c, k.name).FontSize(14).FontWeight(600).TextColor(fg)
				})
			}
			ui.Spacer(c)
			iconButton(c.Key("refresh"), iconReload, 16, p.muted, "刷新搜索").
				Disabled(e != nil && e.loading).OnClick(func() { a.loadSearch(r, true, false) })
		})
	})
}

func (a *app) searchPage(c *ui.Context) {
	e := a.searchEntry()
	if e == nil {
		return
	}
	header := func(c *ui.Context) {
		ui.Column(c).PaddingX(12).Children(func() { a.searchHeader(c) })
	}
	count := searchCount(e.result, a.route.kind)
	if a.route.kind == netease.SearchSongs {
		if e.loading && count == 0 {
			a.trackSkeleton(c, a.pageShimmer(c), true, header)
			return
		}
		if count > 0 {
			a.trackRows(c, e.result.Songs, true, header, a.searchFooter)
			return
		}
	}
	top, bottom := a.insets(c)
	ui.Scroll(c.Key(a.route)).TrackScroll(&a.pageScroll).Grow(1).
		Padding(top+pageGap, pagePad, bottom+32, pagePad).ScrollbarInsets(top, 0, bottom, 0).Children(func() {
		a.startMark(c)
		defer a.endMark(c)
		a.searchHeader(c)
		if count == 0 {
			switch {
			case e.loading:
				s := a.pageShimmer(c)
				a.playlistCardsSkeleton(c, s, a.columns(c, 156)*2, 156, true)
			case e.err != "":
				ui.Column(c).PaddingY(40).Gap(14).AlignItems(ui.Start).Children(func() {
					ui.Text(c, e.err).FontSize(14).TextColor(a.pal.muted)
					r := a.route
					ui.Button(c, "重试").OnClick(func() { a.loadSearch(r, true, false) })
				})
			default:
				ui.Column(c).PaddingY(40).Gap(8).Children(func() {
					ui.Text(c, "没有找到相关结果").FontSize(17).FontWeight(600)
					ui.Text(c, "试试其他关键词，或切换搜索分类").FontSize(14).TextColor(a.pal.muted)
				})
			}
			return
		}
		switch a.route.kind {
		case netease.SearchArtists:
			ui.Grid(c).Columns(a.columns(c, 132)).GapX(20).GapY(28).Children(func() {
				for i := range e.result.Artists {
					a.artistCard(c.Key(e.result.Artists[i].ID), &e.result.Artists[i])
				}
			})
		case netease.SearchAlbums:
			ui.Grid(c).Columns(a.columns(c, 156)).GapX(20).GapY(28).Children(func() {
				for i := range e.result.Albums {
					a.albumCard(c.Key(e.result.Albums[i].ID), &e.result.Albums[i])
				}
			})
		case netease.SearchUsers:
			ui.Column(c).Gap(4).Children(func() {
				for i := range e.result.Users {
					a.userRow(c.Key(e.result.Users[i].UserID), &e.result.Users[i])
				}
			})
		case netease.SearchPodcasts:
			ui.Column(c).Gap(6).Children(func() {
				for i := range e.result.Podcasts {
					a.podcastRow(c.Key(e.result.Podcasts[i].ID), &e.result.Podcasts[i])
				}
			})
		}
		a.searchFooter(c)
	})
}

func (a *app) searchFooter(c *ui.Context) {
	e, r := a.searchEntry(), a.route
	if e == nil {
		return
	}
	ui.Column(c).Gap(12).PaddingY(24).AlignItems(ui.Center).Children(func() {
		ui.Textf(c, "已显示 %d / %d", searchCount(e.result, r.kind), e.result.Total).
			FontSize(12).FontFeatures("tnum").TextColor(a.pal.muted)
		switch {
		case e.loading:
			ui.Text(c, "正在加载…").FontSize(13).TextColor(a.pal.muted)
		case e.err != "":
			ui.Text(c, e.err).FontSize(13).TextColor(a.pal.muted)
			ui.Button(c, "重试加载").OnClick(func() { a.loadSearch(r, false, true) })
		case e.result.More:
			ui.Button(c, "加载更多").OnClick(func() { a.loadSearch(r, false, true) })
		}
	})
}
