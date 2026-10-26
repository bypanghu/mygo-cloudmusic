package app

import "strings"

const searchHistoryLimit = 20

// rememberSearch keeps only submitted keywords, with a repeated one moved to the front.
func (a *app) rememberSearch(q string) {
	q = strings.TrimSpace(q)
	if q == "" {
		return
	}
	history := []string{q}
	for _, previous := range a.settings.SearchHistory {
		if previous != q {
			history = append(history, previous)
			if len(history) == searchHistoryLimit {
				break
			}
		}
	}
	a.settings.SearchHistory = history
	if err := a.settings.save(); err != nil {
		a.toast = "搜索历史保存失败"
	}
}

func (a *app) clearSearchHistory() {
	previous := a.settings.SearchHistory
	a.settings.SearchHistory = []string{}
	if err := a.settings.save(); err != nil {
		a.settings.SearchHistory = previous
		a.toast = "清空搜索历史失败，请重试"
	}
}
