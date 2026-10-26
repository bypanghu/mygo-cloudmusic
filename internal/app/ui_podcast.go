package app

import (
	"fmt"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"

	"netease-music-mygo/internal/netease"
)

func (a *app) podcastRow(c *ui.Context, podcast *netease.Podcast) {
	id := podcast.ID
	row := ui.ButtonBase(c).Gap(16).Padding(12).Radius(10).AlignItems(ui.Center).Label(podcast.Name).
		OnClick(func() { a.open(route{page: pagePodcast, entity: id}) })
	if row.Hovered() {
		row.Background(a.pal.hover)
	}
	row.Children(func() {
		a.cover(c, podcast.PicURL, 180, 72, 8)
		ui.Column(c).Grow(1).Shrink(1).MinWidth(0).Gap(5).Children(func() {
			ui.Text(c, podcast.Name).FontSize(15).FontWeight(600).SingleLine()
			var meta []string
			if podcast.DJ.Nickname != "" {
				meta = append(meta, podcast.DJ.Nickname)
			}
			if podcast.Category != "" {
				meta = append(meta, podcast.Category)
			}
			meta = append(meta, fmt.Sprintf("%d 期节目", podcast.ProgramCount))
			ui.Text(c, strings.Join(meta, " · ")).FontSize(12.5).TextColor(a.pal.muted).SingleLine()
			if podcast.Description != "" {
				ui.Text(c, podcast.Description).FontSize(12.5).TextColor(a.pal.muted).SingleLine()
			}
		})
		ui.Icon(c, iconForward).Size(16, 16).TextColor(a.pal.muted)
	})
}

func (a *app) programPane(c *ui.Context, tr *netease.Track) {
	ui.Scroll(c.Key(tr.ProgramID)).Grow(1).Padding(8, 12, 32, 0).Gap(16).Children(func() {
		ui.Text(c, tr.Name).FontSize(24).FontWeight(700).MaxLines(3)
		text := strings.TrimSpace(tr.Description)
		if text == "" {
			text = "暂无节目简介"
		}
		ui.Text(c, text).FontSize(15).LineHeight(1.6).TextColor(white.Alpha(0.75))
		if tr.PodcastID > 0 {
			id := tr.PodcastID
			pillButton(c, iconRadio, "查看播客", white.Alpha(0.12), white).OnClick(func() {
				a.nowPlaying = false
				a.open(route{page: pagePodcast, entity: id})
			})
		}
	})
}

func (a *app) podcastPage(c *ui.Context) {
	id := a.route.entity
	s := a.catalog.podcasts[id]
	if s == nil || s.data == nil {
		message, loading := "播客不可用", false
		if s != nil {
			message, loading = s.err, s.loading
		}
		a.entityStatus(c, message, loading, func() { a.loadPodcast(id) })
		return
	}
	a.entityScroll(c, func() {
		podcast := s.data
		ui.Row(c).Gap(24).AlignItems(ui.Center).Children(func() {
			a.cover(c, podcast.PicURL, 400, 144, 8)
			ui.Column(c).Grow(1).Shrink(1).MinWidth(0).Gap(8).Children(func() {
				ui.Text(c, podcast.Name).FontSize(28).FontWeight(700).MaxLines(2)
				if podcast.DJ.Nickname != "" {
					uid := podcast.DJ.UserID
					ui.ButtonBase(c).Label("主播：" + podcast.DJ.Nickname).Disabled(uid <= 0).
						OnClick(func() { a.open(route{page: pageUser, entity: uid}) }).Children(func() {
						ui.Text(c, podcast.DJ.Nickname).FontSize(14).TextColor(a.pal.muted)
					})
				}
				ui.Textf(c, "%d 期节目 · %s 人订阅", podcast.ProgramCount, count(int(podcast.SubCount))).FontSize(13).TextColor(a.pal.muted)
				ui.Row(c).Gap(10).Children(func() {
					pillButton(c.Key("play-podcast"), iconPlay, "播放节目", a.pal.accent, a.pal.onAccent).
						Disabled(len(s.programs) == 0).OnClick(func() {
						if len(s.programs) > 0 {
							a.playPodcastProgram(id, s.programs[0].ID)
						}
					})
					label, action, icon := "订阅播客", "订阅播客", iconPlus
					if podcast.Subed {
						label, action, icon = "已订阅", "取消订阅播客", iconCheck
					}
					pillButton(c.Key("subscribe-podcast"), icon, label, a.pal.hover, a.pal.text).
						Label(action).Disabled(s.subscribing).OnClick(func() { a.togglePodcastSubscription(id) })
				})
			})
		})
		if desc := strings.TrimSpace(podcast.Description); desc != "" {
			ui.Text(c, desc).FontSize(14).LineHeight(1.5).Margin(20, 0, 0, 0)
		}
		a.section(c, "节目")
		if s.playErr != "" {
			ui.Text(c, s.playErr).FontSize(13).TextColor(a.pal.accent).Margin(0, 0, 12, 0)
		}
		for _, program := range s.programs {
			pid := program.ID
			current := a.pl.Current()
			playing := current != nil && current.ProgramID == pid
			row := ui.ButtonBase(c.Key(pid)).Gap(14).Padding(10).Radius(8).AlignItems(ui.Center).
				Label("播放节目：" + program.Name).Disabled(s.resolving[pid]).OnClick(func() {
				a.playPodcastProgram(id, pid)
			})
			if playing {
				row.Background(a.pal.accent.Alpha(0.08))
			} else if row.Hovered() {
				row.Background(a.pal.hover)
			}
			row.Children(func() {
				cover := program.CoverURL
				if cover == "" {
					cover = podcast.PicURL
				}
				a.cover(c, cover, 120, 48, 6)
				ui.Column(c).Grow(1).Shrink(1).MinWidth(0).Gap(4).Children(func() {
					ui.Text(c, program.Name).FontSize(14).FontWeight(500).SingleLine()
					if program.CreateTime > 0 {
						ui.Text(c, time.UnixMilli(program.CreateTime).Format("2006-01-02")).FontSize(12).TextColor(a.pal.muted)
					}
				})
				if s.resolving[pid] {
					ui.Text(c, "正在加载…").FontSize(12).TextColor(a.pal.muted)
				} else {
					ui.Text(c, clock(time.Duration(program.Duration)*time.Millisecond)).FontSize(12).FontFeatures("tnum").TextColor(a.pal.muted)
				}
				if playing {
					a.nowPlayingBars(c)
				} else {
					ui.Icon(c, iconPlay).Size(14, 14).TextColor(a.pal.muted)
				}
			})
		}
		ui.Column(c).Gap(12).PaddingY(20).AlignItems(ui.Start).Children(func() {
			switch {
			case s.programsLoading:
				ui.Text(c, "正在加载节目…").FontSize(13).TextColor(a.pal.muted)
			case s.programsErr != "":
				ui.Text(c, s.programsErr).FontSize(13).TextColor(a.pal.muted)
				ui.Button(c, "重试加载节目").OnClick(func() { a.loadPodcastEpisodes(id) })
			case s.more:
				ui.Button(c, "加载更多节目").OnClick(func() { a.loadPodcastEpisodes(id) })
			case len(s.programs) == 0:
				ui.Text(c, "暂无节目").FontSize(13).TextColor(a.pal.muted)
			}
		})
	})
}
