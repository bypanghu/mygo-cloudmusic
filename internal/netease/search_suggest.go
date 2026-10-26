package netease

import "net/url"

// SearchSuggestions are the matches returned while a keyword is being typed.
// The web endpoint suggests songs, artists, albums and playlists, not users.
type SearchSuggestions struct {
	Songs     []Track    `json:"songs"`
	Artists   []Artist   `json:"artists"`
	Albums    []Album    `json:"albums"`
	Playlists []Playlist `json:"playlists"`
	Order     []string   `json:"order"`
}

func SearchSuggest(query string) (SearchSuggestions, error) {
	var res struct {
		Result SearchSuggestions `json:"result"`
	}
	err := get("/api/search/suggest/web", url.Values{"s": {query}}, &res)
	return res.Result, err
}
