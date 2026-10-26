package app

import (
	"strings"

	"github.com/egoist/mygo/ui"
)

func (a *app) searchField(c *ui.Context) {
	p := a.pal
	field := ui.Row(c).Height(32).PaddingX(10).Gap(7).Radius(9).AlignItems(ui.Center).Margin(0, 0, 14, 0)
	if c.Focused(a.searchBox) {
		field.Background(p.bg).Shadow(0, 0, 0, 3, p.accent.Alpha(0.25)).Border(1, p.accent.Alpha(0.6))
	} else {
		field.Background(p.hover).Border(1, p.hairline)
	}
	field.Children(func() {
		ui.Icon(c, iconSearch).Size(14, 14).TextColor(p.muted)
		combo := ui.ComboboxBase(c.Key("search"), &a.query)
		in := combo.Input.Bind(&a.searchBox).Grow(1).MinWidth(0).
			Placeholder("歌曲、歌手、专辑、播客…").FontSize(13).Label("搜索")
		q := strings.TrimSpace(a.query)
		s := &a.searches.suggestions
		items := s.items
		if q != s.query {
			items = nil
		}
		if in.Focused() && combo.Open() && q != "" {
			a.suggest(q)
		} else {
			if s.query != "" {
				a.dismissSuggestions()
			}
			if !in.Focused() || len(a.settings.SearchHistory) == 0 {
				combo.SetOpen(false)
			}
		}
		combo.Popup(func(panel ui.Element) {
			panel.AttachTo(field, ui.AnchorBottomStart, ui.AnchorTopStart).
				Width(360).Padding(6).Margin(8, 0, 0, 0).Radius(12).
				Background(p.bg).Border(1, p.hairline).
				Shadow(0, 8, 24, -4, ui.RGBA(0, 0, 0, 0.2)).Label("搜索建议")
			if q == "" {
				panel.Label("搜索历史")
				ui.Row(c).Padding(2, 10).AlignItems(ui.Center).Children(func() {
					ui.Text(c, "搜索历史").FontSize(12).TextColor(p.muted).Grow(1)
					iconButton(c.Key("clear-history"), iconTrash, 14, p.muted, "清空搜索历史").
						KeepFocus().OnClick(a.clearSearchHistory)
				})
			}
			_, height := c.Size()
			ui.Scroll(c).MaxHeight(min(480, height-140)).Gap(2).Children(func() {
				if q == "" {
					for _, keyword := range a.settings.SearchHistory {
						row := combo.Item("history:" + keyword).
							Height(38).PaddingX(10).Gap(8).Radius(7).AlignItems(ui.Center).
							Label("搜索历史：" + keyword)
						if row.Highlighted() {
							row.Background(p.hover)
						}
						row.Children(func() {
							ui.Icon(c, iconSearch).Size(14, 14).TextColor(p.muted)
							ui.Text(c, keyword).FontSize(13).SingleLine().Grow(1).Shrink(1).MinWidth(0)
						})
					}
					return
				}
				// The typed keyword remains the first option: Enter searches it
				// unchanged; Down reaches the server's suggested matches.
				row := combo.Item("query").Height(38).PaddingX(10).Gap(8).Radius(7).AlignItems(ui.Center).
					Label("搜索：" + q)
				if row.Highlighted() {
					row.Background(p.hover)
				}
				row.Children(func() {
					ui.Icon(c, iconSearch).Size(14, 14).TextColor(p.muted)
					ui.Text(c, "搜索「"+q+"」").FontSize(13).FontWeight(500).SingleLine().Grow(1).Shrink(1).MinWidth(0)
				})
				if s.loading {
					ui.Text(c, "正在获取联想…").FontSize(12).TextColor(p.muted).Padding(6, 10)
				} else if s.failed {
					ui.Text(c, "联想暂不可用，仍可搜索关键词").FontSize(12).TextColor(p.muted).Padding(6, 10)
				}
				for _, suggestion := range items {
					item := combo.Item(suggestion.key).Padding(8, 10).Gap(12).Radius(7).AlignItems(ui.Center).
						Label(suggestion.category + "：" + suggestion.name)
					if item.Highlighted() {
						item.Background(p.hover)
					}
					item.Children(func() {
						ui.Column(c).Grow(1).Shrink(1).MinWidth(0).Gap(3).Children(func() {
							ui.Text(c, suggestion.name).FontSize(13).FontWeight(500).SingleLine()
							if suggestion.detail != "" {
								ui.Text(c, suggestion.detail).FontSize(12).TextColor(p.muted).SingleLine()
							}
						})
						ui.Text(c, suggestion.category).FontSize(12).TextColor(p.muted).SingleLine()
					})
				}
			})
		})
		if chosen, ok := combo.Chosen(); ok {
			combo.SetOpen(false)
			if chosen == "query" {
				a.search(q)
			} else if keyword, ok := strings.CutPrefix(chosen, "history:"); ok {
				a.query = keyword
				a.search(keyword)
			} else {
				for _, suggestion := range items {
					if suggestion.key == chosen {
						a.chooseSuggestion(suggestion)
						break
					}
				}
			}
		} else if in.Submitted() && q != "" {
			combo.SetOpen(false)
			a.search(q)
		}
	})
}
